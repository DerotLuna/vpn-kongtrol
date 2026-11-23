//go:build linux

package autostart

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

const entryName = "kongtrol.desktop"

// userEntry is $XDG_CONFIG_HOME/autostart/kongtrol.desktop (install.sh
// writes it; the user's copy wins over the system one).
func userEntry() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "autostart", entryName)
}

// systemEntryExists: a package installed /etc/xdg/autostart/kongtrol.desktop
// (or another $XDG_CONFIG_DIRS folder has one).
func systemEntryExists() bool {
	dirs := os.Getenv("XDG_CONFIG_DIRS")
	if dirs == "" {
		dirs = "/etc/xdg"
	}
	for _, d := range strings.Split(dirs, ":") {
		if d == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(d, "autostart", entryName)); err == nil {
			return true
		}
	}
	return false
}

func readUser() (string, bool) {
	data, err := os.ReadFile(userEntry())
	if err != nil {
		return "", false
	}
	return string(data), true
}

func writeUser(content string) error {
	path := userEntry()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return config.WriteFileAtomic(path, []byte(content), 0o644)
}

// Status follows the XDG lookup: the user's entry when there is one,
// else the system entry.
func Status() State {
	_, hasTray := TrayPath()
	if content, ok := readUser(); ok {
		return State{Supported: true, Enabled: entryEnabled(content)}
	}
	if systemEntryExists() {
		return State{Supported: true, Enabled: true}
	}
	return State{Supported: hasTray}
}

// Set: off writes Hidden=true into the user's entry (or a hidden override
// when only the system one exists); on removes it again, or writes a
// full entry for the tray next to this program.
func Set(on bool) error {
	content, hasUser := readUser()
	if !on {
		switch {
		case hasUser:
			return writeUser(withHidden(content, true))
		case systemEntryExists():
			return writeUser(hiddenOverride)
		}
		return nil
	}
	if hasUser {
		shown := withHidden(content, false)
		if entryHasExec(shown) {
			return writeUser(shown)
		}
		if systemEntryExists() {
			// Only an override hiding the system entry: drop it.
			if err := os.Remove(userEntry()); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		}
	} else if systemEntryExists() {
		return nil
	}
	tray, ok := TrayPath()
	if !ok {
		return ErrNoTray
	}
	return writeUser(desktopEntryFor(tray))
}

// Remove deletes the user's entry when it starts tray or only hides the
// system entry (uninstall; another program's file is never touched).
func Remove(tray string) error {
	content, ok := readUser()
	if !ok {
		return nil
	}
	if exec := entryExec(content); exec != "" && !execStarts(exec, tray) {
		return nil
	}
	if err := os.Remove(userEntry()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
