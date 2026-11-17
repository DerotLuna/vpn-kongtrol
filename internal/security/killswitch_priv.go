package security

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// privKillSwitch applies the kill switch through privops: Windows Firewall
// rules, a dedicated iptables chain on Linux, a pf anchor on macOS (beta).
type privKillSwitch struct {
	mu      sync.Mutex
	enabled bool
}

// NewKillSwitch returns the kill switch for this OS.
func NewKillSwitch() KillSwitch { return &privKillSwitch{} }

// parseKillSwitchSpec splits "iface1,iface2|endpoint1,endpoint2" into the
// privops request. Interfaces that no longer exist are dropped (the helper
// rejects unknown interfaces); unparsable endpoints are skipped.
func parseKillSwitchSpec(spec string, allowLAN bool, exists func(string) bool) privops.KillSwitchRequest {
	req := privops.KillSwitchRequest{AllowLAN: allowLAN}
	left, right, _ := strings.Cut(spec, "|")
	for _, i := range strings.Split(left, ",") {
		if i = strings.TrimSpace(i); i != "" && exists(i) {
			req.Interfaces = append(req.Interfaces, i)
		}
	}
	for _, e := range strings.Split(right, ",") {
		if a, err := netip.ParseAddr(strings.TrimSpace(e)); err == nil {
			req.Endpoints = append(req.Endpoints, a.Unmap())
		}
	}
	return req
}

func ifaceExists(name string) bool {
	_, err := net.InterfaceByName(name)
	return err == nil
}

// Enable activates the kill switch. tunnelSpec format:
//
//	"iface1,iface2|endpoint1,endpoint2"
//
// Left of | = tunnel interfaces (traffic through them is allowed).
// Right of | = VPN endpoint IPs (allowed so the encrypted tunnel works).
func (k *privKillSwitch) Enable(tunnelSpec string, allowLAN bool) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	req := parseKillSwitchSpec(tunnelSpec, allowLAN, ifaceExists)
	if err := withOps(func(ctx context.Context, ops privops.Ops) error {
		return ops.KillSwitchEnable(ctx, req)
	}); err != nil {
		return fmt.Errorf("kill switch: %w", err)
	}
	k.enabled = true
	return nil
}

// Disable removes the kill switch rules (safe when never enabled).
func (k *privKillSwitch) Disable() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	err := withOps(func(ctx context.Context, ops privops.Ops) error { return ops.KillSwitchDisable(ctx) })
	if err != nil && !k.enabled {
		// Nothing of ours is in place; a missing helper is not an error here.
		return nil
	}
	k.enabled = false
	if err != nil {
		return fmt.Errorf("kill switch: %w", err)
	}
	return nil
}

func (k *privKillSwitch) IsEnabled() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.enabled
}
