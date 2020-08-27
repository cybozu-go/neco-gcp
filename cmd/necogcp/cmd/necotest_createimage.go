package cmd

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco-gcp/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
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
			f, err := ioutil.TempFile("", "*.yml")
			if err != nil {
				return err
			}
			defer func() {
				f.Close()
				os.Remove(f.Name())
			}()

			data, err := yaml.Marshal(necotestCfg)
			if err != nil {
				return err
			}
			_, err = f.Write(data)
			if err != nil {
				return err
			}

			err = f.Sync()
			if err != nil {
				return err
			}

			return gcp.CreateVMXEnabledImage(ctx, cc, f.Name())
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	necotestCreateImageCmd.Flags().StringVarP(&projectID, "project-id", "p", "neco-test", "Project ID for GCP")
	necotestCreateImageCmd.Flags().StringVarP(&zone, "zone", "z", "asia-northeast2-c", "Zone name for GCP")
	necotestCmd.AddCommand(necotestCreateImageCmd)
}
