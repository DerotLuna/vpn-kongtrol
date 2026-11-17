//go:build !windows

package forticlient

import (
	"context"
	"fmt"
)

// guiConnect is not implemented on non-Windows platforms.
// The tryGUIConnect wrapper in cli_v6.go returns an error before reaching this.
func guiConnect(context.Context, string, string, string, bool) error {
	return fmt.Errorf("forticlient gui: not supported on this platform")
}

// guiDisconnect is not implemented on non-Windows platforms.
func guiDisconnect(string) error {
	return fmt.Errorf("forticlient gui: not supported on this platform")
}
