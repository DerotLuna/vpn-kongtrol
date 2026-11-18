//go:build !windows

package main

import (
	"errors"
	"os"
	"os/exec"
)

// stopTray closes a running tray app.
func stopTray() {
	_ = exec.Command("pkill", "-x", "kongtrol-tray").Run()
}

// startDetached is only used for the Windows installer.
func startDetached(string) error { return errors.ErrUnsupported }

// deleteAfterExit: Unix can unlink a running executable right away.
func deleteAfterExit(path string) error { return os.Remove(path) }

// removeUserPathEntry: install.sh never edits shell profiles, so there is
// no PATH entry of ours to remove.
func removeUserPathEntry(string) error { return errNotInPath }
