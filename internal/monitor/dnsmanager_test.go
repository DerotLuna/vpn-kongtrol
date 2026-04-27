package monitor_test

import (
	"net"
	"sync"
	"testing"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// mockDNSGuard is a thread-safe DNSGuard stub that records calls.
type mockDNSGuard struct {
	mu       sync.Mutex
	active   bool
	applied  []net.IP
	restored int
}

func (m *mockDNSGuard) Apply(_ string, servers []net.IP) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active = true
	m.applied = servers
	return nil
}

func (m *mockDNSGuard) Restore() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.active = false
	m.restored++
	return nil
}

func (m *mockDNSGuard) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active
}

func (m *mockDNSGuard) appliedIPs() []net.IP {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]net.IP, len(m.applied))
	copy(out, m.applied)
	return out
}

func (m *mockDNSGuard) restoreCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restored
}

func newMgr(t *testing.T) (*monitor.DNSManager, *mockDNSGuard) {
	t.Helper()
	g := &mockDNSGuard{}
	return monitor.NewDNSManager(g, "", zap.NewNop()), g
}

var (
	dns1 = net.ParseIP("1.1.1.1")
	dns2 = net.ParseIP("8.8.8.8")
	dns3 = net.ParseIP("9.9.9.9")
)

// TestDNSManager_ApplyOnFirstConnect verifies the guard is applied when the
// first profile connects.
func TestDNSManager_ApplyOnFirstConnect(t *testing.T) {
	mgr, g := newMgr(t)
	mgr.OnConnect("warp", []net.IP{dns1})

	if !g.IsActive() {
		t.Fatal("guard should be active after first connect")
	}
	ips := g.appliedIPs()
	if len(ips) != 1 || !ips[0].Equal(dns1) {
		t.Errorf("applied IPs = %v, want [%v]", ips, dns1)
	}
}

// TestDNSManager_MergedOnSecondConnect verifies that connecting a second
// profile merges both DNS server lists (deduped).
func TestDNSManager_MergedOnSecondConnect(t *testing.T) {
	mgr, g := newMgr(t)
	mgr.OnConnect("warp", []net.IP{dns1})
	mgr.OnConnect("tailscale", []net.IP{dns2})

	ips := g.appliedIPs()
	if len(ips) != 2 {
		t.Fatalf("expected 2 merged IPs, got %d: %v", len(ips), ips)
	}
}

// TestDNSManager_DedupedMerge ensures duplicate IPs from two profiles appear once.
func TestDNSManager_DedupedMerge(t *testing.T) {
	mgr, g := newMgr(t)
	mgr.OnConnect("a", []net.IP{dns1, dns2})
	mgr.OnConnect("b", []net.IP{dns2, dns3}) // dns2 duplicated

	ips := g.appliedIPs()
	if len(ips) != 3 {
		t.Fatalf("expected 3 unique IPs, got %d: %v", len(ips), ips)
	}
}

// TestDNSManager_RestoreWhenLastDisconnects verifies the guard is restored only
// when all profiles have disconnected.
func TestDNSManager_RestoreWhenLastDisconnects(t *testing.T) {
	mgr, g := newMgr(t)
	mgr.OnConnect("a", []net.IP{dns1})
	mgr.OnConnect("b", []net.IP{dns2})

	// Disconnect first profile — guard must stay active.
	mgr.OnDisconnect("a")
	if !g.IsActive() {
		t.Fatal("guard should remain active while 'b' is still connected")
	}
	if g.restoreCount() > 0 {
		t.Fatal("Restore should not have been called yet")
	}

	// Disconnect second profile — now guard must restore.
	mgr.OnDisconnect("b")
	if g.IsActive() {
		t.Fatal("guard should be inactive after all profiles disconnect")
	}
	if g.restoreCount() != 1 {
		t.Fatalf("Restore called %d times, want 1", g.restoreCount())
	}
}

// TestDNSManager_ReapplyOnDisconnectWithRemaining verifies the guard is
// re-applied with the surviving profile's DNS after one of two disconnects.
func TestDNSManager_ReapplyOnDisconnectWithRemaining(t *testing.T) {
	mgr, g := newMgr(t)
	mgr.OnConnect("a", []net.IP{dns1})
	mgr.OnConnect("b", []net.IP{dns2})

	mgr.OnDisconnect("a")

	// Guard still active; applied IPs should now only contain dns2.
	if !g.IsActive() {
		t.Fatal("guard should still be active")
	}
	ips := g.appliedIPs()
	if len(ips) != 1 || !ips[0].Equal(dns2) {
		t.Errorf("after a disconnects, applied IPs = %v, want [%v]", ips, dns2)
	}
}

// TestDNSManager_ForceRestore clears all state and restores unconditionally.
func TestDNSManager_ForceRestore(t *testing.T) {
	mgr, g := newMgr(t)
	mgr.OnConnect("a", []net.IP{dns1})
	mgr.OnConnect("b", []net.IP{dns2})

	mgr.ForceRestore()

	if g.IsActive() {
		t.Fatal("guard should be inactive after ForceRestore")
	}
	if g.restoreCount() != 1 {
		t.Fatalf("Restore called %d times after ForceRestore, want 1", g.restoreCount())
	}
}

// TestDNSManager_ForceRestoreIdempotent verifies ForceRestore is safe when
// the guard is already inactive.
func TestDNSManager_ForceRestoreIdempotent(t *testing.T) {
	mgr, g := newMgr(t)
	// No connects — guard never activated.
	mgr.ForceRestore()

	if g.restoreCount() > 0 {
		t.Errorf("Restore called %d times on inactive guard, want 0", g.restoreCount())
	}
}

// TestDNSManager_SkipEmptyDNS verifies that a profile with no DNS servers
// does not activate the guard.
func TestDNSManager_SkipEmptyDNS(t *testing.T) {
	mgr, g := newMgr(t)
	mgr.OnConnect("wg-home", nil) // WireGuard manages its own DNS

	if g.IsActive() {
		t.Fatal("guard should not activate for a profile with no DNS servers")
	}
}
