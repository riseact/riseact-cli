package app

import (
	"os"
	"os/exec"
)

func ExecCommand(path string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = path

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
