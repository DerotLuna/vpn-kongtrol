package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

// ADR-0113: program (app:) rules are recognised, never routed yet; every
// CLI surface that names one says so.
const appNotApplied = "Programas: solo salen por la VPN en Linux"

func TestPolicyAdd_AppTargetWarnsItIsNotAppliedYet(t *testing.T) {
	withEditConfig(t)
	out, err := runFlags(t, policyAddCmd, map[string]string{"via": "oficina"}, "app:claude.exe")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, appNotApplied) {
		t.Fatalf("no app warning:\n%s", out)
	}

	out, err = runFlags(t, policyAddCmd, map[string]string{"via": "oficina"}, "github.com")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, appNotApplied) {
		t.Fatalf("warning without an app target:\n%s", out)
	}
}

func TestPolicyListShowTest_MarkAppRules(t *testing.T) {
	prevEngine, prevCfg, prevJSON := engine, cfg, outputJSON
	t.Cleanup(func() { engine, cfg, outputJSON = prevEngine, prevCfg, prevJSON })
	t.Setenv("KONGTROL_LANG", "es")
	cfg = &config.Config{
		VPNs: map[string]config.VPNConfig{"brainbox": {Type: "openvpn"}},
		Policies: []config.PolicyRule{
			{Name: "ia", Via: "brainbox", Match: config.MatchSpec{Apps: []string{"claude.exe"}}},
			{Name: "web", Via: "brainbox", Match: config.MatchSpec{Domains: []string{"claude.ai"}}},
		},
	}
	e, err := policy.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	engine = e
	outputJSON = false

	out, err := runFlags(t, policyListCmd, nil)
	if err != nil || !strings.Contains(out, "app: aún no se aplica") || !strings.Contains(out, appNotApplied) {
		t.Fatalf("list: %v %q", err, out)
	}

	out, err = runFlags(t, policyShowCmd, nil, "app:claude.exe")
	if err != nil || !strings.Contains(out, "(aún no se aplica)") || !strings.Contains(out, appNotApplied) {
		t.Fatalf("show app: %v %q", err, out)
	}
	out, err = runFlags(t, policyShowCmd, nil, "claude.ai")
	if err != nil || strings.Contains(out, appNotApplied) || strings.Contains(out, "aún no se aplica") {
		t.Fatalf("show site: %v %q", err, out)
	}

	list := filepath.Join(t.TempDir(), "targets.txt")
	if err := os.WriteFile(list, []byte("app:claude.exe\nclaude.ai\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err = runFlags(t, policyTestCmd, nil, list)
	if err != nil || strings.Count(out, appNotApplied) != 1 {
		t.Fatalf("test: %v %q", err, out)
	}
}
