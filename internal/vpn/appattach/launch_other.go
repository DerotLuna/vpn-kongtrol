//go:build !windows && !darwin

package appattach

import "os/exec"

// osLaunch starts the first GUI launcher command found on PATH.
func osLaunch(app App) error {
	for _, c := range app.LinuxCmds {
		if p, err := exec.LookPath(c); err == nil {
			return startDetached(p)
		}
	}
	return ErrAppNotFound
}
