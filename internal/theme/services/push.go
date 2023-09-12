package theme

import (
	"context"
	"fmt"
	"riseact/internal/gql"
	"riseact/internal/theme"
	"riseact/internal/utils/logger"

	"github.com/AlecAivazis/survey/v2"
)

type PushThemeData struct {
	Create          bool
	ExistingThemeId int
}

func Push(path string) error {
	t, err := theme.NewTheme(path)

	if err != nil {
		return fmt.Errorf("Error loading theme: %s", err.Error())
	}
	// ask if user want to create new theme or ovveride existing one
	data, err := pushThemeForm()

	if err != nil {
		return err
	}

	logger.Debugf("Push theme data: %+v", data)

	if data.Create {
		_, err = t.Upload(nil)

		if err != nil {
			return fmt.Errorf("Error uploading new theme: %s", err.Error())
		}

	} else {
		_, err := t.Upload(&data.ExistingThemeId)
		if err != nil {
			return fmt.Errorf("Error uploading existing theme: %s", err.Error())
		}
	}

	return nil
}

func pushThemeForm() (*PushThemeData, error) {
	d := &PushThemeData{
		ExistingThemeId: -1,
	}

	// ask if user want to create new theme or ovveride existing one
	prompt := &survey.Confirm{
		Message: "Create a new theme?",
	}
	survey.AskOne(prompt, &d.Create)

	if !d.Create {
		// ask for existing theme id
		themes, err := themeList()

		if err != nil {
			return nil, err
		}

		options := make([]string, len(themes.Themes.Edges))

		for i, theme := range themes.Themes.Edges {
			options[i] = fmt.Sprintf("%d - %s", theme.Node.Id, theme.Node.Name)
		}

		prompt := &survey.Select{
			Message: "Select a theme:",
			Options: options,
		}

		survey.AskOne(prompt, &d.ExistingThemeId)

		if d.ExistingThemeId < 0 {
			return nil, fmt.Errorf("Invalid theme id")
		}

		logger.Debugf("Selected theme id: %d", d.ExistingThemeId)

		d.ExistingThemeId = themes.Themes.Edges[d.ExistingThemeId].Node.Id
	}
	return d, nil
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
