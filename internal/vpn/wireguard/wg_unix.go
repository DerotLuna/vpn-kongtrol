//go:build linux || darwin

package wireguard

import (
	"fmt"
	"strings"
)

// wgUp brings up a WireGuard interface using wg-quick.
func wgUp(configPath string) error {
	out, err := runCmd("wg-quick", "up", configPath)
	if err != nil {
		return fmt.Errorf("wg-quick up: %w\n%s", err, out)
	}
	return nil
}

// wgDown brings down a WireGuard interface using wg-quick.
func wgDown(ifaceName string) error {
	out, err := runCmd("wg-quick", "down", ifaceName)
	if err != nil {
		return fmt.Errorf("wg-quick down: %w\n%s", err, out)
	}
	return nil
}

// wgShow returns the output of `wg show <iface>`.
func wgShow(ifaceName string) (string, error) {
	return runCmd("wg", "show", ifaceName)
}

// wgTransfer returns byte counts: `wg show <iface> transfer`.
// Output: "<peer-pubkey>\t<rx-bytes>\t<tx-bytes>"
func wgTransfer(ifaceName string) (string, error) {
	return runCmd("wg", "show", ifaceName, "transfer")
}

// tunnelServiceRunning is a no-op on Unix; WireGuard is managed by wg-quick,
// not a Windows service. Always returns false.
func tunnelServiceRunning(_ string) bool { return false }

// WgSetAllowedIPs updates AllowedIPs for a peer on a live WireGuard interface.
// On Unix, this updates the kernel crypto-routing table only — the caller must
// also add OS routes via routeMgr.Add() for the new CIDRs.
func WgSetAllowedIPs(ifaceName, peerPubKey string, cidrs []string) error {
	_, err := runCmd("wg", "set", ifaceName, "peer", peerPubKey,
		"allowed-ips", strings.Join(cidrs, ","))
	return err
}
