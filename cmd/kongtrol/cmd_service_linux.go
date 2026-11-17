//go:build linux

package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

// fileIsTTY reports whether f is a terminal (/dev/null is not).
func fileIsTTY(f *os.File) bool {
	_, err := unix.IoctlGetTermios(int(f.Fd()), unix.TCGETS)
	return err == nil
}

// runElevatedGUI runs argv as root through pkexec, which shows the desktop's
// polkit password dialog. pkexec exits 126 when the dialog is dismissed and
// 127 when the user is not authorized.
func runElevatedGUI(argv []string) error {
	pkexec := "/usr/bin/pkexec"
	if _, err := os.Stat(pkexec); err != nil {
		if pkexec, err = exec.LookPath("pkexec"); err != nil {
			return err
		}
	}
	cmd := exec.Command(pkexec, argv...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		cancelled := errors.As(err, &ee) && (ee.ExitCode() == 126 || ee.ExitCode() == 127)
		return guiElevationError("pkexec", err, stderr.String(), cancelled)
	}
	return nil
}
