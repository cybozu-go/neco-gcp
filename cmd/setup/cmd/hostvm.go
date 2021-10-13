package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var hostvmCmd = &cobra.Command{
	Use:   "host-vm",
	Short: "setup host-vm instance",
	Long: `setup host-vm instance.

Please run this command on host-vm instance.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}
			if hostname != "host-vm" {
				return errors.New("this host is not supported")
			}
			return gcp.SetupHostVM(ctx)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(hostvmCmd)
}
