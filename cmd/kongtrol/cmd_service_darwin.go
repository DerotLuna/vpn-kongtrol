//go:build darwin

package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
)

// fileIsTTY reports whether f is a terminal (/dev/null is not).
func fileIsTTY(f *os.File) bool {
	_, err := unix.IoctlGetTermios(int(f.Fd()), unix.TIOCGETA)
	return err == nil
}

// runElevatedGUI runs argv as root behind the macOS administrator dialog.
// osascript reports a dismissed dialog as error -128.
func runElevatedGUI(argv []string) error {
	script, err := appleScriptAdminCommand(argv)
	if err != nil {
		return err
	}
	cmd := exec.Command("/usr/bin/osascript", "-e", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return guiElevationError("osascript", err, stderr.String(), strings.Contains(stderr.String(), "-128"))
	}
	return nil
}
