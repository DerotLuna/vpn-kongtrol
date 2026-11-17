//go:build windows

package routing

import (
	"net"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// listOS returns VPN-related routes from the OS routing table (read-only,
// no elevation needed).
// Includes both kongtrol-managed routes and routes added by VPN clients
// (WireGuard, FortiClient, etc.), plus default route(s) for visibility.
func listOS() ([]Route, bool) {
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
		`$_.InterfaceDescription -like '*NordLynx*' -or ` +
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
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: the caller returns kongtrol-managed routes only.
		return nil, false
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
	return routes, true
}
