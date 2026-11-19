package monitor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

// PolicyResolver steers each policy's traffic into its tunnel, for every
// adapter type:
//
//   - It resolves the policies' domains periodically (whether the tunnel is
//     up or not: the kill switch, on_tunnel_down: block and the "traffic is
//     leaving through the normal internet" notices need the IPs too).
//   - For a connected tunnel it installs OS routes for the policy's CIDRs
//     and resolved IPs through the tunnel interface (helper / privops),
//     adding and removing them as IPs change. WireGuard additionally gets
//     its AllowedIPs updated (crypto-routing needs it).
//   - When the tunnel goes down (Disconnect, a drop, or the user closing a
//     provider app) its routes are removed, so the traffic falls back to the
//     normal internet — unless the policy says otherwise (on_tunnel_down,
//     enforced by app.KillSwitchService).
//
// Overlapping policies follow the policy engine's order (VPN priority,
// then position): a CIDR gets one route, through the winning policy's
// tunnel. An equal route that already exists on the tunnel interface
// (pushed by the VPN server) is left alone.
type PolicyResolver struct {
	mu       sync.Mutex
	opMu     sync.Mutex // serializes route / AllowedIPs changes
	cfg      *config.Config
	routeMgr routing.RouteManager
	log      *zap.Logger
	profiles map[string]*profileState
	resolved map[string]map[string]seenIP // policy → CIDR → last seen
	probe    TunnelProbe
	cancel   context.CancelFunc

	// test hooks
	now    func() time.Time
	lookup func(domain string, dns []net.IP) []net.IP
	wgSet  func(iface, peer string, cidrs []string, configPath string) error
}

// TunnelState is what the resolver needs to know about one tunnel.
type TunnelState struct {
	Connected  bool
	Type       string // adapter type; "wireguard" also gets AllowedIPs
	Interface  string
	DNS        []net.IP
	ConfigFile string
}

// TunnelProbe reports a profile's tunnel; the resolver polls it to follow
// tunnels that come up or drop without going through Connect/Disconnect.
type TunnelProbe func(profile string) TunnelState

// seenIP is one resolved address of a policy domain.
type seenIP struct {
	last time.Time
	host string // the name it was resolved from (for notices)
}

// profileState holds the live routing state of one connected tunnel.
type profileState struct {
	wg         bool
	ifaceName  string
	peerPubKey string
	configPath string   // profile .conf (lets the helper adopt a pre-existing tunnel)
	endpointIP net.IP   // never routed into the tunnel (routing loop)
	dnsServers []net.IP // VPN DNS servers for resolution
	vpnSubnet  string   // WireGuard: e.g. "10.2.0.0/24"

	installed map[string]routing.Route // CIDR → route Kongtrol added
	external  map[string]bool          // CIDR already routed on the interface by someone else
	gateway   net.IP                   // next hop seen on the interface's own routes
	wgApplied string                   // last AllowedIPs set (joined)
}

const (
	// staleAfter drops a resolved IP not seen for this long (CDN IPs
	// rotate; long enough not to cut a connection to an old edge).
	staleAfter = 6 * time.Hour
	// routeMetric for policy routes; the longest prefix decides anyway.
	routeMetric = 10
)

// commonSubdomains are probed for wildcard domain patterns to capture
// CDN edges that resolve differently per subdomain.
var commonSubdomains = []string{"www.", "api.", "cdn.", "app.", "docs.", "console.", "static.", "assets."}

// NewPolicyResolver creates a resolver.
func NewPolicyResolver(cfg *config.Config, routeMgr routing.RouteManager, log *zap.Logger) *PolicyResolver {
	if log == nil {
		log = zap.NewNop()
	}
	r := &PolicyResolver{
		cfg:      cfg,
		routeMgr: routeMgr,
		log:      log,
		profiles: make(map[string]*profileState),
		resolved: make(map[string]map[string]seenIP),
		now:      time.Now,
		wgSet:    wireguard.WgSetAllowedIPs,
	}
	r.lookup = r.lookupAll
	return r
}

// SetTunnelProbe sets the tunnel state source used by the background loop.
func (r *PolicyResolver) SetTunnelProbe(p TunnelProbe) {
	r.mu.Lock()
	r.probe = p
	r.mu.Unlock()
}

