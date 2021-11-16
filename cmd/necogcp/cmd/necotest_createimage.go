package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	necogcp "github.com/cybozu-go/neco-gcp"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var necotestCreateImageCmd = &cobra.Command{
	Use:   "create-image",
	Short: "Create vmx-enabled image on neco-test",
	Long:  `Create vmx-enabled image on neco-test.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		necotestCfg := gcp.NecoTestConfig(projectID, zone)
		cc := gcp.NewComputeCLIClient(necotestCfg, "vmx-enabled")
		well.Go(func(ctx context.Context) error {
			return gcp.CreateVMXEnabledImage(ctx, cc, necogcp.VMXEnabledBaseImageProject, necogcp.VMXEnabledBaseImage)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	necotestCmd.AddCommand(necotestCreateImageCmd)
}
