package app

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// childShutdownGrace is how long the dev server gets to exit on its own after
// being asked to stop, before it is killed outright.
const childShutdownGrace = 5 * time.Second

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

// ExecCommandContext runs a long-lived command and stops it, and everything it
// spawned, when ctx is cancelled.
//
// Signalling the direct child is not enough: `npm run dev` exits immediately on
// SIGTERM without passing it on, so concurrently, ts-node-dev and the server
// listening on port 3000 were all left running while the CLI reported a clean
// shutdown. The child therefore gets its own process group, and the whole group
// is signalled.
//
// A consequence of that group is that the terminal no longer delivers Ctrl-C to
// the dev server directly — the CLI catches it and forwards it here, which is the
// deterministic path anyway. The child also gets no stdin: a background process
// group that reads from the terminal is stopped with SIGTTIN, and nothing in the
// dev server needs input.
func ExecCommandContext(ctx context.Context, path string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = path

	cmd.Stdin = nil

	// Filtered so a dev server restarting cannot erase the tunnel URL printed
	// above it. Going through a writer means the child's output is a pipe rather
	// than the terminal, and most tools drop colour when they cannot see a tty —
	// hence the env below.
	cmd.Stdout = newClearFilter(os.Stdout)
	cmd.Stderr = newClearFilter(os.Stderr)
	cmd.Env = forceColor(os.Environ())

	setProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		return err
	}

	exited := make(chan error, 1)

	go func() {
		exited <- cmd.Wait()
	}()

	select {
	case err := <-exited:
		return err

	case <-ctx.Done():
		terminateTree(cmd)

		select {
		case <-exited:
		case <-time.After(childShutdownGrace):
		}

		// npm exits without waiting for what it started, so the direct child
		// having been reaped says nothing about its descendants. Give them the
		// rest of the grace period to finish on their own, then make sure.
		if !waitForTree(cmd, childShutdownGrace) {
			killTree(cmd)
		}

		// Cancellation is how this command is meant to end, not a failure.
		return nil
	}
}
