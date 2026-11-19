package api

import (
	"bytes"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// tunnelRouteDTO is one destination that goes through a tunnel.
type tunnelRouteDTO struct {
	Destination string `json:"destination"`
	// Origin is "server" (pushed by the VPN server), "policy" (installed by
	// Kongtrol for a policy) or "vpn" (the VPN's own config / OS route on
	// the tunnel interface).
	Origin  string `json:"origin"`
	Policy  string `json:"policy,omitempty"`
	Gateway string `json:"gateway,omitempty"`
}

// tunnelRoutesDTO is what goes through one tunnel.
type tunnelRoutesDTO struct {
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	Interface string `json:"interface,omitempty"`
	// AllTraffic: the tunnel took the default route (redirect-gateway,
	// AllowedIPs 0.0.0.0/0): all IPv4 traffic goes through it.
	AllTraffic bool             `json:"all_traffic"`
	Total      int              `json:"total"`
	Counts     map[string]int   `json:"counts"`
	Routes     []tunnelRouteDTO `json:"routes"`
}

// tunnelRoutesResponse answers GET /api/v1/tunnels/routes and
// GET /api/v1/tunnels/{name}/routes.
type tunnelRoutesResponse struct {
	// RestVia is the interface everything else leaves through (the
	// default route not on a tunnel, e.g. "Wi-Fi").
	RestVia string `json:"rest_via,omitempty"`
	// RestViaTunnel names the profile that took all traffic, if any.
	RestViaTunnel string            `json:"rest_via_tunnel,omitempty"`
	Tunnels       []tunnelRoutesDTO `json:"tunnels"`
}

// osRoutesCache keeps the OS route table for a few seconds: reading it can
// take a second on Windows and the dashboard + CLI ask often.
var osRoutesCache struct {
	sync.Mutex
	at     time.Time
	routes []routing.Route
}

var listAllRoutes = routing.ListAll

func cachedOSRoutes() []routing.Route {
	osRoutesCache.Lock()
	defer osRoutesCache.Unlock()
	if time.Since(osRoutesCache.at) < 5*time.Second {
		return osRoutesCache.routes
	}
	routes, err := listAllRoutes()
	if err != nil {
		routes = nil
	}
	osRoutesCache.routes, osRoutesCache.at = routes, time.Now()
	return routes
}

// GET /api/v1/tunnels/routes — what goes through every connected tunnel.
func (s *Server) handleAllTunnelRoutes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.tunnelRoutes(""))
}

// GET /api/v1/tunnels/{name}/routes — what goes through one tunnel.
func (s *Server) handleTunnelRoutes(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, ok := s.adapters.Get(name); !ok {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}
	writeJSON(w, http.StatusOK, s.tunnelRoutes(name))
}