// SetConfig swaps the configuration (policies edited in the dashboard)
// and re-applies every connected tunnel's routes.
func (r *PolicyResolver) SetConfig(cfg *config.Config) {
	r.mu.Lock()
	r.cfg = cfg
	keep := map[string]bool{}
	for _, p := range cfg.Policies {
		keep[p.Name] = true
	}
	for name := range r.resolved {
		if !keep[name] {
			delete(r.resolved, name)
		}
	}
	r.mu.Unlock()
	go func() {
		r.resolve("", 12*time.Second)
		r.applyAll()
	}()
}

func (r *PolicyResolver) policies() []config.PolicyRule {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cfg == nil {
		return nil
	}
	return append([]config.PolicyRule(nil), r.cfg.Policies...)
}

// hasPolicies reports whether any policy routes through profile.
func (r *PolicyResolver) hasPolicies(profile string) bool {
	for _, p := range r.policies() {
		if p.Via == profile {
			return true
		}
	}
	return false
}

// RegisterProfile starts routing for a connected WireGuard profile: it
// parses the config and performs a first resolution with a bounded budget
// so `kongtrol up` is not blocked by slow DNS.
func (r *PolicyResolver) RegisterProfile(name, ifaceName, configPath string) error {
	if !r.hasPolicies(name) {
		return nil
	}
	pubKey, err := wireguard.ParsePeerPublicKey(configPath)
	if err != nil {
		return fmt.Errorf("policyresolver: %w", err)
	}
	endpointIP, err := wireguard.ParseEndpoint(configPath)
	if err != nil {
		return fmt.Errorf("policyresolver: %w", err)
	}
	ps := newProfileState(ifaceName, wireguard.ParseConfigDNS(configPath))
	ps.wg, ps.peerPubKey, ps.configPath, ps.endpointIP = true, pubKey, configPath, endpointIP
	// Derive VPN subnet from assigned address: 10.2.0.2/32 → 10.2.0.0/24.
	if addr, err := wireguard.ParseConfigAddress(configPath); err == nil && addr != nil {
		if v4 := addr.To4(); v4 != nil {
			ps.vpnSubnet = fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2])
		}
	}
	r.register(name, ps)
	return nil
}

// RegisterTunnel starts routing for a connected tunnel of any other type
// (OpenVPN, provider apps, Tailscale, WARP, …): policy routes go through
// iface. dns are the tunnel's resolvers (used to resolve policy domains).
func (r *PolicyResolver) RegisterTunnel(name, iface string, dns []net.IP) error {
	if !r.hasPolicies(name) {
		return nil
	}
	if strings.TrimSpace(iface) == "" {
		return fmt.Errorf("policyresolver: %s: tunnel interface unknown", name)
	}
	r.register(name, newProfileState(iface, dns))
	return nil
}

func newProfileState(iface string, dns []net.IP) *profileState {
	return &profileState{
		ifaceName:  iface,
		dnsServers: dns,
		installed:  map[string]routing.Route{},
		external:   map[string]bool{},
	}
}

func (r *PolicyResolver) register(name string, ps *profileState) {
	r.mu.Lock()
	old := r.profiles[name]
	r.profiles[name] = ps
	r.mu.Unlock()
	if old != nil && old.ifaceName != ps.ifaceName {
		r.flush(old)
	}
	r.resolve(name, 12*time.Second)
	r.apply(name)
}

// UnregisterProfile stops routing a profile and removes its routes, so its
// policies' traffic uses the normal internet again.
func (r *PolicyResolver) UnregisterProfile(name string) {
	r.mu.Lock()
	ps, ok := r.profiles[name]
	delete(r.profiles, name)
	r.mu.Unlock()
	if ok {
		r.flush(ps)
	}
}

func (r *PolicyResolver) flush(ps *profileState) {
	r.opMu.Lock()
	defer r.opMu.Unlock()
	if r.routeMgr == nil {
		return
	}
	for _, rt := range ps.installed {
		_ = r.routeMgr.Delete(rt)
	}
	_ = r.routeMgr.Flush(ps.ifaceName)
	ps.installed = map[string]routing.Route{}
}

// Start launches the background loop: tunnel follow-up every 10s, DNS
// re-resolution every 30s.
func (r *PolicyResolver) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	go r.loop(ctx)
}

