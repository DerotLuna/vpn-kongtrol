//go:build !windows

package vpndetect

// The provider apps are found by their Windows install folders only; on
// other OSes the adapters use the CLI on PATH, the app name (macOS) or a
// launcher command (Linux).

// ProtonVPNAppPaths is empty outside Windows.
func ProtonVPNAppPaths() []string { return nil }

// NordVPNAppPaths is empty outside Windows.
func NordVPNAppPaths() []string { return nil }

// SurfsharkAppPaths is empty outside Windows.
func SurfsharkAppPaths() []string { return nil }
