package app

import (
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// Only direct profiles from the provider's files that have rules are
// offered, by name (ADR-0104).
func TestRuleMoveCandidates(t *testing.T) {
	cfg := &config.Config{
		VPNs: map[string]config.VPNConfig{
			"proton-us": {Type: "wireguard", ConfigFile: writeProfileFile(t, "us.conf", protonWG)},
			"proton-ch": {Type: "wireguard", ConfigFile: writeProfileFile(t, "ch.conf", protonWG)},
			"proton-jp": {Type: "wireguard", ConfigFile: writeProfileFile(t, "jp.conf", protonWG)}, // no rules
			"office":    {Type: "wireguard", ConfigFile: writeProfileFile(t, "office.conf", "[Interface]\nAddress = 192.168.9.2/32\n")},
			"Proton":    {Type: "protonvpn"},
		},
		Policies: []config.PolicyRule{
			{Name: "a", Via: "proton-us"}, {Name: "b", Via: "proton-ch"}, {Name: "c", Via: "proton-ch"},
			{Name: "d", Via: "office"}, {Name: "e", Via: "Proton"},
		},
	}
	got := RuleMoveCandidates(cfg, "protonvpn")
	want := []RuleMoveCandidate{{Name: "proton-ch", Rules: 2}, {Name: "proton-us", Rules: 1}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if got := RuleMoveCandidates(cfg, "nordvpn"); len(got) != 0 {
		t.Fatalf("nordvpn: %+v", got)
	}
}
