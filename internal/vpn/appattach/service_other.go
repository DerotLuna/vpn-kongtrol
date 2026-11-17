//go:build !darwin && !windows

package appattach

// osServiceTunnel: macOS-only probe; nothing on this OS.
func osServiceTunnel([]string) (Tunnel, bool) { return Tunnel{}, false }
