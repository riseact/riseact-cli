package cli

import (
	app "riseact/internal/app/services"
	"riseact/internal/auth"
	"riseact/internal/utils/logger"

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

var appProxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Init app and start proxy",
	Run: func(cmd *cobra.Command, args []string) {
		if err := auth.IsAuthenticated(); err != nil {
			errorExit(err.Error())
		}

		port, _ := cmd.Flags().GetString("port")

		logger.Debug("Port: " + port)

		if err := app.ProxyApp(port); err != nil {
			errorExit(err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(appCmd)

	appCmd.AddCommand(appInitCmd)

	appProxyCmd.Flags().StringP("port", "p", ".", "Port")
	appCmd.AddCommand(appProxyCmd)

	appCmd.AddCommand(appStartDevelopmentCmd)
}
