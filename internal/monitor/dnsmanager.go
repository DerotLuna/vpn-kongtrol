package monitor

import (
	"net"
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
	iface   string
	active  map[string][]net.IP // profile → dns servers
	log     *zap.Logger
}

// NewDNSManager creates a manager backed by the given guard implementation.
// iface is the network interface name to protect (empty = OS default).
func NewDNSManager(guard security.DNSGuard, iface string, log *zap.Logger) *DNSManager {
	return &DNSManager{
		guard:  guard,
		iface:  iface,
		active: make(map[string][]net.IP),
		log:    log,
	}
}

// OnConnect registers profile's DNS servers and (re)applies the guard with
// the merged server list. Safe to call multiple times for the same profile.
func (m *DNSManager) OnConnect(profile string, dnsServers []net.IP) {
	if len(dnsServers) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.active[profile] = dnsServers
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
