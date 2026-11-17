package main

import (
	"reflect"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// The user's real config: similar names and priorities, no failover
// declared. Priority must never become failover (ADR-0018).
func TestProfileFailover_OnlyExplicitDeclarations(t *testing.T) {
	old := cfg
	t.Cleanup(func() { cfg = old })
	cfg = &config.Config{VPNs: map[string]config.VPNConfig{
		"ekiipago":      {Type: "forticlient", Priority: 1},
		"us-content":    {Type: "protonvpn", Priority: 1},
		"brainbox":      {Type: "wireguard", Priority: 15},
		"ekiipago-ovpn": {Type: "openvpn", Priority: 20},
		"brainbox-ovpn": {Type: "openvpn", Priority: 20},
	}}
	if got := profileFailover(); len(got) != 0 {
		t.Fatalf("failover derived from priority/names: %v", got)
	}

	v := cfg.VPNs["brainbox"]
	v.Failover = []string{"brainbox-ovpn"}
	cfg.VPNs["brainbox"] = v
	want := map[string][]string{"brainbox": {"brainbox-ovpn"}}
	if got := profileFailover(); !reflect.DeepEqual(got, want) {
		t.Fatalf("profileFailover() = %v, want %v", got, want)
	}
}
