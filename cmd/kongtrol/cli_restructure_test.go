package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Old names keep working, hidden from help/docs, as deprecated aliases.
func TestCLIRestructure_DeprecatedAliasesHidden(t *testing.T) {
	for path, replacement := range map[string]string{
		"map":                       "kongtrol policy list · kongtrol policy show",
		"check":                     "kongtrol ip",
		"routes list":               "kongtrol routes --system",
		"policy explain":            "kongtrol policy show",
		"config defaults show":      "kongtrol config default-group",
		"config defaults set-group": "kongtrol config default-group",
	} {
		c, _, err := rootCmd.Find(strings.Fields(path))
		if err != nil || c.CommandPath() != "kongtrol "+path {
			t.Fatalf("%s: not found (%v)", path, err)
		}
		if !c.Hidden || c.Annotations[annotationDeprecated] != replacement {
			t.Errorf("%s: hidden=%v replacement=%q, want hidden alias of %q", path, c.Hidden, c.Annotations[annotationDeprecated], replacement)
		}
	}
	if f := reloadCmd.Flags().Lookup("tunnel"); f == nil || !f.Hidden {
		t.Error("reload --tunnel must stay as a hidden flag")
	}
}

func TestCLIRestructure_NewCommandsVisible(t *testing.T) {
	for _, path := range []string{
		"ip", "policy list", "policy show", "config default-group",
		"config killswitch", "config dns-guard",
	} {
		c, _, err := rootCmd.Find(strings.Fields(path))
		if err != nil || c.CommandPath() != "kongtrol "+path || c.Hidden {
			t.Errorf("%s: want a visible command (%v)", path, err)
		}
	}
	if routesCmd.Flags().Lookup("system") == nil {
		t.Error("routes --system missing")
	}
}

func TestCheckAlias_WarnsOnStderrAndRuns(t *testing.T) {
	stubUp(t)
	oldLeak := leak
	t.Cleanup(func() { leak = oldLeak })
	leak = nil // lookup off: the command says how to turn it on
	var runErr error
	stderr := captureStderr(t, func() {
		_, runErr = captureStdout(t, func() error { return checkCmd.RunE(checkCmd, nil) })
	})
	if !strings.Contains(stderr, "kongtrol check") || !strings.Contains(stderr, "kongtrol ip") {
		t.Errorf("no deprecation line on stderr: %q", stderr)
	}
	if runErr == nil {
		t.Error("check must still run ip (disabled lookup → error)")
	}
}

func TestDefaultGroup_SetShowAndDeprecatedAlias(t *testing.T) {
	stubUp(t)
	if _, err := captureStdout(t, func() error { return runDefaultGroup(configDefaultGroupCmd, []string{"work"}) }); err != nil {
		t.Fatal(err)
	}
	out, _ := captureStdout(t, func() error { return runDefaultGroup(configDefaultGroupCmd, nil) })
	if !strings.Contains(out, "work") {
		t.Fatalf("show: %q", out)
	}
	_ = captureStderr(t, func() {
		_, _ = captureStdout(t, func() error {
			return configDefaultsSetGroupCmd.RunE(configDefaultsSetGroupCmd, []string{"home"})
		})
	})
	p, err := loadPreferences()
	if err != nil || p.DefaultGroup != "home" {
		t.Fatalf("alias did not save: %+v %v", p, err)
	}
}

func TestRoutesSystem_RejectsProfile(t *testing.T) {
	routesSystem = true
	t.Cleanup(func() { routesSystem = false })
	if err := routesCmd.RunE(routesCmd, []string{"office"}); err == nil {
		t.Fatal("--system with a profile must fail with a hint")
	}
}

