package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"

	"github.com/cybozu-go/neco-gcp/pkg/autodctest"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete all GCP instances in a project",
	Long: `delete all GCP instances in a project.

NOTE:
This command is created only for testing.
Please DO NOT use this command except for the purpose.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		well.Go(func(ctx context.Context) error {
			if projectID == "neco-test" || projectID == "neco-dev" {
				log.Info("this operation is not permitted", map[string]any{})
				return nil
			}

			cc, err := gcp.NewComputeClient(ctx, projectID, zone)
			if err != nil {
				log.Error("failed to create compute client", map[string]any{
					log.FnError: err,
				})
				return err
			}
			log.Info("start deleting instance", map[string]any{
				"project": projectID,
				"zone":    zone,
			})
			runner := autodctest.NewRunner(cc)
			return runner.DeleteFilteredInstances(ctx, "")
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
	deleteCmd.Flags().StringVarP(&projectID, "project-id", "p", "", "Project ID for GCP")
	deleteCmd.Flags().StringVarP(&zone, "zone", "z", "asia-northeast1-c", "Zone name for GCP")
	rootCmd.AddCommand(deleteCmd)
}
