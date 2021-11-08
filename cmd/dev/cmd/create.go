package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/autodctest"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

const (
	machineType        = "n1-highmem-32"
	serviceAccountName = "default"
)

var (
	projectID          string
	zone               string
	instanceNamePrefix string
	instancesNum       int
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create multiple GCP instances with running dctest",
	Long: `Create multiple GCP instances with running dctest.

NOTE:
This command is created only for testing.
Please push "Run Now" button on Cloud Scheduler when running dctest`,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			if len(instanceNamePrefix) == 0 {
				log.ErrorExit(errors.New("instance name is required"))
			}
			if len(projectID) == 0 {
				log.ErrorExit(errors.New("project ID is required"))
			}

			cc, err := gcp.NewComputeClient(ctx, projectID, zone)
			if err != nil {
				log.Error("failed to create compute client", map[string]interface{}{
					log.FnError: err,
				})
				return err
			}
			log.Info("start creating instance", map[string]interface{}{
				"project":            projectID,
				"zone":               zone,
				"instancenameprefix": instanceNamePrefix,
				"instancesnum":       instancesNum,
			})
			runner := autodctest.NewRunner(cc)
			return runner.CreateInstancesIfNotExist(
				ctx,
				instanceNamePrefix,
				instancesNum,
				serviceAccountName,
				machineType,
				gcp.MakeVMXEnabledImageURL(projectID),
				"",
			)
		})

		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
		return nil
	},
}

func init() {
	createCmd.Flags().StringVarP(&projectID, "project-id", "p", "", "Project ID for GCP")
	createCmd.Flags().StringVarP(&zone, "zone", "z", "asia-northeast1-c", "Zone name for GCP")
	createCmd.Flags().StringVarP(&instanceNamePrefix, "name-prefix", "n", "", "Instance name prefix")
	createCmd.Flags().IntVarP(&instancesNum, "instances-num", "i", 1, "Instance num to create")
	rootCmd.AddCommand(createCmd)
}
