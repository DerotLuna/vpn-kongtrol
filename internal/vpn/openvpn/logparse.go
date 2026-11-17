package openvpn

import (
	"net"
	"regexp"
	"strings"
	"sync"
)

// Patterns for the tunnel device line openvpn prints at --verb 3:
//
//	Windows: "DCO device [OpenVPN Data Channel Offload] opened"
//	         "Wintun device [OpenVPN Wintun] opened"
//	         "TAP-WIN32 device [OpenVPN TAP-Windows6] opened: \\.\Global\{...}.tap"
//	Linux:   "TUN/TAP device tun0 opened" / "DCO device tun0 opened"
//	macOS:   "Opened utun device utun4"
var ifacePatterns = []*regexp.Regexp{
	regexp.MustCompile(`device \[([^\]]+)\] opened`),
	regexp.MustCompile(`(?:TUN/TAP|DCO) device (\S+) opened`),
	regexp.MustCompile(`Opened utun device (\S+)`),
}

// parseInterface returns the tunnel interface name if line announces it.
func parseInterface(line string) (string, bool) {
	for _, re := range ifacePatterns {
		if m := re.FindStringSubmatch(line); m != nil {
			return m[1], true
		}
	}
	return "", false
}

// parsePushedDNS extracts DNS servers from a PUSH_REPLY log line. It handles
// both the classic "dhcp-option DNS <ip>" and the OpenVPN 2.6
// "dns server <n> address <ip>[:port] [<ip>...]" forms.
func parsePushedDNS(line string) []net.IP {
	idx := strings.Index(line, "PUSH_REPLY")
	if idx < 0 {
		return nil
	}
	var out []net.IP
	for _, opt := range strings.Split(line[idx:], ",") {
		f := strings.Fields(strings.Trim(opt, "' "))
		switch {
		case len(f) >= 3 && f[0] == "dhcp-option" && strings.EqualFold(f[1], "DNS"):
			out = appendIP(out, f[2])
		case len(f) >= 5 && f[0] == "dns" && f[1] == "server" && f[3] == "address":
			for _, a := range f[4:] {
				out = appendIP(out, a)
			}
		}
	}
	return out
}

func appendIP(out []net.IP, s string) []net.IP {
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	s = strings.Trim(s, "[]")
	if ip := net.ParseIP(s); ip != nil {
		for _, e := range out {
			if e.Equal(ip) {
				return out
			}
		}
		out = append(out, ip)
	}
	return out
}

// pushedRoutes is what one PUSH_REPLY line says about routing.
type pushedRoutes struct {
	routes    []string // CIDRs from route / route-ipv6
	redirect4 bool     // redirect-gateway without !ipv4: all IPv4 traffic
	cont      bool     // push-continuation 2: more PUSH_REPLY lines follow
}

// parsePushedRoutes extracts the routing options of a PUSH_REPLY line:
//
//	route <network> [<netmask>] [<gateway>] [<metric>]
//	route-ipv6 <addr>/<bits> [<gateway>] [<metric>]
//	redirect-gateway [def1] [ipv6] [!ipv4] [bypass-dhcp] ...
//
// redirect-gateway with "ipv6" adds ::/0. Routes whose network is a
// hostname are skipped.
func parsePushedRoutes(line string) (pushedRoutes, bool) {
	idx := strings.Index(line, "PUSH_REPLY")
	if idx < 0 {
		return pushedRoutes{}, false
	}
	var out pushedRoutes
	add := func(cidr string) {
		for _, e := range out.routes {
			if e == cidr {
				return
			}
		}
		out.routes = append(out.routes, cidr)
	}
	for _, opt := range strings.Split(line[idx:], ",") {
		f := strings.Fields(strings.Trim(opt, "' "))
		if len(f) == 0 {
			continue
		}
		switch f[0] {
		case "route":
			if len(f) < 2 {
				continue
			}
			ip := net.ParseIP(f[1]).To4()
			if ip == nil {
				continue
			}
			mask := net.CIDRMask(32, 32)
			if len(f) >= 3 {
				m := net.ParseIP(f[2]).To4()
				if m == nil {
					continue
				}
				mask = net.IPMask(m)
				if ones, bits := mask.Size(); bits == 0 && ones == 0 {
					continue // not a contiguous netmask
				}
			}
			add((&net.IPNet{IP: ip.Mask(mask), Mask: mask}).String())
		case "route-ipv6":
			if len(f) < 2 {
				continue
			}
			s := f[1]
			if !strings.Contains(s, "/") {
				s += "/128"
			}
			if _, n, err := net.ParseCIDR(s); err == nil {
				add(n.String())
			}
		case "redirect-gateway":
			v4 := true
			for _, flag := range f[1:] {
				switch flag {
				case "!ipv4":
					v4 = false
				case "ipv6":
					add("::/0")
				}
			}
			out.redirect4 = out.redirect4 || v4
		case "push-continuation":
			out.cont = len(f) >= 2 && f[1] == "2"
		}
	}
	return out, true
}

