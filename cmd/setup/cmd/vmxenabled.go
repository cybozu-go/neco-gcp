package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/pkg/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var vmxenabledCmd = &cobra.Command{
	Use:   "vmx-enabled PROJECT OPTIONAL_PACKAGES_FILE",
	Short: "setup vmx-enabled instance",
	Long: `setup vmx-enabled instance.

Please run this command on vmx-enabled instance.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}
			if hostname != "vmx-enabled" {
				return errors.New("this host is not supported")
			}

			data, err := os.ReadFile(args[1])
			if err != nil {
				return err
			}
			var optionalPackages []string
			if err := json.Unmarshal(data, &optionalPackages); err != nil {
				return err
			}

			return gcp.SetupVMXEnabled(ctx, args[0], optionalPackages)
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(vmxenabledCmd)
}
