package configedit

import (
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func TestMovePolicies(t *testing.T) {
	c, path := load(t)
	c.VPNs["proton"] = config.VPNConfig{Type: "protonvpn", Auth: config.AuthConfig{Method: "none"}}
	for _, r := range []config.PolicyRule{
		{Name: "a", Via: "office", Match: config.MatchSpec{Domains: []string{"a.com"}}},
		{Name: "b", Via: "proton", Match: config.MatchSpec{Domains: []string{"b.com"}}},
		{Name: "c", Via: "office", Match: config.MatchSpec{IPRanges: []string{"10.0.0.0/8"}}},
	} {
		if _, err := AddPolicy(c, r); err != nil {
			t.Fatal(err)
		}
	}
	if n := CountPolicies(c, "office"); n != 2 {
		t.Fatalf("count = %d", n)
	}
	if _, err := MovePolicies(c, "office", "office"); key(err) != ErrKeyPolicyMoveSame {
		t.Fatalf("same: %v", err)
	}
	if _, err := MovePolicies(c, "office", "nope"); key(err) != "api.err.policy_via_unknown" {
		t.Fatalf("unknown: %v", err)
	}
	moved, err := MovePolicies(c, "office", "proton")
	if err != nil || len(moved) != 2 || moved[0] != "a" || moved[1] != "c" {
		t.Fatalf("moved = %v, %v", moved, err)
	}
	if _, err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	c2, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if CountPolicies(c2, "office") != 0 || CountPolicies(c2, "proton") != 3 {
		t.Fatalf("after save: %+v", c2.Policies)
	}
	if moved, err := MovePolicies(c2, "office", "proton"); err != nil || len(moved) != 0 {
		t.Fatalf("nothing left: %v %v", moved, err)
	}
}
