//go:build darwin

package routing

import (
	"fmt"
	"os/exec"
	"sync"
)

// darwinRouteManager manages routes on macOS using the route(8) command.
type darwinRouteManager struct {
	mu     sync.Mutex
	routes []Route
}

// NewRouteManager returns the macOS route manager.
func NewRouteManager() RouteManager {
	return &darwinRouteManager{}
}

func (m *darwinRouteManager) Add(r Route) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := []string{"add", "-net", r.Destination.String()}
	if r.Gateway != nil {
		args = append(args, r.Gateway.String())
	} else {
		args = append(args, "-interface", r.Interface)
	}

	if err := routeCmd(args...); err != nil {
		return fmt.Errorf("routing: add %s via %s: %w", r.Destination.String(), r.Interface, err)
	}
	m.routes = append(m.routes, r)
	return nil
}

func (m *darwinRouteManager) Delete(r Route) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := []string{"delete", "-net", r.Destination.String()}
	if r.Gateway != nil {
		args = append(args, r.Gateway.String())
	}

	if err := routeCmd(args...); err != nil {
		return fmt.Errorf("routing: delete %s: %w", r.Destination.String(), err)
	}
	m.routes = removeRoute(m.routes, r)
	return nil
}

func (m *darwinRouteManager) List() ([]Route, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Route, len(m.routes))
	copy(out, m.routes)
	return out, nil
}

func (m *darwinRouteManager) Flush(tunnelInterface string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var remaining []Route
	for _, r := range m.routes {
		if r.Interface == tunnelInterface {
			args := []string{"delete", "-net", r.Destination.String()}
			if r.Gateway != nil {
				args = append(args, r.Gateway.String())
			}
			_ = routeCmd(args...) // best-effort
		} else {
			remaining = append(remaining, r)
		}
	}
	m.routes = remaining
	return nil
}

func routeCmd(args ...string) error {
	cmd := exec.Command("/sbin/route", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("route %v: %w\noutput: %s", args, err, out)
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
