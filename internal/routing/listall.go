package routing

import (
	"net"
	"strconv"
	"strings"
)

// ListAll returns every unicast route in the OS routing table (IPv4 and
// IPv6, any interface), read-only and without elevation. Callers filter by
// Route.Interface to see what goes through one tunnel.
func ListAll() ([]Route, error) { return listAllOS() }

// parseNetRouteLines parses "dest|nexthop|alias|metric" lines (Windows
// Get-NetRoute output).
func parseNetRouteLines(out string) []Route {
	var routes []Route
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 4)
		if len(parts) < 4 {
			continue
		}
		_, n, err := net.ParseCIDR(parts[0])
		if err != nil {
			continue
		}
		r := Route{Destination: *n, Interface: parts[2]}
		r.Metric, _ = strconv.Atoi(parts[3])
		if gw := net.ParseIP(parts[1]); gw != nil && !gw.IsUnspecified() {
			r.Gateway = gw
		}
		routes = append(routes, r)
	}
	return routes
}

// parseIPRouteLines parses `ip route show` / `ip -6 route show` output
// (Linux): "<dest> [via <gw>] dev <iface> ... [metric <n>]".
func parseIPRouteLines(out string, v6 bool) []Route {
	var routes []Route
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		dest := f[0]
		if dest == "default" {
			dest = "0.0.0.0/0"
			if v6 {
				dest = "::/0"
			}
		}
		n := parseDest(dest, v6)
		if n == nil {
			continue // unreachable, blackhole, broadcast, …
		}
		r := Route{Destination: *n}
		for i := 1; i+1 < len(f); i++ {
			switch f[i] {
			case "via":
				r.Gateway = net.ParseIP(f[i+1])
			case "dev":
				r.Interface = f[i+1]
			case "metric":
				r.Metric, _ = strconv.Atoi(f[i+1])
			}
		}
		if r.Interface != "" {
			routes = append(routes, r)
		}
	}
	return routes
}

// parseNetstatRoutes parses `netstat -rn` output (macOS): sections
// "Internet:" / "Internet6:", columns Destination Gateway Flags Netif ….
func parseNetstatRoutes(out string) []Route {
	var routes []Route
	v6, netifCol, flagsCol := false, -1, -1
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		switch {
		case len(f) == 0:
			continue
		case f[0] == "Internet:":
			v6 = false
			continue
		case f[0] == "Internet6:":
			v6 = true
			continue
		case f[0] == "Destination":
			netifCol, flagsCol = -1, -1
			for i, h := range f {
				switch h {
				case "Netif":
					netifCol = i
				case "Flags":
					flagsCol = i
				}
			}
			continue
		}
		if netifCol < 0 || len(f) <= netifCol {
			continue
		}
		dest := f[0]
		if i := strings.IndexByte(dest, '%'); i >= 0 { // fe80::%utun4/64
			rest := dest[i:]
			dest = dest[:i]
			if j := strings.IndexByte(rest, '/'); j >= 0 {
				dest += rest[j:]
			}
		}
		if dest == "default" {
			dest = "0.0.0.0/0"
			if v6 {
				dest = "::/0"
			}
		} else if !v6 && !strings.Contains(dest, "/") {
			host := flagsCol >= 0 && flagsCol < len(f) && strings.Contains(f[flagsCol], "H")
			dest = expandV4(dest, host)
		} else if !v6 {
			addr, bits, _ := strings.Cut(dest, "/")
			dest = expandV4(addr, true)
			dest = dest[:strings.IndexByte(dest, '/')] + "/" + bits
		}
		n := parseDest(dest, v6)
		if n == nil {
			continue
		}
		r := Route{Destination: *n, Interface: f[netifCol]}
		if gw := net.ParseIP(f[1]); gw != nil {
			r.Gateway = gw
		}
		routes = append(routes, r)
	}
	return routes
}

// expandV4 turns netstat's abbreviated "10.200" into "10.200.0.0/16"
// (host: "/32").
func expandV4(s string, host bool) string {
	octets := strings.Split(s, ".")
	n := len(octets)
	for len(octets) < 4 {
		octets = append(octets, "0")
	}
	bits := 32
	if !host {
		bits = 8 * n
	}
	return strings.Join(octets[:4], ".") + "/" + strconv.Itoa(bits)
}

// parseDest parses "a.b.c.d/n", a bare address (host route) or "::/0".
func parseDest(dest string, v6 bool) *net.IPNet {
	if !strings.Contains(dest, "/") {
		ip := net.ParseIP(dest)
		if ip == nil {
			return nil
		}
		if ip.To4() != nil && !v6 {
			dest += "/32"
		} else {
			dest += "/128"
		}
	}
	_, n, err := net.ParseCIDR(dest)
	if err != nil {
		return nil
	}
	return n
}
