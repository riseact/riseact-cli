package auth

import (
	"fmt"
	"riseact/internal/config"
	"time"
)

func Login() error {
	fmt.Println("Logging in...")

	url, code_verifier := generateAuthorizationURL()

	err := launchBrowser(url)

	if err != nil {
		return err
	}

	code, err := awaitAuthorizationCode()

	if err != nil {
		return err
	}

	token, err := getAccessToken(code, code_verifier)

	if err != nil {
		return err
	}

	userInfo, err := getUserInfo(token.AccessToken)

	if err != nil {
		return err
	}

	err = config.SaveUserSettings(&config.UserSettings{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpireAt:     token.ExpiresAt,
		Name:         userInfo.Name,
		Email:        userInfo.Email,
		PartnerID:    userInfo.PartnerID,
		PartnerName:  userInfo.PartnerName,
	})

	if err != nil {
		return err
	}

	fmt.Println("Login successful")

	return nil
}

func Logout() {
	// TODO: implement logout
	fmt.Println("Logout")
}

// check if user is authenticated
func IsAuthenticated() error {
	userSettings, err := config.GetUserSettings()

	if err != nil {
		return err
	}

	// if is expired
	if userSettings.AccessToken == "" || userSettings.ExpireAt == 0 {
		return fmt.Errorf("Not authenticated. Login first with 'riseact auth login'")
	}

	// ExpiresAt < now
	if userSettings.ExpireAt < (int)(time.Now().Unix()) {
		err = refreshToken()

		if err != nil {
			return err
		}
	}

	return nil
}
