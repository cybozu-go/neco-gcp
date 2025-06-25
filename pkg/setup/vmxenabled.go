package setup

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

const (
	assetDir = "/assets"
	ctPath   = "/usr/local/bin/ct"
)

var (
	staticFiles = []string{
		"/etc/apt/apt.conf.d/20auto-upgrades",
		"/etc/containers/registries.conf",
		"/etc/docker/daemon.json",
		"/etc/profile.d/go.sh",
	}
)

// VMXEnabled setup vmx-enabled instance
func VMXEnabled(ctx context.Context, project string, artifacts *ArtifactSet, optionalPackages []string) error {
	err := configureDNS(ctx)
	if err != nil {
		return err
	}

	err = configureApt(ctx)
	if err != nil {
		return err
	}

	err = apt(ctx, "install", "-y", "software-properties-common", "dirmngr")
	if err != nil {
		return err
	}

	err = configureDocker(ctx)
	if err != nil {
		return err
	}

	debPackages := append(artifacts.DebPackages, optionalPackages...)
	err = installAptPackages(ctx, debPackages)
	if err != nil {
		return err
	}

	transport := &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Minute,
	}

	err = installSeaBIOS(client, artifacts.seaBIOSURLs())
	if err != nil {
		return err
	}

	err = installGo(client, artifacts.goURL())
	if err != nil {
		return err
	}

	err = installDebianPackage(ctx, client, artifacts.placematURL())
	if err != nil {
		return err
	}

	err = installBinaryFile(ctx, client, artifacts.ctURL(), ctPath)
	if err != nil {
		return err
	}

	err = dumpStaticFiles()
	if err != nil {
		return err
	}

	if project == "neco-test" || project == "neco-dev" {
		err = downloadAssets(client, artifacts.assetURLs(), artifacts.bz2Files())
		if err != nil {
			return err
		}
	}
	return nil
}

func configureDNS(ctx context.Context) error {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return err
	}

	for _, line := range strings.Fields(string(data)) {
		param := strings.SplitN(line, "=", 2)
		if param[0] != "ID" {
			continue
		}
		// Skip DNS configuration if distribution is not Ubuntu
		if param[1] != "ubuntu" {
			return nil
		}
	}

	err = DisableService(ctx, "systemd-resolved")
	if err != nil {
		return err
	}

	err = StopService(ctx, "systemd-resolved")
	if err != nil {
		return err
	}

	data, err = os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return err
	}

	newData := strings.Replace(string(data), "nameserver 127.0.0.53", "nameserver 169.254.169.254", -1)

	err = os.Remove("/etc/resolv.conf")
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir("/etc/resolv.conf"), 0755)
	if err != nil {
		return err
	}
	return os.WriteFile("/etc/resolv.conf", []byte(newData), 0644)
}

func apt(ctx context.Context, args ...string) error {
	return well.CommandContext(ctx, "apt-get", args...).Run()
}

func configureApt(ctx context.Context) error {
	err := StopTimer(ctx, "apt-daily-upgrade")
	if err != nil {
		return err
	}
	err = DisableTimer(ctx, "apt-daily-upgrade")
	if err != nil {
		return err
	}
	err = StopService(ctx, "apt-daily-upgrade")
	if err != nil {
		return err
	}
	err = StopTimer(ctx, "apt-daily")
	if err != nil {
		return err
	}
	err = DisableTimer(ctx, "apt-daily")
	if err != nil {
		return err
	}
	err = StopService(ctx, "apt-daily")
	if err != nil {
		return err
	}

	err = apt(ctx, "purge", "-y", "--autoremove", "unattended-upgrades")
	if err != nil {
		return err
	}
	err = apt(ctx, "update")
	if err != nil {
		return err
	}
	err = apt(ctx, "install", "-y", "apt-transport-https")
	if err != nil {
		return err
	}

	return nil
}

