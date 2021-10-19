package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var createInstanceCmd = &cobra.Command{
	Use:   "create-instance",
	Short: "Launch host-vm instance",
	Long: `Launch host-vm instance using vmx-enabled image.

If host-vm instance already exists in the project, it is re-created.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cc := gcp.NewComputeCLIClient(cfg, "host-vm")
		well.Go(func(ctx context.Context) error {
			cc.DeleteInstance(ctx)

			err := cc.CreateHostVMInstance(ctx)
			if err != nil {
				return err
			}

			err = cc.WaitInstance(ctx)
			if err != nil {
				return err
			}

			err = cc.CreateHomeDisk(ctx)
			if err != nil {
				return err
			}

			err = cc.ResizeHomeDisk(ctx)
			if err != nil {
				return err
			}

			err = cc.AttachHomeDisk(ctx)
			if err != nil {
				return err
			}

			return cc.RunSetupHostVM(ctx)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
		fmt.Println("host-vm has been created! Ready to login")
	},
}

func init() {
	rootCmd.AddCommand(createInstanceCmd)
}
