package app

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func TestFullTunnelConflicts(t *testing.T) {
	wg := TunnelClaim{Profile: "wg-home", Label: "wg-home", Interface: "wg-home"}
	proton := ExternalTunnel{Type: "protonvpn", Label: "Proton VPN", Interface: "ProtonVPN"}
	cases := []struct {
		name     string
		cand     string
		typ      string
		full     bool
		managed  []TunnelClaim
		ifaces   []string
		external []ExternalTunnel
		want     string // label of the first conflict, "" = none
	}{
		{"split tunnel never conflicts", "office", "wireguard", false, []TunnelClaim{wg}, nil, []ExternalTunnel{proton}, ""},
		{"nothing else up", "wg-home", "wireguard", true, nil, nil, nil, ""},
		{"second managed full tunnel", "proton", "protonvpn", true, []TunnelClaim{wg}, []string{"wg-home"}, nil, "wg-home"},
		{"itself is not a conflict", "wg-home", "wireguard", true, []TunnelClaim{wg}, nil, nil, ""},
		{"provider app up outside Kongtrol", "wg-home", "wireguard", true, nil, nil, []ExternalTunnel{proton}, "Proton VPN"},
		{"same provider app = attach", "proton", "protonvpn", true, nil, nil, []ExternalTunnel{proton}, ""},
		{"matcher hit on Kongtrol's own interface", "nord", "nordvpn", true, nil, []string{"protonvpn"}, []ExternalTunnel{proton}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := fullTunnelConflicts(c.cand, c.typ, c.full, c.managed, c.ifaces, c.external)
			label := ""
			if len(got) > 0 {
				label = got[0].Label
			}
			if label != c.want {
				t.Fatalf("conflict = %q, want %q (%+v)", label, c.want, got)
			}
		})
	}
}

func TestProfileCarriesAllTraffic(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	redirect := write("full.ovpn", "client\nremote vpn.example 1194\nredirect-gateway def1\n")
	commented := write("split.ovpn", "client\n# redirect-gateway def1\n;redirect-gateway\nroute 10.0.0.0 255.0.0.0\n")
	for _, c := range []struct {
		v    config.VPNConfig
		want bool
	}{
		{config.VPNConfig{Type: "protonvpn"}, true},
		{config.VPNConfig{Type: "nordvpn"}, true},
		{config.VPNConfig{Type: "surfshark"}, true},
		{config.VPNConfig{Type: "forticlient"}, false},
		{config.VPNConfig{Type: "wireguard"}, false},
		{config.VPNConfig{Type: "wireguard", DefaultEgress: true}, true},
		{config.VPNConfig{Type: "openvpn", ConfigFile: redirect}, true},
		{config.VPNConfig{Type: "openvpn", ConfigFile: commented}, false},
		{config.VPNConfig{Type: "openvpn", ConfigFile: filepath.Join(dir, "missing.ovpn")}, false},
	} {
		if got := ProfileCarriesAllTraffic(c.v); got != c.want {
			t.Errorf("%s %q default_egress=%v: got %v, want %v", c.v.Type, filepath.Base(c.v.ConfigFile), c.v.DefaultEgress, got, c.want)
		}
	}
}

// A connect that would start a second full tunnel is refused before the
// adapter runs, with a translated reason, its "what to do" line and a
// sentinel the retrying callers can test.
func TestProfileService_RefusesSecondFullTunnel(t *testing.T) {
	cfg := &config.Config{VPNs: map[string]config.VPNConfig{
		"wg-home": {Type: "wireguard", DefaultEgress: true},
		"proton":  {Type: "protonvpn"},
		"office":  {Type: "wireguard"},
	}}
	wg := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "wg-home"}}
	proton := &countingAdapter{}
	office := &countingAdapter{}
	var external []ExternalTunnel
	svc := NewProfileService(ProfileServiceDeps{
		Cfg:             cfg,
		Adapters:        vpn.NewSet(map[string]vpn.VPNAdapter{"wg-home": wg, "proton": proton, "office": office}),
		ExternalTunnels: func() []ExternalTunnel { return external },
	})

	err := svc.Connect(context.Background(), "proton")
	var ve *config.ValidationError
	if !errors.Is(err, ErrFullTunnelConflict) || !errors.As(err, &ve) || ve.Key != "api.err.connect_full_conflict" {
		t.Fatalf("err = %v", err)
	}
	if proton.connects != 0 {
		t.Fatal("adapter ran despite the conflict")
	}
	// A split tunnel next to a full tunnel is allowed.
	if err := svc.Connect(context.Background(), "office"); err != nil || office.connects != 1 {
		t.Fatalf("split connect: err=%v connects=%d", err, office.connects)
	}

	// The reverse: Proton's app is up and the direct full tunnel is off.
	wg.status = vpn.StatusDisconnected
	external = []ExternalTunnel{{Type: "protonvpn", Label: "Proton VPN", Interface: "ProtonVPN"}}
	err = svc.CheckFullTunnelConflict("wg-home")
	if !errors.As(err, &ve) || ve.Key != "api.err.connect_full_conflict_app" || ve.Args[1] != "Proton VPN" {
		t.Fatalf("err = %v", err)
	}
	// Connecting the Proton profile itself attaches to the app: no conflict.
	if err := svc.CheckFullTunnelConflict("proton"); err != nil {
		t.Fatalf("attach refused: %v", err)
	}
}

