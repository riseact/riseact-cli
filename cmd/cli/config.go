package cli

import (
	config "riseact/internal/config/services"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print CLI configuration",

	Run: func(cmd *cobra.Command, args []string) {
		config.PrintSettings()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