// tunnelRoutes builds the response for one profile (only) or every
// connected profile (only == "").
func (s *Server) tunnelRoutes(only string) tunnelRoutesResponse {
	osRoutes := cachedOSRoutes()
	var rules []policy.Rule
	if pe := s.policyEngine.Load(); pe != nil {
		rules = pe.Rules()
	}
	resolved := map[string][]string{} // profile → policy CIDRs Kongtrol installed
	if s.policyResolver != nil {
		for _, snap := range s.policyResolver.Snapshot() {
			resolved[snap.Name] = append(append(resolved[snap.Name], snap.IPRanges...), snap.ResolvedCIDRs...)
		}
	} else {
		for _, rule := range rules {
			for _, n := range rule.Match.IPRanges {
				resolved[rule.Via] = append(resolved[rule.Via], n.String())
			}
		}
	}

	var snapshot map[string]tunnelSnapshot
	if s.collector != nil {
		snapshot = map[string]tunnelSnapshot{}
		for name, m := range s.collector.Snapshot() {
			snapshot[name] = tunnelSnapshot{
				connected: m.Status.Normalize() == vpn.StatusConnected,
				iface:     m.InterfaceName,
				ip:        net.ParseIP(m.AssignedIP),
			}
		}
	}

	out := tunnelRoutesResponse{Tunnels: []tunnelRoutesDTO{}}
	tunnelIfaces := map[string]bool{}
	names := s.adapters.Names()
	for _, name := range names {
		snap := snapshot[name]
		if only != "" && name != only {
			if snap.connected {
				for _, i := range tunnelIfaceNames(snap) {
					tunnelIfaces[i] = true
				}
			}
			continue
		}
		if !snap.connected {
			if only != "" {
				out.Tunnels = append(out.Tunnels, tunnelRoutesDTO{Name: name, Counts: map[string]int{}, Routes: []tunnelRouteDTO{}})
			}
			continue
		}
		ifaces := tunnelIfaceNames(snap)
		for _, i := range ifaces {
			tunnelIfaces[i] = true
		}
		var reported *vpn.TunnelRoutes
		if rr, ok := s.adapterOf(name).(vpn.RouteReporter); ok {
			reported = rr.TunnelRoutes()
		}
		dto := buildTunnelRoutes(name, ifaces, snap.ip, osRoutes, reported, policyNames(name, resolved[name], rules))
		dto.Connected = true
		if dto.AllTraffic && out.RestViaTunnel == "" {
			out.RestViaTunnel = name
		}
		out.Tunnels = append(out.Tunnels, dto)
	}
	out.RestVia = restVia(osRoutes, tunnelIfaces)
	return out
}

type tunnelSnapshot struct {
	connected bool
	iface     string
	ip        net.IP
}

// tunnelIfaceNames returns the names the OS may use for the tunnel: the
// adapter's (OpenVPN prints the device name, which on Windows is not
// always the interface alias) and the one holding the tunnel IP.
func tunnelIfaceNames(t tunnelSnapshot) []string {
	var out []string
	if t.iface != "" {
		out = append(out, t.iface)
	}
	if n := interfaceHolding(t.ip); n != "" && n != t.iface {
		out = append(out, n)
	}
	return out
}

func interfaceHolding(ip net.IP) string {
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

// policyNames maps each policy CIDR installed for profile to the name of
// the policy that asked for it.
func policyNames(profile string, cidrs []string, rules []policy.Rule) map[string]string {
	out := make(map[string]string, len(cidrs))
	var own []policy.Rule
	fallback := ""
	for _, r := range rules {
		if r.Via == profile {
			own = append(own, r)
			if fallback == "" && len(r.Match.Domains) > 0 {
				fallback = r.Name
			}
		}
	}
	if fallback == "" && len(own) > 0 {
		fallback = own[0].Name
	}
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			continue
		}
		_, name := matchStaticPolicy(*n, own)
		if name == "" {
			name = fallback
		}
		out[n.String()] = name
	}
	return out
}

// restVia returns the interface of the best default route that is not a
// tunnel: where everything else goes.
func restVia(routes []routing.Route, tunnelIfaces map[string]bool) string {
	best, bestMetric := "", 0
	for _, r := range routes {
		if !isDefaultRoute(r.Destination) || r.Destination.IP.To4() == nil || tunnelIfaces[r.Interface] {
			continue
		}
		if best == "" || r.Metric < bestMetric {
			best, bestMetric = r.Interface, r.Metric
		}
	}
	return best
}

var originRank = map[string]int{string(vpn.OriginServer): 0, string(vpn.OriginPolicy): 1, string(vpn.OriginVPN): 2}

