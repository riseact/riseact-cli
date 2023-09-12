package cli

import (
	"riseact/internal/auth"
	"riseact/internal/config"
	c "riseact/internal/theme/services"
	"riseact/internal/utils/logger"

	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Theme management commands",
}

var themeInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Clones a Git repository to your local machine to use as the starting point for building a theme.",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Cloning remote theme repository...")
		repo, _ := cmd.Flags().GetString("repository")

		err := c.InitTheme(repo)

		if err != nil {
			errorExit(err.Error())
		}

		logger.Info("Theme successfully initialized")
	},
}

var themeDevCmd = &cobra.Command{
	Use:   "dev",
	Short: "Uploads the current theme to an organization so you can preview it.",
	Run: func(cmd *cobra.Command, args []string) {
		path, _ := cmd.Flags().GetString("path")

		if err := auth.IsAuthenticated(); err != nil {
			errorExit(err.Error())
		}

		err := c.StartDevEnvironment(path)

		if err != nil {
			errorExit(err.Error())
		}
	},
}

var themePushCmd = &cobra.Command{
	Use:   "push",
	Short: "Uploads your local theme files to Riseact, overwriting the remote version if specified.",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Push theme")
		if err := auth.IsAuthenticated(); err != nil {
			errorExit(err.Error())
		}

		path, _ := cmd.Flags().GetString("path")

		logger.Debug("Push theme path: " + path)

		if err := c.Push(path); err != nil {
			errorExit(err.Error())
		}
	},
}

var themePackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Creates a zip file of your theme that you can use to upload to Riseact.",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Package theme")
		path, _ := cmd.Flags().GetString("path")
		_, err := c.PackageCreate(path, ".")

		if err != nil {
			errorExit(err.Error())
		}

	},
}

func init() {
	rootCmd.AddCommand(themeCmd)

	themeInitCmd.Flags().StringP("repository", "r", config.BaseThemeRepo, "Theme remote repository")
	themeCmd.AddCommand(themeInitCmd)

	themeDevCmd.Flags().StringP("path", "p", ".", "Theme path")
	themeCmd.AddCommand(themeDevCmd)

	themePushCmd.Flags().StringP("path", "p", ".", "Theme path")
	themeCmd.AddCommand(themePushCmd)

	themePackageCmd.Flags().StringP("path", "p", ".", "Theme path")
	themeCmd.AddCommand(themePackageCmd)
}
