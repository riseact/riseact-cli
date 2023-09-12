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
		return nil, err
	}

	a := &AppEnv{}

	a.DatabaseUrl = os.Getenv("DATABASE_URL")
	a.SessionSecret = os.Getenv("SESSION_SECRET")
	a.ClientId = os.Getenv("CLIENT_ID")
	a.ClientSecret = os.Getenv("CLIENT_SECRET")
	a.RiseactAppUrl = os.Getenv("RISEACT_APP_URL")

	return a, nil
}

func (a *AppEnv) store() error {
	data := map[string]string{
		"DATABASE_URL":    a.DatabaseUrl,
		"SESSION_SECRET":  a.SessionSecret,
		"CLIENT_ID":       a.ClientId,
		"CLIENT_SECRET":   a.ClientSecret,
		"RISEACT_APP_URL": a.RiseactAppUrl,
	}

	err := godotenv.Write(data, ".env")

	if err != nil {
		return err
	}

	return nil
}