package monitor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/wireguard"
)

// PolicyResolver periodically re-resolves DNS for domain-based policy rules
// and dynamically updates WireGuard's AllowedIPs via `wg set`. IPs accumulate
// over the session lifetime (never removed) to handle CDN IP rotation.
type PolicyResolver struct {
	mu       sync.Mutex
	cfg      *config.Config
	routeMgr routing.RouteManager
	log      *zap.Logger
	profiles map[string]*profileState
	cancel   context.CancelFunc
}

// profileState holds the live routing state for a single WireGuard profile.
type profileState struct {
	ifaceName  string
	peerPubKey string
	endpointIP net.IP   // must never appear in AllowedIPs (routing loop)
	dnsServers []net.IP // VPN DNS servers for resolution
	vpnSubnet  string   // e.g. "10.2.0.0/24"

	domains  []string // domain patterns from policies (e.g. "*.anthropic.com")
	ipRanges []string // static CIDR ranges from policies

	seenCIDRs map[string]bool // all CIDRs ever resolved, accumulate-only
}

// commonSubdomains are probed for wildcard domain patterns to capture
// CDN edges that resolve differently per subdomain.
var commonSubdomains = []string{"www.", "api.", "cdn.", "app.", "docs.", "console.", "static.", "assets."}

// NewPolicyResolver creates a resolver. Call RegisterProfile for each profile
// that needs dynamic DNS-based split tunneling.
func NewPolicyResolver(cfg *config.Config, routeMgr routing.RouteManager, log *zap.Logger) *PolicyResolver {
	return &PolicyResolver{
		cfg:      cfg,
		routeMgr: routeMgr,
		log:      log,
		profiles: make(map[string]*profileState),
	}
}

// RegisterProfile parses the WireGuard config and collects policy domains for
// the named profile. It performs an immediate first resolution with a bounded
// budget so `kongtrol up` is not blocked by slow DNS.
func (r *PolicyResolver) RegisterProfile(name, ifaceName, configPath string) error {
	pubKey, err := wireguard.ParsePeerPublicKey(configPath)
	if err != nil {
		return fmt.Errorf("policyresolver: %w", err)
	}
	endpointIP, err := wireguard.ParseEndpoint(configPath)
	if err != nil {
		return fmt.Errorf("policyresolver: %w", err)
	}
	dnsServers := wireguard.ParseConfigDNS(configPath)

	// Derive VPN subnet from assigned address: 10.2.0.2/32 → 10.2.0.0/24.
	vpnSubnet := ""
	if addr, err := wireguard.ParseConfigAddress(configPath); err == nil && addr != nil {
		if v4 := addr.To4(); v4 != nil {
			vpnSubnet = fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2])
		}
	}

	// Collect domains and static IP ranges from policies routed via this profile.
	var domains []string
	var ipRanges []string
	for _, pol := range r.cfg.Policies {
		if pol.Via != name {
			continue
		}
		domains = append(domains, pol.Match.Domains...)
		ipRanges = append(ipRanges, pol.Match.IPRanges...)
	}

	if len(domains) == 0 && len(ipRanges) == 0 {
		return nil // no domain policies for this profile
	}

	ps := &profileState{
		ifaceName:  ifaceName,
		peerPubKey: pubKey,
		endpointIP: endpointIP,
		dnsServers: dnsServers,
		vpnSubnet:  vpnSubnet,
		domains:    domains,
		ipRanges:   ipRanges,
		seenCIDRs:  make(map[string]bool),
	}

	r.mu.Lock()
	r.profiles[name] = ps
	r.mu.Unlock()

	// Immediate first resolution with a bounded time budget.
	r.resolve(name, ps, 12*time.Second)
	return nil
}

// UnregisterProfile stops tracking a profile and flushes its routes.
func (r *PolicyResolver) UnregisterProfile(name string) {
	r.mu.Lock()
	ps, ok := r.profiles[name]
	delete(r.profiles, name)
	r.mu.Unlock()

	if ok && r.routeMgr != nil {
		_ = r.routeMgr.Flush(ps.ifaceName)
	}
}

// Start launches the background resolution loop (30s interval).
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

// ProfileSnapshot is a read-only view of a profile's resolved routing state.
type ProfileSnapshot struct {
	Name          string   `json:"name"`
	Interface     string   `json:"interface"`
	Domains       []string `json:"domains"`
	IPRanges      []string `json:"ip_ranges"`
	ResolvedCIDRs []string `json:"resolved_cidrs"`
}

// Snapshot returns the current resolved state for all registered profiles.
func (r *PolicyResolver) Snapshot() []ProfileSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]ProfileSnapshot, 0, len(r.profiles))
	for name, ps := range r.profiles {
		cidrs := make([]string, 0, len(ps.seenCIDRs))
		for c := range ps.seenCIDRs {
			cidrs = append(cidrs, c)
		}
		out = append(out, ProfileSnapshot{
			Name:          name,
			Interface:     ps.ifaceName,
			Domains:       ps.domains,
			IPRanges:      ps.ipRanges,
			ResolvedCIDRs: cidrs,
		})
	}
	return out
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
	ips := r.lookupAll(domain, ps.dnsServers)
	if len(ips) == 0 {
		return nil, fmt.Errorf("policyresolver: no answers for %q via %s DNS", domain, profile)
	}
	return ips, nil
}

