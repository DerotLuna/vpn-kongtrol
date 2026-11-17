//go:build linux

package nordvpn

import (
	"fmt"
	"net"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// defaultLocations: the official `nordvpn` CLI on PATH.
var defaultLocations = []string{"nordvpn"}

// CheckPlatform reports whether the NordVPN CLI connect path exists here.
func CheckPlatform() error { return nil }

func connect(override string, t target) error {
	bin, err := findBinary(override, defaultLocations)
	if err != nil {
		return err
	}
	args := linuxConnectArgs(t)
	out, err := runCmd(bin, args...)
	if isNotLoggedIn(out) {
		return ErrNotLoggedIn
	}
	if err != nil {
		return fmt.Errorf("nordvpn %v: %w\n%s", args, err, out)
	}
	return nil
}

func disconnect(override string) error {
	bin, err := findBinary(override, defaultLocations)
	if err != nil {
		return err
	}
	if out, err := runCmd(bin, "disconnect"); err != nil {
		return fmt.Errorf("nordvpn disconnect: %w\n%s", err, out)
	}
	return nil
}

func version(override string) string {
	bin, err := findBinary(override, defaultLocations)
	if err != nil {
		return ""
	}
	out, err := runCmd(bin, "--version")
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(strings.TrimSpace(out), "\n")
	return strings.TrimSpace(line)
}

func cliStatus(override string) (linuxStatus, error) {
	bin, err := findBinary(override, defaultLocations)
	if err != nil {
		return linuxStatus{}, err
	}
	out, err := runCmd(bin, "status")
	if isNotLoggedIn(out) {
		return linuxStatus{}, ErrNotLoggedIn
	}
	if err != nil {
		return linuxStatus{}, fmt.Errorf("nordvpn status: %w\n%s", err, out)
	}
	return parseLinuxStatus(out), nil
}

// cliTunnelInfo returns what `nordvpn status` adds over interface
// detection (the server IP), or nil when the CLI is unavailable.
func cliTunnelInfo(override string) (*vpn.TunnelInfo, error) {
	st, err := cliStatus(override)
	if err != nil || !st.Connected || st.ServerIP == "" {
		return nil, err
	}
	return &vpn.TunnelInfo{RemoteIP: net.ParseIP(st.ServerIP)}, nil
}
