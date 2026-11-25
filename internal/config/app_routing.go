package config

import (
	"os"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// AppRouteSupported reports whether executable routing can be enabled on this
// host for the saved profile, without starting a tunnel.
func (v VPNConfig) AppRouteSupported() bool {
	return v.Type == "wireguard" && !v.DefaultEgress && v.CheckAppRoute() == nil
}

// CheckAppRoute verifies the local platform and the current WireGuard file.
// Config validation separately rejects conflicting egress modes.
func (v VPNConfig) CheckAppRoute() error {
	if !privops.WGAppRoutingSupported() {
		return newValidationError("config.err.app_route_platform")
	}
	home, _ := os.UserHomeDir()
	v.ConfigFile = expandHome(v.ConfigFile, home)
	return v.CheckDefaultEgress()
}