// buildTunnelRoutes merges what the adapter reported, the policy routes
// Kongtrol installed for the profile, and the OS routes on its interface
// into one list grouped by origin.
func buildTunnelRoutes(name string, ifaces []string, assigned net.IP, osRoutes []routing.Route,
	reported *vpn.TunnelRoutes, policies map[string]string) tunnelRoutesDTO {
	dto := tunnelRoutesDTO{Name: name, Counts: map[string]int{}, Routes: []tunnelRouteDTO{}}
	if len(ifaces) > 0 {
		dto.Interface = ifaces[len(ifaces)-1]
	}
	byDest := map[string]*tunnelRouteDTO{}
	add := func(cidr, origin, pol string) *tunnelRouteDTO {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil
		}
		key := n.String()
		if e, ok := byDest[key]; ok {
			return e
		}
		e := &tunnelRouteDTO{Destination: key, Origin: origin, Policy: pol}
		byDest[key] = e
		return e
	}

	// Policies first: Kongtrol installed them, the user named them.
	for cidr, pol := range policies {
		add(cidr, string(vpn.OriginPolicy), pol)
	}
	if reported != nil {
		dto.AllTraffic = reported.AllTraffic
		origin := string(reported.Origin)
		if origin == "" {
			origin = string(vpn.OriginVPN)
		}
		for _, cidr := range reported.Routes {
			add(cidr, origin, "")
		}
	}

	onIface := map[string]bool{}
	for _, i := range ifaces {
		onIface[i] = true
	}
	var mine []routing.Route
	var connected []net.IPNet
	for _, r := range osRoutes {
		if !onIface[r.Interface] {
			continue
		}
		mine = append(mine, r)
		ones, bits := r.Destination.Mask.Size()
		if r.Gateway == nil && assigned != nil && ones < bits && r.Destination.Contains(assigned) && ones > 0 {
			connected = append(connected, r.Destination)
		}
	}
	half1, half2 := false, false
	for _, r := range mine {
		dest := r.Destination.String()
		switch dest {
		case "0.0.0.0/0":
			dto.AllTraffic = true
			continue
		case "0.0.0.0/1":
			half1 = true
			continue
		case "128.0.0.0/1":
			half2 = true
			continue
		}
		if isNoiseRoute(r, assigned, connected) {
			continue
		}
		e := add(dest, string(vpn.OriginVPN), "")
		if e != nil && e.Gateway == "" && r.Gateway != nil {
			e.Gateway = r.Gateway.String()
		}
	}
	if half1 && half2 {
		dto.AllTraffic = true
	}

	for _, e := range byDest {
		dto.Routes = append(dto.Routes, *e)
		dto.Counts[e.Origin]++
	}
	dto.Total = len(dto.Routes)
	sort.Slice(dto.Routes, func(i, j int) bool {
		a, b := dto.Routes[i], dto.Routes[j]
		if originRank[a.Origin] != originRank[b.Origin] {
			return originRank[a.Origin] < originRank[b.Origin]
		}
		if a.Policy != b.Policy {
			return a.Policy < b.Policy
		}
		return lessCIDR(a.Destination, b.Destination)
	})
	return dto
}

// isNoiseRoute drops what the OS adds to every interface on its own:
// multicast, broadcast, link-local, the tunnel's own address and its
// point-to-point subnet.
func isNoiseRoute(r routing.Route, assigned net.IP, connected []net.IPNet) bool {
	ip := r.Destination.IP
	ones, bits := r.Destination.Mask.Size()
	if ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.Equal(net.IPv4bcast) {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil && (ip4[0] >= 224) {
		return true
	}
	if assigned != nil && ones == bits && ip.Equal(assigned) {
		return true
	}
	for _, c := range connected {
		if c.String() == r.Destination.String() {
			return true
		}
		if r.Gateway == nil && ones == bits && c.Contains(ip) {
			return true // network / broadcast address of the tunnel subnet
		}
	}
	return false
}

func lessCIDR(a, b string) bool {
	ipA, nA, errA := net.ParseCIDR(a)
	ipB, nB, errB := net.ParseCIDR(b)
	if errA != nil || errB != nil {
		return a < b
	}
	a4, b4 := ipA.To4() != nil, ipB.To4() != nil
	if a4 != b4 {
		return a4
	}
	if c := bytes.Compare(ipA.To16(), ipB.To16()); c != 0 {
		return c < 0
	}
	oa, _ := nA.Mask.Size()
	ob, _ := nB.Mask.Size()
	return oa < ob
}
