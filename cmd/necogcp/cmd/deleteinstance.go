package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var deleteInstanceCmd = &cobra.Command{
	Use:   "delete-instance",
	Short: "Delete host-vm instance",
	Long:  `Delete host-vm instance manually.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cc := gcp.NewComputeCLIClient(cfg, "host-vm")
		well.Go(func(ctx context.Context) error {
			err := cc.DeleteInstance(ctx)
			if err != nil {
				return err
			}

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
	rootCmd.AddCommand(deleteInstanceCmd)
}
