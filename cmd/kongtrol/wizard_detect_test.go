package main

import "testing"

func TestBinaryPathPlan(t *testing.T) {
	standard := func(_, path string) bool { return path == "/std/bin" }
	cases := []struct {
		name, adapter, detected string
		ask                     bool
		suggestion              string
	}{
		{"detected in standard location: no prompt, no yaml", "tailscale", "/std/bin", false, ""},
		{"detected elsewhere: prompt prefilled", "openvpn", "/opt/x/openvpn", true, "/opt/x/openvpn"},
		{"not detected: empty prompt", "wireguard", "", true, ""},
		{"warp never asks", "cloudflarewarp", "/opt/warp-cli", false, ""},
	}
	for _, c := range cases {
		ask, sug := binaryPathPlan(c.adapter, c.detected, standard)
		if ask != c.ask || sug != c.suggestion {
			t.Errorf("%s: got (%v,%q) want (%v,%q)", c.name, ask, sug, c.ask, c.suggestion)
		}
	}
}

func TestDetectedBinaryPath(t *testing.T) {
	d := []detectedVPN{{label: "TunnelBear", path: "/tb"}, {label: "WireGuard", adapterType: "wireguard", path: "/wg"}}
	if got := detectedBinaryPath(d, "wireguard"); got != "/wg" {
		t.Fatalf("got %q", got)
	}
	if got := detectedBinaryPath(d, "openvpn"); got != "" {
		t.Fatalf("got %q", got)
	}
	if keys := detectedAdapterKeys(d); len(keys) != 1 || !keys["wireguard"] {
		t.Fatalf("keys = %v", keys)
	}
}
