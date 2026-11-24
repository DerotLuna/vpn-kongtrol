//go:build windows || darwin

package filepicker

// hasDesktop: the daemon runs in the user's desktop session on Windows and
// macOS (it is started from the tray, the installer or a terminal).
func hasDesktop() bool { return true }
