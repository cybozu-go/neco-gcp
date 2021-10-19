package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var deleteInstanceName string

var necotestDeleteInstanceCmd = &cobra.Command{
	Use:   "delete-instance",
	Short: "Delete instance",
	Long:  `Delete instance.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if len(deleteInstanceName) == 0 {
			log.ErrorExit(errors.New("instance name is required"))
		}
		if len(projectID) == 0 {
			log.ErrorExit(errors.New("project id is required"))
		}
		well.Go(func(ctx context.Context) error {
			cc, err := gcp.NewComputeClient(ctx, projectID, zone)
			if err != nil {
				log.Error("failed to create compute client", map[string]interface{}{
					log.FnError: err,
				})
				return err
			}
			log.Info("start deleting instance", map[string]interface{}{
				"project": projectID,
				"zone":    zone,
				"name":    deleteInstanceName,
			})
			return cc.Delete(deleteInstanceName)
		})

		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	necotestDeleteInstanceCmd.Flags().StringVarP(&deleteInstanceName, "instance-name", "n", "", "Instance name")
	necotestCmd.AddCommand(necotestDeleteInstanceCmd)
}
