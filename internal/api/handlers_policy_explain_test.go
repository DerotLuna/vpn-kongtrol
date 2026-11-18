package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

// GET /api/v1/policies/explain answers like `kongtrol policy show --json`:
// URLs are reduced to their host, app: targets keep their path, and the
// reason comes as a code for the client to translate.
func TestExplainPolicy(t *testing.T) {
	cfg := &config.Config{
		VPNs: map[string]config.VPNConfig{"office": {Type: "openvpn"}, "home": {Type: "wireguard"}},
		Policies: []config.PolicyRule{
			{Name: "work", Via: "office", Match: config.MatchSpec{Domains: []string{"*.corp.example"}, IPRanges: []string{"10.0.0.0/8"}}},
			{Name: "lab", Via: "home", Match: config.MatchSpec{IPRanges: []string{"10.2.0.0/16"}, Apps: []string{"code.exe"}}},
		},
	}
	e, err := policy.New(cfg)
	if err != nil {
		t.Fatalf("policy.New: %v", err)
	}
	s := &Server{}
	s.policyEngine.Store(e)

	cases := []struct {
		target, via, rule, code string
		matched                 bool
	}{
		{"https://git.corp.example/repo", "office", "work", "matched_domain", true},
		{"10.2.3.4", "home", "lab", "matched_cidr", true},
		{"10.9.9.9", "office", "work", "matched_cidr", true},
		{`app:C:\Tools\Code.exe`, "home", "lab", "matched_app", true},
		{"example.org", "", "", "no_domain_rule", false},
	}
	for _, c := range cases {
		q := url.Values{"target": {c.target}}.Encode()
		rec := httptest.NewRecorder()
		s.handleExplainPolicy(rec, httptest.NewRequest(http.MethodGet, "/api/v1/policies/explain?"+q, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", c.target, rec.Code)
		}
		var got policy.ExplainResult
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("%s: decode: %v", c.target, err)
		}
		if got.Matched != c.matched || got.Via != c.via || got.RuleName != c.rule || got.ReasonCode != c.code {
			t.Errorf("%s: got %+v", c.target, got)
		}
	}

	rec := httptest.NewRecorder()
	s.handleExplainPolicy(rec, httptest.NewRequest(http.MethodGet, "/api/v1/policies/explain", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty target: status %d, want 400", rec.Code)
	}
}
