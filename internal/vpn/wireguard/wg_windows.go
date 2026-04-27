//go:build windows

package wireguard

import (
	"fmt"
	"path/filepath"
)

// wgUp installs and starts a WireGuard tunnel on Windows using wireguard.exe.
// Windows WireGuard does not use wg-quick; it manages tunnels as Windows services.
func wgUp(configPath string) error {
	binary := wireguardBinary()
	out, err := runCmd(binary, "/installtunnel", configPath)
	if err != nil {
		return fmt.Errorf("wireguard /installtunnel: %w\n%s", err, out)
	}
	return nil
}

// wgDown uninstalls and stops a WireGuard tunnel service on Windows.
func wgDown(ifaceName string) error {
	binary := wireguardBinary()
	out, err := runCmd(binary, "/uninstalltunnel", ifaceName)
	if err != nil {
		return fmt.Errorf("wireguard /uninstalltunnel: %w\n%s", err, out)
	}
	return nil
}

// wgShow returns the output of `wg show <iface>` on Windows.
// The `wg` CLI on Windows communicates over a named pipe managed by wireguard.exe.
func wgShow(ifaceName string) (string, error) {
	return runCmd("wg", "show", ifaceName)
}

// wgTransfer returns byte counts for a tunnel.
func wgTransfer(ifaceName string) (string, error) {
	return runCmd("wg", "show", ifaceName, "transfer")
}

func wireguardBinary() string {
	// Default WireGuard for Windows installation path.
	return filepath.Join(
		"C:", "Program Files", "WireGuard", "wireguard.exe",
	)
}
