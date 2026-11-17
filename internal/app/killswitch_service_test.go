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
	s := NewKillSwitchService(cfg, adapters, nil)
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
	// The user turns it off: no block, traffic goes to the normal internet.
	s.NoteIntent("brainbox", true)
	_ = s.Apply()
	if len(b.set) != 0 || s.Status().State != "armed" {
		t.Fatalf("blocked after down: %v %+v", b.set, s.Status())
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
