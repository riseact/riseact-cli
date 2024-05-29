package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const BaseThemeRepo = "https://github.com/riseact/lancio.git"

type UserSettings struct {
	AccessToken  string `json:"access_token" mapstructure:"access_token"`
	RefreshToken string `json:"refresh_token" mapstructure:"refresh_token"`
	ExpireAt     int    `json:"expire_at" mapstructure:"expire_at"`
	Name         string `json:"name" mapstructure:"name"`
	Email        string `json:"email" mapstructure:"email"`
	PartnerID    int    `json:"partner_id" mapstructure:"partner_id"`
	PartnerName  string `json:"partner_name" mapstructure:"partner_name"`
	NgrokToken   string `json:"ngrok_token" mapstructure:"ngrok_token"`
}

type AppSettings struct {
	AccountsHost string
	CoreHost     string
	AdminHost    string
	ClientId     string
	RedirectUri  string
	DebugLevel   logrus.Level
}

var production = &AppSettings{
	AccountsHost: "https://accounts.riseact.org",
	CoreHost:     "https://core.riseact.org",
	AdminHost:    "https://admin.riseact.org",
	RedirectUri:  "http://localhost:55443",
	ClientId:     "oigtjb908t2i3lnkjvgfjdSFGHY43gk90ufsdfsd",
	DebugLevel:   logrus.InfoLevel,
}

var staging = &AppSettings{
	AccountsHost: "https://accounts.riseact.xyz",
	CoreHost:     "https://core.riseact.xyz",
	AdminHost:    "https://admin.riseact.xyz",
	RedirectUri:  "http://localhost:55443",
	ClientId:     "oigtjb908t2i3lnkjvgfjdSFGHY43gk90ufsdfsd",
	DebugLevel:   logrus.DebugLevel,
}

var development = &AppSettings{
	AccountsHost: "http://accounts.localhost:8000",
	CoreHost:     "http://core.localhost:8000",
	AdminHost:    "http://admin.localhost:4500",
	RedirectUri:  "http://localhost:55443",
	ClientId:     "VUpZM6CDKimnGgCSkzxLWsHP60DytxfHeRJXgVA2",
	DebugLevel:   logrus.DebugLevel,
}

func GetAppSettings() *AppSettings {
	env := os.Getenv("RA_ENV")

	if env == "dev" {
		return development
	}

	if env == "staging" {
		return staging
	}

	return production
}

func getConfigDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Adjust based on your platform/app
	path := filepath.Join(home, ".config", "riseact.yml")
	// create directory if it doesn't exist

	if _, err := os.Stat(filepath.Join(home, ".config")); os.IsNotExist(err) {
		// Create directory with 0700 permissions (read/write/execute only for the user)
		os.MkdirAll(path, 0700)
	}

	return path, nil
}

func initDefaultConfig() {
	// set default user settings

	viper.SetDefault("access_token", "")
	viper.SetDefault("refresh_token", "")
	viper.SetDefault("expire_at", 0)
	viper.SetDefault("name", "")
	viper.SetDefault("email", "")
	viper.SetDefault("partner_id", 0)
	viper.SetDefault("partner_name", "")
	viper.SetDefault("ngrok_token", "")

	viper.AutomaticEnv()
}

func GetUserSettings() (*UserSettings, error) {
	settings := &UserSettings{
		AccessToken:  viper.GetString("access_token"),
		RefreshToken: viper.GetString("refresh_token"),
		ExpireAt:     viper.GetInt("expire_at"),
		Name:         viper.GetString("name"),
		Email:        viper.GetString("email"),
		PartnerID:    viper.GetInt("partner_id"),
		PartnerName:  viper.GetString("partner_name"),
		NgrokToken:   viper.GetString("ngrok_token"),
	}

	return settings, nil
}

func SaveUserSettings(settings *UserSettings) error {
	viper.Set("access_token", settings.AccessToken)
	viper.Set("refresh_token", settings.RefreshToken)
	viper.Set("expire_at", settings.ExpireAt)
	viper.Set("name", settings.Name)
	viper.Set("email", settings.Email)
	viper.Set("partner_id", settings.PartnerID)
	viper.Set("partner_name", settings.PartnerName)
	viper.Set("ngrok_token", settings.NgrokToken)

	configDir, err := getConfigDirectory()

	if err != nil {
		return fmt.Errorf("error getting config directory: %s", err)
	}

	if err := viper.WriteConfigAs(configDir); err != nil {
		return fmt.Errorf("error writing config file: %s", err)
	}

	return nil
}

func LoadConfig() error {
	initDefaultConfig()
	configDir, err := getConfigDirectory()

	if err != nil {
		return fmt.Errorf("error getting config directory: %s", err)
	}

	viper.SetConfigName("riseact.yml")
	viper.AddConfigPath(configDir)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Fprintf(os.Stderr, "No configuration file loaded - using defaults\n")
		} else {
			panic("Error reading config file:" + err.Error())
		}
	}

	return nil
}
