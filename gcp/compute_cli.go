package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

var startUpScriptTmpl = template.Must(template.New("").Parse(`#!/bin/sh

STARTUP_PATH=/tmp/startup.sh
cat << 'EOF' > $STARTUP_PATH
export NAME=$(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/name -H 'Metadata-Flavor: Google')
export ZONE=$(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/zone -H 'Metadata-Flavor: Google')
/snap/bin/gcloud --quiet compute instances delete $NAME --zone=$ZONE
EOF
chmod 755 $STARTUP_PATH

at {{ .ShutdownAt }} -f $STARTUP_PATH
`))

const (
	retryCount   = 300
	imageLicense = "https://www.googleapis.com/compute/v1/projects/vm-options/global/licenses/enable-vmx"
	// MetadataKeyShutdownAt is the instance key to represents the time that this instance should be deleted.
	MetadataKeyShutdownAt = "shutdown-at"
	timeFormat            = "2006-01-02 15:04:05"
)

// ComputeCLIClient is GCP compute client using "gcloud compute"
type ComputeCLIClient struct {
	cfg      *Config
	instance string
	user     string
	image    string
}

// NewComputeCLIClient returns ComputeClient
func NewComputeCLIClient(cfg *Config, instance string) *ComputeCLIClient {
	user := os.Getenv("USER")
	if cfg.Common.Project == "neco-test" {
		user = "cybozu"
	}

	return &ComputeCLIClient{
		cfg:      cfg,
		instance: instance,
		user:     user,
		image:    "vmx-enabled",
	}
}

