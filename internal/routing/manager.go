// Package routing provides route table management. Changes go through the
// privileged operations contract (internal/privops): directly when this
// process is elevated, else through the Kongtrol helper service.
package routing

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// Route represents a single entry in the OS routing table.
type Route struct {
	Destination net.IPNet
	Gateway     net.IP
	Interface   string // network interface name (e.g. "tun0", "vpn0")
	Metric      int
}

// RouteManager manages routes in the OS routing table.
type RouteManager interface {
	// Add inserts a route into the routing table.
	Add(r Route) error

	// Delete removes a route from the routing table.
	Delete(r Route) error

	// List returns VPN-related routes (Windows: read from the OS; elsewhere:
	// the routes Kongtrol added).
	List() ([]Route, error)

	// Flush removes all routes Kongtrol added on a tunnel interface.
	Flush(tunnelInterface string) error
}

const opTimeout = 20 * time.Second

// opsFunc returns the privileged implementation (privops.Default; tests
// replace it).
var opsFunc = privops.Default

// manager implements RouteManager on top of privops. The OS-specific part
// is only listOS (build-tagged).
type manager struct {
	mu     sync.Mutex
	routes []Route // tracks routes added by kongtrol
}

// NewRouteManager returns the route manager.
func NewRouteManager() RouteManager { return &manager{} }

func toPrivRoute(r Route) (privops.Route, error) {
	ones, bits := r.Destination.Mask.Size()
	addr, ok := netip.AddrFromSlice(r.Destination.IP)
	if !ok || bits == 0 {
		return privops.Route{}, fmt.Errorf("routing: invalid destination %s", r.Destination.String())
	}
	addr = addr.Unmap()
	if addr.Is4() && bits == 128 {
		ones -= 96
	}
	pr := privops.Route{Destination: netip.PrefixFrom(addr, ones), Interface: r.Interface, Metric: r.Metric}
	if r.Gateway != nil {
		g, ok := netip.AddrFromSlice(r.Gateway)
		if !ok {
			return privops.Route{}, fmt.Errorf("routing: invalid gateway %s", r.Gateway)
		}
		pr.Gateway = g.Unmap()
	}
	return pr, nil
}

func withOps(fn func(ctx context.Context, ops privops.Ops) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	ops, err := opsFunc(ctx)
	if err != nil {
		return err
	}
	return fn(ctx, ops)
}

func (m *manager) Add(r Route) error {
	pr, err := toPrivRoute(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := withOps(func(ctx context.Context, ops privops.Ops) error { return ops.AddRoute(ctx, pr) }); err != nil {
		return fmt.Errorf("routing: add %s via %s: %w", r.Destination.String(), r.Interface, err)
	}
	addTrackedRoute(&m.routes, r)
	return nil
}

func (m *manager) Delete(r Route) error {
	pr, err := toPrivRoute(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := withOps(func(ctx context.Context, ops privops.Ops) error { return ops.DeleteRoute(ctx, pr) }); err != nil {
		return fmt.Errorf("routing: delete %s via %s: %w", r.Destination.String(), r.Interface, err)
	}
	m.routes = removeRoute(m.routes, r)
	return nil
}

func (m *manager) Flush(tunnelInterface string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routes = filterInterface(m.routes, tunnelInterface)
	return withOps(func(ctx context.Context, ops privops.Ops) error {
		return ops.FlushRoutes(ctx, tunnelInterface)
	})
}

func (m *manager) List() ([]Route, error) {
	if routes, ok := listOS(); ok {
		return routes, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Route, len(m.routes))
	copy(out, m.routes)
	return out, nil
}

func removeRoute(routes []Route, target Route) []Route {
	out := routes[:0]
	for _, r := range routes {
		if r.Destination.String() != target.Destination.String() || r.Interface != target.Interface {
			out = append(out, r)
		}
	}
	return out
}

func filterInterface(routes []Route, iface string) []Route {
	out := routes[:0]
	for _, r := range routes {
		if r.Interface != iface {
			out = append(out, r)
		}
	}
	return out
}

func addTrackedRoute(routes *[]Route, r Route) {
	for _, cur := range *routes {
		if cur.Destination.String() == r.Destination.String() && cur.Interface == r.Interface {
			return
		}
	}
	*routes = append(*routes, r)
}
