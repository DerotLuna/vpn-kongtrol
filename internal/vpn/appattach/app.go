package appattach

import (
	"errors"
	"os"
	"os/exec"
)

// ErrAppNotFound is returned by Launch when the provider app is not
// installed in any known location (and no app_path override is set).
var ErrAppNotFound = errors.New("provider app not found")

// App identifies a provider's official app on each OS. Locations are best
// effort; the profile's `app_path` overrides them.
type App struct {
	DisplayName  string   // shown to the user ("NordVPN", "Proton VPN", …)
	WindowsPaths []string // absolute .exe paths, first existing wins
	// WindowsFind lists more .exe paths, worked out only when the app is
	// launched (versioned folders, registry install location); tried after
	// WindowsPaths. nil = none.
	WindowsFind func() []string
	MacApp      string   // application name for `open -a`
	LinuxCmds   []string // GUI launcher commands looked up on PATH
}

// Launch starts the provider app (or brings it forward when already
// running) without waiting for it. override is the profile's app_path.
func Launch(app App, override string) error {
	if override != "" {
		if _, err := os.Stat(override); err != nil {
			return err
		}
		return startDetached(override)
	}
	return osLaunch(app)
}

// launchFunc is swapped in tests.
var launchFunc = Launch

// startDetached starts a GUI program and releases it.
func startDetached(path string, args ...string) error {
	cmd := exec.Command(path, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// firstExisting returns the first path that exists as a regular file.
func firstExisting(paths []string) (string, bool) {
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
			return p, true
		}
	}
	return "", false
}
