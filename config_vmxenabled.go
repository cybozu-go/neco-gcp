package necogcp

import "github.com/cybozu-go/neco-gcp/pkg/setup"

// The base image of the VMXEnabled image.
// This setting is used by both "necogcp create-image" and "necogcp neco-test create-image".
const (
	VMXEnabledBaseImageProject = "ubuntu-os-cloud"
	VMXEnabledBaseImage        = "ubuntu-2604-resolute-amd64-v20260527"
)

// The settings of software which installed in the VMXEnabled image.
// This setting is used by both "necogcp create-image" and "necogcp neco-test create-image".
var VMXEnabledArtifacts = setup.ArtifactSet{
	GoVersion:       "1.25.3",
	EtcdVersion:     "3.6.5",
	PlacematVersion: "2.5.0",
	CoreOSVersion:   "4230.2.4",
	CtVersion:       "0.9.3", //If upgrading a version, make sure the binary is included in the GitHub release
	DebPackages: []string{
		"git",
		"build-essential",
		"less",
		"wget",
		"systemd-container",
		"lldpd",
		"qemu-system-x86",
		"qemu-utils",
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
		"libgpgme45",
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
