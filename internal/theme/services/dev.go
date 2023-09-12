package theme

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"riseact/internal/gql"
	"riseact/internal/organizations"
	"riseact/internal/theme"
	"riseact/internal/utils/logger"
	"syscall"
)

type ThemePreviewResponse struct {
	PreviewId  int    `json:"preview_id"`
	PreviewUrl string `json:"preview_url"`
	AdminUrl   string `json:"admin_url"`
}

func StartDevEnvironment(path string) error {
	t, err := theme.NewTheme(path)

	if err != nil {
		return fmt.Errorf("Error loading theme: %s", err.Error())
	}

	// upload theme in partner area, as development theme
	themeId, err := t.Upload(nil)

	if err != nil {
		return fmt.Errorf("Error uploading theme: %s", err.Error())
	}

	defer func() {
		err := t.Delete()
		if err != nil {
			logger.Errorf("Error deleting base theme: %d %s", t.Id, err.Error())
		}
	}()

	// choose a dev organization
	organization, err := organizations.PickOrganizationForm()

	if err != nil {
		return fmt.Errorf("Error picking organization: %s", err.Error())
	}

	// install theme in organization
	preview, err := themeInstall(themeId, organization.Id)

	if err != nil {
		return fmt.Errorf("Error installing theme: %s", err.Error())
	}

	previewTheme := &theme.Theme{
		Id:   preview.PreviewId,
		Path: t.Path,
	}
	defer func() {

		err := previewTheme.Delete()
		if err != nil {
			logger.Errorf("Error deleting preview theme: %s", err.Error())
		}
	}()

	// print preview url and development url
	logger.Info("")
	logger.Infof("Theme preview: %s", preview.PreviewUrl)
	logger.Infof("Theme admin: %s", preview.AdminUrl)
	logger.Info("")

	// watch theme for changes
	go previewTheme.Watch()

	// wait for ctrl+c
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	return nil
}

func themeInstall(themeId int, organizationId int) (*ThemePreviewResponse, error) {
	client, err := gql.GetClient()

	if err != nil {
		return nil, err
	}

	resp, err := gql.ThemePreview(context.Background(), *client, themeId, organizationId)

	if err != nil {
		return nil, err
	}

	return &ThemePreviewResponse{
		PreviewId:  resp.ThemePreview.Id,
		PreviewUrl: resp.ThemePreview.PreviewUrl,
		AdminUrl:   resp.ThemePreview.AdminUrl,
	}, nil
}
