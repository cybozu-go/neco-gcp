package setup

import (
	"fmt"
)

// ArtifactSet represents a set of artifacts for GCP instance.
type ArtifactSet struct {
	GoVersion       string
	EtcdVersion     string
	PlacematVersion string
	CoreOSVersion   string
	CtVersion       string
	DebPackages     []string
}

func (a ArtifactSet) seaBIOSURLs() []string {
	return []string{
		"https://github.com/qemu/qemu/raw/master/pc-bios/bios.bin",
		"https://github.com/qemu/qemu/raw/master/pc-bios/bios-256k.bin",
	}
}

func (a ArtifactSet) goURL() string {
	return fmt.Sprintf("https://dl.google.com/go/go%s.linux-amd64.tar.gz", a.GoVersion)
}

func (a ArtifactSet) placematURL() string {
	return fmt.Sprintf("https://github.com/cybozu-go/placemat/releases/download/v%s/placemat2_%s_amd64.deb", a.PlacematVersion, a.PlacematVersion)
}

func (a ArtifactSet) ctURL() string {
	// Download ct from coreos repository because github.com/flatcar-linux/container-linux-config-transpiler does not provide binary assets.
	return fmt.Sprintf("https://github.com/coreos/container-linux-config-transpiler/releases/download/v%s/ct-v%s-x86_64-unknown-linux-gnu", a.CtVersion, a.CtVersion)
}

func (a ArtifactSet) assetURLs() []string {
	return []string{
		fmt.Sprintf("https://github.com/etcd-io/etcd/releases/download/v%s/etcd-v%s-linux-amd64.tar.gz", a.EtcdVersion, a.EtcdVersion),
		fmt.Sprintf("https://stable.release.flatcar-linux.net/amd64-usr/%s/flatcar_production_qemu_image.img.bz2", a.CoreOSVersion),
		fmt.Sprintf("https://stable.release.flatcar-linux.net/amd64-usr/%s/flatcar_production_pxe.vmlinuz", a.CoreOSVersion),
		fmt.Sprintf("https://stable.release.flatcar-linux.net/amd64-usr/%s/flatcar_production_pxe_image.cpio.gz", a.CoreOSVersion),
		"https://cloud-images.ubuntu.com/releases/20.04/release/ubuntu-20.04-server-cloudimg-amd64.img",
	}
}

func (a ArtifactSet) bz2Files() []string {
	return []string{
		"flatcar_production_qemu_image.img.bz2",
	}
}
