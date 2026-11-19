package app

import (
	"bufio"
	"context"
	"net"
	"net/netip"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

// KillSwitchService owns the set of blocked destinations, built from two
// sources (split-tunnel aware; see config.KillSwitchConfig):
//
//   - Kill switch: a protected tunnel that dropped unexpectedly (it was up,
//     the user did not turn it off) gets only its destinations blocked —
//     routes captured while it was up + its policies' IPs/CIDRs — until it
//     reconnects or the user runs `down`. A full-tunnel profile (default
//     route through the tunnel) blocks everything except the VPN endpoints
//     (and the LAN when allowed). A tunnel the user turned off is never
//     blocked: its traffic goes to the normal internet.
//   - Policies with on_tunnel_down: block whose tunnel is not connected.
//
// Destinations of a connected tunnel are never blocked (firewall rules
// apply on every interface, so they would cut that tunnel too).
// Everything goes to the helper as one leased set of BLOCK rules
// (privops.SetBlockedDestinations): fail-open if the daemon dies.
type KillSwitchService struct {
	mu       sync.Mutex
	applyMu  sync.Mutex
	cfg      *config.Config
	adapters *vpn.Set
	blocker  security.DestinationBlocker
	legacy   security.KillSwitch // legacy all-traffic rules, removed once
	dests    DestinationSource
	routes   func() ([]routing.Route, error)
	now      func() time.Time
	lookupIP func(host string) []net.IP

	state  map[string]*ksProfile
	status security.KillSwitchStatus
	// blocked is the prefix set last sent to the blocker (Blocked()).
	blocked     []netip.Prefix
	legacyClean bool
	// onBlock tells the daemon a dropped tunnel started being blocked.
	onBlock func(name string, fullTunnel bool)
}

// DestinationSource gives the policy destinations (static CIDRs + resolved
// domain IPs), implemented by monitor.PolicyResolver.
type DestinationSource interface {
	ProfileDestinations(profile string) []netip.Prefix
	PolicyDestinations(policy string) []netip.Prefix
}

// ksProfile is what the service remembers about one profile.
type ksProfile struct {
	wasUp      bool // connected at some point since the user last turned it on
	intended   bool // the user turned it off (down / disconnect)
	iface      string
	captured   []netip.Prefix // routes through the tunnel, captured while up
	capturedAt time.Time
	fullTunnel bool
	endpoints  []netip.Addr
}

// captureEvery bounds how often a connected tunnel's routes are re-read
// (listing the route table may spawn a process on Windows).
const captureEvery = 2 * time.Minute

// NewKillSwitchService returns the service. ks is the legacy kill switch,
// used only to remove all-traffic rules left by older releases.
func NewKillSwitchService(cfg *config.Config, adapters *vpn.Set, ks security.KillSwitch) *KillSwitchService {
	return &KillSwitchService{
		cfg:      cfg,
		adapters: adapters,
		legacy:   ks,
		now:      time.Now,
		state:    map[string]*ksProfile{},
		lookupIP: func(host string) []net.IP {
			ips, _ := net.LookupIP(host)
			return ips
		},
	}
}

// SetBlocker sets the destination blocker (privops-backed in the daemon).
func (s *KillSwitchService) SetBlocker(b security.DestinationBlocker) {
	s.mu.Lock()
	s.blocker = b
	s.mu.Unlock()
}

// SetDestinations sets the policy destination source.
func (s *KillSwitchService) SetDestinations(d DestinationSource) {
	s.mu.Lock()
	s.dests = d
	s.mu.Unlock()
}

// SetRouteLister sets how tunnel routes are captured (routing.RouteManager.List).
func (s *KillSwitchService) SetRouteLister(fn func() ([]routing.Route, error)) {
	s.mu.Lock()
	s.routes = fn
	s.mu.Unlock()
}

// SetConfig swaps the configuration and keeps the per-tunnel memory.
func (s *KillSwitchService) SetConfig(cfg *config.Config) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
}

// NoteIntent records that the user turned a tunnel off (intendedDown) or
// on. Turning it off is never a "drop": its block goes away at once.
func (s *KillSwitchService) NoteIntent(name string, intendedDown bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	st := s.profile(name)
	st.intended = intendedDown
	if intendedDown {
		st.wasUp = false
	}
	s.mu.Unlock()
}

