package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"github.com/cybozu-go/neco-gcp/pkg/github"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

const secretPrefix = "ghatoken"
const defaultRunnerLabel = "gcp-runner"

var (
	runnerMachineType    string
	runnerNumLocalSSDs   int
	runnerNecoBranch     string
	runnerNecoAppsBranch string
	runnerProjectId      string
)

var (
	appId          int64
	installationId int64
	privateKeyPath string
	pat            string
	runnerRepo     string
	runnerName     string
	labels         []string
)

var runnerStartUpScriptTmpl = template.Must(template.New("").Parse(`#!/bin/sh

# Wait for auto-refresh to be done
snap watch --last=auto-refresh

useradd -m runner
usermod -aG google-sudoers runner
RUNNER_HOME=/home/runner
/snap/bin/gcloud secrets versions access latest --secret="{{ .SecretID }}" > $RUNNER_HOME/github_token
cat << EOF >job-cancelled
#!/bin/sh
EOF
chmod 755 job-cancelled
cp job-cancelled /usr/local/bin/job-started
cp job-cancelled /usr/local/bin/job-failure
cp job-cancelled /usr/local/bin/job-success
mv job-cancelled /usr/local/bin/job-cancelled

NUM_DEVICES=$(ls /dev/nvme0n*|wc -l)
mdadm --create /dev/md0 -l stripe --raid-devices=${NUM_DEVICES} /dev/nvme0n*
mkfs -t ext4 -F /dev/md0
mkdir -p /var/scratch
mount -t ext4 /dev/md0 /var/scratch
chmod 1777 /var/scratch

latest=$(curl -sSf https://api.github.com/repos/actions/runner/releases/latest | jq -r '.tag_name' | sed -e 's/v//')
curl -L -o $RUNNER_HOME/runner.tar.gz https://github.com/actions/runner/releases/download/v"$latest"/actions-runner-linux-x64-"$latest".tar.gz
tar xzf $RUNNER_HOME/runner.tar.gz -C $RUNNER_HOME
$RUNNER_HOME/bin/installdependencies.sh

cat << 'EOT' >$RUNNER_HOME/run-actions.sh
#!/bin/sh
REPO_URL=https://github.com/{{ .RunnerRepo }}

export NECO_DIR=${GOPATH}/src/github.com/cybozu-go/neco
export NECO_APPS_DIR=${GOPATH}/src/github.com/cybozu-private/neco-apps

$HOME/config.sh \
	--unattended \
	--replace \
	--name $(hostname) \
	--labels {{ .Labels }} \
	--url $REPO_URL \
	--token $(cat $HOME/github_token) \
	--work $HOME \
	--ephemeral

code=-1
while  [ "$code" -ne 1 ] && [ "$code" -ne 0 ]
do
	$HOME/bin/Runner.Listener run --startuptype service
	code=$?
done

EOT

chmod 755 $RUNNER_HOME/run-actions.sh
sudo -i -u runner $RUNNER_HOME/run-actions.sh
`))

var createRunnerCmd = &cobra.Command{
	Use:   "create-runner",
	Short: "Launch runner instance which runs self-hosted-runner",
	Long: `Launch runner instance which runs self-hosted-runner with vmx-enabled image.

If runner instance already exists in the project, new runner is not created.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		splitRunnerRepo := strings.Split(runnerRepo, "/")
		owner := splitRunnerRepo[0]
		repository := splitRunnerRepo[1]
		secretID := fmt.Sprintf("%s_%s", secretPrefix, runnerName)
		labels := strings.Join(append(labels, defaultRunnerLabel), ",")

		buf := new(bytes.Buffer)
		err := runnerStartUpScriptTmpl.Execute(buf, struct {
			SecretID   string
			RunnerRepo string
			Labels     string
		}{
			SecretID:   secretID,
			RunnerRepo: runnerRepo,
			Labels:     labels,
		})
		if err != nil {
			log.ErrorExit(err)
		}
		startupScript := buf.String()

		well.Go(func(ctx context.Context) error {
			secretClient, err := gcp.NewSecretClient(ctx, runnerProjectId)
			if err != nil {
				return err
			}
			exists, err := secretClient.Exists(secretID)
			if err != nil {
				return err
			} else if !exists {
				registrationToken, expireAt, err := github.CreateRegistrationToken(owner, repository, pat, privateKeyPath, appId, installationId)
				if err != nil {
					return err
				}
				err = secretClient.CreateSecretFromData(expireAt, secretID, []byte(registrationToken))
				if err != nil {
					return err
				}
			}

			computeClient, err := gcp.NewComputeClient(ctx, runnerProjectId, zone)
			if err != nil {
				log.Error("failed to create compute client", map[string]interface{}{
					log.FnError: err,
				})
				return err
			}
			serviceaccount := fmt.Sprintf("bootstrap-dctest@%s.iam.gserviceaccount.com", runnerProjectId)
			log.Info("start creating instance", map[string]interface{}{
				"project":        runnerProjectId,
				"zone":           zone,
				"name":           runnerName,
				"serviceaccount": serviceaccount,
				"machinetype":    runnerMachineType,
				"necobranch":     runnerNecoBranch,
				"necoappsbranch": runnerNecoAppsBranch,
			})
			return computeClient.Create(
				runnerName,
				serviceaccount,
				runnerMachineType,
				runnerNumLocalSSDs,
				gcp.MakeVMXEnabledImageURL(runnerProjectId),
				startupScript,
			)
		})

		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
		fmt.Println("Runner has been created!")
	},
}

func init() {
	createRunnerCmd.Flags().Int64VarP(&appId, "app-id", "", 0, "GitHub App Id")
	createRunnerCmd.Flags().Int64VarP(&installationId, "app-installation-id", "", 0, "GitHub App Installation Id")
	createRunnerCmd.Flags().StringVarP(&privateKeyPath, "private-key-path", "", "", "Path of Private key for GitHub App")
	createRunnerCmd.Flags().StringVarP(&pat, "pat", "", "", "Personal Access Token")
	createRunnerCmd.Flags().StringSliceVarP(&labels, "labels", "", []string{}, "Labels added to runner")
	createRunnerCmd.Flags().StringVarP(&runnerRepo, "runner-repository", "", "", "GitHub Repository name which formatted in owner/repository")
	createRunnerCmd.MarkFlagRequired("runner-repository")
	createRunnerCmd.Flags().StringVarP(&runnerName, "runner-name", "", "", "Actions runner name which formatted in '^[a-z]([-a-z0-9]*[a-z0-9])?'")
	createRunnerCmd.MarkFlagRequired("runner-name")
	createRunnerCmd.Flags().StringVarP(&runnerMachineType, "machine-type", "t", "n1-standard-64", "Machine type")
	createRunnerCmd.Flags().IntVarP(&runnerNumLocalSSDs, "local-ssd", "s", 4, "Number of local SSDs")
	createRunnerCmd.Flags().StringVar(&runnerNecoBranch, "neco-branch", "release", "Branch of cybozu-go/neco to run")
	createRunnerCmd.Flags().StringVar(&runnerNecoAppsBranch, "neco-apps-branch", "release", "Branch of cybozu-private/neco-apps to run")
	createRunnerCmd.Flags().StringVarP(&runnerProjectId, "project-id", "p", "neco-test", "Project ID for GCP")
	rootCmd.AddCommand(createRunnerCmd)
}
