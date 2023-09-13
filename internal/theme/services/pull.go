package theme

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"riseact/internal/config"
	"riseact/internal/theme"
)

const ThemeZipFilename = "theme.zip"

func Pull(path string) error {
	settings := config.GetAppSettings()

	// check path is empty
	if err := verifyPath(path); err != nil {
		return fmt.Errorf("Invalid path: %s", err.Error())
	}

	// pick a theme from partners themes
	uuid, err := theme.ThemePickerForm()

	if err != nil {
		return fmt.Errorf("Error picking theme: %s", err.Error())
	}

	// download .zip file to path
	downloadZipFile(settings.CoreHost + "/partners/themes/download/" + uuid)

	// unzip file to path
	if err := unzip(ThemeZipFilename, path); err != nil {
		return fmt.Errorf("Error unzipping file: %s", err.Error())
	}

	// delete .zip file
	if err := os.Remove(ThemeZipFilename); err != nil {
		return fmt.Errorf("Error deleting file: %s", err.Error())
	}

	return nil
}

func verifyPath(path string) error {
	isEmpty, err := isDirEmpty(path)
	if err != nil {
		return fmt.Errorf("Errore:", err)
	}

	if !isEmpty {
		return fmt.Errorf("Path is not empty")
	}

	// create path if not exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("Error creating path: %s", err.Error())
		}
	}

	return nil
}

func downloadZipFile(url string) error {
	settings, err := config.GetUserSettings()

	if err != nil {
		return fmt.Errorf("Error getting user settings: %s", err.Error())
	}

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.AccessToken))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(ThemeZipFilename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func isDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}

			rc, err := f.Open()
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, rc)
			if err != nil {
				return err
			}

			outFile.Close()
			rc.Close()
		}
	}
	return nil
}
