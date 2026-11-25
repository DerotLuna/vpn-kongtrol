package policy

import (
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"testing"
)

func TestResolveFlowPrefersDestinationUnlessAppOverrides(t *testing.T) {
	cfg := &config.Config{
		VPNs: map[string]config.VPNConfig{"application": {Priority: 50}, "destinations": {Priority: 1}},
		Policies: []config.PolicyRule{
			{Name: "app", Via: "application", Match: config.MatchSpec{Apps: []string{"example.exe"}}},
			{Name: "sites", Via: "destinations", Match: config.MatchSpec{Domains: []string{"example.net"}, IPRanges: []string{"198.18.0.0/24"}}},
		},
	}
	verify := func(mode, target, app string, onlyApp bool, want string) {
		t.Helper()
		profile := cfg.VPNs["application"]
		profile.AppRouteMode = mode
		cfg.VPNs["application"] = profile
		engine, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		var active func(string) bool
		if onlyApp {
			active = func(profile string) bool { return profile == "application" }
		}
		via, _, ok := engine.resolveFlow(target, app, active)
		if !ok || via != want {
			t.Fatalf("mode %q, target %q, app %q: %s; want %s", mode, target, app, via, want)
		}
	}
	verify("prefer_destinations", "example.net", "example.exe", false, "destinations")
	verify("prefer_destinations", "example.net", "browser.exe", false, "destinations")
	verify("prefer_destinations", "198.18.0.4", "example.exe", false, "destinations")
	verify("prefer_destinations", "another.net", "example.exe", false, "application")
	verify("prefer_destinations", "example.net", "example.exe", true, "application")
	verify("all", "example.net", "example.exe", false, "application")
	verify("all", "example.net", "browser.exe", false, "destinations")
}
