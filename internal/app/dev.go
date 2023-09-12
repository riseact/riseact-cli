package app

import (
	"fmt"
	"os"
	"riseact/internal/gql"
	"riseact/internal/utils/logger"

	"github.com/AlecAivazis/survey/v2"
)

func StartDevEnvironment() error {
	logger.Debug("Starting dev environment...")
	var app *Application

	// start ngrok tunnel
	tun, err := startNgrokTunnel()

	if err != nil {
		return err
	}

	// initialize app
	if app == nil {
		app, err = initApp(tun.URL())

		if err != nil {
			logger.Debugf("Error initializing app: %s", err.Error())
			return err
		}
	}

	// print infos
	logger.Info("")
	logger.Infof("App url: %s", tun.URL())
	logger.Info("")

	proxy := NewReverseProxy(&tun, "http://localhost:3000", "http://localhost:3001")

	os.Setenv("RISEACT_APP_URL", tun.URL())

	// start reverse proxy server
	go proxy.Launch()

	// start web app
	err = app.launch()

	if err != nil {
		return err
	}

	return nil
}

func initApp(host string) (*Application, error) {
	var app *Application

	redirectUri := fmt.Sprintf("%s/auth/callback", host)

	appEnv, err := LoadEnv()

	if err != nil {
		return nil, fmt.Errorf("Error loading app env: %s", err.Error())
	}

	appEnv.RiseactAppUrl = host

	err = IsValidApp(".")

	if err != nil {
		return nil, err
	}

	// retrieve app by client_id
	if appEnv.ClientId != "" {
		app, _ = GetAppByClientId(appEnv.ClientId)
		if app != nil {
			logger.Debugf("Existing App: %v\n", app.Name)
			// update app redirect_uri with ngrok url
			appEnv.store()
			app.updateAppRedirectUri(redirectUri)
			return app, nil
		}
	}

	logger.Info("App not configured, do you want to create a new one or link to an existing one?")

	create := false

	prompt := &survey.Confirm{
		Message: "Create a new app?",
	}
	survey.AskOne(prompt, &create)

	if create {
		appData, err := createAppForm()

		if err != nil {
			return nil, err
		}

		app, err = NewApp(appData)

		if err != nil {
			return nil, err
		}

	} else {
		app, err = selectExistingApp(appEnv)

		if err != nil {
			return nil, err
		}

	}

	appEnv.ClientId = app.ClientId
	appEnv.ClientSecret = app.ClientSecret

	appEnv.store()
	app.updateAppRedirectUri(redirectUri)

	logger.Infof("App configured successfully. Client ID: " + appEnv.ClientId)

	return app, nil
}

func selectExistingApp(e *AppEnv) (*Application, error) {
	partnerApps, err := GetApps()

	if err != nil {
		return nil, err
	}

	var appIds []string
	var apps map[string]*Application = make(map[string]*Application)

	for i, _ := range partnerApps {
		appIds = append(appIds, partnerApps[i].ClientId)
		apps[partnerApps[i].ClientId] = &partnerApps[i]
	}
	prompt := &survey.Select{
		Message: "Select an app",
		Options: appIds,
		Description: func(id string, index int) string {
			return apps[id].Name
		},
	}

	survey.AskOne(prompt, &e.ClientId)

	return apps[e.ClientId], nil
}

func createAppForm() (gql.AppInput, error) {
	var name string

	namePrompt := &survey.Input{
		Message: "App name",
	}
	survey.AskOne(namePrompt, &name)

	typePrompt := &survey.Select{
		Message: "Select an app type",
		Options: []string{"PUBLIC", "PRIVATE"},
	}

	var appTypeAnswer string

	survey.AskOne(typePrompt, &appTypeAnswer)

	// TODO: ask other basic questions

	logger.Info("Creating app " + name + " of type " + appTypeAnswer)

	appType := gql.ApplicationType(appTypeAnswer)

	return gql.AppInput{
		Name:         name,
		Type:         appType,
		RedirectUris: "",
	}, nil
}
