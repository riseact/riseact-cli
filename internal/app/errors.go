package app

import "fmt"

type AppPackageJSONMissingError struct {
	Message string
}

func (e *AppPackageJSONMissingError) Error() string {
	return fmt.Sprint(e.Message)
}
