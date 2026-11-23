package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

const protonWG = "[Interface]\n# NetShield = 1\n# VPN Accelerator = on\n# Moderate NAT = off\nPrivateKey = x\nAddress = 10.2.0.2/32\nDNS = 10.2.0.1\n\n[Peer]\nPublicKey = y\nEndpoint = 203.0.113.7:51820\n"

func writeProfileFile(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

var protonApp = ExternalTunnel{Type: "protonvpn", Label: "Proton VPN", Interface: "ProtonVPN"}

func TestDirectProfileProvider(t *testing.T) {
	conf := writeProfileFile(t, "ch.conf", protonWG)
	ovpn := writeProfileFile(t, "us.ovpn", "client\nremote node-us-1.protonvpn.net 1194\n")
	plain := writeProfileFile(t, "home.conf", "[Interface]\nAddress = 10.2.0.2/32\n")
	for _, c := range []struct {
		v    config.VPNConfig
		want string
	}{
		{config.VPNConfig{Type: "wireguard", ConfigFile: conf}, "protonvpn"},
		{config.VPNConfig{Type: "openvpn", ConfigFile: ovpn}, "protonvpn"},
		{config.VPNConfig{Type: "wireguard", ConfigFile: plain}, ""},
		{config.VPNConfig{Type: "protonvpn"}, ""}, // the app profile is not a direct one
		{config.VPNConfig{Type: "wireguard", ConfigFile: filepath.Join(t.TempDir(), "missing.conf")}, ""},
	} {
		if got := DirectProfileProvider(c.v); got != c.want {
			t.Errorf("%s %s: got %q, want %q", c.v.Type, filepath.Base(c.v.ConfigFile), got, c.want)
		}
	}
}

// A direct Proton profile is refused while Proton's app is connected,
// split tunnel or not; a non-Proton split profile is not; Proton's app
// matched on Kongtrol's own tunnel is not the app (ADR-0103).
func TestProfileService_RefusesDirectProtonWhileAppUp(t *testing.T) {
	conf := writeProfileFile(t, "proton-ch.conf", protonWG)
	cfg := &config.Config{VPNs: map[string]config.VPNConfig{
		"proton-ch": {Type: "wireguard", ConfigFile: conf},
		"office":    {Type: "wireguard", ConfigFile: writeProfileFile(t, "office.conf", "[Interface]\nAddress = 192.168.9.2/32\n")},
	}}
	x := &countingAdapter{}
	external := []ExternalTunnel{protonApp}
	svc := NewProfileService(ProfileServiceDeps{
		Cfg:             cfg,
		Adapters:        vpn.NewSet(map[string]vpn.VPNAdapter{"proton-ch": x, "office": &countingAdapter{}}),
		ExternalTunnels: func() []ExternalTunnel { return external },
	})
	err := svc.Connect(context.Background(), "proton-ch")
	var ve *config.ValidationError
	if !errors.Is(err, ErrFullTunnelConflict) || !errors.As(err, &ve) || ve.Key != "api.err.connect_proton_app" || ve.Args[0] != "proton-ch" {
		t.Fatalf("err = %v", err)
	}
	if x.connects != 0 {
		t.Fatal("adapter ran despite the conflict")
	}
	if err := svc.CheckFullTunnelConflict("office"); err != nil {
		t.Fatalf("split non-Proton profile refused: %v", err)
	}
	// The matcher hit is Kongtrol's own direct tunnel (interface proton-ch).
	other := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "ProtonVPN"}}
	svc.adapters = vpn.NewSet(map[string]vpn.VPNAdapter{"proton-ch": x, "office": other})
	if err := svc.CheckFullTunnelConflict("proton-ch"); err != nil {
		t.Fatalf("own interface taken for the app: %v", err)
	}
	external = nil
	if err := svc.Connect(context.Background(), "proton-ch"); err != nil || x.connects != 1 {
		t.Fatalf("no app up: err=%v connects=%d", err, x.connects)
	}
}

