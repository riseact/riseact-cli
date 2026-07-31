//go:build windows

package app

import (
	"os/exec"
	"strconv"
	"time"
)

// Windows has no process groups in the POSIX sense, so the child is started
// normally and the tree is torn down with taskkill instead.
func setProcessGroup(cmd *exec.Cmd) {}

func terminateTree(cmd *exec.Cmd) {
	taskkill(cmd, false)
}

func killTree(cmd *exec.Cmd) {
	taskkill(cmd, true)
}

// taskkill ends the child and its descendants. /T walks the tree, which is what
// reaches the processes npm spawned and then forgot about.
func taskkill(cmd *exec.Cmd, force bool) {
	if cmd.Process == nil {
		return
	}

	args := []string{"/T", "/PID", strconv.Itoa(cmd.Process.Pid)}

	if force {
		args = append([]string{"/F"}, args...)
	}

	_ = exec.Command("taskkill", args...).Run()
}

// waitForTree has nothing to poll on Windows: taskkill /T already walked the
// tree synchronously, so report the tree as gone and skip the forced pass.
func waitForTree(cmd *exec.Cmd, timeout time.Duration) bool {
	return true
}
