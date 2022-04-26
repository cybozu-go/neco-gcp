package autodctest

import (
	"errors"
	"fmt"
)

const (
	necoAppsAccountSecretName    = "cloud-dns-admin-account"
	ghcrDockerConfigName         = "ghcr-readonly-dockerconfigjson"
	quayDockerConfigName         = "quay-readonly-dockerconfigjson"
	cybozuPrivateRepoReadPATName = "cybuzo-private-repo-read-pat"
	autoDCTestServiceAccountName = "auto-dctest-vminstance"
)

// MakeNecoDevServiceAccountEmail returns custom service account name in the project
func MakeNecoDevServiceAccountEmail(projectID string) string {
	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", autoDCTestServiceAccountName, projectID)
}

// StartupScriptBuilder creates startup-script builder to run dctest
type StartupScriptBuilder struct {
	withFluentd    bool
	necoBranch     string
	necoAppsBranch string
}

// NewStartupScriptBuilder creates NecoStartupScriptBuilder
func NewStartupScriptBuilder() *StartupScriptBuilder {
	return &StartupScriptBuilder{}
}

// WithFluentd enables fluentd logging
func (b *StartupScriptBuilder) WithFluentd() *StartupScriptBuilder {
	b.withFluentd = true
	return b
}

// WithNeco sets branch name to run neco
func (b *StartupScriptBuilder) WithNeco(branch string) *StartupScriptBuilder {
	b.necoBranch = branch
	return b
}

// WithNecoApps sets branch name to run neco-apps
func (b *StartupScriptBuilder) WithNecoApps(branch string) (*StartupScriptBuilder, error) {
	if len(b.necoBranch) == 0 {
		return nil, errors.New("please specify neco branch to run neco-apps")
	}
	b.necoAppsBranch = branch
	return b, nil
}

// Build  builds startup script
func (b *StartupScriptBuilder) Build() string {
	s := `#! /bin/sh

echo "starting auto dctest..."

# Set environment variables
HOME=/root
GOPATH=${HOME}/go
GO111MODULE=on
PATH=${PATH}:/usr/local/go/bin:${GOPATH}/bin
NECO_DIR=${GOPATH}/src/github.com/cybozu-go/neco
export HOME GOPATH GO111MODULE PATH NECO_DIR

delete_myself()
{
echo "[auto-dctest] Auto dctest failed. Deleting the instance..."
/snap/bin/gcloud --quiet compute instances delete $NAME --zone=$ZONE
}
`

	if b.withFluentd {
		s += `
with_fluentd()
{
curl -sSO https://dl.google.com/cloudagents/add-logging-agent-repo.sh &&
bash add-logging-agent-repo.sh &&
apt-get update &&
apt-cache madison google-fluentd &&
apt-get install -y google-fluentd &&
apt-get install -y google-fluentd-catch-all-config-structured &&
rm -f /etc/google-fluentd/config.d/*.conf &&
echo '<source>
  @type systemd
  tag systemd
  path /var/log/journal
  read_from_head true
  matches [{ "_SYSTEMD_UNIT": "google-startup-scripts.service" }]
  pos_file /var/lib/google-fluentd/pos/systemd.pos
</source>' > /etc/google-fluentd/config.d/systemd.conf &&
/opt/google-fluentd/embedded/bin/fluent-gem install fluent-plugin-systemd &&
service google-fluentd start &&
# This line is needed to ensure that fluentd is running
service google-fluentd restart
}

if ! with_fluentd ; then delete_myself; fi
`
	}

	s += `
prepare()
{
# fetch NAME and ZONE for automatic deletion and mkfs and mount local SSD on /var/scratch
export NAME=$(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/name -H 'Metadata-Flavor: Google') &&
export ZONE=$(curl -X GET http://metadata.google.internal/computeMetadata/v1/instance/zone -H 'Metadata-Flavor: Google') &&
mkfs -t ext4 -F /dev/nvme0n1 &&
mkdir -p /var/scratch &&
mount -t ext4 /dev/nvme0n1 /var/scratch &&
chmod 1777 /var/scratch
}

if ! prepare ; then delete_myself; fi
`

	if len(b.necoBranch) > 0 {
		s += fmt.Sprintf(`
prepare_neco()
{
mkdir -p ${GOPATH}/src/github.com/cybozu-go &&
cd ${GOPATH}/src/github.com/cybozu-go &&
git clone --depth 1 -b %s https://github.com/cybozu-go/neco
}

if ! prepare_neco ; then
  echo '[auto-dctest] Failed to checkout neco branch "%s"'
  delete_myself
fi
`, b.necoBranch, b.necoBranch)
	}

	if len(b.necoAppsBranch) > 0 {
		s += fmt.Sprintf(`
prepare_necoapps()
{
mkdir -p ${GOPATH}/src/github.com/cybozu-go &&
cd ${GOPATH}/src/github.com/cybozu-go &&
gcloud secrets versions access latest --secret="%s" > cybozu_private_repo_read_pat &&
git clone --depth 1 -b %s https://cybozu-neco:$(cat cybozu_private_repo_read_pat)@github.com/cybozu-go/neco-apps &&
rm cybozu_private_repo_read_pat
}

if ! prepare_necoapps ; then
  echo '[auto-dctest] Failed to checkout neco-apps branch "%s"'
  delete_myself
fi
`, cybozuPrivateRepoReadPATName, b.necoAppsBranch, b.necoAppsBranch)
	}

	if len(b.necoBranch) > 0 {
		s += `
# Run neco
run_neco()
{
cd ${GOPATH}/src/github.com/cybozu-go/neco/dctest &&
make setup placemat MENU_ARG=menu-ss.yml && make test SUITE=bootstrap
}

if run_neco ; then
  echo "[auto-dctest] Neco bootstrap succeeded!"
else
  delete_myself
fi
`
	}

	if len(b.necoAppsBranch) > 0 {
		s += fmt.Sprintf(`
run_necoapps()
{
# Run neco-apps
cd ${GOPATH}/src/github.com/cybozu-go/neco-apps/test &&
gcloud secrets versions access latest --secret="%s" > account.json &&
gcloud secrets versions access latest --secret="%s" > ghcr_dockerconfig.json &&
gcloud secrets versions access latest --secret="%s" > quay_dockerconfig.json &&
gcloud secrets versions access latest --secret="%s" > cybozu_private_repo_read_pat &&
make setup dctest SUITE=bootstrap OVERLAY=neco-dev
}

if run_necoapps ; then
  echo "[auto-dctest] Neco Apps bootstrap succeeded!"
else
  delete_myself
fi
`, necoAppsAccountSecretName, ghcrDockerConfigName, quayDockerConfigName, cybozuPrivateRepoReadPATName)
	}
	return s
}