// logState accumulates what the adapter learns from openvpn's stdout.
type logState struct {
	mu    sync.Mutex
	iface string
	dns   []net.IP
	tail  []string // last maxTail lines, for error reporting

	// Routes the server pushed (latest PUSH_REPLY, continuations merged).
	routes   []string
	redirect bool
	pushCont bool // the last PUSH_REPLY announced a continuation
}

const maxTail = 15

func (s *logState) consume(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parseLocked(line, true)
	s.tail = append(s.tail, line)
	if len(s.tail) > maxTail {
		s.tail = s.tail[len(s.tail)-maxTail:]
	}
}

// parseLocked learns the interface, DNS and (when withPush) routes from line.
func (s *logState) parseLocked(line string, withPush bool) {
	if name, ok := parseInterface(line); ok {
		s.iface = name
	}
	if dns := parsePushedDNS(line); len(dns) > 0 {
		s.dns = dns
	}
	if !withPush {
		return
	}
	if p, ok := parsePushedRoutes(line); ok {
		// A new PUSH_REPLY replaces the previous one, unless the previous
		// one said more lines follow.
		if !s.pushCont {
			s.routes, s.redirect = nil, false
		}
		for _, r := range p.routes {
			if !containsString(s.routes, r) {
				s.routes = append(s.routes, r)
			}
		}
		s.redirect = s.redirect || p.redirect4
		s.pushCont = p.cont
	}
}

func containsString(list []string, v string) bool {
	for _, e := range list {
		if e == v {
			return true
		}
	}
	return false
}

func (s *logState) snapshot() (iface string, dns []net.IP) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.iface, append([]net.IP(nil), s.dns...)
}

// pushed returns the routes the server pushed and whether it redirected
// all IPv4 traffic; ok is false when no PUSH_REPLY was seen.
func (s *logState) pushed() (routes []string, redirect, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.routes...), s.redirect, s.routes != nil || s.redirect
}

// lastErrors returns the most relevant recent lines (those mentioning an
// error), falling back to the raw tail, so a failed start explains itself.
func (s *logState) lastErrors() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var hits []string
	for _, l := range s.tail {
		ll := strings.ToLower(l)
		if strings.Contains(ll, "error") || strings.Contains(ll, "fatal") ||
			strings.Contains(ll, "failed") || strings.Contains(ll, "denied") ||
			strings.Contains(ll, "cannot") || strings.Contains(ll, "exiting") {
			hits = append(hits, strings.TrimSpace(l))
		}
	}
	if len(hits) == 0 {
		start := len(s.tail) - 3
		if start < 0 {
			start = 0
		}
		hits = s.tail[start:]
	}
	return strings.Join(hits, " | ")
}

// replaceTail takes the helper's view of openvpn's recent output (a
// helper-run openvpn has no stdout here): every line is parsed for the
// tunnel interface and pushed DNS, and the error tail is replaced.
// pushReply, when the helper sends it, is the authoritative list of
// PUSH_REPLY lines (it survives the log scrolling); otherwise the routes
// come from the PUSH_REPLY lines still in the tail.
func (s *logState) replaceTail(lines, pushReply []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pushCont = false
	if len(pushReply) > 0 {
		for _, line := range pushReply {
			s.parseLocked(line, true)
		}
	}
	for _, line := range lines {
		s.parseLocked(line, len(pushReply) == 0)
	}
	if len(lines) > maxTail {
		lines = lines[len(lines)-maxTail:]
	}
	s.tail = append(s.tail[:0], lines...)
}

// interfaceWithIP returns the name of the interface holding ip, used when
// the log did not name the tunnel device.
func interfaceWithIP(ip net.IP) string {
	if ip == nil {
		return ""
	}
	ifaces, _ := net.Interfaces()
	for _, ifi := range ifaces {
		addrs, _ := ifi.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.Equal(ip) {
				return ifi.Name
			}
		}
	}
	return ""
}
