package gcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

const (
	homeDisk           = "/dev/disk/by-id/google-home"
	homeFSType         = "ext4"
	homeMountPoint     = "/home"
	localSSDDisk       = "/dev/nvme0n1"
	localSSDFSType     = "ext4"
	localSSDMountPoint = "/var/scratch"
)

type temporaryer interface {
	Temporary() bool
}

// SetupHostVM setup vmx-enabled instance
func SetupHostVM(ctx context.Context) error {
	err := enableXForwarding()
	if err != nil {
		return err
	}

	err = mountHomeDisk(ctx)
	if err != nil {
		return err
	}

	return setupLocalSSD(ctx)
}

func enableXForwarding() error {
	reFrom := regexp.MustCompile(`SSHD_OPTS=.*`)
	reTo := `SSHD_OPTS="-o X11UseLocalhost=no"`
	destFile := "/etc/default/ssh"

	f, err := os.OpenFile(destFile, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	replaced := reFrom.ReplaceAll(data, []byte(reTo))
	st, err := f.Stat()
	if err != nil {
		return err
	}
	return os.WriteFile(destFile, replaced, st.Mode())
}

func mountHomeDisk(ctx context.Context) error {
	f, err := os.OpenFile("/etc/fstab", os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if bytes.Contains([]byte(homeDisk), data) {
		return nil
	}

	err = StopService(ctx, "ssh")
	if err != nil {
		return err
	}

	/*
	 * Those daemons touch $HOME to install ssh keys.
	 * We stop them during mounting /home just in case.
	 */
	accountDaemons := []string{"google-accounts-daemon", "google-guest-agent"}
	accountDaemon := ""
	for _, ad := range accountDaemons {
		exists, err := ExistsService(ctx, ad)
		if err != nil {
			return err
		}
		if exists {
			accountDaemon = ad
			break
		}
	}
	if accountDaemon == "" {
		return errors.New("no known account daemon found")
	}

	err = RetryWithSleep(ctx, retryCount, time.Second,
		func(ctx context.Context) error {
			active, err := IsActiveService(ctx, accountDaemon)
			if err != nil {
				return err
			}
			if !active {
				return errors.New(accountDaemon + ".service is not yet active")
			}
			return nil

		},
		func(err error) {
			log.Error("timeout for checking service is active", map[string]interface{}{
				log.FnError: err,
				"service":   accountDaemon + ".service",
			})
		},
	)
	if err != nil {
		return err
	}

	err = StopService(ctx, accountDaemon)
	if err != nil {
		return err
	}

	err = well.CommandContext(ctx, "/sbin/dumpe2fs", "-h", homeDisk).Run()
	if err != nil {
		err := formatHomeDisk(ctx)
		if err != nil {
			return err
		}
	}

	_, err = io.WriteString(f, fmt.Sprintf("%s %s %s defaults 1 1", homeDisk, homeMountPoint, homeFSType))
	if err != nil {
		return err
	}

	for {
		err = well.CommandContext(ctx, "/bin/mount", "-t", homeFSType, "-o", "relatime", homeDisk, homeMountPoint).Run()
		if err == nil {
			break
		}
		if e, ok := err.(temporaryer); ok && e.Temporary() {
			continue
		}
		return err
	}

	err = StartService(ctx, accountDaemon)
	if err != nil {
		return err
	}

	return StartService(ctx, "ssh")
}

func formatHomeDisk(ctx context.Context) error {
	err := well.CommandContext(ctx, "/sbin/mkfs", "-t", homeFSType, homeDisk).Run()
	if err != nil {
		return err
	}

	for {
		err = well.CommandContext(ctx, "/bin/mount", "-t", homeFSType, "-o", "relatime", homeDisk, "/mnt").Run()
		if err == nil {
			break
		}
		if e, ok := err.(temporaryer); ok && e.Temporary() {
			continue
		}
		return err
	}

	err = well.CommandContext(ctx, "/bin/cp", "-a", "/home/.", "/mnt").Run()
	if err != nil {
		return err
	}

	for {
		err := well.CommandContext(ctx, "/bin/umount", "/mnt", "-f").Run()
		if err == nil {
			break
		}
		if e, ok := err.(temporaryer); ok && e.Temporary() {
			continue
		}
		return err
	}
	return nil
}

func setupLocalSSD(ctx context.Context) error {
	err := well.CommandContext(ctx, "/sbin/mkfs", "-t", localSSDFSType, "-F", localSSDDisk).Run()
	if err != nil {
		return err
	}

	err = os.MkdirAll(localSSDMountPoint, 0755)
	if err != nil {
		return err
	}

	for {
		err = well.CommandContext(ctx, "/bin/mount", "-t", localSSDFSType, "-o", "relatime", localSSDDisk, localSSDMountPoint).Run()
		if err == nil {
			break
		}
		if e, ok := err.(temporaryer); ok && e.Temporary() {
			continue
		}
		return err
	}

	return os.Chmod(localSSDMountPoint, 0777)
}
