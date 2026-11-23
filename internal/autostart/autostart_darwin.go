//go:build darwin

package autostart

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// agentPath is ~/Library/LaunchAgents/app.kongtrol.tray.plist.
func agentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
}

// Status: on when Kongtrol's LaunchAgent exists.
func Status() State {
	_, hasTray := TrayPath()
	_, err := os.Stat(agentPath())
	exists := err == nil
	return State{Supported: hasTray || exists, Enabled: exists}
}

// Set writes or deletes the LaunchAgent; launchd reads it at the next
// sign-in, so nothing is loaded now.
func Set(on bool) error {
	if !on {
		if err := os.Remove(agentPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	tray, ok := TrayPath()
	if !ok {
		return ErrNoTray
	}
	if err := os.MkdirAll(filepath.Dir(agentPath()), 0o755); err != nil {
		return err
	}
	return config.WriteFileAtomic(agentPath(), []byte(launchAgentFor(tray)), 0o644)
}

// Remove deletes the LaunchAgent when it opens tray or its app bundle.
func Remove(tray string) error {
	data, err := os.ReadFile(agentPath())
	if err != nil {
		return nil
	}
	target := tray
	if app := appBundleOf(tray); app != "" {
		target = app
	}
	if !strings.Contains(string(data), target) {
		return nil
	}
	if err := os.Remove(agentPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
