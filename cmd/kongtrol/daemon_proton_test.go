package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type protonFakeAdapter struct {
	status vpn.Status
	iface  string
}

func (f *protonFakeAdapter) Name() string                                     { return "fake" }
func (f *protonFakeAdapter) Version() string                                  { return "" }
func (f *protonFakeAdapter) Capabilities() vpn.Capabilities                   { return vpn.Capabilities{} }
func (f *protonFakeAdapter) Connect(context.Context, vpn.AdapterConfig) error { return nil }
func (f *protonFakeAdapter) Disconnect(context.Context) error                 { return nil }
func (f *protonFakeAdapter) Reconnect(context.Context) error                  { return nil }
func (f *protonFakeAdapter) Status() vpn.Status                               { return f.status }
func (f *protonFakeAdapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	return &vpn.TunnelInfo{InterfaceName: f.iface}, nil
}

// Proton's app connecting while a direct Proton profile is up: the daemon
// disconnects the profile (a system disconnect) and leaves one notice that
// offers "Reintentar" and, without a protonvpn profile, to add one
// (ADR-0103).
func TestCheckProtonHandover(t *testing.T) {
	conf := filepath.Join(t.TempDir(), "proton-ch.conf")
	body := "[Interface]\n# NetShield = 1\n# VPN Accelerator = on\nAddress = 10.2.0.2/32\nDNS = 10.2.0.1\n"
	if err := os.WriteFile(conf, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	oldEvents, oldDisc, oldSvc := daemonEvents, handoverDisconnect, profileSvc.Load()
	t.Cleanup(func() {
		daemonEvents, handoverDisconnect = oldEvents, oldDisc
		profileSvc.Store(oldSvc)
	})

	for _, c := range []struct {
		name      string
		appProf   bool
		wantKey   string
		wantArgs  []string
		wantOffer string
	}{
		{"no protonvpn profile", false, "watchdog.event.proton_handover_add", []string{"proton-ch", "protonvpn"}, "add_provider"},
		{"protonvpn profile", true, "watchdog.event.proton_handover", []string{"proton-ch", "proton"}, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			daemonEvents = monitor.NewEventLog(10)
			x := &protonFakeAdapter{status: vpn.StatusConnected, iface: "proton-ch"}
			c2 := &config.Config{VPNs: map[string]config.VPNConfig{"proton-ch": {Type: "wireguard", ConfigFile: conf}}}
			if c.appProf {
				c2.VPNs["proton"] = config.VPNConfig{Type: "protonvpn"}
			}
			ext := []app.ExternalTunnel{{Type: "protonvpn", Label: "Proton VPN", Interface: "ProtonVPN"}}
			profileSvc.Store(newTestProfileService(c2, map[string]vpn.VPNAdapter{"proton-ch": x}, ext))
			var disconnected []string
			handoverDisconnect = func(_ context.Context, name string) error {
				disconnected = append(disconnected, name)
				x.status = vpn.StatusDisconnected
				return nil
			}

			checkProtonHandover(context.Background())
			checkProtonHandover(context.Background()) // already handed over: nothing more

			if !reflect.DeepEqual(disconnected, []string{"proton-ch"}) {
				t.Fatalf("disconnected = %v", disconnected)
			}
			evs, _ := daemonEvents.After(0)
			if len(evs) != 1 {
				t.Fatalf("events = %+v", evs)
			}
			e := evs[0]
			if e.Key != c.wantKey || !reflect.DeepEqual(e.Args, c.wantArgs) || e.Offer != c.wantOffer ||
				!e.CanConnect || e.Profile != "proton-ch" || e.Kind != monitor.EventTunnelWatchdog {
				t.Fatalf("event = %+v", e)
			}
			for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
				if i18n.T(lang, e.Key) == e.Key {
					t.Errorf("%v: missing %s", lang, e.Key)
				}
			}
		})
	}
}

func newTestProfileService(c *config.Config, adapters map[string]vpn.VPNAdapter, external []app.ExternalTunnel) *app.ProfileService {
	return app.NewProfileService(app.ProfileServiceDeps{
		Cfg:             c,
		Adapters:        vpn.NewSet(adapters),
		ExternalTunnels: func() []app.ExternalTunnel { return external },
	})
}

// A reconnect refused for the same Proton account or tunnel address says
// that reason, not "another VPN carries all traffic".
func TestReconnectBlockedNotice(t *testing.T) {
	for _, c := range []struct {
		err      error
		wantKey  string
		wantArgs []string
	}{
		{&app.FullTunnelConflictError{ValidationError: &config.ValidationError{Key: "api.err.connect_proton_app", Args: []any{"x"}}},
			"watchdog.event.reconnect_blocked_proton", []string{"x"}},
		{&app.FullTunnelConflictError{ValidationError: &config.ValidationError{Key: "api.err.connect_same_address", Args: []any{"x", "y", "10.2.0.2"}}},
			"watchdog.event.reconnect_blocked_address", []string{"x", "y", "10.2.0.2"}},
		{&app.FullTunnelConflictError{ValidationError: &config.ValidationError{Key: "api.err.connect_full_conflict", Args: []any{"x", "y"}}},
			"watchdog.event.reconnect_blocked", []string{"x"}},
	} {
		key, args := reconnectBlockedNotice("x", c.err)
		if key != c.wantKey || !reflect.DeepEqual(args, c.wantArgs) || !offersRetry(key) {
			t.Errorf("%v: got %s %v", c.err, key, args)
		}
	}
}
