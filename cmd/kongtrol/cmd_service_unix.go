//go:build linux || darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
)

func elevationPromptKey() string { return "service.install.sudo" }

func identityArgs() []string {
	args := []string{"--uid", strconv.Itoa(os.Getuid()), "--gid", strconv.Itoa(os.Getgid())}
	if u, err := user.Current(); err == nil {
		args = append(args, "--user-name", u.Username)
	}
	return args
}

// elevatedInstallOptions takes the installing user from the flags sent by
// the unprivileged parent, else from sudo's SUDO_UID/SUDO_GID, else root.
func elevatedInstallOptions() (privhelper.InstallOptions, error) {
	uid, gid, name := svcUID, svcGID, svcUserName
	if uid < 0 {
		uid, gid = 0, 0
		if v, err := strconv.Atoi(os.Getenv("SUDO_UID")); err == nil {
			uid = v
		}
		if v, err := strconv.Atoi(os.Getenv("SUDO_GID")); err == nil {
			gid = v
		}
		name = os.Getenv("SUDO_USER")
	}
	if uid < 0 || gid < 0 {
		return privhelper.InstallOptions{}, fmt.Errorf("invalid uid/gid %d/%d", uid, gid)
	}
	if name == "" {
		if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
			name = u.Username
		}
	}
	return privhelper.InstallOptions{UserID: strconv.Itoa(uid), UserName: name, GID: gid}, nil
}

// runElevatedStep re-runs this executable as root with one prompt: sudo in
// a terminal, else the desktop's own permission dialog (osascript on
// macOS, pkexec on Linux), so the tray / dashboard can install the helper
// with no terminal at all.
func runElevatedStep(action string, extra []string) error {
	exe, err := currentExecutable()
	if err != nil {
		return err
	}
	argv := append([]string{exe, "service", action, "--elevated-step", "--quiet"}, extra...)
	if !fileIsTTY(os.Stdin) {
		return runElevatedGUI(argv)
	}
	sudo := "/usr/bin/sudo"
	if _, err := os.Stat(sudo); err != nil {
		if sudo, err = exec.LookPath("sudo"); err != nil {
			return err
		}
	}
	cmd := exec.Command(sudo, append([]string{"--"}, argv...)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// guiElevationError turns a failed GUI elevation into
// privhelper.ErrElevationCancelled when cancelled is true, else an error
// carrying the tool's stderr.
func guiElevationError(tool string, err error, stderr string, cancelled bool) error {
	if cancelled {
		return privhelper.ErrElevationCancelled
	}
	if msg := strings.TrimSpace(stderr); msg != "" {
		return fmt.Errorf("%s: %w: %s", tool, err, msg)
	}
	return fmt.Errorf("%s: %w", tool, err)
}
