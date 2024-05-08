package necogcp

import "github.com/cybozu-go/neco-gcp/pkg/setup"

// The base image of the VMXEnabled image.
// This setting is used by both "necogcp create-image" and "necogcp neco-test create-image".
const (
	VMXEnabledBaseImageProject = "ubuntu-os-cloud"
	VMXEnabledBaseImage        = "ubuntu-2204-jammy-v20240319"
)

// The settings of software which installed in the VMXEnabled image.
// This setting is used by both "necogcp create-image" and "necogcp neco-test create-image".
var VMXEnabledArtifacts = setup.ArtifactSet{
	GoVersion:       "1.22.3",
	EtcdVersion:     "3.5.13",
	PlacematVersion: "2.4.5",
	CoreOSVersion:   "3815.2.1",
	CtVersion:       "0.9.3", //If upgrading a version, make sure the binary is included in the GitHub release
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
		"docker-buildx-plugin",
		"containerd.io",
	},
}
