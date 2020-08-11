package functions

import (
	"errors"
	"fmt"
)

const (
	accountSecretName  = "cloud-dns-admin-account"
	serviceAccountName = "neco-dev"
)

// NecoStartupScriptBuilder creates startup-script builder to run dctest
type NecoStartupScriptBuilder struct {
	deleteIfFail   bool
	withFluentd    bool
	necoBranch     string
	necoAppsBranch string
}

// NewNecoStartupScriptBuilder creates NecoStartupScriptBuilder
func NewNecoStartupScriptBuilder() *NecoStartupScriptBuilder {
	return &NecoStartupScriptBuilder{}
}

// WithFluentd enables fluentd
func (b *NecoStartupScriptBuilder) WithFluentd() *NecoStartupScriptBuilder {
	b.withFluentd = true
	return b
}

// WithNeco sets which branch to run neco
func (b *NecoStartupScriptBuilder) WithNeco(branch string) *NecoStartupScriptBuilder {
	b.necoBranch = branch
	return b
}

// WithNecoApps sets which branch to run neco-apps
func (b *NecoStartupScriptBuilder) WithNecoApps(branch string) (*NecoStartupScriptBuilder, error) {
	if len(b.necoBranch) == 0 {
		return nil, errors.New("please specify neco branch to run neco-apps")
	}
	b.necoAppsBranch = branch
	return b, nil
}

// Build  builds startup script
func (b *NecoStartupScriptBuilder) Build() string {
	s := `#! /bin/sh -x

delete_myself()
{
export NAME=$(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/name -H 'Metadata-Flavor: Google')
export ZONE=$(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/zone -H 'Metadata-Flavor: Google')
/snap/bin/gcloud --quiet compute instances delete $NAME --zone=$ZONE
}

prepare_scratch()
{
# mkfs and mount local SSD on /var/scratch
mkfs -t ext4 -F /dev/disk/by-id/google-local-ssd-0 &&
mkdir -p /var/scratch &&
mount -t ext4 /dev/disk/by-id/google-local-ssd-0 /var/scratch &&
chmod 1777 /var/scratch
}

if ! prepare_scratch ; then delete_myself; fi
`

	if b.withFluentd {
		s += `
with_fluentd()
{
# Run fluentd to export syslog to Cloud Logging
curl -sSO https://dl.google.com/cloudagents/add-logging-agent-repo.sh &&
bash add-logging-agent-repo.sh &&
apt-get update &&
apt-cache madison google-fluentd &&
apt-get install -y google-fluentd &&
apt-get install -y google-fluentd-catch-all-config-structured &&
service google-fluentd start &&
# This line is needed to ensure that fluentd is running
service google-fluentd restart
}

if ! with_fluentd ; then delete_myself; fi
`
	}

	s += fmt.Sprintf(`
# Set environment variables
HOME=/root
GOPATH=${HOME}/go
GO111MODULE=on
PATH=${PATH}:/usr/local/go/bin:${GOPATH}/bin
NECO_DIR=${GOPATH}/src/github.com/cybozu-go/neco
export HOME GOPATH GO111MODULE PATH NECO_DIR

# Run neco
run_neco()
{
mkdir -p ${GOPATH}/src/github.com/cybozu-go &&
cd ${GOPATH}/src/github.com/cybozu-go &&
git clone https://github.com/cybozu-go/neco &&
cd ${GOPATH}/src/github.com/cybozu-go/neco/dctest &&
git checkout %s &&
make setup placemat MENU_ARG=menu-ss.yml && make test SUITE=bootstrap
}

if ! run_neco ; then delete_myself; fi
`, b.necoBranch)

	if len(b.necoAppsBranch) > 0 {
		s += fmt.Sprintf(`
run_necoapps()
{
# Run neco-apps
cd ${GOPATH}/src/github.com/cybozu-go &&
git clone https://github.com/cybozu-go/neco-apps &&
cd ${GOPATH}/src/github.com/cybozu-go/neco-apps/test &&
git checkout %s &&
gcloud secrets versions access latest --secret="%s" > account.json &&
make setup dctest BOOTSTRAP=1 OVERLAY=neco-dev
}

if ! run_necoapps ; then delete_myself; fi
`, b.necoAppsBranch, accountSecretName)
	}

	return s
}

// MakeVMXEnabledImageURL returns vmx-enabled image URL in the project
func MakeVMXEnabledImageURL(projectID string) string {
	return "https://www.googleapis.com/compute/v1/projects/" + projectID + "/global/images/vmx-enabled"
}

// MakeNecoDevServiceAccountEmail returns custom service account name in the project
func MakeNecoDevServiceAccountEmail(projectID string) string {
	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", serviceAccountName, projectID)
}
