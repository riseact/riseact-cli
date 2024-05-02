package app

import (
	"context"
	"fmt"
	"net"
	"riseact/internal/config"
	"riseact/internal/utils/logger"

	"github.com/AlecAivazis/survey/v2"
	"golang.ngrok.com/ngrok"
	ngrokConfig "golang.ngrok.com/ngrok/config"
)

func isPortInUse(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return true
	}
	defer listener.Close()
	return false
}

func StartNgrokTunnel() (ngrok.Tunnel, error) {
	settings, err := config.GetUserSettings()

	if err != nil {
		return nil, err
	}

	// check ngrok configuration
	if err = ensureNgrokSetup(); err != nil {
		return nil, err
	}

	// check if port is already in use by another ngrok tunnel
	if isPortInUse(8080) {
		return nil, fmt.Errorf("port is already in use by another ngrok tunnel")
	}

	fmt.Println("Starting ngrok tunnel...")
	tun, err := ngrok.Listen(context.Background(),
		ngrokConfig.HTTPEndpoint(),
		ngrok.WithAuthtoken(settings.NgrokToken),
	)

	fmt.Println("Tunnel started successfully, url: ", tun.URL())

	if err != nil {
		logger.Debug("Error starting ngrok tunnel")
		return nil, err
	}

	return tun, nil
}

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

		fmt.Println("Token saved successfully")

		if err != nil {
			return err
		}
	}

	return nil
}
