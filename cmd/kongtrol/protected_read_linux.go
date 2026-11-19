//go:build linux

package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/vpn-kongtrol/kongtrol/internal/profileimport"
)

// elevatedCapture runs argv as root through pkexec (the desktop's polkit
// password dialog, as runElevatedGUI) and returns its stdout. pkexec exits
// 126 when the dialog is dismissed and 127 when the user is not allowed.
func elevatedCapture(ctx context.Context, argv []string) ([]byte, error) {
	pkexec := "/usr/bin/pkexec"
	if _, err := os.Stat(pkexec); err != nil {
		if pkexec, err = exec.LookPath("pkexec"); err != nil {
			return nil, errElevatedReadUnavailable
		}
	}
	var out, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, pkexec, argv...)
	cmd.Stdout = &limitedBuffer{buf: &out, max: profileimport.MaxSize}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && (ee.ExitCode() == 126 || ee.ExitCode() == 127) {
			return nil, errElevatedReadCancelled
		}
		return nil, childReadError(err, stderr.String())
	}
	return out.Bytes(), nil
}

// apiProtectedReader backs the dashboard's "elevate" import (pkexec dialog).
var apiProtectedReader = readProtectedElevated
