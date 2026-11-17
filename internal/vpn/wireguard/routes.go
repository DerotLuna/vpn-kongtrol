package wireguard

import (
	"bufio"
	"net"
	"os"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// parseConfigAllowedIPs reads every [Peer] AllowedIPs entry of a .conf file
// as canonical CIDRs (deduplicated, in file order).
func parseConfigAllowedIPs(configPath string) []string {
	f, err := os.Open(configPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	inPeer := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			inPeer = strings.EqualFold(line, "[Peer]")
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !inPeer || !ok || !strings.EqualFold(strings.TrimSpace(k), "AllowedIPs") {
			continue
		}
		for _, s := range strings.Split(v, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if !strings.Contains(s, "/") {
				if ip := net.ParseIP(s); ip != nil && ip.To4() != nil {
					s += "/32"
				} else {
					s += "/128"
				}
			}
			_, n, err := net.ParseCIDR(s)
			if err != nil {
				continue
			}
			c := n.String()
			dup := false
			for _, e := range out {
				dup = dup || e == c
			}
			if !dup {
				out = append(out, c)
			}
		}
	}
	return out
}

// TunnelRoutes reports the peers' AllowedIPs: what WireGuard sends through
// the tunnel. Nil when not connected.
func (a *Adapter) TunnelRoutes() *vpn.TunnelRoutes {
	a.mu.RLock()
	path, status := a.lastCfg.ConfigPath, a.status
	a.mu.RUnlock()
	if path == "" || status != vpn.StatusConnected {
		return nil
	}
	routes := parseConfigAllowedIPs(path)
	if len(routes) == 0 {
		return nil
	}
	tr := &vpn.TunnelRoutes{Origin: vpn.OriginVPN}
	for _, r := range routes {
		if r == "0.0.0.0/0" {
			tr.AllTraffic = true
			continue
		}
		tr.Routes = append(tr.Routes, r)
	}
	return tr
}
