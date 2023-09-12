package theme

import (
	"context"
	"os"
	"riseact/internal/gql"
	"riseact/internal/utils/fsutils"
	"riseact/internal/utils/logger"
	"strings"
)

type ThemeAsset struct {
	ThemeID int
	Key     string
	Content string
}

func NewThemeAsset(key string, themeId int) *ThemeAsset {
	return &ThemeAsset{
		Key:     key,
		ThemeID: themeId,
	}
}

func (a *ThemeAsset) Read() error {

	content, err := os.ReadFile("./" + a.Key)
	if err != nil {
		logger.Debugf("Error reading asset file: %s", err.Error())
		return err
	}

	a.Content = string(content)

	return nil
}

func (a *ThemeAsset) IsValidAsset() bool {
	path := a.Key

	if !strings.HasPrefix(path, "./") {
		path = "./" + a.Key
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	if fileInfo.IsDir() {
		return false
	}

	return true
}

func (a *ThemeAsset) Create() error {
	client, err := gql.GetClient()

	if err != nil {
		return err
	}

	if a.IsValidAsset() == false {
		return nil
	}

	logger.Debugf("Start Creating asset %s", a.Key)

	err = a.Read()

	if err != nil {
		logger.Debugf("Error reading asset file: %s", err.Error())
		return err
	}

	mimetype, err := fsutils.GetMimeType(a.Key)

	if err != nil {
		return err
	}
	logger.Debugf("Post Mimetype: %s", mimetype)
	_, err = gql.AssetCreate(context.Background(), *client, gql.AssetInput{
		ThemeId:     a.ThemeID,
		Key:         a.Key,
		Value:       a.Content,
		ContentType: mimetype,
	})
	logger.Debugf("Post Mimetype: %s", mimetype)

	if err != nil {
		return err
	}

	logger.Infof("Create asset: %s", a.Key)

	return nil
}

func (a *ThemeAsset) Update() error {
	client, err := gql.GetClient()

	if err != nil {
		return err
	}
	a.Read()

	mimetype, err := fsutils.GetMimeType(a.Key)

	if err != nil {
		return err
	}

	_, err = gql.AssetUpdate(context.Background(), *client, a.ThemeID, gql.AssetInput{
		ThemeId:     a.ThemeID,
		Key:         a.Key,
		Value:       a.Content,
		ContentType: mimetype,
	})

	if err != nil {
		return err
	}

	logger.Infof("Update asset: %s", a.Key)

	return nil
}

func (a *ThemeAsset) Delete() error {
	client, err := gql.GetClient()

	if err != nil {
		return err
	}

	_, err = gql.AssetDelete(context.Background(), *client, a.ThemeID, a.Key)

	if err != nil {
		return err
	}

	logger.Infof("Delete asset: %s", a.Key)

	return nil
}
