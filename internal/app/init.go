package app

import (
	"fmt"
	"os"
	"os/exec"
	"riseact/internal/utils/git"
	"riseact/internal/utils/stringutils"

	"github.com/AlecAivazis/survey/v2"
)

const noteTemplateRepo = "https://github.com/riseact/riseact-app-template-node.git"
const remixTemplateRepo = "https://github.com/riseact/riseact-app-template-remix.git"

type AppData struct {
	path     string
	template string
	manifest *AppManifest
}

type AppManifest struct {
	Name        string
	DevCommands []string
	ClientId    string
}

func DoAppInit() error {
	appData := appDataForm()

	if _, err := os.Stat(appData.path); !os.IsNotExist(err) {
		return fmt.Errorf("App folder already exists")
	}

	if err := os.Mkdir(appData.path, os.ModePerm); err != nil {
		return err
	}

	// clone app template
	err := appCloneTemplate(appData)

	if err != nil {
		return err
	}

	// install dependencies
	err = appInstallDependencies(appData)

	if err != nil {
		return err
	}

	fmt.Println("App created successfully")

	return nil
}

func appDataForm() *AppData {
	appManifest := &AppManifest{}
	appData := &AppData{
		manifest: appManifest,
	}

	namePrompt := &survey.Input{
		Message: "App name",
	}
	survey.AskOne(namePrompt, &appManifest.Name)

	pathPrompt := &survey.Input{
		Message: "App path",
		Default: stringutils.Slugify(appManifest.Name),
	}
	survey.AskOne(pathPrompt, &appData.path)

	templatePrompt := &survey.Select{
		Message: "App template",
		Options: []string{"Remix", "Node"},
	}
	survey.AskOne(templatePrompt, &appData.template)

	return appData
}

func appCloneTemplate(appData *AppData) error {

	var repo string

	switch appData.template {
	case "Remix":
		repo = remixTemplateRepo
	case "Node":
		repo = noteTemplateRepo
	}

	err := git.Clone(repo, appData.path)

	if err != nil {
		return err
	}

	// remove .git folder
	err = os.RemoveAll(fmt.Sprintf("%s/.git", appData.path))

	if err != nil {
		return err
	}

	return nil
}

func execCommand(path string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = path

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func appInstallDependencies(appData *AppData) error {
	switch appData.template {
	case "Remix":
		return appInstallRemixDependencies(appData)
	case "Node":
		return appInstallNodeDependencies(appData)
	}

	return nil
}

func appInstallRemixDependencies(appData *AppData) error {
	err := execCommand(appData.path, "npm", "install")

	return err
}

func appInstallNodeDependencies(appData *AppData) error {

	err := execCommand(appData.path, "npm", "install")

	if err != nil {
		return err
	}

	err = execCommand(appData.path+"/src/frontend", "npm", "run", "build")

	if err != nil {
		return err
	}

	return nil
}
