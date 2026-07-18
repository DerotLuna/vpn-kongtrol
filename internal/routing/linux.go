//go:build linux

package routing

import (
	"fmt"
	"strings"
	"sync"

	"github.com/vishvananda/netlink"
)

// linuxRouteManager manages routes on Linux using netlink.
type linuxRouteManager struct {
	mu     sync.Mutex
	routes []Route
}

// NewRouteManager returns the Linux route manager.
func NewRouteManager() RouteManager {
	return &linuxRouteManager{}
}

func (m *linuxRouteManager) Add(r Route) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	link, err := netlink.LinkByName(r.Interface)
	if err != nil {
		return fmt.Errorf("routing: interface %q not found: %w", r.Interface, err)
	}

	dst := r.Destination
	nlRoute := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       &dst,
		Gw:        r.Gateway,
		Priority:  r.Metric,
	}

	if err := netlink.RouteAdd(nlRoute); err != nil {
		if routeAlreadyExists(err) {
			addTrackedRoute(&m.routes, r)
			return nil
		}
		return fmt.Errorf("routing: add %s via %s: %w", r.Destination.String(), r.Interface, err)
	}
	addTrackedRoute(&m.routes, r)
	return nil
}

func (m *linuxRouteManager) Delete(r Route) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	link, err := netlink.LinkByName(r.Interface)
	if err != nil {
		return fmt.Errorf("routing: interface %q not found: %w", r.Interface, err)
	}

	dst := r.Destination
	nlRoute := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       &dst,
		Gw:        r.Gateway,
	}

	if err := netlink.RouteDel(nlRoute); err != nil {
		return fmt.Errorf("routing: delete %s via %s: %w", r.Destination.String(), r.Interface, err)
	}
	m.routes = removeRoute(m.routes, r)
	return nil
}

func (m *linuxRouteManager) List() ([]Route, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Route, len(m.routes))
	copy(out, m.routes)
	return out, nil
}

func (m *linuxRouteManager) Flush(tunnelInterface string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	link, err := netlink.LinkByName(tunnelInterface)
	if err != nil {
		// Interface already gone — nothing to flush.
		m.routes = filterInterface(m.routes, tunnelInterface)
		return nil
	}

	existingRoutes, err := netlink.RouteList(link, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("routing: list routes for %s: %w", tunnelInterface, err)
	}

	for i := range existingRoutes {
		_ = netlink.RouteDel(&existingRoutes[i]) // best-effort
	}
	m.routes = filterInterface(m.routes, tunnelInterface)
	return nil
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

func routeAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "file exists") || strings.Contains(msg, "already exists")
}
