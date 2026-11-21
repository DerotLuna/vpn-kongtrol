package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

const editTestYAML = "vpns:\n  oficina:\n    type: wireguard\n    config: /etc/wg.conf\n    auth:\n      method: certificate\n  casa:\n    type: nordvpn\n"

// withEditConfig points the CLI at a fresh kongtrol.yaml (no daemon) and
// returns its path. cfg is what loadConfig would have loaded.
func withEditConfig(t *testing.T) string {
	t.Helper()
	// Hermetic: default config paths, api-token and preferences resolve under
	// a temp home, never the developer's real ~/.kongtrol.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	stubUp(t)
	t.Setenv("KONGTROL_LANG", "es")
	daemonProbeFn = func(string) bool { return false }
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	if err := os.WriteFile(path, []byte(editTestYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	oldPath, oldJSON, oldQuiet := activeCfgPath, outputJSON, outputQuiet
	oldPrompt := promptInteractiveFn
	t.Cleanup(func() {
		activeCfgPath, outputJSON, outputQuiet = oldPath, oldJSON, oldQuiet
		promptInteractiveFn = oldPrompt
	})
	activeCfgPath = path
	promptInteractiveFn = func() bool { return false }
	reloadEditCfg(t, path)
	return path
}

func reloadEditCfg(t *testing.T, path string) *config.Config {
	t.Helper()
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg = c
	return c
}

// runFlags sets flags on cmd for one run and resets them afterwards.
func runFlags(t *testing.T, cmd *cobra.Command, flags map[string]string, args ...string) (string, error) {
	t.Helper()
	for k, v := range flags {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
	defer cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	return captureStdout(t, func() error {
		if err := cmd.Args(cmd, args); err != nil {
			return err
		}
		return cmd.RunE(cmd, args)
	})
}

func TestPolicyAdd_OfflineWritesTheSameRuleAsTheDashboard(t *testing.T) {
	path := withEditConfig(t)

	out, err := runFlags(t, policyAddCmd, map[string]string{"via": "oficina"},
		"https://GitHub.com/x", "10.20.0.0/16", "app:C:\\Apps\\teams.exe")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "github.com por oficina") || !strings.Contains(out, "kongtrol policy show github.com") {
		t.Fatalf("output:\n%s", out)
	}
	c := reloadEditCfg(t, path)
	if len(c.Policies) != 1 {
		t.Fatalf("policies = %+v", c.Policies)
	}
	p := c.Policies[0]
	if p.Name != "github.com por oficina" || p.Via != "oficina" || p.OnTunnelDown != "" ||
		strings.Join(p.Match.Domains, ",") != "github.com,*.github.com" ||
		strings.Join(p.Match.IPRanges, ",") != "10.20.0.0/16" ||
		strings.Join(p.Match.Apps, ",") != `C:\Apps\teams.exe` {
		t.Fatalf("policy = %+v", p)
	}

	// Same first target again: the generated name is made unique;
	// --no-subdomains and --block-when-down; --json.
	outputJSON = true
	out, err = runFlags(t, policyAddCmd, map[string]string{"via": "oficina", "no-subdomains": "true", "block-when-down": "true"}, "github.com")
	outputJSON = false
	if err != nil {
		t.Fatal(err)
	}
	var res policyAddResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if res.Name != "github.com por oficina 2" || res.OnTunnelDown != "block" || len(res.Domains) != 1 || res.ViaDaemon {
		t.Fatalf("json result = %+v", res)
	}
	if got := reloadEditCfg(t, path).Policies[1]; got.OnTunnelDown != "block" || len(got.Match.Domains) != 1 {
		t.Fatalf("second policy = %+v", got)
	}
}

func TestTunnelDownCommandsPersistProfileAndExplicitOverride(t *testing.T) {
	path := withEditConfig(t)
	if _, err := runFlags(t, profileTunnelDownCmd, nil, "oficina", "connect"); err != nil {
		t.Fatal(err)
	}
	reloadEditCfg(t, path)
	if _, err := runFlags(t, policyAddCmd, map[string]string{"via": "oficina", "name": "intranet"}, "corp.example"); err != nil {
		t.Fatal(err)
	}
	if _, err := runFlags(t, policyTunnelDownCmd, nil, "intranet", "allow"); err != nil {
		t.Fatal(err)
	}
	c := reloadEditCfg(t, path)
	if c.VPNs["oficina"].OnTunnelDown != "connect" || c.Policies[0].OnTunnelDown != "allow" {
		t.Fatalf("profile = %q, policy override = %q", c.VPNs["oficina"].OnTunnelDown, c.Policies[0].OnTunnelDown)
	}
	if _, err := runFlags(t, policyTunnelDownCmd, nil, "intranet", "inherit"); err != nil {
		t.Fatal(err)
	}
	c = reloadEditCfg(t, path)
	if c.Policies[0].OnTunnelDown != "" || c.Policies[0].TunnelDownAction(c.VPNs["oficina"]) != "connect" {
		t.Fatalf("expected inherited connect: %+v", c.Policies[0])
	}
}

func TestPolicyAdd_ErrorsCarryWhatToDo(t *testing.T) {
	withEditConfig(t)
	cases := []struct {
		flags map[string]string
		args  []string
		want  string
	}{
		{map[string]string{}, []string{"github.com"}, "--via"},
		{map[string]string{"via": "nada"}, []string{"github.com"}, "casa, oficina"},
		{map[string]string{"via": "oficina"}, []string{"no es un host"}, "Ejemplos:"},
		{map[string]string{"via": "oficina", "name": "x"}, []string{"a.com"}, ""},
		// The name is taken now: the shared check answers, translated.
		{map[string]string{"via": "oficina", "name": "X"}, []string{"b.com"}, "Ya hay una política"},
	}
	for _, c := range cases {
		_, err := runFlags(t, policyAddCmd, c.flags, c.args...)
		if c.want == "" {
			if err != nil {
				t.Fatalf("%v: %v", c.args, err)
			}
			reloadEditCfg(t, activeCfgPath)
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "\n    ") {
			t.Fatalf("%v %v: %v", c.flags, c.args, err)
		}
	}
	if err := policyAddCmd.Args(policyAddCmd, nil); err == nil {
		t.Fatal("policy add with no target accepted")
	}
}

func TestPolicyRemove_AsksAndHonorsYes(t *testing.T) {
	path := withEditConfig(t)
	if _, err := runFlags(t, policyAddCmd, map[string]string{"via": "oficina", "name": "corp"}, "corp.example"); err != nil {
		t.Fatal(err)
	}
	reloadEditCfg(t, path)

	// No terminal and no --yes: nothing is removed, the line says how.
	out, err := runFlags(t, policyRemoveCmd, nil, "corp")
	if err != nil || !strings.Contains(out, "--yes") || len(reloadEditCfg(t, path).Policies) != 1 {
		t.Fatalf("without --yes: %v\n%s", err, out)
	}
	// A terminal that answers yes.
	promptInteractiveFn = func() bool { return true }
	promptReader = strings.NewReader("s\n")
	t.Cleanup(func() { promptReader = os.Stdin })
	if _, err := runFlags(t, policyRemoveCmd, nil, "CORP"); err != nil || len(reloadEditCfg(t, path).Policies) != 0 {
		t.Fatalf("answered yes: %v", err)
	}
	if _, err := runFlags(t, policyRemoveCmd, map[string]string{"yes": "true"}, "corp"); err == nil || !strings.Contains(err.Error(), "No existe la política") {
		t.Fatalf("missing policy: %v", err)
	}
}

func TestPolicyAddRemove_DaemonUsesAPI(t *testing.T) {
	withEditConfig(t)
	var got []string
	var body map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		got = append(got, "POST lang="+r.Header.Get("X-Kongtrol-Lang"))
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("DELETE /api/v1/policies/{name}", func(w http.ResponseWriter, r *http.Request) {
		got = append(got, "DELETE "+r.PathValue("name"))
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":"Ya hay una política llamada \"x\".","hint":"Usa otro nombre.","error_key":"api.err.policy_exists"}`)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	host, port := fakeDaemonBase(t, ts)
	cfg.Monitor.Dashboard.Bind, cfg.Monitor.Dashboard.Port = host, port
	daemonProbeFn = func(string) bool { return true }

	if _, err := runFlags(t, policyAddCmd, map[string]string{"via": "casa"}, "10.0.0.5"); err != nil {
		t.Fatal(err)
	}
	m, _ := body["match"].(map[string]any)
	if body["name"] != "10.0.0.5 por casa" || body["via"] != "casa" || m == nil || len(m["ip_ranges"].([]any)) != 1 {
		t.Fatalf("body = %v", body)
	}
	cfg.Policies = append(cfg.Policies, config.PolicyRule{Name: "a b", Via: "casa"})
	_, err := runFlags(t, policyRemoveCmd, map[string]string{"yes": "true"}, "a b")
	if err == nil || err.Error() != "Ya hay una política llamada \"x\".\n    Usa otro nombre." {
		t.Fatalf("remote error: %v", err)
	}
	if strings.Join(got, ",") != "POST lang=es,DELETE a b" {
		t.Fatalf("requests = %v", got)
	}
}