func configureDocker(ctx context.Context) error {
	resp, err := http.Get("https://download.docker.com/linux/ubuntu/gpg")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get docker repository GPG key: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Docker PGP key: %w", err)
	}

	if err := os.WriteFile("/etc/apt/keyrings/docker-key.asc", data, 0644); err != nil {
		return err
	}

	cmd := well.CommandContext(ctx, "lsb_release", "-cs")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to invoke lsb_release -cs: %w", err)
	}

	codename := strings.TrimSuffix(string(out), "\n")
	repo := fmt.Sprintf("deb [arch=amd64 signed-by=%s] https://download.docker.com/linux/ubuntu %s stable\n", "/etc/apt/keyrings/docker-key.asc", codename)

	return os.WriteFile("/etc/apt/sources.list.d/docker.list", []byte(repo), 0644)
}

func installAptPackages(ctx context.Context, debPackages []string) error {
	err := apt(ctx, "update")
	if err != nil {
		return err
	}

	args := []string{"install", "-y", "--no-install-recommends"}
	args = append(args, debPackages...)
	err = apt(ctx, args...)
	if err != nil {
		return err
	}
	return apt(ctx, "clean")
}

func installSeaBIOS(client *http.Client, urls []string) error {
	for _, url := range urls {
		err := downloadFile(client, url, "/usr/share/seabios")
		if err != nil {
			return err
		}
	}

	return nil
}

func installGo(client *http.Client, url string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return untargz(resp.Body, "/usr/local")
}

func untargz(r io.Reader, dst string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(dst, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			f.Close()
		}
	}
}

func installDebianPackage(ctx context.Context, client *http.Client, url string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	command := []string{"sh", "-c", "dpkg -i " + f.Name() + " && rm " + f.Name()}
	return well.CommandContext(ctx, command[0], command[1:]...).Run()
}

func installBinaryFile(ctx context.Context, client *http.Client, url, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return writeToFile(dest, resp.Body, 0755)
}

func dumpStaticFiles() error {
	for _, file := range staticFiles {
		err := copyStatic(file)
		if err != nil {
			log.Error("failed to copy file: "+file, map[string]interface{}{
				log.FnError: err,
			})
			return err
		}
		log.Info("wrote", map[string]interface{}{
			"file": file,
		})
	}

	return nil
}

func copyStatic(fileName string) error {
	src, err := assets.Open(path.Join("assets", fileName))
	if err != nil {
		return err
	}
	defer src.Close()

	fi, err := src.Stat()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(fileName), 0755)
	if err != nil {
		return err
	}

	dst, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer dst.Close()

	err = dst.Chmod(fi.Mode())
	if err != nil {
		return err
	}

	_, err = io.Copy(dst, src)
	return err
}

func downloadAssets(client *http.Client, assetURLs, bz2Files []string) error {
	err := os.MkdirAll(assetDir, 0755)
	if err != nil {
		return err
	}

	// Download files
	for _, url := range assetURLs {
		err := downloadFile(client, url, assetDir)
		if err != nil {
			return err
		}
		log.Info("downloaded", map[string]interface{}{
			"url": url,
		})
	}

	// Decompress bzip2 archives
	for _, bz2file := range bz2Files {
		bz2, err := os.Open(filepath.Join(assetDir, bz2file))
		if err != nil {
			return err
		}
		defer func() {
			bz2.Close()
			os.Remove(bz2.Name())
		}()
		f := bzip2.NewReader(bz2)
		extName := strings.TrimRight(bz2.Name(), ".bz2")
		err = writeToFile(extName, f, 0644)
		if err != nil {
			return err
		}
		log.Info("decompressed", map[string]interface{}{
			"from": bz2.Name(),
			"to":   extName,
		})
	}

	return nil
}

func downloadFile(client *http.Client, url, destDir string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to download %s, status code: %d, failed to read body: %w", url, resp.StatusCode, err)
		}
		return fmt.Errorf("failed to download %s, status code: %d, body: %s", url, resp.StatusCode, body)
	}
	f, err := os.OpenFile(filepath.Join(destDir, filepath.Base(url)), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	return f.Sync()
}

func writeToFile(p string, r io.Reader, perm os.FileMode) error {
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()

	err = f.Chmod(perm)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, r)
	if err != nil {
		return err
	}

	return f.Sync()
}
