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
		{Name: "d", OnTunnelDown: "connect", Match: MatchSpec{IPRanges: []string{"10.20.30.0/24"}}},
	}
	for _, p := range ok {
		if err := ValidateOnTunnelDown(p, VPNConfig{}); err != nil {
			t.Errorf("%s: %v", p.Name, err)
		}
	}
	var ve *ValidationError
	err := ValidateOnTunnelDown(PolicyRule{Name: "x", OnTunnelDown: "deny"}, VPNConfig{})
	if !errors.As(err, &ve) || ve.Key != "config.err.on_tunnel_down" || ve.Hint(i18n.ES) == "" {
		t.Fatalf("bad value: %v", err)
	}
	profile := VPNConfig{OnTunnelDown: OnTunnelDownConnect}
	rule := PolicyRule{Name: "x", Match: MatchSpec{Apps: []string{"x.exe"}}}
	err = ValidateOnTunnelDown(rule, profile)
	if !errors.As(err, &ve) || ve.Key != "config.err.on_tunnel_down_apps" {
		t.Fatalf("apps-only block: %v", err)
	}
	if rule.TunnelDownAction(profile) != OnTunnelDownConnect {
		t.Fatal("expected inherited connect")
	}
	rule.OnTunnelDown = OnTunnelDownAllow
	if err := ValidateOnTunnelDown(rule, profile); err != nil || rule.TunnelDownAction(profile) != OnTunnelDownAllow {
		t.Fatalf("explicit allow must override connect: %v", err)
	}
	if (PolicyRule{}).TunnelDownAction(VPNConfig{}) != "allow" {
		t.Fatal("default action")
	}
	if !(KillSwitchConfig{Mode: "loose"}).LANAllowed() || (KillSwitchConfig{Mode: "strict"}).LANAllowed() {
		t.Fatal("legacy mode mapping")
	}
}
