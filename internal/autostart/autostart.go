// Package autostart turns "open Kongtrol when you sign in" on and off for
// the current user, with each OS's own per-user mechanism (ADR-0098):
//
//   - Windows: the HKCU Run value "Kongtrol" that the installer and
//     install.ps1 write (ADR-0075), switched with the StartupApproved
//     value that Settings > Apps > Startup and Task Manager use, so an
//     update that writes the Run value again keeps the user's choice.
//   - Linux: the XDG Autostart entry kongtrol.desktop; turning it off
//     writes Hidden=true in the user's own copy, which also overrides the
//     packages' /etc/xdg/autostart entry (XDG Autostart spec §2.2–2.3).
//   - macOS: a per-user LaunchAgent in ~/Library/LaunchAgents.
//
// Opening the tray starts Kongtrol (the tray starts the daemon), which then
// reconnects the VPNs the user had on when that setting is on.
package autostart

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// State is the sign-in start for this user.
type State struct {
	// Supported: this install has a tray to start (or an entry exists).
	Supported bool `json:"supported"`
	Enabled   bool `json:"enabled"`
}

// ErrNoTray: there is no kongtrol-tray next to this program (a CLI-only
// install), so there is nothing to open at sign-in.
var ErrNoTray = errors.New("autostart: kongtrol-tray is not installed next to kongtrol")

// trayName is the tray executable's file name on this OS.
func trayName() string {
	if runtime.GOOS == "windows" {
		return "kongtrol-tray.exe"
	}
	return "kongtrol-tray"
}

// executable is os.Executable with symlinks resolved (a test seam).
var executable = func() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return exe, nil
}

// TrayPath returns the tray next to the running kongtrol (every installer
// puts both in one folder), or false when there is none.
func TrayPath() (string, bool) {
	exe, err := executable()
	if err != nil {
		return "", false
	}
	tray := filepath.Join(filepath.Dir(exe), trayName())
	if fi, err := os.Stat(tray); err != nil || fi.IsDir() {
		return "", false
	}
	return tray, true
}
