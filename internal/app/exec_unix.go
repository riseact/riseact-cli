//go:build !windows

package app

import (
	"os/exec"
	"syscall"
	"time"
)

// setProcessGroup puts the child in a process group of its own, so the whole
// tree can be signalled as a unit.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateTree(cmd *exec.Cmd) {
	signalGroup(cmd, syscall.SIGTERM)
}

func killTree(cmd *exec.Cmd) {
	signalGroup(cmd, syscall.SIGKILL)
}

// signalGroup signals the child's whole process group. A negative pid means
// "the group", which is what reaches the grandchildren npm leaves behind.
func signalGroup(cmd *exec.Cmd, sig syscall.Signal) {
	if cmd.Process == nil {
		return
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	if err != nil {
		// The child is already gone, or never got its own group.
		_ = cmd.Process.Signal(sig)
		return
	}

	_ = syscall.Kill(-pgid, sig)
}

// waitForTree reports whether the child's process group emptied on its own
// within timeout. Signal 0 checks for existence without delivering anything, so
// this can poll without disturbing a shutdown already in progress.
func waitForTree(cmd *exec.Cmd, timeout time.Duration) bool {
	if cmd.Process == nil {
		return true
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)

	if err != nil {
		return true
	}

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if err := syscall.Kill(-pgid, 0); err != nil {
			return true
		}

		time.Sleep(100 * time.Millisecond)
	}

	return false
}
