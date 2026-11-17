package security

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// privDNSGuard forces resolvers through privops: netsh per interface on
// Windows, networksetup per service on macOS, /etc/resolv.conf on Linux.
// The helper snapshots the previous configuration and puts it back.
type privDNSGuard struct {
	mu     sync.Mutex
	active bool
	iface  string
}

// NewDNSGuard returns the DNS guard for this OS.
func NewDNSGuard() DNSGuard { return &privDNSGuard{} }

func toAddrs(ips []net.IP) []netip.Addr {
	out := make([]netip.Addr, 0, len(ips))
	for _, ip := range ips {
		if a, ok := netip.AddrFromSlice(ip); ok {
			out = append(out, a.Unmap())
		}
	}
	return out
}

func (g *privDNSGuard) Apply(iface string, dnsServers []net.IP) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(dnsServers) == 0 {
		return fmt.Errorf("dnsguard: no DNS servers provided")
	}
	if iface == "" {
		return fmt.Errorf("dnsguard: tunnel interface unknown")
	}
	req := privops.DNSRequest{Interface: iface, Servers: toAddrs(dnsServers)}
	err := withOps(func(ctx context.Context, ops privops.Ops) error {
		// The tunnel interface changed: give the old one its DNS back first.
		if g.active && g.iface != iface {
			_ = ops.RestoreDNS(ctx, g.iface)
			g.active = false
		}
		return ops.SetDNS(ctx, req)
	})
	if err != nil {
		return fmt.Errorf("dnsguard: set DNS: %w", err)
	}
	g.iface, g.active = iface, true
	return nil
}

func (g *privDNSGuard) Restore() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.active {
		return nil
	}
	g.active = false // never retry in a loop; the helper lease is the backstop
	if err := withOps(func(ctx context.Context, ops privops.Ops) error {
		return ops.RestoreDNS(ctx, g.iface)
	}); err != nil {
		return fmt.Errorf("dnsguard: restore DNS: %w", err)
	}
	return nil
}

func (g *privDNSGuard) IsActive() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.active
}
