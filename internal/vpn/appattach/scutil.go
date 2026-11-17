package appattach

import (
	"net"
	"strings"
)

// parseNCList returns the names of the connected services in
// `scutil --nc list` output, whose lines look like
//
//   - (Connected)      7A1F… VPN (com.nordvpn.macos.NEPacketTunnel) "NordVPN NordLynx" [VPN:com.nordvpn.macos]
//
// (macOS lists Network Extension VPN configurations of apps such as
// NordVPN, Proton VPN or Surfshark there). Best effort: the exact columns
// are not documented, so only the "(Connected)" marker and the first
// double-quoted field are relied on.
func parseNCList(out string) []string {
	var names []string
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "(Connected)") {
			continue
		}
		start := strings.IndexByte(line, '"')
		if start < 0 {
			continue
		}
		end := strings.IndexByte(line[start+1:], '"')
		if end < 0 {
			continue
		}
		names = append(names, line[start+1:start+1+end])
	}
	return names
}

// parseNCStatus extracts the tunnel interface ("InterfaceName : utun4")
// and its first IPv4 address from `scutil --nc status <service>` output.
// Best effort: both may be empty.
func parseNCStatus(out string) Tunnel {
	var t Tunnel
	for _, line := range strings.Split(out, "\n") {
		key, val, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "InterfaceName" && t.Name == "" {
			t.Name = val
			continue
		}
		if t.IP == nil {
			if ip := net.ParseIP(val); ip != nil && ip.To4() != nil && !ip.IsUnspecified() {
				t.IP = ip.To4()
			}
		}
	}
	return t
}

// pickService returns the first connected service matching a pattern.
func pickService(connected, patterns []string) (string, bool) {
	for _, name := range connected {
		if anyGlob(patterns, name) {
			return name, true
		}
	}
	return "", false
}
