package gcp

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

//go:embed bin
var binDir embed.FS

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
	defaultVolumeName     = "home"
)

// ComputeCLIClient is GCP compute client using "gcloud compute"
type ComputeCLIClient struct {
	cfg      *Config
	instance string
	user     string
	image    string
}

// Snapshot is GCP snapshot object
type Snapshot struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// NewComputeCLIClient returns ComputeCLIClient
func NewComputeCLIClient(cfg *Config, instance string) *ComputeCLIClient {
	user := os.Getenv("USER")
	if cfg.Common.Project == "neco-test" || cfg.Common.Project == "neco-dev" {
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

func (cc *ComputeCLIClient) gCloudDiskSnapshot() []string {
	return []string{"gcloud", "--quiet", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute", "disks", "snapshot"}
}

func (cc *ComputeCLIClient) gCloudComputeSSH(command []string) []string {
	return []string{"gcloud", "--quiet", "--account", cc.cfg.Common.ServiceAccount, "--project", cc.cfg.Common.Project, "compute", "ssh",
		"--zone", cc.cfg.Common.Zone,
		fmt.Sprintf("%s@%s", cc.user, cc.instance),
		fmt.Sprintf("--command=%s", strings.Join(command, " "))}
}

// CreateVMXEnabledInstance creates vmx-enabled instance
func (cc *ComputeCLIClient) CreateVMXEnabledInstance(ctx context.Context, baseImageProject, baseImage string) error {
	gcmd := cc.gCloudComputeInstances()
	bootDiskSize := strconv.Itoa(cc.cfg.Compute.BootDiskSizeGB) + "GB"
	gcmd = append(gcmd, "create", cc.instance,
		"--zone", cc.cfg.Common.Zone,
		"--image-project", baseImageProject,
		"--image", baseImage,
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
	log.Info("the instance will shutdown at UTC "+shutdownAt, nil)
	buf := new(bytes.Buffer)
	err = startUpScriptTmpl.Execute(buf, struct {
		ShutdownAt string
	}{
		ShutdownAt: shutdownAt,
	})
	if err != nil {
		return err
	}
	tmpfile, err := os.CreateTemp("", "gcp-start-up-script-*.sh")
	if err != nil {
		return err
	}
	defer func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	}()
	log.Info("create a temporary file (start up script): "+tmpfile.Name(), nil)
	_, err = tmpfile.Write(buf.Bytes())
	if err != nil {
		return err
	}
	gcmd = append(gcmd, "create", cc.instance,
		"--zone", cc.cfg.Common.Zone,
		"--image", cc.image,
		"--boot-disk-type", "pd-ssd",
		"--boot-disk-size", bootDiskSize,
		"--machine-type", cc.cfg.Compute.MachineType,
		"--metadata-from-file", "startup-script="+tmpfile.Name(),
		"--scopes", "compute-rw,storage-rw",
	)
	for i := 0; i < cc.cfg.Compute.NumLocalSSDs; i++ {
		gcmd = append(gcmd, "--local-ssd", "interface=nvme")
	}
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
	gcmdResize = append(gcmdResize, "resize", "home", "--size", configSize, "--zone", cc.cfg.Common.Zone)
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
func (cc *ComputeCLIClient) Upload(ctx context.Context, file, target, mode string) error {
	gcmd := cc.gCloudCompute()
	gcmd = append(gcmd, "scp", "--zone", cc.cfg.Common.Zone, file, fmt.Sprintf("%s@%s:%s", cc.user, cc.instance, target))
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return err
	}

	gcmd = cc.gCloudComputeSSH([]string{"sudo", "chmod", mode, target})
	c = well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// UploadSetupCommand uploads the "setup" command on the instance through ssh
func (cc *ComputeCLIClient) UploadSetupCommand(ctx context.Context) error {
	tmpfile, err := os.CreateTemp("", "necogcp-setup-command-")
	if err != nil {
		return err
	}
	defer func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	}()
	log.Info("create a temporary file (setup command): "+tmpfile.Name(), nil)

	src, err := binDir.Open(filepath.Join("bin", "setup"))
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(tmpfile, src)
	if err != nil {
		return err
	}

	return cc.Upload(ctx, tmpfile.Name(), "/tmp/setup", "775")
}

// RunSetupHostVM executes "setup host-vm" on the instance through ssh
func (cc *ComputeCLIClient) RunSetupHostVM(ctx context.Context) error {
	err := cc.UploadSetupCommand(ctx)
	if err != nil {
		return err
	}

	gcmd := cc.gCloudComputeSSH([]string{"sudo", "/tmp/setup", "host-vm"})
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// RunSetupVMXEnabled executes "setup vmx-enabled" on the instance through ssh
func (cc *ComputeCLIClient) RunSetupVMXEnabled(ctx context.Context, optionalPackages []string) error {
	err := cc.UploadSetupCommand(ctx)
	if err != nil {
		return err
	}

	tmpfile, err := os.CreateTemp("", "necogcp-setup-packages-*.json")
	if err != nil {
		return err
	}
	defer func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
	}()
	log.Info("create a temporary file (optional packages file): "+tmpfile.Name(), nil)

	buf, err := json.Marshal(optionalPackages)
	if err != nil {
		panic(err)
	}
	_, err = tmpfile.Write(buf)
	if err != nil {
		return err
	}

	err = cc.Upload(ctx, tmpfile.Name(), "/tmp/package.json", "664")
	if err != nil {
		return err
	}

	gcmd := cc.gCloudComputeSSH([]string{"sudo", "/tmp/setup", "vmx-enabled", cc.cfg.Common.Project, "/tmp/package.json"})
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

// CreateVolumeSnapshot creates home volume snapshot
func (cc *ComputeCLIClient) CreateVolumeSnapshot(ctx context.Context) error {
	gcmd := cc.gCloudDiskSnapshot()
	gcmd = append(gcmd, defaultVolumeName,
		"--zone", cc.cfg.Common.Zone)
	c := well.CommandContext(ctx, gcmd[0], gcmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// RestoreVolumeFromSnapshot restores home volume in the target zone
func (cc *ComputeCLIClient) RestoreVolumeFromSnapshot(ctx context.Context, destZone string) error {
	gcmdSnapshot := []string{"gcloud", "--quiet", "--account", cc.cfg.Common.ServiceAccount,
		"--project", cc.cfg.Common.Project, "compute", "snapshots", "list",
		"--sort-by=date", "--limit=1", "--filter=sourceDisk:disks/home", "--format=json"}

	outBuf := new(bytes.Buffer)
	c := well.CommandContext(ctx, gcmdSnapshot[0], gcmdSnapshot[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = outBuf
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return err
	}

	snapshot := []Snapshot{}
	err = json.Unmarshal(outBuf.Bytes(), &snapshot)
	if err != nil {
		return err
	}

	if len(snapshot) == 0 {
		return fmt.Errorf("no availabe snapshot exists")
	}
	if len(snapshot) > 1 {
		return fmt.Errorf("more than 1 snapshot was selected. num: %v", len(snapshot))
	}
	target := snapshot[0]
	if target.Status != "READY" {
		return fmt.Errorf("target snapshot %v stauts is %v and not ready", target.Name, target.Status)
	}

	// Confirm there is no home disk in the target zone to prevent unexpected volume restore.
	gcmdInfo := cc.gCloudComputeDisks()
	gcmdInfo = append(gcmdInfo, "describe", "home",
		"--zone", destZone,
		"--format", "json")
	c = well.CommandContext(ctx, gcmdInfo[0], gcmdInfo[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err = c.Run()
	if err == nil {
		return fmt.Errorf("home disk already exists in the zone %v. please delete the disk first. ", destZone)
	}

	// Restore volume in the target zone
	configSize := strconv.Itoa(cc.cfg.Compute.HostVM.HomeDiskSizeGB) + "GB"
	gcmdCreate := cc.gCloudComputeDisks()
	gcmdCreate = append(gcmdCreate, "create", "home", "--size", configSize,
		"--type", "pd-ssd", "--zone", destZone, "--source-snapshot", target.Name)
	c = well.CommandContext(ctx, gcmdCreate[0], gcmdCreate[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}
