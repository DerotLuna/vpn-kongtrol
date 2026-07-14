//go:build !windows

package forticlient

import "fmt"

// guiConnect is not implemented on non-Windows platforms.
// The tryGUIConnect wrapper in cli_v6.go returns an error before reaching this.
func guiConnect(tunnelName, username, password string, allowInsecureCert bool) error {
	return fmt.Errorf("forticlient gui: not supported on this platform")
}

// guiDisconnect is not implemented on non-Windows platforms.
func guiDisconnect(tunnelName string) error {
	return fmt.Errorf("forticlient gui: not supported on this platform")
}