// Two direct profiles with one tunnel address are never up together.
func TestProfileService_RefusesSharedTunnelAddress(t *testing.T) {
	cfg := &config.Config{VPNs: map[string]config.VPNConfig{
		"proton-ch": {Type: "wireguard", ConfigFile: writeProfileFile(t, "a.conf", protonWG)},
		"proton-us": {Type: "wireguard", ConfigFile: writeProfileFile(t, "b.conf", "[Interface]\nAddress = 10.2.0.2/32, 2a07:b944::2:2/128\n")},
		"proton-nl": {Type: "wireguard", ConfigFile: writeProfileFile(t, "c.conf", "[Interface]\nAddress = 10.3.0.2/32\n")},
	}}
	ch := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "proton-ch"}}
	us := &countingAdapter{}
	nl := &countingAdapter{}
	svc := NewProfileService(ProfileServiceDeps{
		Cfg:      cfg,
		Adapters: vpn.NewSet(map[string]vpn.VPNAdapter{"proton-ch": ch, "proton-us": us, "proton-nl": nl}),
	})
	err := svc.Connect(context.Background(), "proton-us")
	var ve *config.ValidationError
	if !errors.Is(err, ErrFullTunnelConflict) || !errors.As(err, &ve) || ve.Key != "api.err.connect_same_address" ||
		!slices.Equal(ve.Args, []any{"proton-us", "proton-ch", "10.2.0.2"}) {
		t.Fatalf("err = %v", err)
	}
	if us.connects != 0 {
		t.Fatal("adapter ran despite the conflict")
	}
	if err := svc.Connect(context.Background(), "proton-nl"); err != nil || nl.connects != 1 {
		t.Fatalf("10.3.0.2 refused: err=%v", err)
	}
}

// Proton's app appearing while a direct Proton profile is up hands the
// traffic over: the profile is listed, with the protonvpn profile that now
// shows the app when one exists.
func TestProfileService_ProtonAppHandovers(t *testing.T) {
	conf := writeProfileFile(t, "proton-ch.conf", protonWG)
	cfg := &config.Config{VPNs: map[string]config.VPNConfig{
		"proton-ch": {Type: "wireguard", ConfigFile: conf},
		"office":    {Type: "wireguard"},
	}}
	x := &fakeAdapter{status: vpn.StatusConnected, info: &vpn.TunnelInfo{InterfaceName: "proton-ch"}}
	var external []ExternalTunnel
	svc := NewProfileService(ProfileServiceDeps{
		Cfg:             cfg,
		Adapters:        vpn.NewSet(map[string]vpn.VPNAdapter{"proton-ch": x, "office": &fakeAdapter{status: vpn.StatusConnected}}),
		ExternalTunnels: func() []ExternalTunnel { return external },
	})
	if got := svc.ProtonAppHandovers(); got != nil {
		t.Fatalf("no app: %v", got)
	}
	// Proton's matcher sees Kongtrol's own direct tunnel: not the app.
	external = []ExternalTunnel{{Type: "protonvpn", Label: "Proton VPN", Interface: "proton-ch"}}
	if got := svc.ProtonAppHandovers(); got != nil {
		t.Fatalf("own interface: %v", got)
	}
	external = []ExternalTunnel{protonApp}
	if got := svc.ProtonAppHandovers(); !slices.Equal(got, []ProtonHandover{{Profile: "proton-ch"}}) {
		t.Fatalf("got %v", got)
	}
	cfg.VPNs["proton"] = config.VPNConfig{Type: "protonvpn"}
	if got := svc.ProtonAppHandovers(); !slices.Equal(got, []ProtonHandover{{Profile: "proton-ch", AppProfile: "proton"}}) {
		t.Fatalf("got %v", got)
	}
	x.status = vpn.StatusDisconnected
	if got := svc.ProtonAppHandovers(); got != nil {
		t.Fatalf("disconnected profile handed over: %v", got)
	}

	// The kill switch hint: split, not Proton, Proton's app up.
	if !svc.ProtonAppMayBlock("office") {
		t.Fatal("office: want hint")
	}
	if svc.ProtonAppMayBlock("proton-ch") {
		t.Fatal("direct Proton profile: no hint (its own conflict applies)")
	}
	external = nil
	if svc.ProtonAppMayBlock("office") {
		t.Fatal("no app: no hint")
	}
}
