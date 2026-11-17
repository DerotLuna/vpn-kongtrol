//go:build !windows && !linux

package nordvpn

import "github.com/vpn-kongtrol/kongtrol/internal/vpn"

// CheckPlatform reports whether the NordVPN CLI connect path exists on the
// current OS. NordVPN ships no CLI on macOS (or other OSes): attach mode
// opens the app and waits for the user, and Status() still works through
// the app's network service / interface detection.
func CheckPlatform() error { return ErrUnsupportedOS }

func connect(string, target) error                  { return ErrUnsupportedOS }
func disconnect(string) error                       { return ErrUnsupportedOS }
func version(string) string                         { return "" }
func cliTunnelInfo(string) (*vpn.TunnelInfo, error) { return nil, nil }
