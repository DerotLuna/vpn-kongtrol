package monitor

import (
	"net"
	"sort"
	"sync"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// DNSManager reference-counts DNS guard activations across multiple VPN
// adapters. The guard is applied on the first OnConnect and released only
// when the last active connection calls OnDisconnect.
//
// All DNS servers from all active adapters are merged (deduped) when the
// guard is (re)applied.
type DNSManager struct {
	mu      sync.Mutex
	guard   security.DNSGuard
	iface   string            // last known tunnel interface name
	active  map[string][]net.IP // profile → dns servers
	log     *zap.Logger
}

// NewDNSManager creates a manager backed by the given guard implementation.
func NewDNSManager(guard security.DNSGuard, log *zap.Logger) *DNSManager {
	return &DNSManager{
		guard:  guard,
		active: make(map[string][]net.IP),
		log:    log,
	}
}

// OnConnect registers profile's DNS servers and (re)applies the guard with
// the merged server list. iface is the tunnel interface name (e.g. "wg0").
// Safe to call multiple times for the same profile.
func (m *DNSManager) OnConnect(profile string, iface string, dnsServers []net.IP) {
	if len(dnsServers) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.active[profile] = dnsServers
	if iface != "" {
		m.iface = iface
	}
	m.apply()
}

// OnDisconnect removes profile from the active set.
// If no profiles remain active the guard is restored.
func (m *DNSManager) OnDisconnect(profile string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.active, profile)

	if len(m.active) == 0 {
		m.restore()
	} else {
		m.apply()
	}
}

// ForceRestore unconditionally restores original DNS (call on SIGTERM/panic).
func (m *DNSManager) ForceRestore() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active = make(map[string][]net.IP)
	m.restore()
}

// apply must be called with m.mu held.
func (m *DNSManager) apply() {
	merged := m.merged()
	if err := m.guard.Apply(m.iface, merged); err != nil {
		m.log.Error("dnsmanager: failed to apply DNS guard", zap.Error(err))
		return
	}
	m.log.Info("dnsmanager: DNS guard applied", zap.Int("servers", len(merged)))
}

// restore must be called with m.mu held.
func (m *DNSManager) restore() {
	if !m.guard.IsActive() {
		return
	}
	if err := m.guard.Restore(); err != nil {
		m.log.Error("dnsmanager: failed to restore DNS", zap.Error(err))
		return
	}
	m.log.Info("dnsmanager: DNS restored")
}

// IsActive reports whether the DNS guard is currently applied to any profile.
func (m *DNSManager) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.active) > 0
}

// DNSGuardSnapshot is what the DNS guard is doing now, for status views.
type DNSGuardSnapshot struct {
	Interface string              `json:"interface,omitempty"`
	Servers   []string            `json:"servers"`  // merged, deduped, in profile name order
	Profiles  map[string][]string `json:"profiles"` // profile → its DNS servers
}

// Snapshot returns the servers the guard applies now (empty when idle).
// Deterministic: profiles are visited in name order.
func (m *DNSManager) Snapshot() DNSGuardSnapshot {
	out := DNSGuardSnapshot{Servers: []string{}, Profiles: map[string][]string{}}
	if m == nil {
		return out
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.active) == 0 {
		return out
	}
	out.Interface = m.iface
	names := make([]string, 0, len(m.active))
	for n := range m.active {
		names = append(names, n)
	}
	sort.Strings(names)
	seen := map[string]bool{}
	for _, n := range names {
		var list []string
		for _, ip := range m.active[n] {
			s := ip.String()
			list = append(list, s)
			if !seen[s] {
				seen[s] = true
				out.Servers = append(out.Servers, s)
			}
		}
		out.Profiles[n] = list
	}
	return out
}

// merged returns a deduped list of all DNS servers from active profiles.
func (m *DNSManager) merged() []net.IP {
	seen := make(map[string]struct{})
	var result []net.IP
	for _, servers := range m.active {
		for _, ip := range servers {
			key := ip.String()
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				result = append(result, ip)
			}
		}
	}
	return result
}
