//go:build !windows

package forticlient

import (
	"context"
	"errors"
)

var errNoGUI = errors.New("forticlient gui: not supported on this platform")

// GUI automation and FortiVPN.exe exist only on Windows; SelectPlan never
// picks them elsewhere.
func guiConnect(context.Context, string, string, string, string, bool) error { return errNoGUI }

func guiDisconnect(context.Context, string) error { return errNoGUI }

func fortiVPNConnect(context.Context, Install, string, string, string, bool, func() bool) error {
	return errNoGUI
}

func fortiVPNDisconnect(context.Context, string, string) error { return errNoGUI }
