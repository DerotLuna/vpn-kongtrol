package app

import (
	"bufio"
	"context"
	"fmt"
	"iter"
	"net"
	"net/netip"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
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
//     (and the LAN when allowed). This source excludes intentional stops.
//   - Winning policies with effective block or connect while their tunnel
//     is disconnected or being disconnected.
//
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

	state         map[string]*ksProfile
	disconnecting map[string]int
	status        security.KillSwitchStatus
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
	PolicyDestinations() iter.Seq[monitor.PolicyDestination]
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
		cfg:           cfg,
		adapters:      adapters,
		legacy:        ks,
		now:           time.Now,
		state:         map[string]*ksProfile{},
		disconnecting: map[string]int{},
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
	for name, st := range s.state {
		if s.cfg == nil || cfg == nil {
			break
		}
		old, next := s.cfg.VPNs[name], cfg.VPNs[name]
		if old.Host != next.Host || old.ConfigFile != next.ConfigFile || old.Type != next.Type {
			st.endpoints = nil
			st.capturedAt = time.Time{}
		}
	}
	s.cfg = cfg
	s.mu.Unlock()
}

// BeginDisconnect installs policy blocks before the adapter removes routes.
func (s *KillSwitchService) BeginDisconnect(name string) (func(), error) {
	s.mu.Lock()
	if s.cfg == nil {
		s.mu.Unlock()
		return nil, nil
	}
	protected := slices.ContainsFunc(s.cfg.Policies, func(p config.PolicyRule) bool {
		return !p.Disabled && p.Via == name && config.BlocksWhenDown(p.TunnelDownAction(s.cfg.VPNs[name]))
	})
	if !protected {
		s.mu.Unlock()
		return nil, nil
	}
	s.disconnecting[name]++
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		s.disconnecting[name]--
		if s.disconnecting[name] == 0 {
			delete(s.disconnecting, name)
		}
		s.mu.Unlock()
		_ = s.Apply()
	}, s.Apply()
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
	if s == nil {
		return nil
	}
	s.applyMu.Lock()
	defer s.applyMu.Unlock()
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
	s.mu.Lock()
	unlocked := false
	defer func() {
		if !unlocked {
			s.mu.Unlock()
		}
	}()
	if s.cfg == nil {
		return nil
	}
	s.cleanLegacy()

	// 1. Remember what each connected tunnel carries.
	var routes []routing.Route
	routesRead := false
	for name, o := range seen {
		st := s.profile(name)
		if !o.connected {
			continue
		}
		if s.disconnecting[name] == 0 {
			st.wasUp, st.intended = true, false
		}
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
	block = subtractAll(dedupePrefixes(block), append(slices.Clone(keep), local...))
	block = append(block, fullBlock...)
	if s.dests != nil {
		var exemptions, claimed []netip.Prefix
		seenPrefix := map[netip.Prefix]bool{}
		blocking := map[string]bool{}
		for dst := range s.dests.PolicyDestinations() {
			if seenPrefix[dst.Prefix] {
				continue
			}
			seenPrefix[dst.Prefix] = true
			m := dst.Match
			earlier := claimed
			claimed = append(claimed, dst.Prefix)
			if !config.BlocksWhenDown(m.Action) || (seen[m.Via].connected && s.disconnecting[m.Via] == 0) {
				continue
			}
			if exemptions == nil {
				exemptions = s.policyExceptions(allEndpoints)
			}
			winning := []netip.Prefix{dst.Prefix}
			if dst.Prefix.Bits() < dst.Prefix.Addr().BitLen() {
				winning = subtractPrefix(dst.Prefix, earlier)
			}
			winning = subtractAll(winning, exemptions)
			if len(winning) > 0 {
				blocking[m.Policy] = true
				block = append(block, winning...)
			}
		}
		for name := range blocking {
			status.PolicyBlocking = append(status.PolicyBlocking, name)
		}
		sort.Strings(status.PolicyBlocking)
	}
	block = dedupePrefixes(block)

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
	if len(block) > privops.MaxBlockPrefixes {
		err = fmt.Errorf("blocked destinations exceed firewall limit of %d", privops.MaxBlockPrefixes)
	} else if blocker != nil {
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

func (s *KillSwitchService) policyExceptions(known []netip.Addr) []netip.Prefix {
	exceptions := slices.Clone(neverBlock)
	for _, addr := range known {
		exceptions = append(exceptions, hostPrefix(addr))
	}
	for name, v := range s.cfg.VPNs {
		st := s.profile(name)
		if st.endpoints != nil {
			continue
		}
		st.endpoints = []netip.Addr{}
		for _, ip := range s.profileEndpointIPs(v) {
			if addr, err := netip.ParseAddr(ip); err == nil {
				st.endpoints = append(st.endpoints, addr.Unmap())
				exceptions = append(exceptions, hostPrefix(addr))
			}
		}
	}
	return exceptions
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
			if !p.Disabled && p.Via == name {
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
