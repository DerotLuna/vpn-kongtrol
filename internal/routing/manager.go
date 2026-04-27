// Package routing provides OS-specific route table management.
package routing

import "net"

// Route represents a single entry in the OS routing table.
type Route struct {
	Destination net.IPNet
	Gateway     net.IP
	Interface   string // network interface name (e.g. "tun0", "vpn0")
	Metric      int
}

// RouteManager manages routes in the OS routing table.
// Platform-specific implementations are in windows.go, linux.go, darwin.go.
type RouteManager interface {
	// Add inserts a route into the routing table.
	Add(r Route) error

	// Delete removes a route from the routing table.
	Delete(r Route) error

	// List returns all routes currently managed by kongtrol.
	List() ([]Route, error)

	// Flush removes all routes associated with a given tunnel interface.
	Flush(tunnelInterface string) error
}