// Status returns the last computed state.
func (s *KillSwitchService) Status() security.KillSwitchStatus {
	if s == nil {
		return security.KillSwitchStatus{State: security.KillSwitchOff}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status
	if st.State == "" {
		st.State = security.KillSwitchOff
		if s.anyEnabled() {
			st.State = security.KillSwitchArmed
		}
	}
	return st
}

// Run re-applies every interval until ctx ends (drop detection).
func (s *KillSwitchService) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = s.Apply()
		}
	}
}

func (s *KillSwitchService) profile(name string) *ksProfile {
	st, ok := s.state[name]
	if !ok {
		st = &ksProfile{}
		s.state[name] = st
	}
	return st
}

// Apply recomputes the block set from the tunnels' state and sends it.
func (s *KillSwitchService) Apply() error {
	if s == nil || s.cfg == nil {
		return nil
	}
	// Adapter status / tunnel info may run commands: read them unlocked.
	type obs struct {
		connected bool
		info      *vpn.TunnelInfo
	}
	seen := map[string]obs{}
	for name, a := range s.adapters.Snapshot() {
		o := obs{connected: a.Status().Normalize() == vpn.StatusConnected}
		if o.connected {
			if info, err := a.TunnelInfo(); err == nil {
				o.info = info
			}
		}
		seen[name] = o
	}

	// applyMu serializes whole Apply runs; s.mu is released while the
	// helper works so Status never waits for it.
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
	s.mu.Lock()
	unlocked := false
	defer func() {
		if !unlocked {
			s.mu.Unlock()
		}
	}()
	s.cleanLegacy()

	// 1. Remember what each connected tunnel carries.
	var routes []routing.Route
	routesRead := false
	for name, o := range seen {
		st := s.profile(name)
		if !o.connected {
			continue
		}
		st.wasUp, st.intended = true, false
		iface := ""
		if o.info != nil {
			iface = o.info.InterfaceName
		}
		if iface != st.iface || st.capturedAt.IsZero() || s.now().Sub(st.capturedAt) >= captureEvery {
			if !routesRead && s.routes != nil {
				routes, _ = s.routes()
				routesRead = true
			}
			s.capture(name, st, iface, o.info, routes)
		}
	}

	// 2. Destinations that must keep working: connected tunnels' routes
	// and policies, every known VPN endpoint, never-block ranges. The
	// directly connected subnets (local) stay reachable around a split
	// tunnel's block; a full-tunnel block exempts them only with
	// allow_lan, because the LAN router is usually also the resolver the
	// system falls back to once the tunnel's DNS is gone (a DNS leak).
	local := localSubnets()
	keep := slices.Clone(neverBlock)
	var allEndpoints []netip.Addr
	for name, st := range s.state {
		allEndpoints = append(allEndpoints, st.endpoints...)
		if seen[name].connected {
			keep = append(keep, st.captured...)
			if s.dests != nil {
				keep = append(keep, s.dests.ProfileDestinations(name)...)
			}
		}
	}
	for _, e := range allEndpoints {
		keep = append(keep, hostPrefix(e))
	}

	// 3. What to block.
	status := security.KillSwitchStatus{State: security.KillSwitchOff}
	var block, fullBlock []netip.Prefix
	names := s.adapters.Names()
	for _, name := range names {
		if !s.profileKillSwitchEnabled(name) {
			continue
		}
		st := s.profile(name)
		if seen[name].connected {
			status.Protected = append(status.Protected, name)
			continue
		}
		if !st.wasUp || st.intended {
			continue // never up, or turned off by the user: normal internet
		}
		status.Blocking = append(status.Blocking, name)
		if st.fullTunnel {
			status.FullTunnel = append(status.FullTunnel, name)
			excl := slices.Clone(keep)
			if s.cfg.Security.KillSwitch.LANAllowed() {
				excl = append(append(excl, lanRanges...), local...)
			}
			fullBlock = append(fullBlock, subtractAll([]netip.Prefix{allIPv4, allIPv6}, excl)...)
			continue
		}
		block = append(block, st.captured...)
		if s.dests != nil {
			block = append(block, s.dests.ProfileDestinations(name)...)
		}
	}
	for _, pol := range s.cfg.Policies {
		if pol.TunnelDownAction() != config.OnTunnelDownBlock || seen[pol.Via].connected || s.dests == nil {
			continue
		}
		if d := s.dests.PolicyDestinations(pol.Name); len(d) > 0 {
			status.PolicyBlocking = append(status.PolicyBlocking, pol.Name)
			block = append(block, d...)
		}
	}
	block = subtractAll(dedupePrefixes(block), append(slices.Clone(keep), local...))
	block = dedupePrefixes(append(block, fullBlock...))
	if len(block) > privops.MaxBlockPrefixes {
		block = block[:privops.MaxBlockPrefixes]
	}

	switch {
	case len(status.Blocking) > 0:
		status.State = security.KillSwitchActive
	case s.anyEnabled():
		status.State = security.KillSwitchArmed
	}
	status.BlockedCount = len(block)
	blocker := s.blocker
	unlocked = true
	s.mu.Unlock()

	var err error
	if blocker != nil {
		err = blocker.Set(block)
	}
	if err != nil {
		status.Error = err.Error()
		if key, ok := privops.GuidanceKey(err); ok {
			status.ErrorKey = key
		}
	}
	s.mu.Lock()
	prev := s.status.Blocking
	s.status = status
	if err == nil {
		s.blocked = block // what is in place now; a failed Set keeps the last one
	}
	onBlock := s.onBlock
	s.mu.Unlock()
	// Tell the user once per outage what is blocked and what to do (#37).
	if onBlock != nil {
		for _, name := range status.Blocking {
			if !slices.Contains(prev, name) {
				onBlock(name, slices.Contains(status.FullTunnel, name))
			}
		}
	}
	return err
}

