package policy

import (
	"net"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func mustParseCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

func buildEngine(t *testing.T, policies []config.PolicyRule, vpns map[string]config.VPNConfig) *Engine {
	t.Helper()
	cfg := &config.Config{
		VPNs:     vpns,
		Policies: policies,
	}
	e, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

// ── IP resolution tests ───────────────────────────────────────────────────────

func TestResolveIP_ExactMatch(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "office", Match: config.MatchSpec{IPRanges: []string{"10.10.0.0/16"}}, Via: "office"},
	}, nil)

	vpn, ok := e.ResolveIP(net.ParseIP("10.10.5.1"))
	if !ok || vpn != "office" {
		t.Errorf("got (%q, %v), want (office, true)", vpn, ok)
	}
}

func TestResolveIP_NoMatch(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "office", Match: config.MatchSpec{IPRanges: []string{"10.10.0.0/16"}}, Via: "office"},
	}, nil)

	_, ok := e.ResolveIP(net.ParseIP("192.168.1.1"))
	if ok {
		t.Error("expected no match for 192.168.1.1")
	}
}

func TestResolveIP_LongestPrefixWins(t *testing.T) {
	// /24 is more specific than /16 — should win for 10.10.5.x
	e := buildEngine(t, []config.PolicyRule{
		{Name: "broad", Match: config.MatchSpec{IPRanges: []string{"10.10.0.0/16"}}, Via: "broad-vpn"},
		{Name: "narrow", Match: config.MatchSpec{IPRanges: []string{"10.10.5.0/24"}}, Via: "narrow-vpn"},
	}, nil)

	vpn, ok := e.ResolveIP(net.ParseIP("10.10.5.100"))
	if !ok || vpn != "narrow-vpn" {
		t.Errorf("got (%q, %v), want (narrow-vpn, true) — longest prefix should win", vpn, ok)
	}

	// IP outside the /24 should fall to the /16
	vpn, ok = e.ResolveIP(net.ParseIP("10.10.8.1"))
	if !ok || vpn != "broad-vpn" {
		t.Errorf("got (%q, %v), want (broad-vpn, true)", vpn, ok)
	}
}

func TestResolveIP_MultipleRanges(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "aws", Match: config.MatchSpec{IPRanges: []string{"172.31.0.0/16", "10.200.0.0/16"}}, Via: "aws"},
	}, nil)

	for _, ip := range []string{"172.31.5.1", "10.200.3.4"} {
		vpn, ok := e.ResolveIP(net.ParseIP(ip))
		if !ok || vpn != "aws" {
			t.Errorf("IP %s: got (%q, %v), want (aws, true)", ip, vpn, ok)
		}
	}
}

func TestResolveIP_PriorityOverridesPosition(t *testing.T) {
	// Both rules match, the one with higher priority should win.
	e := buildEngine(t,
		[]config.PolicyRule{
			{Name: "low-prio", Match: config.MatchSpec{IPRanges: []string{"10.0.0.0/8"}}, Via: "low"},
			{Name: "high-prio", Match: config.MatchSpec{IPRanges: []string{"10.0.0.0/8"}}, Via: "high"},
		},
		map[string]config.VPNConfig{
			"low":  {Priority: 1},
			"high": {Priority: 99},
		},
	)

	vpn, ok := e.ResolveIP(net.ParseIP("10.5.5.5"))
	if !ok || vpn != "high" {
		t.Errorf("got (%q, %v), want (high, true)", vpn, ok)
	}
}

// ── Domain resolution tests ───────────────────────────────────────────────────

func TestResolveDomain_ExactMatch(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "us", Match: config.MatchSpec{Domains: []string{"netflix.com"}}, Via: "us-vpn"},
	}, nil)

	vpn, ok := e.ResolveDomain("netflix.com")
	if !ok || vpn != "us-vpn" {
		t.Errorf("got (%q, %v), want (us-vpn, true)", vpn, ok)
	}
}

func TestResolveDomain_WildcardMatch(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "aws", Match: config.MatchSpec{Domains: []string{"*.amazonaws.com"}}, Via: "aws"},
	}, nil)

	cases := []struct {
		domain  string
		matches bool
	}{
		{"s3.amazonaws.com", true},
		{"ec2.amazonaws.com", true},
		{"amazonaws.com", false},         // wildcard requires a prefix
		{"sub.sub.amazonaws.com", false}, // wildcard is single-label only
		{"other.com", false},
	}

	for _, tc := range cases {
		_, ok := e.ResolveDomain(tc.domain)
		if ok != tc.matches {
			t.Errorf("domain %q: got %v, want %v", tc.domain, ok, tc.matches)
		}
	}
}

