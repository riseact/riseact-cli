package services

import (
	"fmt"
	"riseact/internal/config"
	"strconv"
)

func PrintSettings() {
	PrintAppSettings()
	PrintUserSettings()
}

func PrintAppSettings() {
	settings := config.GetAppSettings()

	println("Accounts host: " + settings.AccountsHost)
	println("Core host: " + settings.CoreHost)
	println("Admin host: " + settings.AdminHost)
	println("Client ID: " + settings.ClientId)
	println("Redirect URI: " + settings.RedirectUri)
	println("Debug level: " + settings.DebugLevel.String())

}

func PrintUserSettings() error {
	settings, err := config.GetUserSettings()

	if err != nil {
		return fmt.Errorf("error reading user settings: %w", err)
	}

	println("Name: " + settings.Name)
	println("Email: " + settings.Email)
	println("Partner ID: " + strconv.Itoa(settings.PartnerID))
	println("Partner name: " + settings.PartnerName)
	println("Access token: " + settings.AccessToken)
	println("Refresh token: " + settings.RefreshToken)
	println("Expire at: " + strconv.Itoa(settings.ExpireAt))

	return nil
}