// SetOnBlock sets the callback run once when a dropped tunnel starts being
// blocked (fullTunnel: everything but the VPN servers, plus the LAN when
// LANAllowed).
func (s *KillSwitchService) SetOnBlock(fn func(name string, fullTunnel bool)) {
	s.mu.Lock()
	s.onBlock = fn
	s.mu.Unlock()
}

// LANAllowed reports whether a full-tunnel block leaves the local network
// reachable (allow_lan, or the legacy mode: loose). When false, a dropped
// full tunnel also cuts printers, NAS and the router itself.
func (s *KillSwitchService) LANAllowed() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg != nil && s.cfg.Security.KillSwitch.LANAllowed()
}

// HoldsAllTraffic reports, for a tunnel that just dropped, whether the
// kill switch blocks (or, once the next Apply runs, will block) all
// traffic because that tunnel carried everything. Then its DNS servers stay set: giving the
// system resolver back would send lookups outside the tunnel (a leak)
// while every other destination is blocked anyway.
func (s *KillSwitchService) HoldsAllTraffic(name string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg == nil || !s.profileKillSwitchEnabled(name) {
		return false
	}
	st, ok := s.state[name]
	return ok && st.fullTunnel && st.wasUp && !st.intended
}

// Blocked returns the prefixes blocked now, sorted (empty when nothing is).
func (s *KillSwitchService) Blocked() []string {
	out := []string{}
	if s == nil {
		return out
	}
	s.mu.Lock()
	for _, p := range s.blocked {
		out = append(out, p.String())
	}
	s.mu.Unlock()
	sort.Strings(out)
	return out
}

