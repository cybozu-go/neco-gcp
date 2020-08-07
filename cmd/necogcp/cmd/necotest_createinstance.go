package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/gcp/functions"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var (
	projectID      string
	zone           string
	machineType    string
	instanceName   string
	necoBranch     string
	necoAppsBranch string
)

const serviceAccountName = "neco-dev"

var necotestCreateInstanceCmd = &cobra.Command{
	Use:   "create-instance",
	Short: "Create dctest env for neco (and neco-apps)",
	Long:  `Create dctest env for neco (and neco-apps).`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if len(instanceName) == 0 {
			log.ErrorExit(errors.New("instance name is required"))
		}
		if len(projectID) == 0 {
			log.ErrorExit(errors.New("project id is required"))
		}
		builder := functions.NewNecoStartupScriptBuilder().WithFluentd()
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
			_, err := builder.WithNecoApps(necoBranch)
			if err != nil {
				log.ErrorExit(fmt.Errorf("failed to create startup script: %v", err))
			}
		}

		well.Go(func(ctx context.Context) error {
			cc, err := functions.NewComputeClient(ctx, projectID, zone)
			if err != nil {
				log.Error("failed to create compute client", map[string]interface{}{
					log.FnError: err,
				})
				return err
			}

			sa := functions.MakeCustomServiceAccountEmail(serviceAccountName, projectID)
			log.Info("start creating instance", map[string]interface{}{
				"project":        projectID,
				"zone":           zone,
				"name":           instanceName,
				"serviceaccount": sa,
				"machinetype":    machineType,
				"necobranch":     necoBranch,
				"necoappsbranch": necoAppsBranch,
			})
			return cc.Create(
				instanceName,
				sa,
				machineType,
				functions.MakeVMXEnabledImageURL(projectID),
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
	necotestCreateInstanceCmd.Flags().StringVarP(&projectID, "project-id", "p", "", "Project ID for GCP")
	necotestCreateInstanceCmd.Flags().StringVarP(&zone, "zone", "z", "asia-northeast1-c", "Zone name for GCP")
	necotestCreateInstanceCmd.Flags().StringVarP(&machineType, "machine-type", "t", "n1-standard-32", "Machine type")
	necotestCreateInstanceCmd.Flags().StringVarP(&instanceName, "instance-name", "n", "", "Instance name")
	necotestCreateInstanceCmd.Flags().StringVar(&necoBranch, "neco-branch", "release", "Branch of neco to run")
	necotestCreateInstanceCmd.Flags().StringVar(&necoAppsBranch, "neco-apps-branch", "release", "Branch of neco-apps to run")
	necotestCmd.AddCommand(necotestCreateInstanceCmd)
}