// fakeReloadDaemon serves the endpoints reload uses and records hits.
func fakeReloadDaemon(t *testing.T, newProfile bool) (hits *[]string) {
	t.Helper()
	hits = &[]string{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tunnels", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	})
	mux.HandleFunc("/api/v1/policies/reload", func(w http.ResponseWriter, r *http.Request) {
		*hits = append(*hits, r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "reloaded"})
	})
	mux.HandleFunc("/api/v1/tunnels/{name}/reload", func(w http.ResponseWriter, r *http.Request) {
		*hits = append(*hits, r.URL.Path)
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "restarting", "restarted": []string{r.PathValue("name")}})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	host, port := fakeDaemonBase(t, ts)
	withTestCfg(t, host, port)
	cfg.VPNs = map[string]config.VPNConfig{"office": {Type: "wireguard"}}
	if newProfile {
		cfg.VPNs["added"] = config.VPNConfig{Type: "wireguard"}
	}
	oldStatuses, oldRestart, oldYes := daemonTunnelStatusesFn, restartDaemonFn, reloadYes
	t.Cleanup(func() { daemonTunnelStatusesFn, restartDaemonFn, reloadYes = oldStatuses, oldRestart, oldYes })
	daemonTunnelStatusesFn = func(string) (map[string]vpn.Status, error) {
		return map[string]vpn.Status{"office": vpn.StatusConnected}, nil
	}
	restartDaemonFn = func(string) error { *hits = append(*hits, "restart"); return nil }
	return hits
}

func hit(hits []string, s string) bool {
	for _, h := range hits {
		if h == s {
			return true
		}
	}
	return false
}

func TestReload_NoFlagsAppliesEverything(t *testing.T) {
	hits := fakeReloadDaemon(t, false)
	stubPrompt(t, false, "")
	if _, err := captureStdout(t, func() error { return reloadCmd.RunE(reloadCmd, nil) }); err != nil {
		t.Fatal(err)
	}
	if !hit(*hits, "/api/v1/policies/reload") || !hit(*hits, "/api/v1/tunnels/office/reload") || hit(*hits, "restart") {
		t.Fatalf("hits = %v, want policy + office reload, no restart", *hits)
	}
}

