package gcp

import (
	"archive/tar"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/rakyll/statik/fs"
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

// SetupVMXEnabled setup vmx-enabled instance
func SetupVMXEnabled(ctx context.Context, project string, option []string) error {
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

	err = configureSWTPM(ctx)
	if err != nil {
		return err
	}

	err = configureProjectAtomic(ctx)
	if err != nil {
		return err
	}

	err = configureDocker(ctx)
	if err != nil {
		return err
	}

	err = installAptPackages(ctx, option)
	if err != nil {
		return err
	}

	transport := &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
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

	err = installSeaBIOS(client)
	if err != nil {
		return err
	}

	err = installGo(client)
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
		err = downloadAssets(client)
		if err != nil {
			return err
		}
	}
	return nil
}

func configureDNS(ctx context.Context) error {
	data, err := ioutil.ReadFile("/etc/os-release")
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

	data, err = ioutil.ReadFile("/etc/resolv.conf")
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
	return ioutil.WriteFile("/etc/resolv.conf", []byte(newData), 0644)
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

func configureSWTPM(ctx context.Context) error {
	return well.CommandContext(ctx, "add-apt-repository", "-y", "ppa:smoser/swtpm").Run()
}

func configureProjectAtomic(ctx context.Context) error {
	err := well.CommandContext(ctx, "apt-key", "adv", "--keyserver", "keyserver.ubuntu.com", "--recv", "7AD8C79D").Run()
	if err != nil {
		return err
	}

	return well.CommandContext(ctx, "add-apt-repository", "deb http://ppa.launchpad.net/projectatomic/ppa/ubuntu bionic main").Run()
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
	key, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	cmd := well.CommandContext(ctx, "apt-key", "add", "-")
	cmd.Stdin = bytes.NewReader(key)
	err = cmd.Run()
	if err != nil {
		return err
	}

	return well.CommandContext(ctx, "add-apt-repository", "deb [arch=amd64] https://download.docker.com/linux/ubuntu bionic stable").Run()
}

func installAptPackages(ctx context.Context, optionalPackages []string) error {
	err := apt(ctx, "update")
	if err != nil {
		return err
	}

	args := []string{"install", "-y", "--no-install-recommends"}
	args = append(args, artifacts.debPackages...)
	err = apt(ctx, args...)
	if err != nil {
		return err
	}
	if len(optionalPackages) != 0 {
		args := []string{"install", "-y", "--no-install-recommends"}
		args = append(args, optionalPackages...)
		err = apt(ctx, args...)
		if err != nil {
			return err
		}
	}
	return apt(ctx, "clean")
}

func installSeaBIOS(client *http.Client) error {
	for _, url := range artifacts.seaBIOSURLs() {
		err := downloadFile(client, url, "/usr/share/seabios")
		if err != nil {
			return err
		}
	}

	return nil
}

func installGo(client *http.Client) error {
	resp, err := client.Get(artifacts.goURL())
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

	f, err := ioutil.TempFile("", "")
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
	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	for _, file := range staticFiles {
		err := copyStatic(statikFS, file)
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

func copyStatic(fs http.FileSystem, fileName string) error {
	src, err := fs.Open(fileName)
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

func downloadAssets(client *http.Client) error {
	err := os.MkdirAll(assetDir, 0755)
	if err != nil {
		return err
	}

	// Download files
	for _, url := range artifacts.assetURLs() {
		err := downloadFile(client, url, assetDir)
		if err != nil {
			return err
		}
		log.Info("downloaded", map[string]interface{}{
			"url": url,
		})
	}

	// Decompress bzip2 archives
	for _, bz2file := range artifacts.bz2Files() {
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