// capture records a connected tunnel's interface, endpoints, routes and
// whether it carries all traffic.
func (s *KillSwitchService) capture(name string, st *ksProfile, iface string, info *vpn.TunnelInfo, routes []routing.Route) {
	st.iface = iface
	st.capturedAt = s.now()
	var captured []netip.Prefix
	full := false
	if iface != "" {
		for _, r := range routes {
			if !strings.EqualFold(r.Interface, iface) {
				continue
			}
			p, ok := prefixFromIPNet(r.Destination)
			if !ok {
				continue
			}
			if isDefaultish(p) {
				full = true
				continue
			}
			captured = append(captured, p)
		}
	}
	// Routes the adapter itself reports (OpenVPN server pushes, WireGuard
	// AllowedIPs) — vpn.RouteReporter is the shared contract.
	a, _ := s.adapters.Get(name)
	if rr, ok := a.(vpn.RouteReporter); ok {
		if tr := rr.TunnelRoutes(); tr != nil {
			if tr.AllTraffic {
				full = true
			}
			for _, r := range tr.Routes {
				p, err := netip.ParsePrefix(r)
				if err != nil {
					continue
				}
				if isDefaultish(p) {
					full = true
				} else {
					captured = append(captured, p.Masked())
				}
			}
		}
	}
	vpnCfg := s.cfg.VPNs[name]
	if !full && len(captured) == 0 {
		full = s.defaultFullTunnel(name, vpnCfg)
	}
	st.captured = dedupePrefixes(captured)
	st.fullTunnel = full

	var eps []netip.Addr
	for _, ip := range s.profileEndpointIPs(vpnCfg) {
		if a, err := netip.ParseAddr(ip); err == nil {
			eps = append(eps, a.Unmap())
		}
	}
	if info != nil && info.RemoteIP != nil {
		if a, ok := netip.AddrFromSlice(info.RemoteIP); ok && a.Unmap().IsGlobalUnicast() {
			eps = append(eps, a.Unmap())
		}
	}
	slices.SortFunc(eps, func(a, b netip.Addr) int { return a.Compare(b) })
	st.endpoints = slices.Compact(eps)
}

// defaultFullTunnel decides for a tunnel whose routes could not be read:
// a WireGuard config without Kongtrol policies whose AllowedIPs hold a
// default route, or a consumer VPN app (they send everything through
// the tunnel by default).
func (s *KillSwitchService) defaultFullTunnel(name string, v config.VPNConfig) bool {
	switch v.Type {
	case "protonvpn", "nordvpn", "surfshark":
		return true
	case "wireguard":
		for _, p := range s.cfg.Policies {
			if p.Via == name {
				return false // Kongtrol replaced AllowedIPs with the policies
			}
		}
		return wgConfigFullTunnel(v.ConfigFile)
	}
	return false
}

// wgConfigFullTunnel reports whether a WireGuard config routes 0.0.0.0/0
// or ::/0 through the tunnel.
func wgConfigFullTunnel(path string) bool {
	if path == "" {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(k), "AllowedIPs") {
			continue
		}
		for _, c := range strings.Split(v, ",") {
			if p, err := netip.ParsePrefix(strings.TrimSpace(c)); err == nil && isDefaultish(p) {
				return true
			}
		}
	}
	return false
}

// cleanLegacy removes, once per daemon, the all-traffic rules an older
// release may have left (they blocked everything on Windows, where a block
// rule beats any allow rule). Caller holds s.mu.
func (s *KillSwitchService) cleanLegacy() {
	if s.legacyClean || s.legacy == nil {
		return
	}
	s.legacyClean = true
	_ = s.legacy.Disable()
}

func (s *KillSwitchService) anyEnabled() bool {
	if s.cfg == nil {
		return false
	}
	if s.cfg.Security.KillSwitch.Enabled {
		return true
	}
	for _, v := range s.cfg.VPNs {
		if v.KillSwitch != nil && *v.KillSwitch {
			return true
		}
	}
	return false
}

func (s *KillSwitchService) profileKillSwitchEnabled(profile string) bool {
	v, ok := s.cfg.VPNs[profile]
	if !ok {
		return s.cfg.Security.KillSwitch.Enabled
	}
	if v.KillSwitch != nil {
		return *v.KillSwitch
	}
	return s.cfg.Security.KillSwitch.Enabled
}

func (s *KillSwitchService) profileEndpointIPs(vpnCfg config.VPNConfig) []string {
	seen := map[string]bool{}
	var out []string
	add := func(ip string) {
		if ip != "" && !seen[ip] {
			seen[ip] = true
			out = append(out, ip)
		}
	}
	if vpnCfg.Host != "" {
		if ip := net.ParseIP(vpnCfg.Host); ip != nil {
			add(ip.String())
		} else if s.lookupIP != nil {
			for _, ip := range s.lookupIP(vpnCfg.Host) {
				add(ip.String())
			}
		}
	}
	if vpnCfg.Type == "wireguard" && vpnCfg.ConfigFile != "" {
		if epIP, err := wireguard.ParseEndpoint(vpnCfg.ConfigFile); err == nil && epIP != nil {
			add(epIP.String())
		}
	}
	return out
}
