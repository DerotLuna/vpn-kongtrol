//go:build windows

package routing

import (
	"fmt"
	"net"
	"os/exec"
	"sync"
)

// windowsRouteManager manages routes on Windows using netsh.
type windowsRouteManager struct {
	mu     sync.Mutex
	routes []Route // tracks routes added by kongtrol
}

// NewRouteManager returns the Windows route manager.
func NewRouteManager() RouteManager {
	return &windowsRouteManager{}
}

func (m *windowsRouteManager) Add(r Route) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mask := net.IP(r.Destination.Mask)
	args := []string{
		"interface", "ipv4", "add", "route",
		r.Destination.String(),
		r.Interface,
	}
	if r.Gateway != nil {
		args = append(args, r.Gateway.String())
	}
	args = append(args, fmt.Sprintf("metric=%d", r.Metric))
	args = append(args, "store=active")
	_ = mask // used implicitly in Destination.String()

	if err := netsh(args...); err != nil {
		return fmt.Errorf("routing: add %s via %s: %w", r.Destination.String(), r.Interface, err)
	}
	m.routes = append(m.routes, r)
	return nil
}

func (m *windowsRouteManager) Delete(r Route) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := []string{
		"interface", "ipv4", "delete", "route",
		r.Destination.String(),
		r.Interface,
	}
	if err := netsh(args...); err != nil {
		return fmt.Errorf("routing: delete %s via %s: %w", r.Destination.String(), r.Interface, err)
	}
	m.routes = removeRoute(m.routes, r)
	return nil
}

func (m *windowsRouteManager) List() ([]Route, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Route, len(m.routes))
	copy(out, m.routes)
	return out, nil
}

func (m *windowsRouteManager) Flush(tunnelInterface string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var remaining []Route
	for _, r := range m.routes {
		if r.Interface == tunnelInterface {
			args := []string{
				"interface", "ipv4", "delete", "route",
				r.Destination.String(),
				r.Interface,
			}
			_ = netsh(args...) // best-effort
		} else {
			remaining = append(remaining, r)
		}
	}
	m.routes = remaining
	return nil
}

func netsh(args ...string) error {
	cmd := exec.Command("netsh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh %v: %w\noutput: %s", args, err, out)
	}
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
