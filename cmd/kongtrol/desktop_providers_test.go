package main

import (
	"net"
	"reflect"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

func TestNewDesktopProviders(t *testing.T) {
	for _, tc := range []struct {
		typ, description string
		extra            map[string]string
		commands         [][]string
	}{
		{"mullvad", "wg0-mullvad", map[string]string{"country": "SE", "city": "got"}, [][]string{{"relay", "set", "location", "se", "got"}, {"connect"}}},
		{"pia", "PIA OpenVPN WinTUN Adapter", map[string]string{"server": "us_california"}, [][]string{{"set", "region", "us_california"}, {"connect"}}},
		{"expressvpn", "ExpressVPN TUN Driver", map[string]string{"server": "Germany - Frankfurt - 1"}, [][]string{{"connect", "Germany - Frankfurt - 1"}}},
		{"windscribe", "WindscribeWireguard", map[string]string{"server": "New York"}, [][]string{{"connect", "New York"}}},
		{"cyberghost", "CyberGhost WireGuard Adapter", nil, nil},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			adapter, err := vpn.New(tc.typ)
			if err != nil || adapter.Name() != tc.typ {
				t.Fatalf("registry: %v", err)
			}
			cfg := config.VPNConfig{Type: tc.typ, Server: tc.extra["server"], Country: tc.extra["country"], City: tc.extra["city"]}
			if err := config.Validate(&config.Config{VPNs: map[string]config.VPNConfig{"example": cfg.WithDefaults()}}); err != nil {
				t.Fatal(err)
			}
			if !vpndetect.Known(tc.typ) || !app.ProfileCarriesAllTraffic(cfg) {
				t.Fatal("provider is missing installation detection or coexistence safeguards")
			}
			desktop, _ := appattach.DesktopDefinition(tc.typ)
			address := net.ParseIP("10.7.0.2")
			if !desktop.Matcher.Match(appattach.Iface{Name: tc.description, Up: true, Addrs: []net.IP{address}}) {
				t.Fatal("missed the branded interface")
			}
			for _, unrelated := range []string{"utun2", "wg0", "tun0", "TAP-Windows Adapter V9", "Ethernet"} {
				if desktop.Matcher.Match(appattach.Iface{Name: unrelated, Up: true, Addrs: []net.IP{address}}) {
					t.Fatalf("claimed unrelated interface %q", unrelated)
				}
			}
			if !desktop.Matcher.WithOverride("utun2").Match(appattach.Iface{Name: "utun2", Up: true, Addrs: []net.IP{address}}) {
				t.Fatal("interface override did not work")
			}
			if tc.commands == nil {
				if desktop.ConnectArgs != nil || !adapter.Capabilities().ManualReconnect {
					t.Fatal("manual provider unexpectedly enables automation")
				}
				return
			}
			args, err := desktop.ConnectArgs(vpn.AdapterConfig{Extra: tc.extra})
			if err != nil || !reflect.DeepEqual(args, tc.commands) {
				t.Fatalf("commands = %v, %v; want %v", args, err, tc.commands)
			}
			if _, err := desktop.ConnectArgs(vpn.AdapterConfig{Extra: map[string]string{"server": "--help"}}); err == nil {
				t.Fatal("CLI option was accepted as a region")
			}
		})
	}
}
