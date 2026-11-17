package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestMarshal_MinimalConfigHasNoEmptyKeys guards the omitempty tags: a
// minimal config must not round-trip into kongtrol.yaml with blank keys.
func TestMarshal_MinimalConfigHasNoEmptyKeys(t *testing.T) {
	cfg := Config{VPNs: map[string]VPNConfig{
		"office": {Type: "openvpn", ConfigFile: "/etc/office.ovpn", Auth: AuthConfig{Method: "certificate"}},
	}}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	want := "vpns:\n    office:\n        type: openvpn\n        config: /etc/office.ovpn\n        auth:\n            method: certificate\n"
	if out != want {
		t.Fatalf("unexpected YAML:\n%s\nwant:\n%s", out, want)
	}

	// An explicit per-profile kill_switch: false must survive (pointer).
	off := false
	v := cfg.VPNs["office"]
	v.KillSwitch = &off
	cfg.VPNs["office"] = v
	data, _ = yaml.Marshal(&cfg)
	if !strings.Contains(string(data), "kill_switch: false") {
		t.Fatalf("explicit kill_switch: false dropped:\n%s", data)
	}
}
