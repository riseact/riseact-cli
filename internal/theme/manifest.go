package theme

import (
	"encoding/json"
	"fmt"
	"os"
)

type ThemeManifest struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Version          string `json:"version"`
	Author           string `json:"author"`
	DocumentationUrl string `json:"documentation_url"`
	SupportUrl       string `json:"support_url"`
}

func (m *ThemeManifest) Load(path string) error {

	data, err := os.ReadFile(fmt.Sprintf("%s/manifest.json", path))

	if err != nil {
		return err
	}

	err = json.Unmarshal(data, m)

	if err != nil {
		return err
	}

	return nil
}

func (m *ThemeManifest) Write() error {
	manifest, err := json.Marshal(m)

	if err != nil {
		return err
	}

	return os.WriteFile(fmt.Sprintf("%s/manifest.json", m.Slug), []byte(manifest), os.ModePerm)
}

func NewThemeManifest() *ThemeManifest {
	return &ThemeManifest{}
}
