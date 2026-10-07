package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cybozu-go/neco-gcp/pkg/gcp"
)

var (
	cfgFile string
	cfg     *gcp.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "necogcp",
	Short: "necogcp is GCP management tool for Neco project",
	Long:  `necogcp is GCP management tool for Neco project.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// without this, each subcommand's RunE would display usage text.
		cmd.SilenceUsage = true

		err := well.LogConfig{}.Apply()
		if err != nil {
			return err
		}

		cfg, err = gcp.NewConfig()
		if err != nil {
			return err
		}

		yamlTagOption := func(c *mapstructure.DecoderConfig) {
			c.TagName = "yaml"
		}
		return viper.Unmarshal(cfg, yamlTagOption)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", filepath.Join(os.Getenv("HOME"), ".necogcp.yml"), "config file")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.ErrorExit(err)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".necogcp")
		viper.SetConfigType("yml")
	}

	if err := viper.ReadInConfig(); err != nil {
		// the config file is optional when --config is left at its default path,
		// so only report errors other than "file not found" in that case
		var notFound viper.ConfigFileNotFoundError
		explicit := rootCmd.PersistentFlags().Changed("config")
		if explicit || (!errors.As(err, &notFound) && !errors.Is(err, fs.ErrNotExist)) {
			log.ErrorExit(err)
		}
	}
}
