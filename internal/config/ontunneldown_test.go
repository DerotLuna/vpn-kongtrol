package config

import (
	"errors"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func TestValidateOnTunnelDown(t *testing.T) {
	ok := []PolicyRule{
		{Name: "a"},
		{Name: "b", OnTunnelDown: "allow"},
		{Name: "c", OnTunnelDown: "block", Match: MatchSpec{Domains: []string{"claude.ai"}}},
		{Name: "d", OnTunnelDown: "connect", Match: MatchSpec{Apps: []string{"x.exe"}}},
	}
	for _, p := range ok {
		if err := ValidateOnTunnelDown(p); err != nil {
			t.Errorf("%s: %v", p.Name, err)
		}
	}
	var ve *ValidationError
	err := ValidateOnTunnelDown(PolicyRule{Name: "x", OnTunnelDown: "deny"})
	if !errors.As(err, &ve) || ve.Key != "config.err.on_tunnel_down" || ve.Hint(i18n.ES) == "" {
		t.Fatalf("bad value: %v", err)
	}
	err = ValidateOnTunnelDown(PolicyRule{Name: "x", OnTunnelDown: "block", Match: MatchSpec{Apps: []string{"x.exe"}}})
	if !errors.As(err, &ve) || ve.Key != "config.err.on_tunnel_down_apps" {
		t.Fatalf("apps-only block: %v", err)
	}
	if (PolicyRule{}).TunnelDownAction() != "allow" {
		t.Fatal("default action")
	}
	if !(KillSwitchConfig{Mode: "loose"}).LANAllowed() || (KillSwitchConfig{Mode: "strict"}).LANAllowed() {
		t.Fatal("legacy mode mapping")
	}
}
