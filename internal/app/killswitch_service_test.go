package app

import (
	"context"
	"net"
	"net/netip"
	"slices"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type fakeBlocker struct{ set []netip.Prefix }

func (f *fakeBlocker) Set(d []netip.Prefix) error { f.set = dedupePrefixes(d); return nil }
func (f *fakeBlocker) Blocked() []netip.Prefix    { return f.set }

type fakeAdapter struct {
	status vpn.Status
	info   *vpn.TunnelInfo
}

func (f *fakeAdapter) Name() string                                     { return "fake" }
func (f *fakeAdapter) Version() string                                  { return "" }
func (f *fakeAdapter) Capabilities() vpn.Capabilities                   { return vpn.Capabilities{} }
func (f *fakeAdapter) Connect(context.Context, vpn.AdapterConfig) error { return nil }
func (f *fakeAdapter) Disconnect(context.Context) error                 { return nil }
func (f *fakeAdapter) Reconnect(context.Context) error                  { return nil }
func (f *fakeAdapter) Status() vpn.Status                               { return f.status }
func (f *fakeAdapter) TunnelInfo() (*vpn.TunnelInfo, error)             { return f.info, nil }

type fakeDests map[string][]netip.Prefix

func (f fakeDests) ProfileDestinations(p string) []netip.Prefix { return f["profile:"+p] }
func (f fakeDests) PolicyDestinations(p string) []netip.Prefix  { return f["policy:"+p] }

func pfx(ss ...string) []netip.Prefix {
	var out []netip.Prefix
	for _, s := range ss {
		out = append(out, netip.MustParsePrefix(s))
	}
	return out
}

func rt(cidr, iface string) routing.Route {
	_, n, _ := net.ParseCIDR(cidr)
	return routing.Route{Destination: *n, Interface: iface}
}

func newKS(t *testing.T, cfg *config.Config, adapters map[string]vpn.VPNAdapter, routes []routing.Route, d fakeDests) (*KillSwitchService, *fakeBlocker) {
	t.Helper()
	localSubnets = func() []netip.Prefix { return nil }
	s := NewKillSwitchService(cfg, vpn.NewSet(adapters), nil)
	s.lookupIP = func(string) []net.IP { return nil }
	b := &fakeBlocker{}
	s.SetBlocker(b)
	s.SetDestinations(d)
	s.SetRouteLister(func() ([]routing.Route, error) { return routes, nil })
	return s, b
}

func ksCfg() *config.Config {
	return &config.Config{
		VPNs: map[string]config.VPNConfig{
			"brainbox": {Type: "openvpn", Host: "198.51.100.7"},
			"full":     {Type: "openvpn", Host: "198.51.100.9"},
		},
		Security: config.SecurityConfig{KillSwitch: config.KillSwitchConfig{Enabled: true, Mode: "strict"}},
	}
}

func TestKillSwitch_SplitTunnelDropBlocksOnlyItsDestinations(t *testing.T) {
	bb := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun0"}}
	s, b := newKS(t, ksCfg(), map[string]vpn.VPNAdapter{"brainbox": bb},
		[]routing.Route{rt("10.20.0.0/16", "tun0"), rt("0.0.0.0/0", "Wi-Fi")},
		fakeDests{"profile:brainbox": pfx("160.79.104.10/32")})
	if err := s.Apply(); err != nil {
		t.Fatal(err)
	}
	if len(b.set) != 0 {
		t.Fatalf("blocked while up: %v", b.set)
	}
	if st := s.Status(); st.State != "armed" || len(st.Protected) != 1 {
		t.Fatalf("status = %+v", st)
	}
	bb.status = vpn.StatusDisconnected // unexpected drop
	_ = s.Apply()
	want := pfx("10.20.0.0/16", "160.79.104.10/32")
	if !slices.Equal(b.set, dedupePrefixes(want)) {
		t.Fatalf("blocked = %v, want %v", b.set, want)
	}
	if st := s.Status(); st.State != "active" || st.Blocking[0] != "brainbox" {
		t.Fatalf("status = %+v", st)
	}
	if got := s.Blocked(); !slices.Equal(got, []string{"10.20.0.0/16", "160.79.104.10/32"}) {
		t.Fatalf("Blocked() = %v", got)
	}
	// The user turns it off: no block, traffic goes to the normal internet.
	s.NoteIntent("brainbox", true)
	_ = s.Apply()
	if len(b.set) != 0 || s.Status().State != "armed" {
		t.Fatalf("blocked after down: %v %+v", b.set, s.Status())
	}
	if got := s.Blocked(); len(got) != 0 {
		t.Fatalf("Blocked() after down = %v", got)
	}
}

func TestKillSwitch_FullTunnelBlocksAllButEndpointAndLAN(t *testing.T) {
	cfg := ksCfg()
	cfg.Security.KillSwitch.Mode = "loose" // legacy: LAN allowed
	full := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun1", RemoteIP: net.ParseIP("203.0.113.5")}}
	s, b := newKS(t, cfg, map[string]vpn.VPNAdapter{"full": full},
		[]routing.Route{rt("0.0.0.0/1", "tun1"), rt("128.0.0.0/1", "tun1")}, fakeDests{})
	_ = s.Apply()
	full.status = vpn.StatusConnecting // watchdog reconnecting
	_ = s.Apply()
	if s.Status().State != "active" || len(s.Status().FullTunnel) != 1 {
		t.Fatalf("status = %+v", s.Status())
	}
	blocked := func(ip string) bool {
		a := netip.MustParseAddr(ip)
		for _, p := range b.set {
			if p.Contains(a) {
				return true
			}
		}
		return false
	}
	for ip, want := range map[string]bool{
		"8.8.8.8": true, "2606:4700::1111": true, "198.51.100.9": false, "203.0.113.5": false,
		"192.168.1.10": false, "127.0.0.1": false, "224.0.0.251": false,
	} {
		if blocked(ip) != want {
			t.Errorf("%s blocked=%v, want %v", ip, !want, want)
		}
	}
	full.status = vpn.StatusConnected
	_ = s.Apply()
	if len(b.set) != 0 {
		t.Fatal("still blocked after reconnect")
	}
}

// A full-tunnel drop with allow_lan off blocks the directly connected
// subnets too: the router there is usually the resolver the system falls
// back to once the tunnel's DNS is gone (systemd-resolved forgets a link's
// DNS when it disappears), so exempting it leaked lookups. Link-local,
// broadcast/multicast (DHCP, NDP) and the VPN endpoints stay reachable.
// With allow_lan on, the local subnets stay reachable. A split tunnel's
// block keeps exempting them either way.
func TestKillSwitch_FullTunnelLocalSubnetsFollowAllowLAN(t *testing.T) {
	for _, allowLAN := range []bool{false, true} {
		cfg := ksCfg()
		cfg.Security.KillSwitch.AllowLAN = allowLAN
		full := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun1"}}
		bb := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun0"}}
		s, b := newKS(t, cfg, map[string]vpn.VPNAdapter{"full": full, "brainbox": bb},
			[]routing.Route{rt("0.0.0.0/1", "tun1"), rt("128.0.0.0/1", "tun1"), rt("192.168.0.0/16", "tun0")},
			fakeDests{})
		localSubnets = func() []netip.Prefix { return pfx("192.168.1.0/24", "2001:db8:1::/64") }
		t.Cleanup(func() { localSubnets = func() []netip.Prefix { return nil } })
		blocked := func(ip string) bool {
			a := netip.MustParseAddr(ip)
			for _, p := range b.set {
				if p.Contains(a) {
					return true
				}
			}
			return false
		}
		_ = s.Apply() // both up: routes captured

		// Only the split tunnel drops: its captured 192.168/16 is blocked
		// except the directly connected subnet, whatever allow_lan says.
		bb.status = vpn.StatusError
		_ = s.Apply()
		if blocked("192.168.1.1") || !blocked("192.168.2.1") || blocked("8.8.8.8") {
			t.Errorf("allow_lan=%v split drop: blocked = %v", allowLAN, b.set)
		}

		// The user turns the split tunnel off; then the full tunnel drops.
		s.NoteIntent("brainbox", true)
		bb.status = vpn.StatusDisconnected
		full.status = vpn.StatusError
		_ = s.Apply()
		lanBlocked := !allowLAN
		for ip, want := range map[string]bool{
			"192.168.1.1": lanBlocked, "192.168.1.255": lanBlocked, "2001:db8:1::1": lanBlocked,
			"8.8.8.8": true, "198.51.100.9": false, "255.255.255.255": false, "0.0.0.0": false,
			"169.254.10.1": false, "fe80::1": false, "ff02::1:2": false, "224.0.0.251": false,
		} {
			if blocked(ip) != want {
				t.Errorf("allow_lan=%v: %s blocked=%v, want %v", allowLAN, ip, !want, want)
			}
		}
		if rest := subtractAll(pfx("192.168.1.0/24", "2001:db8:1::/64"), b.set); !allowLAN && len(rest) != 0 {
			t.Errorf("local subnets not fully blocked, still open: %v", rest)
		}
	}
}

// A drop is announced once per outage, saying whether everything is cut
// (full tunnel) or only the tunnel's destinations; HoldsAllTraffic is what
// keeps the DNS guard of a full tunnel (#37, ADR-0058).
func TestKillSwitch_OnBlockOncePerOutageAndHoldsAllTraffic(t *testing.T) {
	bb := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun0"}}
	full := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun1"}}
	s, _ := newKS(t, ksCfg(), map[string]vpn.VPNAdapter{"brainbox": bb, "full": full},
		[]routing.Route{rt("10.20.0.0/16", "tun0"), rt("0.0.0.0/0", "tun1")}, fakeDests{})
	got := map[string]bool{}
	calls := 0
	s.SetOnBlock(func(name string, fullTunnel bool) { got[name] = fullTunnel; calls++ })
	_ = s.Apply()
	if calls != 0 {
		t.Fatalf("notice while up: %d", calls)
	}
	bb.status, full.status = vpn.StatusError, vpn.StatusError
	if !s.HoldsAllTraffic("full") || s.HoldsAllTraffic("brainbox") {
		t.Fatal("HoldsAllTraffic must be true only for the full tunnel")
	}
	_ = s.Apply()
	_ = s.Apply() // same outage: no second notice
	if calls != 2 || got["brainbox"] || !got["full"] {
		t.Fatalf("notices = %v (%d calls)", got, calls)
	}
	s.NoteIntent("full", true) // the user turned it off
	if s.HoldsAllTraffic("full") {
		t.Fatal("a tunnel the user turned off never holds traffic")
	}
}

func TestKillSwitch_PolicyBlockAndConnectedTunnelKept(t *testing.T) {
	cfg := ksCfg()
	cfg.Security.KillSwitch.Enabled = false
	cfg.VPNs["office"] = config.VPNConfig{Type: "forticlient"}
	cfg.Policies = []config.PolicyRule{
		{Name: "corp", Via: "office", OnTunnelDown: "block", Match: config.MatchSpec{IPRanges: []string{"10.0.0.0/8"}}},
	}
	off := &fakeAdapter{status: vpn.StatusDisconnected}
	bb := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun0"}}
	s, b := newKS(t, cfg, map[string]vpn.VPNAdapter{"office": off, "brainbox": bb},
		[]routing.Route{rt("10.20.0.0/16", "tun0")},
		fakeDests{"policy:corp": pfx("10.0.0.0/8")})
	_ = s.Apply()
	// 10/8 is blocked except what the connected brainbox tunnel carries.
	if slices.Contains(b.set, netip.MustParsePrefix("10.0.0.0/8")) || len(b.set) == 0 {
		t.Fatalf("blocked = %v", b.set)
	}
	for _, p := range b.set {
		if p.Overlaps(netip.MustParsePrefix("10.20.0.0/16")) {
			t.Fatalf("blocked a connected tunnel's route: %v", p)
		}
	}
	if st := s.Status(); st.State != "off" || st.PolicyBlocking[0] != "corp" {
		t.Fatalf("status = %+v", st)
	}

}

func TestSubtractPrefix(t *testing.T) {
	got := subtractPrefix(netip.MustParsePrefix("10.0.0.0/8"), pfx("10.1.0.0/16"))
	if len(got) != 8 {
		t.Fatalf("got %d prefixes: %v", len(got), got)
	}
	for _, p := range got {
		if p.Overlaps(netip.MustParsePrefix("10.1.0.0/16")) {
			t.Fatalf("%v overlaps the exclusion", p)
		}
	}
	if subtractPrefix(netip.MustParsePrefix("10.1.2.0/24"), pfx("10.0.0.0/8")) != nil {
		t.Fatal("covered prefix not removed")
	}
}
