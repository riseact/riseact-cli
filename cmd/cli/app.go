package cli

import (
	app "riseact/internal/app"
	"riseact/internal/auth"

	"github.com/spf13/cobra"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "App management commands",
}

var appInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize app",
	Run: func(cmd *cobra.Command, args []string) {
		if err := auth.IsAuthenticated(); err != nil {
			errorExit(err.Error())
		}

		err := app.DoAppInit()

		if err != nil {
			errorExit(err.Error())
		}
	},
}

var appStartDevelopmentCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start app",
	Run: func(cmd *cobra.Command, args []string) {
		if err := auth.IsAuthenticated(); err != nil {
			errorExit(err.Error())
		}

		err := app.StartDevEnvironment()

		if err != nil {
			errorExit(err.Error())
		}

	},
}

func init() {
	rootCmd.AddCommand(appCmd)
	appCmd.AddCommand(appInitCmd)
	appCmd.AddCommand(appStartDevelopmentCmd)
}
