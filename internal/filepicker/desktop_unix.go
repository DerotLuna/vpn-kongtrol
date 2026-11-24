//go:build !windows && !darwin

package filepicker

import "os"

// hasDesktop: a graphical session in the daemon's environment (it inherits
// the session's from the tray or the terminal). A server with no desktop,
// or an SSH session, falls back to the browser's file input.
func hasDesktop() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}
