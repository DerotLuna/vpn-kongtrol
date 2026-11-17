package protonvpn

import (
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

func extra(kv ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func TestConnectArgs(t *testing.T) {
	cases := []struct {
		extra map[string]string
		proto string
		want  string
	}{
		{extra("server", "fastest"), "wireguard", "connect --fastest -p wireguard"},
		{extra("server", "ch"), "", "connect --cc CH"},
		{extra("server", "CH#4"), "udp", "connect CH#4 -p udp"},
		{extra("country", "de"), "", "connect --cc DE"},
		{extra("city", "Zurich"), "", "connect --fastest"}, // v3 has no city selection
	}
	for _, c := range cases {
		if got := strings.Join(connectArgs(targetFrom(c.extra), c.proto), " "); got != c.want {
			t.Errorf("connectArgs(%v,%q) = %q, want %q", c.extra, c.proto, got, c.want)
		}
	}
}

func TestModernConnectArgs(t *testing.T) {
	cases := []struct {
		extra map[string]string
		want  string
	}{
		{extra("server", "fastest"), "connect"},
		{extra(), "connect"},
		{extra("server", "us"), "connect --country US"},
		{extra("server", "CH#242"), "connect CH#242"},
		{extra("country", "United States"), "connect --country United States"},
		{extra("country", "US", "city", "New York"), "connect --city New York"},
	}
	for _, c := range cases {
		if got := strings.Join(modernConnectArgs(targetFrom(c.extra)), " "); got != c.want {
			t.Errorf("modernConnectArgs(%v) = %q, want %q", c.extra, got, c.want)
		}
	}
	// Dispatch: protocol only reaches the legacy CLI.
	if got := strings.Join(connectArgsFor(modernCLI, targetFrom(extra("server", "CH")), "udp"), " "); got != "connect --country CH" {
		t.Errorf("modern dispatch = %q", got)
	}
	if got := strings.Join(connectArgsFor(legacyCLI, targetFrom(extra("server", "CH")), "udp"), " "); got != "connect --cc CH -p udp" {
		t.Errorf("legacy dispatch = %q", got)
	}
}

func TestMatcher(t *testing.T) {
	for name, want := range map[string]bool{
		"proton0": true, "pvpn-wg": true, "ProtonVPN": true,
		"pvpnksintrf0": false, "pvpnroutintrf0": false, "ipv6leakintrf0": false,
		"wg0": false, "tun0": false, "eth0": false, "nordlynx": false,
	} {
		if got := matcher.Match(appattach.Iface{Name: name, Up: true}); got != want {
			t.Errorf("matcher.Match(%q) = %v, want %v", name, got, want)
		}
	}
	if !matcher.Match(appattach.Iface{Name: "Ethernet 2", Description: "TAP-ProtonVPN Windows Adapter V9", Up: true}) {
		t.Error("TAP-ProtonVPN adapter not matched by description")
	}
}

func TestLabel(t *testing.T) {
	if got := targetFrom(extra("server", "fastest")).label(); got != "" {
		t.Errorf("fastest label = %q", got)
	}
	if got := targetFrom(extra("country", "ch")).label(); got != "CH" {
		t.Errorf("country label = %q", got)
	}
}

func TestCapabilitiesManualWithoutCLI(t *testing.T) {
	a := &Adapter{}
	a.Configure(vpn.AdapterConfig{Extra: extra("auto_connect", "false")})
	if !a.Capabilities().ManualReconnect {
		t.Error("auto_connect: false must be ManualReconnect")
	}
	if CheckPlatform() != nil {
		a.Configure(vpn.AdapterConfig{})
		if !a.Capabilities().ManualReconnect {
			t.Error("no Proton CLI on this OS must be ManualReconnect")
		}
	}
}