func (cc *ComputeCLIClient) gCloudCompute() []string {
	return []string{"gcloud", "--quiet", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute"}
}

func (cc *ComputeCLIClient) gCloudComputeInstances() []string {
	return []string{"gcloud", "--quiet", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute", "instances"}
}

func (cc *ComputeCLIClient) gCloudComputeImages() []string {
	return []string{"gcloud", "--quiet", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute", "images"}
}

func (cc *ComputeCLIClient) gCloudComputeDisks() []string {
	return []string{"gcloud", "--quiet", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute", "disks"}
}

func (cc *ComputeCLIClient) gCloudComputeSSH(command []string) []string {
	return []string{"gcloud", "--quiet", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute", "ssh",
		"--zone", cc.cfg.Common.Zone,
		fmt.Sprintf("%s@%s", cc.user, cc.instance),
		fmt.Sprintf("--command=%s", strings.Join(command, " "))}
}

// CreateVMXEnabledInstance creates vmx-enabled instance
func (cc *ComputeCLIClient) CreateVMXEnabledInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeInstances()
	bootDiskSize := strconv.Itoa(cc.cfg.Compute.BootDiskSizeGB) + "GB"
	gcmd = append(gcmd, "create", cc.instance,
		"--zone", cc.cfg.Common.Zone,
		"--image", artifacts.baseImage,
		"--image-project", artifacts.baseImageProject,
		"--boot-disk-type", "pd-ssd",
		"--boot-disk-size", bootDiskSize,
		"--machine-type", cc.cfg.Compute.MachineType)
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// ConvertLocalTimeToUTC converts local time to UTC
func ConvertLocalTimeToUTC(timezone, shutdownAt string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}
	now := time.Now().In(loc)
	localTime := fmt.Sprintf("%d-%02d-%02d "+shutdownAt+":00",
		now.Year(), now.Month(), now.Day())
	t, err := time.ParseInLocation(timeFormat, localTime, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// CreateHostVMInstance creates host-vm instance
func (cc *ComputeCLIClient) CreateHostVMInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeInstances()
	bootDiskSize := strconv.Itoa(cc.cfg.Compute.BootDiskSizeGB) + "GB"
	shutdownTime, err := ConvertLocalTimeToUTC(cc.cfg.Compute.AutoShutdown.Timezone, cc.cfg.Compute.AutoShutdown.ShutdownAt)
	if err != nil {
		return err
	}
	shutdownAt := shutdownTime.Format("15:04")
	log.Info("the instance will shutdown at UTC "+shutdownAt, map[string]interface{}{})
	buf := new(bytes.Buffer)
	err = startUpScriptTmpl.Execute(buf, struct {
		ShutdownAt string
	}{
		ShutdownAt: shutdownAt,
	})
	if err != nil {
		return err
	}
	tmpfile, err := ioutil.TempFile("/tmp", "gcp-start-up-script-*.sh")
	if err != nil {
		return err
	}
	defer func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	}()
	log.Info("start up script for "+tmpfile.Name(), map[string]interface{}{})
	_, err = tmpfile.Write(buf.Bytes())
	if err != nil {
		return err
	}
	gcmd = append(gcmd, "create", cc.instance,
		"--zone", cc.cfg.Common.Zone,
		"--image", cc.image,
		"--boot-disk-type", "pd-ssd",
		"--boot-disk-size", bootDiskSize,
		"--local-ssd", "interface=scsi",
		"--machine-type", cc.cfg.Compute.MachineType,
		"--metadata-from-file", "startup-script="+tmpfile.Name(),
		"--scopes", "compute-rw,storage-rw",
	)
	if cc.cfg.Compute.HostVM.Preemptible {
		gcmd = append(gcmd, "--preemptible")
	}
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// CreateHomeDisk creates home disk image
func (cc *ComputeCLIClient) CreateHomeDisk(ctx context.Context) error {
	if !cc.cfg.Compute.HostVM.HomeDisk {
		return nil
	}
	gcmdInfo := cc.gCloudComputeDisks()
	gcmdInfo = append(gcmdInfo, "describe", "home",
		"--zone", cc.cfg.Common.Zone,
		"--format", "json")
	outBuf := new(bytes.Buffer)
	c := well.CommandContext(ctx, gcmdInfo[0], gcmdInfo[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = outBuf
	c.Stderr = os.Stderr
	err := c.Run()
	if err == nil {
		log.Info("home disk already exists", nil)
		return nil
	}

	configSize := strconv.Itoa(cc.cfg.Compute.HostVM.HomeDiskSizeGB) + "GB"
	gcmdCreate := cc.gCloudComputeDisks()
	gcmdCreate = append(gcmdCreate, "create", "home", "--size", configSize, "--type", "pd-ssd", "--zone", cc.cfg.Common.Zone)
	c = well.CommandContext(ctx, gcmdCreate[0], gcmdCreate[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// AttachHomeDisk attaches home disk image to host-vm instance
func (cc *ComputeCLIClient) AttachHomeDisk(ctx context.Context) error {
	if !cc.cfg.Compute.HostVM.HomeDisk {
		return nil
	}
	gcmd := cc.gCloudComputeInstances()
	gcmd = append(gcmd, "attach-disk", cc.instance,
		"--zone", cc.cfg.Common.Zone,
		"--disk", "home",
		"--device-name", "home")
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// ResizeHomeDisk resizes home disk image
func (cc *ComputeCLIClient) ResizeHomeDisk(ctx context.Context) error {
	if !cc.cfg.Compute.HostVM.HomeDisk {
		return nil
	}
	gcmdInfo := cc.gCloudComputeDisks()
	gcmdInfo = append(gcmdInfo, "describe", "home",
		"--zone", cc.cfg.Common.Zone,
		"--format", "json")
	outBuf := new(bytes.Buffer)
	c := well.CommandContext(ctx, gcmdInfo[0], gcmdInfo[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = outBuf
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return err
	}

	var info map[string]interface{}
	err = json.Unmarshal(outBuf.Bytes(), &info)
	if err != nil {
		return err
	}

	currentSize, ok := info["sizeGb"].(string)
	if !ok {
		return errors.New("failed to convert sizeGb")
	}
	currentSizeInt, err := strconv.Atoi(currentSize)
	if err != nil {
		return err
	}
	configSize := strconv.Itoa(cc.cfg.Compute.HostVM.HomeDiskSizeGB) + "GB"
	configSizeInt := cc.cfg.Compute.HostVM.HomeDiskSizeGB
	if currentSizeInt >= configSizeInt {
		log.Info("current home disk size is smaller or equal to the size in configuration file", map[string]interface{}{
			"currentSize": currentSizeInt,
			"configSize":  configSizeInt,
		})
		return nil
	}

	gcmdResize := cc.gCloudComputeDisks()
	gcmdResize = append(gcmdResize, "resize", "home", "--size", configSize)
	c = well.CommandContext(ctx, gcmdResize[0], gcmdResize[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// DeleteInstance deletes given instance
func (cc *ComputeCLIClient) DeleteInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeInstances()
	gcmd = append(gcmd, "delete", cc.instance, "--zone", cc.cfg.Common.Zone)
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// WaitInstance waits given instance until online
func (cc *ComputeCLIClient) WaitInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeSSH([]string{"date"})
	return RetryWithSleep(ctx, retryCount, time.Second,
		func(ctx context.Context) error {
			c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
		func(err error) {
			log.Error("failed to check online of the instance", map[string]interface{}{
				log.FnError: err,
				"instance":  cc.instance,
			})
		},
	)
}

// StopInstance stops given instance
func (cc *ComputeCLIClient) StopInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeInstances()
	gcmd = append(gcmd, "stop", cc.instance,
		"--zone", cc.cfg.Common.Zone)
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// CreateVMXEnabledImage create GCE vmx-enabled image
func (cc *ComputeCLIClient) CreateVMXEnabledImage(ctx context.Context) error {
	gcmd := cc.gCloudComputeImages()
	gcmd = append(gcmd, "create", cc.image,
		"--source-disk", cc.instance,
		"--source-disk-zone", cc.cfg.Common.Zone,
		"--licenses", imageLicense)
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// DeleteVMXEnabledImage create GCE vmx-enabled image
func (cc *ComputeCLIClient) DeleteVMXEnabledImage(ctx context.Context) error {
	gcmd := cc.gCloudComputeImages()
	gcmd = append(gcmd, "delete", cc.image)
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// Upload uploads a file to the instance through ssh
func (cc *ComputeCLIClient) Upload(ctx context.Context, file string) error {
	gcmd := cc.gCloudCompute()
	gcmd = append(gcmd, "scp", "--zone", cc.cfg.Common.Zone, file, fmt.Sprintf("%s@%s:/tmp", cc.user, cc.instance))
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// RunSetup executes "necogcp setup" on the instance through ssh
func (cc *ComputeCLIClient) RunSetup(ctx context.Context, progFile, cfgFile string) error {
	err := cc.Upload(ctx, progFile)
	if err != nil {
		return err
	}

	err = cc.Upload(ctx, cfgFile)
	if err != nil {
		return err
	}

	gcmd := cc.gCloudComputeSSH([]string{"sudo", "/tmp/" + filepath.Base(progFile), "--config", "/tmp/" + filepath.Base(cfgFile), "setup-instance"})
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// ExtendInstance extends 2 hours from now for given instance to prevent auto deletion
func (cc *ComputeCLIClient) ExtendInstance(ctx context.Context) error {
	gcmd := cc.gCloudComputeInstances()
	gcmd = append(gcmd, "add-metadata", cc.instance,
		"--zone", cc.cfg.Common.Zone,
		"--metadata", MetadataKeyShutdownAt+"="+time.Now().UTC().Add(2*time.Hour).Format(time.RFC3339))
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}