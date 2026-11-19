//go:build windows

package nordvpn

import (
	"fmt"
	"slices"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

// defaultLocations is where the NordVPN Windows installer puts the app.
var defaultLocations = []string{`C:\Program Files\NordVPN\NordVPN.exe`}

// windowsLocations adds the NordVPN.exe of a custom install folder (from
// "Apps & features", shared with the detect list) at call time; only
// connect / disconnect use it, never the 5 s status poll.
func windowsLocations() []string {
	return append(slices.Clone(defaultLocations), vpndetect.NordVPNAppPaths()...)
}

// CheckPlatform reports whether the CLI connect path exists on this OS.
func CheckPlatform() error { return nil }

func connect(override string, t target) error {
	bin, err := findBinary(override, windowsLocations())
	if err != nil {
		return err
	}
	args := windowsConnectArgs(t)
	if out, err := runCmd(bin, args...); err != nil {
		return fmt.Errorf("NordVPN.exe %v: %w\n%s", args, err, out)
	}
	return nil
}

func disconnect(override string) error {
	bin, err := findBinary(override, windowsLocations())
	if err != nil {
		return err
	}
	if out, err := runCmd(bin, windowsDisconnectArgs()...); err != nil {
		return fmt.Errorf("NordVPN.exe -d: %w\n%s", err, out)
	}
	return nil
}

// version: NordVPN.exe has no version switch (running it would start the
// GUI), so report nothing and let callers read the file version.
func version(string) string { return "" }

// cliTunnelInfo: the Windows app has no status command; nothing beyond
// interface detection is available.
func cliTunnelInfo(string) (*vpn.TunnelInfo, error) { return nil, nil }
