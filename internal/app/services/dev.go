package services

import (
	"fmt"
	"os"
	"riseact/internal/app"
	"riseact/internal/config"
	"riseact/internal/gql"
	"riseact/internal/organizations"
	"riseact/internal/utils/logger"

	"github.com/AlecAivazis/survey/v2"
)

func StartDevEnvironment() error {
	logger.Debug("Starting dev environment...")

	var a *app.Application

	settings := config.GetAppSettings()

	// start ngrok tunnel
	tun, err := app.StartNgrokTunnel()

	if err != nil {
		return err
	}

	// initialize app
	if a == nil {
		a, err = initApp(tun.URL())

		if err != nil {
			logger.Debugf("Error initializing app: %s", err.Error())
			return err
		}
	}

	// print infos
	logger.Infof("App url: %s", tun.URL())

	proxy := app.NewReverseProxy(&tun, "http://localhost:3000", "http://localhost:3001")

	// FIXME: this is a hack, it should already be set in app env
	os.Setenv("RISEACT_APP_URL", tun.URL())
	os.Setenv("ACCOUNTS_HOST", settings.AccountsHost)

	// start reverse proxy server
	go proxy.Launch()

	// err = osutils.LaunchBrowser(tun.URL())

	// if err != nil {
	// 	return err
	// }

	// start web app
	err = a.Launch()

	if err != nil {
		return err
	}

	return nil
}

func initApp(host string) (*app.Application, error) {
	var a *app.Application

	appEnv, err := app.LoadEnv()

	if err != nil {
		return nil, fmt.Errorf("Error loading app env: %s", err.Error())
	}

	appEnv.RiseactAppUrl = host

	err = app.IsValidApp(".")

	if err != nil {
		return nil, err
	}

	// retrieve app by client_id
	if appEnv.ClientId != "" {
		a, _ = app.GetAppByClientId(appEnv.ClientId)
		if a != nil {
			logger.Debugf("Existing App: %v\n", a.Name)
			// update app redirect_uri with ngrok url
			appEnv.Store()
			a.UpdateAppUris(host)
			return a, nil
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

		a, err = app.NewApp(appData)

		if err != nil {
			return nil, err
		}

	} else {
		a, err = selectExistingApp(appEnv)
		if err != nil {
			return nil, err
		}

	}

	if a.ClientId == "" || a.ClientSecret == "" {
		return nil, fmt.Errorf("Error creating app, client ID or client Secret are empty")
	}

	appEnv.ClientId = a.ClientId
	appEnv.ClientSecret = a.ClientSecret

	appEnv.Store()
	a.UpdateAppUris(host)

	logger.Infof("App configured successfully. Client ID: " + appEnv.ClientId)

	return a, nil
}

func selectExistingApp(e *app.AppEnv) (*app.Application, error) {
	partnerApps, err := app.GetPrivateApps()

	if err != nil {
		return nil, err
	}

	if len(partnerApps) == 0 {
		return nil, fmt.Errorf("No apps found. Please create a new one.")
	}

	var appIds []string
	var apps map[string]*app.Application = make(map[string]*app.Application)

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

	// typePrompt := &survey.Select{
	// 	Message: "Select an app type",
	// 	Options: []string{"PUBLIC", "PRIVATE"},
	// }

	// var appTypeAnswer string

	// survey.AskOne(typePrompt, &appTypeAnswer)

	// choose a dev organization
	organization, err := organizations.PickOrganizationForm()

	if err != nil {
		return gql.AppInput{}, err
	}

	// TODO: ask other basic questions

	logger.Info("Creating private app " + name + " for organization " + organization.Name)

	appType := gql.ApplicationType(gql.ApplicationTypePrivate)

	return gql.AppInput{
		Name:           name,
		Type:           appType,
		OrganizationId: organization.Id,
		RedirectUris:   "", // TODO: remove
	}, nil
}
