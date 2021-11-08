package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/autodctest"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var (
	machineType         string
	createInstanceName  string
	serviceAccountEmail string
	necoBranch          string
	necoAppsBranch      string
)

var necotestCreateInstanceCmd = &cobra.Command{
	Use:   "create-instance",
	Short: "Create dctest env for neco (and neco-apps)",
	Long:  `Create dctest env for neco (and neco-apps).`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if len(createInstanceName) == 0 {
			log.ErrorExit(errors.New("instance name is required"))
		}
		if len(projectID) == 0 {
			log.ErrorExit(errors.New("project id is required"))
		}
		if len(serviceAccountEmail) == 0 {
			serviceAccountEmail = autodctest.MakeNecoDevServiceAccountEmail(projectID)
			log.Info("Use default service account", map[string]interface{}{
				"serviceaccount": serviceAccountEmail,
			})
		}
		builder := autodctest.NewStartupScriptBuilder().WithFluentd()
		if len(necoBranch) > 0 {
			log.Info("run neco", map[string]interface{}{
				"branch": necoBranch,
			})
			builder.WithNeco(necoBranch)
		}
		if len(necoAppsBranch) > 0 {
			log.Info("run neco-apps", map[string]interface{}{
				"branch": necoAppsBranch,
			})
			_, err := builder.WithNecoApps(necoAppsBranch)
			if err != nil {
				log.ErrorExit(fmt.Errorf("failed to create startup script: %v", err))
			}
		}

		well.Go(func(ctx context.Context) error {
			cc, err := gcp.NewComputeClient(ctx, projectID, zone)
			if err != nil {
				log.Error("failed to create compute client", map[string]interface{}{
					log.FnError: err,
				})
				return err
			}

			log.Info("start creating instance", map[string]interface{}{
				"project":        projectID,
				"zone":           zone,
				"name":           createInstanceName,
				"serviceaccount": serviceAccountEmail,
				"machinetype":    machineType,
				"necobranch":     necoBranch,
				"necoappsbranch": necoAppsBranch,
			})
			return cc.Create(
				createInstanceName,
				serviceAccountEmail,
				machineType,
				gcp.MakeVMXEnabledImageURL(projectID),
				builder.Build(),
			)
		})

		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	necotestCreateInstanceCmd.Flags().StringVarP(&machineType, "machine-type", "t", "n1-highmem-32", "Machine type")
	necotestCreateInstanceCmd.Flags().StringVarP(&createInstanceName, "instance-name", "n", "", "Instance name")
	necotestCreateInstanceCmd.Flags().StringVarP(&serviceAccountEmail, "service-account", "a", "", "Service account email address")
	necotestCreateInstanceCmd.Flags().StringVar(&necoBranch, "neco-branch", "release", "Branch of cybozu-go/neco to run")
	necotestCreateInstanceCmd.Flags().StringVar(&necoAppsBranch, "neco-apps-branch", "release", "Branch of cybozu-go/neco-apps to run")
	necotestCmd.AddCommand(necotestCreateInstanceCmd)
}
