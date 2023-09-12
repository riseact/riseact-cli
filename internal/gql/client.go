package gql

import (
	"net/http"
	"riseact/internal/config"

	"github.com/Khan/genqlient/graphql"
)

type authedTransport struct {
	key     string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.key)
	return t.wrapped.RoundTrip(req)
}

func GetClient() (*graphql.Client, error) {
	appSettings := config.GetAppSettings()
	userSettings, err := config.GetUserSettings()

	if err != nil {
		return nil, err
	}

	httpClient := http.Client{
		Transport: &authedTransport{
			key:     userSettings.AccessToken,
			wrapped: http.DefaultTransport,
		},
	}
	graphqlClient := graphql.NewClient(appSettings.CoreHost+"/partners/graphql/", &httpClient)

	return &graphqlClient, nil
}
