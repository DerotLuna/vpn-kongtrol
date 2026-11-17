//go:build darwin

package appattach

import "os/exec"

// osLaunch opens (or focuses) the app through LaunchServices.
func osLaunch(app App) error {
	if app.MacApp == "" {
		return ErrAppNotFound
	}
	if err := exec.Command("open", "-a", app.MacApp).Run(); err != nil {
		return ErrAppNotFound
	}
	return nil
}
