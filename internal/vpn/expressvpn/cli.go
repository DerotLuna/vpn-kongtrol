package expressvpn

import (
	"runtime"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func cli() []string {
	// The official Windows control requires an elevated terminal.
	if runtime.GOOS == "windows" {
		return nil
	}
	return appattach.CommandPaths("expressvpnctl", nil, []string{"/Applications/ExpressVPN.app/Contents/MacOS/expressvpnctl"}, nil)
}

func connectArgs(cfg vpn.AdapterConfig) ([][]string, error) {
	target, err := appattach.ExactLocation(cfg.Extra["server"])
	if err != nil {
		return nil, err
	}
	if target == "" {
		return [][]string{{"connect"}}, nil
	}
	return [][]string{{"connect", target}}, nil
}
