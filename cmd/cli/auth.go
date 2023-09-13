package cli

import (
	"riseact/internal/auth"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Do login'",
	Run: func(cmd *cobra.Command, args []string) {
		if err := auth.Login(); err != nil {
			errorExit(err.Error())
		}
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Do logout'",
	Run: func(cmd *cobra.Command, args []string) {
		auth.Logout()
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
}
