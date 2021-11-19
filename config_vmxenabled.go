package necogcp

import "github.com/cybozu-go/neco-gcp/pkg/setup"

// The base image of the VMXEnabled image.
// This setting is used by both "necogcp create-image" and "necogcp neco-test create-image".
const (
	VMXEnabledBaseImageProject = "ubuntu-os-cloud"
	VMXEnabledBaseImage        = "ubuntu-2004-focal-v20210908"
)

// The settings of software which installed in the VMXEnabled image.
// This setting is used by both "necogcp create-image" and "necogcp neco-test create-image".
var VMXEnabledArtifacts = setup.ArtifactSet{
	GoVersion:       "1.17.1",
	EtcdVersion:     "3.4.16",
	PlacematVersion: "2.0.5",
	CoreOSVersion:   "2905.2.3",
	CtVersion:       "0.6.1",
	DebPackages: []string{
		"git",
		"build-essential",
		"less",
		"wget",
		"systemd-container",
		"lldpd",
		"qemu",
		"qemu-kvm",
		"socat",
		"picocom",
		"swtpm",
		"cloud-utils",
		"bird2",
		"squid",
		"chrony",
		"dnsmasq",
		"xauth",
		"bash-completion",
		"dbus",
		"jq",
		"libgpgme11",
		"freeipmi-tools",
		"unzip",
		"skopeo",
		// required by building neco
		"fakeroot",
		// docker CE
		"docker-ce",
		"docker-ce-cli",
		"containerd.io",
	},
}
