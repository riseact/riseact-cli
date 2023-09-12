package git

import (
	"github.com/go-git/go-git/v5"
)

func Clone(repo string, path string) error {
	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:               repo,
		RecurseSubmodules: 2,
	})

	return err
}
