package cmd

import (
	"github.com/spf13/cobra"
)

var (
	projectID string
	zone      string
)

// necotestCmd is the root subcommand of "necogcp neco-test".
var necotestCmd = &cobra.Command{
	Use:   "neco-test",
	Short: "neco-test related commands",
	Long:  `neco-test related commands.`,
}

func init() {
	necotestCmd.PersistentFlags().StringVarP(&projectID, "project-id", "p", "neco-test", "Project ID for GCP")
	necotestCmd.PersistentFlags().StringVarP(&zone, "zone", "z", "asia-northeast1-c", "Zone name for GCP")
	rootCmd.AddCommand(necotestCmd)
}
