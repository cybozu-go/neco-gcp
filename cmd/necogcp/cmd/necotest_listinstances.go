package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var filter string

var necotestListInstancesCmd = &cobra.Command{
	Use:   "list-instances",
	Short: "Get name list of running instances",
	Long:  `Get name list of running instances. This command is created mainly for testing.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			if len(projectID) == 0 {
				log.ErrorExit(errors.New("project id is required"))
			}
			cc, err := gcp.NewComputeClient(ctx, projectID, zone)
			if err != nil {
				log.Error("failed to create compute client: %v", map[string]interface{}{
					log.FnError: err,
				})
				return err
			}
			log.Info("start getting instance list", map[string]interface{}{
				"project": projectID,
				"zone":    zone,
				"filter":  filter,
			})
			set, err := cc.GetNameSet(filter)
			if err != nil {
				log.Error("failed to get instances", map[string]interface{}{
					log.FnError: err,
				})
				return err
			}

			list := []string{}
			for v := range set {
				list = append(list, v)
			}
			log.Info("fetched instance names successfully", map[string]interface{}{
				"names": list,
			})
			return nil
		})

		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	necotestListInstancesCmd.Flags().StringVarP(&filter, "filter", "f", "", "Filter string")
	necotestCmd.AddCommand(necotestListInstancesCmd)
}
