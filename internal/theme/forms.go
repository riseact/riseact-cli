package theme

import (
	"context"
	"fmt"
	"riseact/internal/gql"
	"riseact/internal/utils/logger"

	"github.com/AlecAivazis/survey/v2"
)

func ThemePickerForm() (string, error) {
	var idx int = -1

	// ask for existing theme id
	themes, err := themeList()

	if err != nil {
		return "", err
	}

	options := make([]string, len(themes.Themes.Edges))

	for i, theme := range themes.Themes.Edges {
		options[i] = fmt.Sprintf("%d - %s", theme.Node.Id, theme.Node.Name)
	}

	prompt := &survey.Select{
		Message: "Select a theme:",
		Options: options,
	}

	survey.AskOne(prompt, &idx)

	if idx < 0 {
		return "", fmt.Errorf("Invalid theme id")
	}

	logger.Debugf("Selected theme id: %d", idx)

	return themes.Themes.Edges[idx].Node.Uuid, nil
}

func themeList() (*gql.ThemeListResponse, error) {
	client, err := gql.GetClient()

	if err != nil {
		return nil, err
	}

	themes, err := gql.ThemeList(context.Background(), *client)

	if err != nil {
		return nil, err
	}

	return themes, nil
}