func TestResolveDomain_NoMatch(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "us", Match: config.MatchSpec{Domains: []string{"netflix.com"}}, Via: "us-vpn"},
	}, nil)

	_, ok := e.ResolveDomain("hulu.com")
	if ok {
		t.Error("expected no match for hulu.com")
	}
}

// ── App resolution tests ───────────────────────────────────────────────────────

func TestResolveApp_ExactExecutableName(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "streaming", Match: config.MatchSpec{Apps: []string{"vlc.exe"}}, Via: "us-vpn"},
	}, nil)

	vpn, ok := e.ResolveApp("C:\\Program Files\\VideoLAN\\VLC\\vlc.exe")
	if !ok || vpn != "us-vpn" {
		t.Errorf("got (%q, %v), want (us-vpn, true)", vpn, ok)
	}
}

func TestResolveApp_BaseNameWithoutExeMatches(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "browser", Match: config.MatchSpec{Apps: []string{"chrome"}}, Via: "work-vpn"},
	}, nil)

	vpn, ok := e.ResolveApp("C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe")
	if !ok || vpn != "work-vpn" {
		t.Errorf("got (%q, %v), want (work-vpn, true)", vpn, ok)
	}
}

func TestResolveApp_GlobPattern(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "dev-tools", Match: config.MatchSpec{Apps: []string{"code*"}}, Via: "dev-vpn"},
	}, nil)

	vpn, ok := e.ResolveApp("code-insiders.exe")
	if !ok || vpn != "dev-vpn" {
		t.Errorf("got (%q, %v), want (dev-vpn, true)", vpn, ok)
	}
}

func TestResolveApp_NoMatch(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "browser", Match: config.MatchSpec{Apps: []string{"chrome"}}, Via: "work-vpn"},
	}, nil)

	_, ok := e.ResolveApp("firefox.exe")
	if ok {
		t.Error("expected no match for firefox.exe")
	}
}

// ── Rule parsing tests ────────────────────────────────────────────────────────

func TestParseRule_InvalidCIDR(t *testing.T) {
	_, err := ParseRule("bad", "vpn", []string{"not-a-cidr"}, nil, nil, 0)
	if err == nil {
		t.Error("expected error for invalid CIDR, got nil")
	}
}

func TestParseRule_ValidCIDR(t *testing.T) {
	r, err := ParseRule("test", "vpn", []string{"10.0.0.0/8"}, []string{"*.example.com"}, []string{"chrome"}, 10)
	if err != nil {
		t.Fatalf("ParseRule: %v", err)
	}
	if r.Via != "vpn" {
		t.Errorf("Via = %q, want %q", r.Via, "vpn")
	}
	if len(r.Match.IPRanges) != 1 {
		t.Errorf("IPRanges len = %d, want 1", len(r.Match.IPRanges))
	}
	if len(r.Match.Apps) != 1 {
		t.Errorf("Apps len = %d, want 1", len(r.Match.Apps))
	}
}

// ── Glob matching tests ───────────────────────────────────────────────────────

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern string
		s       string
		want    bool
	}{
		{"*.example.com", "sub.example.com", true},
		{"*.example.com", "example.com", false},
		{"*.example.com", "deep.sub.example.com", false},
		{"exact.com", "exact.com", true},
		{"exact.com", "other.com", false},
		{"*.amazonaws.com", "s3.amazonaws.com", true},
		{"*.amazonaws.com", "amazonaws.com", false},
	}

	for _, tc := range cases {
		got := matchGlob(tc.pattern, tc.s)
		if got != tc.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tc.pattern, tc.s, got, tc.want)
		}
	}
}

// ── Engine.Rules() tests ──────────────────────────────────────────────────────

func TestEngine_Rules_ReturnsCopy(t *testing.T) {
	e := buildEngine(t, []config.PolicyRule{
		{Name: "r1", Match: config.MatchSpec{IPRanges: []string{"10.0.0.0/8"}}, Via: "vpn1"},
	}, nil)

	rules := e.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	// Mutating the returned slice ([]Rule = values) must not affect the engine.
	original := rules[0].Via
	rules[0].Via = "mutated"
	if e.rules[0].Via != original {
		t.Error("Rules() returned a reference — engine state was mutated")
	}
}
