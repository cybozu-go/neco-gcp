package gcp

import (
	"fmt"
)

// artifactSet represents a set of artifacts for GCP instance.
type artifactSet struct {
	goVersion           string
	rktVersion          string
	etcdVersion         string
	placematVersion     string
	customUbuntuVersion string
	coreOSVersion       string
	ctVersion           string
	baseImage           string
	baseImageProject    string
	debPackages         []string
}

func (a artifactSet) seaBIOSURLs() []string {
	return []string{
		"https://github.com/qemu/qemu/raw/master/pc-bios/bios.bin",
		"https://github.com/qemu/qemu/raw/master/pc-bios/bios-256k.bin",
	}
}

func (a artifactSet) goURL() string {
	return fmt.Sprintf("https://dl.google.com/go/go%s.linux-amd64.tar.gz", a.goVersion)
}

func (a artifactSet) rktURL() string {
	return fmt.Sprintf("https://github.com/rkt/rkt/releases/download/v%s/rkt_%s-1_amd64.deb", a.rktVersion, a.rktVersion)
}

func (a artifactSet) placematURL() string {
	return fmt.Sprintf("https://github.com/cybozu-go/placemat/releases/download/v%s/placemat_%s_amd64.deb", a.placematVersion, a.placematVersion)
}

func (a artifactSet) ctURL() string {
	// Download ct from coreos repository because github.com/flatcar-linux/container-linux-config-transpiler does not provide binary assets.
	return fmt.Sprintf("https://github.com/coreos/container-linux-config-transpiler/releases/download/v%s/ct-v%s-x86_64-unknown-linux-gnu", a.ctVersion, a.ctVersion)
}

func (a artifactSet) assetURLs() []string {
	return []string{
		fmt.Sprintf("https://github.com/coreos/etcd/releases/download/v%s/etcd-v%s-linux-amd64.tar.gz", a.etcdVersion, a.etcdVersion),
		"https://cloud-images.ubuntu.com/releases/18.04/release/ubuntu-18.04-server-cloudimg-amd64.img",
		"https://stable.release.flatcar-linux.net/amd64-usr/current/flatcar_production_qemu_image.img.bz2",
		fmt.Sprintf("https://stable.release.flatcar-linux.net/amd64-usr/%s/flatcar_production_pxe.vmlinuz", a.coreOSVersion),
		fmt.Sprintf("https://stable.release.flatcar-linux.net/amd64-usr/%s/flatcar_production_pxe_image.cpio.gz", a.coreOSVersion),
		fmt.Sprintf("https://github.com/cybozu/neco-ubuntu/releases/download/%s/cybozu-ubuntu-18.04-server-cloudimg-amd64.img", a.customUbuntuVersion),
	}
}

func (a artifactSet) bz2Files() []string {
	return []string{
		"flatcar_production_qemu_image.img.bz2",
	}
}