// Stop halts the background loop.
func (r *PolicyResolver) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *PolicyResolver) loop(ctx context.Context) {
	follow := time.NewTicker(10 * time.Second)
	defer follow.Stop()
	resolveT := time.NewTicker(30 * time.Second)
	defer resolveT.Stop()
	r.resolve("", 25*time.Second) // tunnels down need IPs too (block, notices)
	for {
		select {
		case <-ctx.Done():
			return
		case <-follow.C:
			r.Reconcile()
		case <-resolveT.C:
			r.resolve("", 25*time.Second)
			r.applyAll()
		}
	}
}

// Reconcile follows tunnels through the probe: a tunnel that came up
// (attach mode, watchdog reconnect) gets its routes, one that dropped
// loses them.
func (r *PolicyResolver) Reconcile() {
	r.mu.Lock()
	probe := r.probe
	r.mu.Unlock()
	if probe == nil {
		return
	}
	want := map[string]bool{}
	for _, p := range r.policies() {
		want[p.Via] = true
	}
	r.mu.Lock()
	for name := range r.profiles {
		want[name] = true
	}
	r.mu.Unlock()
	for name := range want {
		st := probe(name)
		r.mu.Lock()
		ps, registered := r.profiles[name]
		r.mu.Unlock()
		switch {
		case !st.Connected && registered:
			r.UnregisterProfile(name)
		case st.Connected && (!registered || ps.ifaceName != st.Interface) && st.Interface != "":
			var err error
			if st.Type == "wireguard" && st.ConfigFile != "" {
				err = r.RegisterProfile(name, st.Interface, st.ConfigFile)
			} else {
				err = r.RegisterTunnel(name, st.Interface, st.DNS)
			}
			if err != nil {
				r.log.Warn("policyresolver: register", zap.String("profile", name), zap.Error(err))
			}
		case registered && !r.hasPolicies(name):
			r.UnregisterProfile(name)
		}
	}
}

// ProfileSnapshot is a read-only view of a profile's resolved routing state.
type ProfileSnapshot struct {
	Name          string   `json:"name"`
	Interface     string   `json:"interface"`
	Domains       []string `json:"domains"`
	IPRanges      []string `json:"ip_ranges"`
	ResolvedCIDRs []string `json:"resolved_cidrs"`
}

// Snapshot returns the current state of every connected (registered)
// profile.
func (r *PolicyResolver) Snapshot() []ProfileSnapshot {
	pols := r.policies()
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ProfileSnapshot, 0, len(r.profiles))
	for name, ps := range r.profiles {
		snap := ProfileSnapshot{Name: name, Interface: ps.ifaceName}
		seen := map[string]bool{}
		for _, p := range pols {
			if p.Via != name {
				continue
			}
			snap.Domains = append(snap.Domains, p.Match.Domains...)
			snap.IPRanges = append(snap.IPRanges, p.Match.IPRanges...)
			for c := range r.resolved[p.Name] {
				if !seen[c] {
					seen[c] = true
					snap.ResolvedCIDRs = append(snap.ResolvedCIDRs, c)
				}
			}
		}
		sort.Strings(snap.ResolvedCIDRs)
		out = append(out, snap)
	}
	return out
}

