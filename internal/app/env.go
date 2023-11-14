package app

import (
	"os"

	"github.com/joho/godotenv"
)

type AppEnv struct {
	DatabaseUrl   string
	SessionSecret string
	ClientId      string
	ClientSecret  string
	RiseactAppUrl string
}

func LoadEnv() (*AppEnv, error) {
	err := godotenv.Load()

	if err != nil {
		return &AppEnv{}, nil
	}

	a := &AppEnv{}

	a.ClientId = os.Getenv("CLIENT_ID")
	a.ClientSecret = os.Getenv("CLIENT_SECRET")
	a.RiseactAppUrl = os.Getenv("RISEACT_APP_URL")

	return a, nil
}

func (a *AppEnv) Store() error {
	data, err := godotenv.Read()

	if err != nil {
		return err
	}

	data["CLIENT_ID"] = a.ClientId
	data["CLIENT_SECRET"] = a.ClientSecret
	data["RISEACT_APP_URL"] = a.RiseactAppUrl

	err = godotenv.Write(data, ".env")

	if err != nil {
		return err
	}

	return nil
}
