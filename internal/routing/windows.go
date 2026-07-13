//go:build windows

package routing

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
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

// List returns VPN-related routes from the OS routing table.
// Includes both kongtrol-managed routes and routes added by VPN clients
// (WireGuard, FortiClient, etc.), plus default route(s) for visibility.
func (m *windowsRouteManager) List() ([]Route, error) {
	// Read VPN-related routes from the OS route table.
	// First, find VPN adapter interface indexes (by description keywords).
	// Then, get routes for those interfaces + default route(s).
	ps := `$vpnAdapters = Get-NetAdapter | Where-Object { ` +
		`$_.InterfaceDescription -like '*Fortinet*' -or ` +
		`$_.InterfaceDescription -like '*WireGuard*' -or ` +
		`$_.InterfaceDescription -like '*TAP*' -or ` +
		`$_.InterfaceDescription -like '*TUN*' -or ` +
		`$_.InterfaceDescription -like '*VPN*' -or ` +
		`$_.InterfaceDescription -like '*OpenVPN*' -or ` +
		`$_.InterfaceDescription -like '*ProtonVPN*' -or ` +
		`$_.InterfaceDescription -like '*Cloudflare*' -or ` +
		`$_.InterfaceDescription -like '*Tailscale*' -or ` +
		`$_.InterfaceDescription -like '*GlobalProtect*' -or ` +
		`$_.InterfaceDescription -like '*Cisco AnyConnect*' ` +
		`}; ` +
		`$idxs = @(); if ($vpnAdapters) { $idxs = $vpnAdapters.InterfaceIndex }; ` +
		`Get-NetRoute -AddressFamily IPv4 -ErrorAction SilentlyContinue | ` +
		`Where-Object { ($idxs -contains $_.InterfaceIndex -and $_.DestinationPrefix -ne '255.255.255.255/32') -or $_.DestinationPrefix -eq '0.0.0.0/0' } | ` +
		`ForEach-Object { ` +
		`$idx = $_.InterfaceIndex; ` +
		`$name = $null; ` +
		`if ($vpnAdapters) { $name = ($vpnAdapters | Where-Object { $_.InterfaceIndex -eq $idx }).Name }; ` +
		`if (-not $name) { $name = (Get-NetAdapter -InterfaceIndex $idx -ErrorAction SilentlyContinue).Name }; ` +
		`if (-not $name) { $name = "if-" + $idx }; ` +
		`"$($_.DestinationPrefix)|$($_.NextHop)|$name|$($_.RouteMetric)" }`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: return kongtrol-managed routes only.
		m.mu.Lock()
		defer m.mu.Unlock()
		result := make([]Route, len(m.routes))
		copy(result, m.routes)
		return result, nil
	}

	var routes []Route
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		dest, gw, iface, metricStr := parts[0], parts[1], parts[2], parts[3]

		// Skip link-local and multicast.
		if strings.HasPrefix(dest, "169.254.") || strings.HasPrefix(dest, "224.") {
			continue
		}

		_, ipnet, err := net.ParseCIDR(dest)
		if err != nil {
			continue
		}
		metric, _ := strconv.Atoi(metricStr)

		r := Route{
			Destination: *ipnet,
			Interface:   iface,
			Metric:      metric,
		}
		if gwIP := net.ParseIP(gw); gwIP != nil && !gwIP.Equal(net.IPv4zero) {
			r.Gateway = gwIP
		}
		routes = append(routes, r)
	}
	return routes, nil
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
