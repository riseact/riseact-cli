package cli

import (
	"fmt"
	"os"
	"riseact/internal/config"
	"riseact/internal/utils/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "riseact",
	Short: "Manage your riseact resources from the command line",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		settings := config.GetAppSettings()

		logger.InitLogging(settings.DebugLevel)

		if dir := viper.GetString("working-directory"); dir != "" {
			fmt.Println("Setting working directory to", dir)
			err := os.Chdir(dir)
			if err != nil {
				logger.Errorf("Cannot set working directory: %s", err.Error())
			}
		}

		config.LoadConfig()
	},
}

func errorExit(message string) {
	logger.Error(message)
	os.Exit(1)
}

func init() {
	rootCmd.PersistentFlags().StringP("working-dir", "w", "", "Set current working directory")
	viper.BindPFlag("working-directory", rootCmd.PersistentFlags().Lookup("w"))
}

func Execute() error {
	return rootCmd.Execute()
}