func TestReload_NewProfile_NonInteractivePrintsHint(t *testing.T) {
	hits := fakeReloadDaemon(t, true)
	stubPrompt(t, false, "")
	out, err := captureStdout(t, func() error { return reloadCmd.RunE(reloadCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	if hit(*hits, "restart") {
		t.Fatal("restarted without --yes or a terminal")
	}
	if !strings.Contains(out, "added") || !strings.Contains(out, ct("cli.reload.restart_hint")) {
		t.Errorf("want the new profile and the restart hint:\n%s", out)
	}
}

func TestReload_NewProfile_YesRestarts(t *testing.T) {
	hits := fakeReloadDaemon(t, true)
	stubPrompt(t, false, "")
	reloadYes = true
	if _, err := captureStdout(t, func() error { return reloadCmd.RunE(reloadCmd, nil) }); err != nil {
		t.Fatal(err)
	}
	if !hit(*hits, "restart") {
		t.Fatalf("hits = %v, want a restart with --yes", *hits)
	}
}

func TestReload_NewProfile_InteractiveDefaultYesRestarts(t *testing.T) {
	hits := fakeReloadDaemon(t, true)
	stubPrompt(t, true, "\n")
	if _, err := captureStdout(t, func() error { return reloadCmd.RunE(reloadCmd, nil) }); err != nil {
		t.Fatal(err)
	}
	if !hit(*hits, "restart") {
		t.Fatalf("hits = %v, want a restart on Enter", *hits)
	}
}

func TestReload_PositionalProfileAndDeprecatedTunnelFlag(t *testing.T) {
	hits := fakeReloadDaemon(t, false)
	stubPrompt(t, false, "")
	if _, err := captureStdout(t, func() error { return reloadCmd.RunE(reloadCmd, []string{"office"}) }); err != nil {
		t.Fatal(err)
	}
	if len(*hits) != 2 || (*hits)[1] != "/api/v1/tunnels/office/reload" {
		t.Fatalf("positional: hits = %v", *hits)
	}

	*hits = nil
	f := reloadCmd.Flags()
	t.Cleanup(func() { _ = f.Set("tunnel", ""); f.Lookup("tunnel").Changed = false; reloadTunnel = "" })
	if err := f.Set("tunnel", "office"); err != nil {
		t.Fatal(err)
	}
	var runErr error
	stderr := captureStderr(t, func() {
		_, runErr = captureStdout(t, func() error { return reloadCmd.RunE(reloadCmd, nil) })
	})
	if runErr != nil {
		t.Fatal(runErr)
	}
	if !strings.Contains(stderr, "--tunnel") || !hit(*hits, "/api/v1/tunnels/office/reload") {
		t.Fatalf("--tunnel: stderr=%q hits=%v", stderr, *hits)
	}
}

// config killswitch / dns-guard without a daemon edit kongtrol.yaml in
// place, keeping comments.
func TestConfigSecurityToggles_NoDaemonEditsFileKeepingComments(t *testing.T) {
	stubUp(t)
	daemonProbeFn = func(string) bool { return false }
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	src := "# mi config\nvpns:\n  office:\n    type: wireguard # túnel de la oficina\nsecurity:\n  kill_switch:\n    enabled: false\n"
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	oldPath := activeCfgPath
	t.Cleanup(func() { activeCfgPath = oldPath })
	activeCfgPath = path
	for _, c := range []struct {
		cmd  string
		args []string
	}{{"killswitch", []string{"on"}}, {"dns-guard", []string{"on"}}} {
		cmd, _, _ := rootCmd.Find([]string{"config", c.cmd})
		if _, err := captureStdout(t, func() error { return cmd.RunE(cmd, c.args) }); err != nil {
			t.Fatalf("%s: %v", c.cmd, err)
		}
	}
	data, _ := os.ReadFile(path)
	var got config.Config
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Security.KillSwitch.Enabled || !got.Security.DNSGuard.Enabled {
		t.Fatalf("not enabled:\n%s", data)
	}
	if !strings.Contains(string(data), "# mi config") || !strings.Contains(string(data), "# túnel de la oficina") {
		t.Fatalf("comments lost:\n%s", data)
	}
}

func TestConfigSecurityToggles_DaemonUsesAPI(t *testing.T) {
	var posted []string
	mux := http.NewServeMux()
	for _, p := range []string{"/api/v1/security/killswitch", "/api/v1/security/dnsguard"} {
		mux.HandleFunc("POST "+p, func(w http.ResponseWriter, r *http.Request) {
			var body struct{ Enabled bool }
			_ = json.NewDecoder(r.Body).Decode(&body)
			posted = append(posted, r.URL.Path+"="+map[bool]string{true: "on", false: "off"}[body.Enabled])
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "updated"})
		})
	}
	mux.HandleFunc("GET /api/v1/security/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"kill_switch_enabled": true, "dns_guard_enabled": true, "dns_guard_state": "idle"})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	stubUp(t)
	host, port := fakeDaemonBase(t, ts)
	cfg.Monitor.Dashboard.Bind, cfg.Monitor.Dashboard.Port = host, port
	daemonProbeFn = func(string) bool { return true }

	if _, err := captureStdout(t, func() error { return configKillSwitchCmd.RunE(configKillSwitchCmd, []string{"off"}) }); err != nil {
		t.Fatal(err)
	}
	if _, err := captureStdout(t, func() error { return configDNSGuardCmd.RunE(configDNSGuardCmd, []string{"on"}) }); err != nil {
		t.Fatal(err)
	}
	if strings.Join(posted, ",") != "/api/v1/security/killswitch=off,/api/v1/security/dnsguard=on" {
		t.Fatalf("posted = %v", posted)
	}
	out, err := captureStdout(t, func() error { return configDNSGuardCmd.RunE(configDNSGuardCmd, []string{"status"}) })
	if err != nil || !strings.Contains(out, ct("cli.dnsguard.state.idle")) {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if err := configDNSGuardCmd.RunE(configDNSGuardCmd, []string{"maybe"}); err == nil {
		t.Fatal("invalid argument must fail with a hint")
	}
}
