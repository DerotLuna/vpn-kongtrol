package monitor

import (
	"context"
	"net"
	"net/netip"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// fakeRoutes is an in-memory routing.RouteManager. existing are routes the
// OS already has (e.g. pushed by the VPN server).
type fakeRoutes struct {
	mu       sync.Mutex
	existing []routing.Route
	added    map[string]routing.Route // dest|iface
	deleted  []string
	flushed  []string
}

func newFakeRoutes(existing ...routing.Route) *fakeRoutes {
	return &fakeRoutes{existing: existing, added: map[string]routing.Route{}}
}

func key(r routing.Route) string { return r.Destination.String() + "|" + r.Interface }

func (f *fakeRoutes) Add(r routing.Route) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.added[key(r)] = r
	return nil
}
func (f *fakeRoutes) Delete(r routing.Route) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.added, key(r))
	f.deleted = append(f.deleted, key(r))
	return nil
}
func (f *fakeRoutes) List() ([]routing.Route, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]routing.Route(nil), f.existing...)
	for _, r := range f.added {
		out = append(out, r)
	}
	return out, nil
}
func (f *fakeRoutes) Flush(iface string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flushed = append(f.flushed, iface)
	for k, r := range f.added {
		if r.Interface == iface {
			delete(f.added, k)
		}
	}
	return nil
}
func (f *fakeRoutes) keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for k := range f.added {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func mustRoute(cidr, iface string, gw string) routing.Route {
	_, n, _ := net.ParseCIDR(cidr)
	r := routing.Route{Destination: *n, Interface: iface}
	if gw != "" {
		r.Gateway = net.ParseIP(gw)
	}
	return r
}

func testResolver(t *testing.T, cfg *config.Config, rm routing.RouteManager, dns map[string][]string) *PolicyResolver {
	t.Helper()
	r := NewPolicyResolver(cfg, rm, nil)
	var mu sync.Mutex
	r.lookup = func(domain string, _ []net.IP) []net.IP {
		mu.Lock()
		defer mu.Unlock()
		var out []net.IP
		for _, s := range dns[domain] {
			out = append(out, net.ParseIP(s))
		}
		return out
	}
	r.wgSet = func(string, string, []string, string) error { return nil }
	return r
}

func policyCfg() *config.Config {
	return &config.Config{
		VPNs: map[string]config.VPNConfig{
			"brainbox": {Type: "openvpn"},
			"office":   {Type: "forticlient"},
		},
		Policies: []config.PolicyRule{
			{Name: "ai", Via: "brainbox", Match: config.MatchSpec{Domains: []string{"claude.ai"}, IPRanges: []string{"10.50.0.0/16"}}},
			{Name: "corp", Via: "office", Match: config.MatchSpec{IPRanges: []string{"10.50.0.0/16", "172.20.0.0/16"}}, OnTunnelDown: "block"},
		},
	}
}

func TestPolicyResolver_RoutesNonWireGuardTunnel(t *testing.T) {
	rm := newFakeRoutes(mustRoute("10.8.0.0/24", "tun0", "10.8.0.1")) // pushed by the server
	r := testResolver(t, policyCfg(), rm, map[string][]string{"claude.ai": {"160.79.104.10"}})
	if err := r.RegisterTunnel("brainbox", "tun0", nil); err != nil {
		t.Fatal(err)
	}
	got := rm.keys()
	want := []string{"10.50.0.0/16|tun0", "160.79.104.10/32|tun0"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("routes = %v, want %v", got, want)
	}
	// Next hop reused from the interface's own routes.
	if gw := rm.added["160.79.104.10/32|tun0"].Gateway; gw == nil || gw.String() != "10.8.0.1" {
		t.Fatalf("gateway = %v", gw)
	}
	// Tunnel goes down: routes removed (traffic falls back to normal internet).
	r.UnregisterProfile("brainbox")
	if len(rm.keys()) != 0 {
		t.Fatalf("routes left after down: %v", rm.keys())
	}
}

func TestPolicyResolver_SkipsEqualServerRoute(t *testing.T) {
	rm := newFakeRoutes(mustRoute("172.20.0.0/16", "forti", ""))
	r := testResolver(t, policyCfg(), rm, nil)
	_ = r.RegisterTunnel("office", "forti", nil)
	for _, k := range rm.keys() {
		if k == "172.20.0.0/16|forti" {
			t.Fatal("re-added a route the VPN already installed")
		}
	}
	// 10.50.0.0/16 belongs to the higher-priority policy (ai via brainbox).
	for _, k := range rm.keys() {
		if k == "10.50.0.0/16|forti" {
			t.Fatal("lower-priority policy got the overlapping CIDR")
		}
	}
}

func TestPolicyResolver_IPChangesAndProbe(t *testing.T) {
	rm := newFakeRoutes()
	dns := map[string][]string{"claude.ai": {"1.1.1.1"}}
	r := testResolver(t, policyCfg(), rm, dns)
	now := time.Unix(1000, 0)
	r.now = func() time.Time { return now }
	connected := true
	r.SetTunnelProbe(func(p string) TunnelState {
		if p == "brainbox" {
			return TunnelState{Connected: connected, Type: "openvpn", Interface: "tun0"}
		}
		return TunnelState{}
	})
	r.Reconcile()
	if !contains(rm.keys(), "1.1.1.1/32|tun0") {
		t.Fatalf("probe did not register: %v", rm.keys())
	}
	// IP rotates: new one added; old one dropped once stale.
	dns["claude.ai"] = []string{"2.2.2.2"}
	now = now.Add(staleAfter + time.Minute)
	r.resolve("", 0)
	r.applyAll()
	if !contains(rm.keys(), "2.2.2.2/32|tun0") || contains(rm.keys(), "1.1.1.1/32|tun0") {
		t.Fatalf("after rotation: %v", rm.keys())
	}
	// Destinations are known while the tunnel is down too.
	connected = false
	r.Reconcile()
	if len(rm.keys()) != 0 {
		t.Fatalf("routes left after drop: %v", rm.keys())
	}
	if d := r.ProfileDestinations("brainbox"); len(d) != 2 {
		t.Fatalf("destinations = %v", d)
	}
	m, ok := r.MatchAddr(netip.MustParseAddr("2.2.2.2"))
	if !ok || m.Policy != "ai" || m.Label != "claude.ai" || m.Via != "brainbox" || m.Action != "allow" {
		t.Fatalf("match = %+v %v", m, ok)
	}
	if m, _ := r.MatchAddr(netip.MustParseAddr("172.20.1.1")); m.Policy != "corp" || m.Action != "block" {
		t.Fatalf("static match = %+v", m)
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

type fakeMatcher map[netip.Addr]PolicyMatch

func (f fakeMatcher) MatchAddr(a netip.Addr) (PolicyMatch, bool) { m, ok := f[a]; return m, ok }

func TestPolicyWatch_NotifyThrottleAndConnect(t *testing.T) {
	a1, a2 := netip.MustParseAddr("2.2.2.2"), netip.MustParseAddr("3.3.3.3")
	log := NewEventLog(10)
	var connects []string
	var mu sync.Mutex
	done := make(chan struct{}, 4)
	now := time.Unix(0, 0)
	w := &PolicyWatch{
		Matcher: fakeMatcher{
			a1: {Policy: "ai", Via: "brainbox", Action: "allow", Label: "claude.ai"},
			a2: {Policy: "git", Via: "office", Action: "connect", Label: "git.corp"},
		},
		Status: func(string) vpn.Status { return vpn.StatusDisconnected },
		Connect: func(_ context.Context, p string) error {
			mu.Lock()
			connects = append(connects, p)
			mu.Unlock()
			done <- struct{}{}
			return nil
		},
		Events: log,
		conns:  func() ([]netip.Addr, error) { return []netip.Addr{a1, a1, a2}, nil },
		now:    func() time.Time { return now },
	}
	w.Check(context.Background())
	<-done
	evs, _ := log.After(0)
	if len(evs) != 2 {
		t.Fatalf("events = %+v", evs)
	}
	var allow Event
	for _, e := range evs {
		if e.Policy == "ai" {
			allow = e
		}
	}
	if allow.Key != "policy.event.allow" || allow.Args[0] != "claude.ai" || allow.Args[1] != "brainbox" || !allow.CanConnect {
		t.Fatalf("allow event = %+v", allow)
	}
	now = now.Add(time.Minute)
	w.Check(context.Background())
	if evs, _ := log.After(2); len(evs) != 0 {
		t.Fatalf("not throttled: %+v", evs)
	}
	now = now.Add(DefaultNoticeEvery)
	w.Check(context.Background())
	<-done
	mu.Lock()
	defer mu.Unlock()
	if len(connects) != 2 || connects[0] != "office" {
		t.Fatalf("connects = %v", connects)
	}
}

func TestParseProcNetTCP(t *testing.T) {
	text := `  sl  local_address rem_address   st tx_queue rx_queue
   0: 0100007F:0277 00000000:0000 0A 00000000:00000000
   1: 0F02000A:C350 0A68A4A0:01BB 01 00000000:00000000
   2: 0F02000A:C351 0B68A4A0:01BB 02 00000000:00000000
   3: 0F02000A:C352 0C68A4A0:01BB 06 00000000:00000000`
	got := parseProcNetTCP(text)
	if len(got) != 2 || got[0].String() != "160.164.104.10" || got[1].String() != "160.164.104.11" {
		t.Fatalf("got %v", got)
	}
}
