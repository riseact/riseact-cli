package theme

import (
	"fmt"
	"riseact/internal/theme"
)

func PackageCreate(path string, dstDir string) (string, error) {
	theme, err := theme.NewTheme(path)

	if err != nil {
		return "", fmt.Errorf("Error loading theme: %s", err.Error())
	}

	return theme.Package(dstDir)
}
