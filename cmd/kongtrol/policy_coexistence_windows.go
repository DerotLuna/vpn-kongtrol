//go:build windows

package main

import "github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"

// Proton for Windows can refuse to connect while a direct WireGuard
// interface is present, even if only selected destinations use it.
func policyAutoConnectConflict(profile string) string {
	a, ok := adapters.Get(profile)
	if !ok || a.Name() != "wireguard" {
		return ""
	}
	if tunnel, up, _ := appattach.ActiveTunnel("protonvpn"); up && tunnel.IP.IsGlobalUnicast() {
		return "Proton VPN"
	}
	return ""
}