func (r *PolicyResolver) loop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.mu.Lock()
			// Snapshot profile names + states to avoid holding lock during DNS.
			snap := make(map[string]*profileState, len(r.profiles))
			for k, v := range r.profiles {
				snap[k] = v
			}
			r.mu.Unlock()

			for name, ps := range snap {
				r.resolve(name, ps, 25*time.Second)
			}
		}
	}
}

// resolve performs DNS lookups for all domains in the profile and updates
// WireGuard AllowedIPs if new IPs are discovered.
func (r *PolicyResolver) resolve(name string, ps *profileState, budget time.Duration) {
	started := time.Now()
	newCount := 0

	for _, pattern := range ps.domains {
		if budget > 0 && time.Since(started) >= budget {
			break
		}
		lookups := expandDomainPattern(pattern)
		for _, domain := range lookups {
			if budget > 0 && time.Since(started) >= budget {
				break
			}
			ips := r.lookupAll(domain, ps.dnsServers)
			for _, ip := range ips {
				cidr := ipToCIDR(ip)
				if cidr == "" {
					continue
				}
				// Exclude the WireGuard endpoint to prevent routing loops.
				if ps.endpointIP != nil && ip.Equal(ps.endpointIP) {
					continue
				}
				if !ps.seenCIDRs[cidr] {
					ps.seenCIDRs[cidr] = true
					newCount++
				}
			}
		}
	}

	// Build full AllowedIPs list: essential + accumulated.
	all := r.buildAllowedIPs(ps)

	if newCount > 0 {
		r.log.Info("policyresolver: discovered new IPs",
			zap.String("profile", name),
			zap.Int("new", newCount),
			zap.Int("total", len(ps.seenCIDRs)),
		)

		// Update WireGuard crypto-routing via `wg set`.
		if err := wireguard.WgSetAllowedIPs(ps.ifaceName, ps.peerPubKey, all); err != nil {
			r.log.Error("policyresolver: wg set failed",
				zap.String("profile", name),
				zap.Error(err),
			)
			return
		}
	}

	// On Linux/macOS, enforce expected OS routes every cycle.
	// This self-heals route drift when the OS or VPN client overwrites entries.
	if runtime.GOOS != "windows" && r.routeMgr != nil {
		r.addOSRoutes(ps, all)
	}
}

// buildAllowedIPs combines essential CIDRs with all accumulated domain IPs.
func (r *PolicyResolver) buildAllowedIPs(ps *profileState) []string {
	seen := make(map[string]bool)
	var result []string
	add := func(cidr string) {
		if cidr != "" && !seen[cidr] {
			seen[cidr] = true
			result = append(result, cidr)
		}
	}

	// Essential: VPN subnet so internal traffic works.
	add(ps.vpnSubnet)

	// Essential: DNS servers so VPN DNS queries route through tunnel.
	for _, dns := range ps.dnsServers {
		add(ipToCIDR(dns))
	}

	// Static IP ranges from policies.
	for _, cidr := range ps.ipRanges {
		add(cidr)
	}

	// All accumulated domain IPs.
	for cidr := range ps.seenCIDRs {
		add(cidr)
	}

	return result
}

// addOSRoutes adds routing table entries for CIDRs on Linux/macOS where
// `wg set` does not manage OS routes.
func (r *PolicyResolver) addOSRoutes(ps *profileState, cidrs []string) {
	for _, cidr := range cidrs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		_ = r.routeMgr.Add(routing.Route{
			Destination: *ipnet,
			Interface:   ps.ifaceName,
			Metric:      10,
		})
	}
}

// lookupAll resolves a domain using both the system resolver and each VPN DNS
// server. This captures different CDN edges and maximises IP coverage.
func (r *PolicyResolver) lookupAll(domain string, vpnDNS []net.IP) []net.IP {
	seen := make(map[string]bool)
	var result []net.IP
	add := func(ips []net.IP) {
		for _, ip := range ips {
			key := ip.String()
			if !seen[key] {
				seen[key] = true
				result = append(result, ip)
			}
		}
	}

	// System resolver with timeout (can otherwise block for a long time).
	sysCtx, sysCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if ips, err := net.DefaultResolver.LookupIPAddr(sysCtx, domain); err == nil {
		ipList := make([]net.IP, len(ips))
		for i, a := range ips {
			ipList[i] = a.IP
		}
		add(ipList)
	}
	sysCancel()

	// VPN DNS servers via custom resolver.
	for _, dns := range vpnDNS {
		if dns.To4() == nil && dns.To16() == nil {
			continue
		}
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, "udp", net.JoinHostPort(dns.String(), "53"))
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if ips, err := resolver.LookupIPAddr(ctx, domain); err == nil {
			ipList := make([]net.IP, len(ips))
			for i, a := range ips {
				ipList[i] = a.IP
			}
			add(ipList)
		}
		cancel()
	}

	return result
}

// expandDomainPattern expands a domain pattern into concrete lookup targets.
// "*.anthropic.com" → ["anthropic.com", "www.anthropic.com", "api.anthropic.com", ...]
// "claude.ai" → ["claude.ai"]
func expandDomainPattern(pattern string) []string {
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
		return ip.String() + "/32"
	}
	return ip.String() + "/128"
}
