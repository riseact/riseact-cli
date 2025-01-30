package theme

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"riseact/internal/config"
	"riseact/internal/gql"
	"riseact/internal/utils/logger"
	"riseact/internal/utils/watcher"
	"strings"
	"time"
)

const debounceDuration = 300 * time.Millisecond

var ASSETS_BLACKLIST = []string{
	".gitignore",
	".git",
	"node_modules",
	".riseactignore",
}

type ThemeUploadResponse struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type ThemeErrorResponse struct {
	Asset string `json:"asset"`
	Theme string `json:"theme"`
	Error string `json:"error"`
}

type Theme struct {
	Id             int
	Manifest       ThemeManifest
	Path           string
	AssetsSnapshot []string
}

func NewTheme(path string) (*Theme, error) {
	theme := &Theme{
		Path: path,
	}

	err := theme.Manifest.Load(path)

	if err != nil {
		return nil, err
	}

	return theme, nil
}

func (t *Theme) Package(dstDir string) (string, error) {
	if !t.IsValid() {
		return "", fmt.Errorf("Not a valid theme folder")
	}

	if dstDir == "" {
		dstDir = "."
	}

	filename := filepath.Join(t.Path, "tmp", "theme.zip")

	err := createZipThemePackage(t.Path, filename)

	if err != nil {
		return "", err
	}

	logger.Debugf("Theme packaged successfully: %s", filename)

	return filename, nil
}

func (t *Theme) Watch() {
	c := make(chan watcher.FileEvent)

	debounceTimers := make(map[watcher.FileEventAction]*time.Timer)

	go watcher.WatchPath(c, t.Path)

	for {
		event := <-c

		if timer, exists := debounceTimers[event.Action]; exists {
			timer.Stop()
		}

		logger.Debugf("Received: %s", event.Action)
		debounceTimers[event.Action] = time.AfterFunc(debounceDuration, func() {
			logger.Debugf("Firing: %s", event.Action)
			themeAssetChanged(t.Id, event)
			delete(debounceTimers, event.Action)
		})
	}
}

func (t *Theme) Upload(remoteThemeId *int) (int, error) {
	appSettings := config.GetAppSettings()

	// create a zip file of the theme

	if !t.IsValid() {
		return -1, fmt.Errorf("Not a valid theme folder")
	}

	filename, err := t.Package(filepath.Join(t.Path, "tmp"))

	if err != nil {
		return -1, err
	}

	// upload the zip file to the partner area with http multipart POST on https://core.riseact.com/partners/themes/upload

	themeData, err := uploadFile(filename, remoteThemeId, appSettings.CoreHost+"/partners/themes/upload/")

	if err != nil {
		return -1, err
	}

	// return the theme id
	t.Id = themeData.Id

	return themeData.Id, nil
}

func (t *Theme) IsValid() bool {
	if t.Path == "" {
		fmt.Println("Path is empty")
		return false
	}

	return true
}

func (t *Theme) Delete() error {
	client, err := gql.GetClient()

	if err != nil {
		return err
	}

	_, err = gql.ThemeDelete(context.Background(), *client, t.Id)

	if err != nil {
		return err
	}

	return nil
}

func createZipThemePackage(srcDir string, destZip string) error {
	fmt.Println(srcDir)
	fmt.Println(destZip)
	zipFile, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// fmt.Println(path)
		if relPath == "." || relPath == ".." || relPath == destZip{
			return nil
		}

		// fmt.Println(path)
		// fmt.Println(destZip)
		if (path == "." || path == ".." || path == destZip) {
			fmt.Println("Hit the thing", path)
			return nil
		}

		for _, blItem := range ASSETS_BLACKLIST {
			if strings.HasPrefix(relPath, blItem) {
				return nil
			}
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		wr, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(wr, file)
		return err
	})
}

func uploadFile(filename string, themeId *int, url string) (*ThemeUploadResponse, error) {
	settings, _ := config.GetUserSettings()

	logger.Debug("Uploading file: " + filename)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}

	if _, err = io.Copy(fw, file); err != nil {
		return nil, err
	}

	if themeId != nil {
		w.WriteField("theme_id", fmt.Sprintf("%d", *themeId))
	}

	w.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+settings.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var themeErr ThemeErrorResponse

		json.NewDecoder(resp.Body).Decode(&themeErr)

		logger.Errorf("Error:\nAsset: %sMessage: %s", themeErr.Asset, themeErr.Error)
		return nil, fmt.Errorf("upload fallito: %s", resp.Status)
	}

	var themeResp ThemeUploadResponse

	if err := json.NewDecoder(resp.Body).Decode(&themeResp); err != nil {
		return nil, err
	}

	return &themeResp, nil
}

func themeAssetChanged(themeId int, e watcher.FileEvent) error {
	// TODO: handle file move
	// case e.Action == watcher.FileMove:
	// 	_ := NewThemeAsset(e.FileName, themeId)
	switch {
	case e.Action == watcher.FileCreate:
		asset := NewThemeAsset(e.FileName, themeId)
		err := asset.Create()
		if err != nil {
			logger.Errorf("Error creating asset: %s", err.Error())
		}
	case e.Action == watcher.FileWrite:
		asset := NewThemeAsset(e.FileName, themeId)
		err := asset.Update()
		if err != nil {
			logger.Errorf("Error updating asset: %s", err.Error())
		}
	case e.Action == watcher.FileDelete:
		asset := NewThemeAsset(e.FileName, themeId)
		err := asset.Delete()
		if err != nil {
			logger.Errorf("Error deleting asset: %s", err.Error())
		}

	}

	return nil
}
