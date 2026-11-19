//go:build linux || darwin

package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/profileimport"
)

// readProtectedElevated runs `kongtrol profile read-protected <path>` as
// root with one prompt: sudo in a terminal, else the desktop's dialog
// (elevatedCapture: pkexec on Linux). Only stdout is kept (at most the
// import limit); the child prints nothing but a validated profile.
func readProtectedElevated(ctx context.Context, path string) ([]byte, error) {
	exe, err := currentExecutable()
	if err != nil {
		return nil, err
	}
	argv := []string{exe, "profile", "read-protected", "--quiet", "--", path}
	if !fileIsTTY(os.Stdin) {
		return elevatedCapture(ctx, argv)
	}
	sudo := "/usr/bin/sudo"
	if _, err := os.Stat(sudo); err != nil {
		if sudo, err = exec.LookPath("sudo"); err != nil {
			return nil, errElevatedReadUnavailable
		}
	}
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, sudo, append([]string{"--"}, argv...)...)
	cmd.Stdin, cmd.Stdout = os.Stdin, &limitedBuffer{buf: &out, max: profileimport.MaxSize}
	// sudo asks for the password on the terminal itself (/dev/tty); the
	// child's own refusal goes to stderr and is shown as is.
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, childReadError(err, stderr.String())
	}
	return out.Bytes(), nil
}

// childReadError keeps the child's translated refusal (its first line +
// "what to do"), else the run error.
func childReadError(err error, stderr string) error {
	if msg := strings.TrimSpace(stderr); msg != "" {
		return errors.New(msg)
	}
	return err
}

// limitedBuffer stops accepting bytes past max (the child never sends
// more; this only bounds memory if something else answers).
type limitedBuffer struct {
	buf *bytes.Buffer
	max int64
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if int64(b.buf.Len()+len(p)) > b.max+1 {
		return 0, errors.New("elevated read: output too large")
	}
	return b.buf.Write(p)
}