type countingAdapter struct {
	fakeAdapter
	connects int
}

func (c *countingAdapter) Connect(context.Context, vpn.AdapterConfig) error {
	c.connects++
	c.status = vpn.StatusConnected
	return nil
}

// A full tunnel that dropped while another VPN carries all traffic (a
// provider app outside Kongtrol, or another Kongtrol profile) is never
// blocked: the block would also cut that VPN, i.e. the whole internet. The
// user is told once, and the drop does not block later either.
func TestKillSwitch_DroppedFullTunnelSupersededIsNotBlocked(t *testing.T) {
	t.Run("provider app took over", func(t *testing.T) {
		full := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun1"}}
		s, b := newKS(t, ksCfg(), map[string]vpn.VPNAdapter{"full": full},
			[]routing.Route{rt("0.0.0.0/1", "tun1"), rt("128.0.0.0/1", "tun1")}, fakeDests{})
		var external []ExternalTunnel
		s.SetExternalTunnels(func() []ExternalTunnel { return external })
		var notices []string
		s.SetOnSuperseded(func(name, by string) { notices = append(notices, name+">"+by) })
		blocks := 0
		s.SetOnBlock(func(string, bool) { blocks++ })
		_ = s.Apply()

		external = []ExternalTunnel{{Type: "protonvpn", Label: "Proton VPN", Interface: "ProtonVPN"}}
		full.status = vpn.StatusError
		_ = s.Apply()
		_ = s.Apply()
		if len(b.set) != 0 || blocks != 0 || s.Status().State == "active" {
			t.Fatalf("blocked = %v, onBlock = %d, status = %+v", b.set, blocks, s.Status())
		}
		if len(notices) != 1 || notices[0] != "full>Proton VPN" {
			t.Fatalf("notices = %v", notices)
		}
		// Proton goes away later: the user had moved on, still no block.
		external = nil
		_ = s.Apply()
		if len(b.set) != 0 {
			t.Fatalf("blocked after the other VPN left: %v", b.set)
		}
	})
	t.Run("another Kongtrol full tunnel", func(t *testing.T) {
		cfg := ksCfg()
		cfg.VPNs["other"] = config.VPNConfig{Type: "openvpn", Host: "198.51.100.11"}
		full := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun1"}}
		other := &fakeAdapter{status: vpn.StatusDisconnected}
		s, b := newKS(t, cfg, map[string]vpn.VPNAdapter{"full": full, "other": other},
			[]routing.Route{rt("0.0.0.0/1", "tun1"), rt("128.0.0.0/1", "tun1"), rt("0.0.0.0/0", "tun2")}, fakeDests{})
		_ = s.Apply()
		full.status = vpn.StatusDisconnected
		other.status, other.info = vpn.StatusConnected, &vpn.TunnelInfo{InterfaceName: "tun2", RemoteIP: net.ParseIP("198.51.100.11")}
		_ = s.Apply()
		if len(b.set) != 0 || len(s.Status().Blocking) != 0 {
			t.Fatalf("blocked = %v, status = %+v", b.set, s.Status())
		}
	})
	t.Run("nobody took over: the usual full block", func(t *testing.T) {
		full := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "tun1"}}
		s, b := newKS(t, ksCfg(), map[string]vpn.VPNAdapter{"full": full},
			[]routing.Route{rt("0.0.0.0/1", "tun1"), rt("128.0.0.0/1", "tun1")}, fakeDests{})
		// Nothing outside Kongtrol: the usual full block with its notice.
		s.SetExternalTunnels(func() []ExternalTunnel { return nil })
		_ = s.Apply()
		full.status = vpn.StatusDisconnected
		_ = s.Apply()
		if len(b.set) == 0 || len(s.Status().FullTunnel) != 1 {
			t.Fatalf("no full block: %v %+v", b.set, s.Status())
		}
	})
}
