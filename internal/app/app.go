package app

import (
	"context"
	"os"
	"path/filepath"
	"riseact/internal/gql"
)

type Application struct {
	Id                int
	Name              string
	Type              gql.ApplicationType
	AppUrl            string
	InstallUrl        string
	AuthorName        string
	AuthorHomepageUrl string
	RedirectUris      string
	ClientId          string
	ClientSecret      string
}

func (a *Application) UpdateAppRedirectUri(redirectUri string) error {
	graphqlClient, err := gql.GetClient()

	if err != nil {
		return err
	}

	_, err = gql.RedirectUriMutation(context.Background(), *graphqlClient, a.Id, redirectUri)

	if err != nil {
		return err
	}

	a.RedirectUris = redirectUri

	return nil
}

func (a *Application) Launch() error {
	return ExecCommand(".", "npm", "run", "dev")
}

func NewApp(input gql.AppInput) (*Application, error) {
	graphqlClient, err := gql.GetClient()

	if err != nil {
		return nil, err
	}
	resp, err := gql.AppCreateMutation(context.Background(), *graphqlClient, input)

	if err != nil {
		return nil, err
	}

	return &Application{
		Id:                resp.AppCreate.App.Id,
		Name:              resp.AppCreate.App.Name,
		Type:              resp.AppCreate.App.Type,
		AppUrl:            resp.AppCreate.App.AppUrl,
		InstallUrl:        resp.AppCreate.App.InstallUrl,
		AuthorName:        resp.AppCreate.App.AuthorName,
		AuthorHomepageUrl: resp.AppCreate.App.AuthorHomepageUrl,
		RedirectUris:      resp.AppCreate.App.RedirectUris,
		ClientId:          resp.AppCreate.App.ClientId,
		ClientSecret:      resp.AppCreate.App.ClientSecret,
	}, nil
}

func IsValidApp(path string) error {
	// check if package.json exists
	if _, err := os.Stat(filepath.Join(path, "package.json")); os.IsNotExist(err) {
		return &AppPackageJSONMissingError{Message: "package.json not found"}
	}

	// TODO
	// check file and folder structure
	// - layout: there is at least one theme.html file
	// - templates: all basic templates are present
	// - config: json files are valid
	// - all: folder size is not too big
	// - all: there are no files that are not allowed (?)

	return nil
}

func GetAppByClientId(clientId string) (*Application, error) {
	appConfig, err := LoadEnv()

	if err != nil {
		return nil, err
	}

	graphqlClient, err := gql.GetClient()

	if err != nil {
		return nil, err
	}

	resp, err := gql.AppByClientIdQuery(context.Background(), *graphqlClient, appConfig.ClientId)

	if err != nil {
		return nil, err
	}

	return &Application{
		Id:                resp.AppByClientId.Id,
		Name:              resp.AppByClientId.Name,
		Type:              resp.AppByClientId.Type,
		AppUrl:            resp.AppByClientId.AppUrl,
		InstallUrl:        resp.AppByClientId.InstallUrl,
		AuthorName:        resp.AppByClientId.AuthorName,
		AuthorHomepageUrl: resp.AppByClientId.AuthorHomepageUrl,
		RedirectUris:      resp.AppByClientId.RedirectUris,
		ClientId:          resp.AppByClientId.ClientId,
		ClientSecret:      resp.AppByClientId.ClientSecret,
	}, nil
}

func GetApps() ([]Application, error) {
	graphqlClient, err := gql.GetClient()

	if err != nil {
		return nil, err
	}
	resp, err := gql.AppSearchQuery(context.Background(), *graphqlClient)

	if err != nil {
		return nil, err
	}

	var apps []Application

	for _, app := range resp.Apps.Edges {
		apps = append(apps, Application{
			Id:                app.Node.Id,
			Name:              app.Node.Name,
			Type:              app.Node.Type,
			AppUrl:            app.Node.AppUrl,
			InstallUrl:        app.Node.InstallUrl,
			AuthorName:        app.Node.AuthorName,
			AuthorHomepageUrl: app.Node.AuthorHomepageUrl,
			RedirectUris:      app.Node.RedirectUris,
			ClientId:          app.Node.ClientId,
			ClientSecret:      app.Node.ClientSecret,
		})
	}

	return apps, nil
}
