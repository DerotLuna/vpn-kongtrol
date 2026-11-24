package openvpn

import (
	"bufio"
	"net"
	"os"
	"strings"
)

// ConfigRoutes reads the routes a client .ovpn asks for itself (ADR-0107):
//
//	route <network> [<netmask>] [<gateway>] [<metric>]
//	route-ipv6 <addr>/<bits> [<gateway>] [<metric>]
//	redirect-gateway / redirect-def1   (all IPv4 traffic: "0.0.0.0/0")
//
// They come from the file, so Kongtrol shows them but never edits them.
// A route whose network is a hostname or a keyword is skipped, and so is
// one whose gateway is net_gateway: that one keeps the destination OFF the
// tunnel (OpenVPN 2.6 manual, --route). Canonical CIDRs, deduplicated, in
// file order. The error is the file's (not found, permission denied).
func ConfigRoutes(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	add := func(cidr string) {
		for _, e := range out {
			if e == cidr {
				return
			}
		}
		out = append(out, cidr)
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		fields := strings.Fields(line)
		switch strings.ToLower(strings.TrimPrefix(fields[0], "--")) {
		case "route":
			if len(fields) < 2 {
				continue
			}
			if len(fields) >= 4 && strings.EqualFold(fields[3], "net_gateway") {
				continue
			}
			ip := net.ParseIP(fields[1]).To4()
			if ip == nil {
				continue
			}
			mask := net.CIDRMask(32, 32)
			if len(fields) >= 3 && !strings.EqualFold(fields[2], "default") {
				m := net.ParseIP(fields[2]).To4()
				if m == nil {
					continue
				}
				mask = net.IPMask(m)
				if ones, bits := mask.Size(); ones == 0 && bits == 0 {
					continue // not a contiguous netmask
				}
			}
			add((&net.IPNet{IP: ip.Mask(mask), Mask: mask}).String())
		case "route-ipv6":
			if len(fields) < 2 {
				continue
			}
			if len(fields) >= 3 && strings.EqualFold(fields[2], "net_gateway") {
				continue
			}
			s := fields[1]
			if !strings.Contains(s, "/") {
				s += "/128"
			}
			if _, n, err := net.ParseCIDR(s); err == nil {
				add(n.String())
			}
		case "redirect-gateway", "redirect-def1":
			v4 := true
			for _, flag := range fields[1:] {
				switch strings.ToLower(flag) {
				case "!ipv4":
					v4 = false
				case "ipv6":
					add("::/0")
				}
			}
			if v4 {
				add("0.0.0.0/0")
			}
		}
	}
	return out, sc.Err()
}
