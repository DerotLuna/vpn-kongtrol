package protonvpn

import (
	"testing"
)

func TestParseStatus(t *testing.T) {
	ok, srv, ip := parseStatus(`{"Status":"Connected","ServerIP":"1.2.3.4","IP":"10.2.0.2"}`)
	if !ok || srv != "1.2.3.4" || ip != "10.2.0.2" {
		t.Fatalf("json: %v %q %q", ok, srv, ip)
	}
	ok, srv, _ = parseStatus("Status: Connected\nServer IP: 5.6.7.8\n")
	if !ok || srv != "5.6.7.8" {
		t.Fatalf("text: %v %q", ok, srv)
	}
}

func TestFlavorOf(t *testing.T) {
	cases := map[string]flavor{
		"/usr/bin/protonvpn":         modernCLI,
		"/usr/bin/protonvpn-cli":     legacyCLI,
		"/opt/x/ProtonVPN-CLI":       legacyCLI,
		"/usr/local/bin/protonvpn":   modernCLI,
		"/home/u/.local/bin/my-wrap": modernCLI,
	}
	for path, want := range cases {
		if got := flavorOf(path); got != want {
			t.Errorf("flavorOf(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestParseModernStatus(t *testing.T) {
	cases := []struct {
		raw              string
		known, connected bool
	}{
		{"Status: Connected\nServer: CH#242\n", true, true},
		{"Status: Disconnected\n", true, false},
		{"Connected to CH#242 (Zurich)\n", true, true},
		{"You are not connected.\n", true, false},
		{"No active connection\n", true, false},
		{"Usage: protonvpn [OPTIONS] COMMAND\n", false, false},
		{"", false, false},
	}
	for _, c := range cases {
		known, connected := parseModernStatus(c.raw)
		if known != c.known || connected != c.connected {
			t.Errorf("parseModernStatus(%q) = %v,%v; want %v,%v", c.raw, known, connected, c.known, c.connected)
		}
	}
}

func TestParseStatusDisconnectedIsNotConnected(t *testing.T) {
	if ok, _, _ := parseStatus("Status: Disconnected\n"); ok {
		t.Fatal(`"Disconnected" must not parse as connected`)
	}
}
