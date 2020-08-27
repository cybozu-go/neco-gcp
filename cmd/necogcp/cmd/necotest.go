package cmd

import (
	"github.com/spf13/cobra"
)

const zone = "asia-northeast1-c"

var projectID string

// necotestCmd is the root subcommand of "necogcp neco-test".
var necotestCmd = &cobra.Command{
	Use:   "neco-test",
	Short: "neco-test related commands",
	Long:  `neco-test related commands.`,
}

func init() {
	necotestCmd.PersistentFlags().StringVarP(&projectID, "project-id", "p", "neco-test", "Project ID for GCP")
	rootCmd.AddCommand(necotestCmd)
}
