package theme

import (
	"fmt"
	"os"
	"riseact/internal/theme"
	"riseact/internal/utils/git"
	"riseact/internal/utils/stringutils"

	"github.com/AlecAivazis/survey/v2"
)

func InitTheme(repository string) error {
	manifest := themeManifestDataForm()

	// clone repository
	err := git.Clone(repository, manifest.Slug)

	if err != nil {
		return fmt.Errorf("Error cloning repository: %s", err.Error())
	}

	// remove .git folder
	err = os.RemoveAll(fmt.Sprintf("%s/.git", manifest.Slug))

	if err != nil {
		return err
	}

	// write manifest
	err = manifest.Write()
	if err != nil {
		return err
	}

	return nil
}

func themeManifestDataForm() *theme.ThemeManifest {
	var data = theme.NewThemeManifest()

	prompt := &survey.Input{
		Message: "Theme name:",
	}
	survey.AskOne(prompt, &data.Name)

	prompt = &survey.Input{
		Message: "Theme folder name:",
		Default: stringutils.Slugify(data.Name),
	}
	survey.AskOne(prompt, &data.Slug)

	prompt = &survey.Input{
		Message: "Version:",
		Default: "1.0.0",
	}
	survey.AskOne(prompt, &data.Version)

	prompt = &survey.Input{
		Message: "Author:",
	}

	survey.AskOne(prompt, &data.Author)

	prompt = &survey.Input{
		Message: "Documentation URL:",
	}

	survey.AskOne(prompt, &data.DocumentationUrl)

	prompt = &survey.Input{
		Message: "Support URL:",
	}

	survey.AskOne(prompt, &data.SupportUrl)

	return data
}
