package app

import (
	"context"
	"fmt"
	"riseact/internal/config"

	"github.com/AlecAivazis/survey/v2"
	"golang.ngrok.com/ngrok"
	ngrokConfig "golang.ngrok.com/ngrok/config"
)

func ensureNgrokSetup() error {
	settings, err := config.GetUserSettings()

	if err != nil {
		return err
	}

	if settings.NgrokToken == "" {
		fmt.Println("Please provide your ngrok token")
		prompt := &survey.Input{
			Message: "Ngrok token",
		}
		survey.AskOne(prompt, &settings.NgrokToken)

		if settings.NgrokToken == "" {
			return fmt.Errorf("Ngrok token is required")
		}

		err = config.SaveUserSettings(settings)

		if err != nil {
			return err
		}
	}

	return nil
}

func startNgrokTunnel() (ngrok.Tunnel, error) {
	settings, err := config.GetUserSettings()

	if err != nil {
		return nil, err
	}

	// check ngrok configuration
	if err = ensureNgrokSetup(); err != nil {
		return nil, err
	}

	tun, err := ngrok.Listen(context.Background(),
		ngrokConfig.HTTPEndpoint(),
		ngrok.WithAuthtoken(settings.NgrokToken),
	)

	return tun, nil
}