// ResolvedIP is one address a policy domain resolved to, with when it was
// last seen and when it expires if not seen again (staleAfter).
type ResolvedIP struct {
	CIDR      string    `json:"cidr"`
	Host      string    `json:"host,omitempty"`
	LastSeen  time.Time `json:"last_seen"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ResolvedIPs returns the addresses one policy's domains resolved to, sorted
// by CIDR (read-only; the dashboard shows them with their expiry).
func (r *PolicyResolver) ResolvedIPs(policy string) []ResolvedIP {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := r.resolved[policy]
	out := make([]ResolvedIP, 0, len(m))
	for c, s := range m {
		out = append(out, ResolvedIP{CIDR: c, Host: s.host, LastSeen: s.last, ExpiresAt: s.last.Add(staleAfter)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CIDR < out[j].CIDR })
	return out
}

// PolicyDestinations returns one policy's static CIDRs plus its resolved
// domain IPs (tunnel up or not).
func (r *PolicyResolver) PolicyDestinations(policy string) []netip.Prefix {
	for _, p := range r.policies() {
		if p.Name == policy {
			return r.policyPrefixes(p)
		}
	}
	return nil
}

// ProfileDestinations returns the destinations of every policy via profile.
func (r *PolicyResolver) ProfileDestinations(profile string) []netip.Prefix {
	var out []netip.Prefix
	for _, p := range r.policies() {
		if p.Via == profile {
			out = append(out, r.policyPrefixes(p)...)
		}
	}
	return out
}

func (r *PolicyResolver) policyPrefixes(p config.PolicyRule) []netip.Prefix {
	var out []netip.Prefix
	for _, c := range p.Match.IPRanges {
		if pf, err := netip.ParsePrefix(strings.TrimSpace(c)); err == nil {
			out = append(out, pf.Masked())
		}
	}
	r.mu.Lock()
	for c := range r.resolved[p.Name] {
		if pf, err := netip.ParsePrefix(c); err == nil {
			out = append(out, pf)
		}
	}
	r.mu.Unlock()
	return out
}

// PolicyMatch is the policy a destination address belongs to.
type PolicyMatch struct {
	Policy string
	Via    string
	Action string // on_tunnel_down
	Label  string // domain it was resolved from, else a policy domain, else the IP
}

// MatchAddr returns the highest-priority policy whose destinations contain
// a (static CIDRs or resolved domain IPs).
func (r *PolicyResolver) MatchAddr(a netip.Addr) (PolicyMatch, bool) {
	a = a.Unmap()
	cfg := r.config()
	for _, p := range orderedPolicies(cfg) {
		m := PolicyMatch{Policy: p.Name, Via: p.Via, Action: p.TunnelDownAction()}
		for _, c := range p.Match.IPRanges {
			if pf, err := netip.ParsePrefix(strings.TrimSpace(c)); err == nil && pf.Contains(a) {
				m.Label = a.String()
				return m, true
			}
		}
		r.mu.Lock()
		s, ok := r.resolved[p.Name][hostCIDR(a)]
		r.mu.Unlock()
		if ok {
			m.Label = s.host
			if m.Label == "" {
				m.Label = a.String()
			}
			return m, true
		}
	}
	return PolicyMatch{}, false
}

// HostOf returns the policy domain a resolved to, "" when no policy
// domain resolved to it (recent destinations, ADR-0054).
func (r *PolicyResolver) HostOf(a netip.Addr) string {
	key := hostCIDR(a.Unmap())
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ips := range r.resolved {
		if s, ok := ips[key]; ok && s.host != "" {
			return s.host
		}
	}
	return ""
}

func (r *PolicyResolver) config() *config.Config {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cfg
}

// ResolveDomainViaProfile resolves a domain using the DNS servers attached to
// a specific profile. It returns deduplicated IPs.
func (r *PolicyResolver) ResolveDomainViaProfile(profile, domain string) ([]net.IP, error) {
	r.mu.Lock()
	ps, ok := r.profiles[profile]
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("policyresolver: profile %q is not registered", profile)
	}
	if strings.TrimSpace(domain) == "" {
		return nil, errors.New("policyresolver: empty domain")
	}
	ips := r.lookup(domain, ps.dnsServers)
	if len(ips) == 0 {
		return nil, fmt.Errorf("policyresolver: no answers for %q via %s DNS", domain, profile)
	}
	return ips, nil
}

// resolve looks up the domains of every policy (only those via onlyVia
// when set) and records the answers. No lock is held during DNS.
func (r *PolicyResolver) resolve(onlyVia string, budget time.Duration) {
	pols := r.policies()
	r.mu.Lock()
	dnsFor := map[string][]net.IP{}
	endpointFor := map[string]net.IP{}
	for name, ps := range r.profiles {
		dnsFor[name] = ps.dnsServers
		endpointFor[name] = ps.endpointIP
	}
	r.mu.Unlock()

	started := time.Now()
	found := map[string]map[string]string{} // policy → CIDR → host
	for _, p := range pols {
		if onlyVia != "" && p.Via != onlyVia {
			continue
		}
		for _, pattern := range p.Match.Domains {
			for _, domain := range expandDomainPattern(pattern) {
				if budget > 0 && time.Since(started) >= budget {
					break
				}
				for _, ip := range r.lookup(domain, dnsFor[p.Via]) {
					if ep := endpointFor[p.Via]; ep != nil && ip.Equal(ep) {
						continue // never route the VPN endpoint into its tunnel
					}
					cidr := ipToCIDR(ip)
					if cidr == "" {
						continue
					}
					if found[p.Name] == nil {
						found[p.Name] = map[string]string{}
					}
					found[p.Name][cidr] = domain
				}
			}
		}
	}

	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()
	for pol, cidrs := range found {
		m := r.resolved[pol]
		if m == nil {
			m = map[string]seenIP{}
			r.resolved[pol] = m
		}
		newCount := 0
		for c, host := range cidrs {
			if _, ok := m[c]; !ok {
				newCount++
			}
			m[c] = seenIP{last: now, host: host}
		}
		if newCount > 0 {
			r.log.Info("policyresolver: discovered new IPs", zap.String("policy", pol), zap.Int("new", newCount), zap.Int("total", len(m)))
		}
	}
	for _, m := range r.resolved {
		for c, s := range m {
			if now.Sub(s.last) > staleAfter {
				delete(m, c)
			}
		}
	}
}

func (r *PolicyResolver) applyAll() {
	r.mu.Lock()
	names := make([]string, 0, len(r.profiles))
	for n := range r.profiles {
		names = append(names, n)
	}
	r.mu.Unlock()
	for _, n := range names {
		r.apply(n)
	}
}

// desiredCIDRs returns the CIDRs whose winning policy routes via profile.
func (r *PolicyResolver) desiredCIDRs(profile string, ps *profileState) []string {
	owner := map[string]string{}
	var order []string
	for _, p := range orderedPolicies(r.config()) {
		for _, pf := range r.policyPrefixes(p) {
			c := pf.String()
			if _, taken := owner[c]; !taken {
				owner[c] = p.Via
				order = append(order, c)
			}
		}
	}
	var out []string
	for _, c := range order {
		if owner[c] != profile {
			continue
		}
		if ps.endpointIP != nil && c == ipToCIDR(ps.endpointIP) {
			continue
		}
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// apply brings one connected profile's AllowedIPs and OS routes in line
// with its policies.
func (r *PolicyResolver) apply(name string) {
	r.opMu.Lock()
	defer r.opMu.Unlock()
	r.mu.Lock()
	ps, ok := r.profiles[name]
	r.mu.Unlock()
	if !ok {
		return
	}
	desired := r.desiredCIDRs(name, ps)

	if ps.wg {
		all := buildAllowedIPs(ps, desired)
		if joined := strings.Join(all, ","); joined != ps.wgApplied {
			if err := r.wgSet(ps.ifaceName, ps.peerPubKey, all, ps.configPath); err != nil {
				r.log.Error("policyresolver: wg set failed", zap.String("profile", name), zap.Error(err))
			} else {
				ps.wgApplied = joined
			}
		}
	}
	if r.routeMgr == nil {
		return
	}

	want := map[string]bool{}
	for _, c := range desired {
		want[c] = true
	}
	for c, rt := range ps.installed {
		if !want[c] {
			if err := r.routeMgr.Delete(rt); err == nil {
				delete(ps.installed, c)
			}
		}
	}
	var pending []string
	for _, c := range desired {
		if _, done := ps.installed[c]; !done && !ps.external[c] {
			pending = append(pending, c)
		}
	}
	if len(pending) == 0 {
		return
	}
	// Leave equal routes the VPN already installed (server-pushed) alone,
	// and reuse the interface's next hop when its routes have one.
	if existing, err := r.routeMgr.List(); err == nil {
		for _, rt := range existing {
			if !strings.EqualFold(rt.Interface, ps.ifaceName) {
				continue
			}
			if rt.Gateway != nil && !rt.Gateway.IsUnspecified() && ps.gateway == nil {
				ps.gateway = rt.Gateway
			}
			c := rt.Destination.String()
			if _, ours := ps.installed[c]; !ours && want[c] {
				if pf, err := netip.ParsePrefix(c); err == nil && pf.Masked().String() == c {
					ps.external[c] = true
				}
			}
		}
	}
	for _, c := range pending {
		if ps.external[c] {
			continue
		}
		_, ipnet, err := net.ParseCIDR(c)
		if err != nil {
			continue
		}
		if ones, _ := ipnet.Mask.Size(); ones <= 1 {
			// A default route (or half) is the VPN client's job: an extra
			// one in the main table can loop the tunnel's own traffic.
			ps.external[c] = true
			continue
		}
		rt := routing.Route{Destination: *ipnet, Interface: ps.ifaceName, Metric: routeMetric}
		if ps.gateway != nil && (ps.gateway.To4() != nil) == (ipnet.IP.To4() != nil) {
			rt.Gateway = ps.gateway
		}
		if err := r.routeMgr.Add(rt); err != nil {
			r.log.Warn("policyresolver: add route", zap.String("profile", name), zap.String("cidr", c), zap.Error(err))
			continue
		}
		ps.installed[c] = rt
	}
}

// buildAllowedIPs is the WireGuard AllowedIPs list: the VPN subnet and DNS
// (so the tunnel itself works) plus the policy CIDRs.
func buildAllowedIPs(ps *profileState, desired []string) []string {
	seen := make(map[string]bool)
	var result []string
	add := func(cidr string) {
		if cidr != "" && !seen[cidr] {
			seen[cidr] = true
			result = append(result, cidr)
		}
	}
	add(ps.vpnSubnet)
	for _, dns := range ps.dnsServers {
		add(ipToCIDR(dns))
	}
	for _, c := range desired {
		add(c)
	}
	return result
}

// orderedPolicies returns the policies in the policy engine's order:
// higher priority first (the VPN profile's priority, else position).
func orderedPolicies(cfg *config.Config) []config.PolicyRule {
	if cfg == nil {
		return nil
	}
	type ranked struct {
		p    config.PolicyRule
		prio int
		idx  int
	}
	rs := make([]ranked, len(cfg.Policies))
	for i, p := range cfg.Policies {
		prio := len(cfg.Policies) - i
		if v, ok := cfg.VPNs[p.Via]; ok && v.Priority > 0 {
			prio = v.Priority
		}
		rs[i] = ranked{p, prio, i}
	}
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].prio > rs[j].prio })
	out := make([]config.PolicyRule, len(rs))
	for i, x := range rs {
		out[i] = x.p
	}
	return out
}

// lookupAll resolves a domain using both the system resolver and each VPN DNS
// server. This captures different CDN edges and maximises IP coverage.
func (r *PolicyResolver) lookupAll(domain string, vpnDNS []net.IP) []net.IP {
	seen := make(map[string]bool)
	var result []net.IP
	add := func(ips []net.IPAddr) {
		for _, a := range ips {
			key := a.IP.String()
			if !seen[key] {
				seen[key] = true
				result = append(result, a.IP)
			}
		}
	}

	// System resolver with timeout (can otherwise block for a long time).
	sysCtx, sysCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if ips, err := net.DefaultResolver.LookupIPAddr(sysCtx, domain); err == nil {
		add(ips)
	}
	sysCancel()

	// VPN DNS servers via custom resolver.
	for _, dns := range vpnDNS {
		if dns.To4() == nil && dns.To16() == nil {
			continue
		}
		server := net.JoinHostPort(dns.String(), "53")
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, "udp", server)
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if ips, err := resolver.LookupIPAddr(ctx, domain); err == nil {
			add(ips)
		}
		cancel()
	}

	return result
}

// expandDomainPattern expands a domain pattern into concrete lookup targets.
// "*.anthropic.com" → ["anthropic.com", "www.anthropic.com", "api.anthropic.com", ...]
// "claude.ai" → ["claude.ai"]
func expandDomainPattern(pattern string) []string {
	pattern = strings.TrimSpace(pattern)
	if strings.HasPrefix(pattern, "*.") {
		base := strings.TrimPrefix(pattern, "*.")
		targets := []string{base}
		for _, prefix := range commonSubdomains {
			targets = append(targets, prefix+base)
		}
		return targets
	}
	return []string{pattern}
}

// ipToCIDR converts an IP to a host-route CIDR string.
func ipToCIDR(ip net.IP) string {
	if ip == nil {
		return ""
	}
	if ip.To4() != nil {
		return ip.To4().String() + "/32"
	}
	return ip.String() + "/128"
}

func hostCIDR(a netip.Addr) string {
	return netip.PrefixFrom(a, a.BitLen()).String()
}
