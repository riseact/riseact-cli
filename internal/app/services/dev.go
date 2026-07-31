package services

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"riseact/internal/app"
	"riseact/internal/config"
	"riseact/internal/gql"
	"riseact/internal/organizations"
	"riseact/internal/utils/logger"
	"syscall"

	"github.com/AlecAivazis/survey/v2"
)

func StartDevEnvironment() error {
	logger.Debug("Starting dev environment...")

	// Everything below is torn down in reverse order when this context is
	// cancelled, which is what frees the tunnel's subdomain immediately instead
	// of leaving frps to notice a dropped heartbeat.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	settings := config.GetAppSettings()

	// Everything reaches the app server, which handles websocket upgrades itself
	// and forwards HMR traffic to Vite.
	proxy, err := app.NewReverseProxy("http://localhost:3000")

	if err != nil {
		return err
	}

	defer proxy.Close()

	go proxy.Serve()

	a, access, err := startDevAccess(proxy.Port())

	if err != nil {
		return err
	}

	defer access.Close()

	logger.Info("")
	logger.Infof("App url: %s", access.URL)
	logger.Info("")

	// FIXME: this is a hack, it should already be set in app env
	os.Setenv("RISEACT_APP_URL", access.URL)
	os.Setenv("ACCOUNTS_HOST", settings.AccountsHost)

	logger.Info("Starting the dev server...")

	// Blocks until the dev server exits or the user interrupts.
	if err := a.Launch(ctx); err != nil {
		return err
	}

	logger.Info("Stopping the tunnel...")

	return nil
}

// initApp resolves the application these credentials belong to, creating or
// linking one if the project is not configured yet.
//
// It deliberately does not touch the app's URLs: the tunnel hostname is derived
// from the credentials this returns, so it is not known yet. See startDevAccess.
func initApp() (*app.Application, *app.AppEnv, error) {
	var a *app.Application

	appEnv, err := app.LoadEnv()

	if err != nil {
		return nil, nil, fmt.Errorf("Error loading app env: %s", err.Error())
	}

	err = app.IsValidApp(".")

	if err != nil {
		return nil, nil, err
	}

	// retrieve app by client_id
	if appEnv.ClientId != "" {
		a, _ = app.GetAppByClientId(appEnv.ClientId)
		if a != nil {
			logger.Debugf("Existing App: %v\n", a.Name)
			appEnv.Store()
			return a, appEnv, nil
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
			return nil, nil, err
		}

		a, err = app.NewApp(appData)

		if err != nil {
			return nil, nil, err
		}

	} else {
		a, err = selectExistingApp(appEnv)
		if err != nil {
			return nil, nil, err
		}

	}

	if a.ClientId == "" || a.ClientSecret == "" {
		return nil, nil, fmt.Errorf("Error creating app, client ID or client Secret are empty")
	}

	appEnv.ClientId = a.ClientId
	appEnv.ClientSecret = a.ClientSecret

	appEnv.Store()

	logger.Infof("App configured successfully. Client ID: %s", appEnv.ClientId)

	return a, appEnv, nil
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
