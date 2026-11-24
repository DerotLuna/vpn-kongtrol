package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

// ADR-0108: profile priority sets, clears and shows the number offline,
// with the same check as kongtrol.yaml.
func TestProfilePriority_OfflineSetClearAndValidate(t *testing.T) {
	path := withEditConfig(t)

	out, err := runFlags(t, profilePriorityCmd, nil, "oficina", "20")
	if err != nil || reloadEditCfg(t, path).VPNs["oficina"].Priority != 20 {
		t.Fatalf("set: %v", err)
	}
	if !strings.Contains(out, "Prioridad 20") {
		t.Fatalf("set output = %q", out)
	}
	out, err = runFlags(t, profilePriorityCmd, nil, "oficina")
	if err != nil || !strings.Contains(out, "oficina: Prioridad 20") || !strings.Contains(out, "mismo rango de IP") {
		t.Fatalf("show: %v %q", err, out)
	}
	for _, clear := range []string{"none", "0", "NONE"} {
		if _, err := runFlags(t, profilePriorityCmd, nil, "oficina", "20"); err != nil {
			t.Fatal(err)
		}
		reloadEditCfg(t, path)
		out, err := runFlags(t, profilePriorityCmd, nil, "oficina", clear)
		if err != nil || reloadEditCfg(t, path).VPNs["oficina"].Priority != 0 || !strings.Contains(out, "por posición") {
			t.Fatalf("clear %q: %v %q", clear, err, out)
		}
	}
	for _, bad := range []string{"-1", "1001", "alta"} {
		_, err := runFlags(t, profilePriorityCmd, nil, "oficina", bad)
		if err == nil || !strings.Contains(err.Error(), "no es válida") || !strings.Contains(err.Error(), "de 1 a 1000") {
			t.Fatalf("bad %q: %v", bad, err)
		}
	}
	if _, err := runFlags(t, profilePriorityCmd, nil, "nadie", "5"); err == nil || !strings.Contains(err.Error(), "nadie") {
		t.Fatalf("unknown profile: %v", err)
	}
}

func TestProfilePriority_DaemonUsesAPIAndJSON(t *testing.T) {
	withEditConfig(t)
	var body map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/vpns/{name}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("name") != "casa" {
			t.Errorf("name = %s", r.PathValue("name"))
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusOK)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	host, port := fakeDaemonBase(t, ts)
	cfg.Monitor.Dashboard.Bind, cfg.Monitor.Dashboard.Port = host, port
	daemonProbeFn = func(string) bool { return true }
	outputJSON = true

	out, err := runFlags(t, profilePriorityCmd, nil, "casa", "none")
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := body["priority"].(float64); !ok || p != 0 || body["type"] != nil || body["on_tunnel_down"] != nil {
		t.Fatalf("body = %v", body)
	}
	var got profilePriorityJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil || got.Name != "casa" || got.Priority != 0 || !got.ByPosition {
		t.Fatalf("json = %q (%v)", out, err)
	}
}

func TestProfileList_ShowsOrder(t *testing.T) {
	path := withEditConfig(t)
	if _, err := runFlags(t, profilePriorityCmd, nil, "oficina", "7"); err != nil {
		t.Fatal(err)
	}
	reloadEditCfg(t, path)
	out, err := runFlags(t, profileListCmd, nil)
	if err != nil || !strings.Contains(out, "Prioridad 7") || !strings.Contains(out, "por posición") || !strings.Contains(out, "ORDEN") {
		t.Fatalf("list: %v %q", err, out)
	}
	outputJSON = true
	out, err = runFlags(t, profileListCmd, nil)
	var got struct {
		Profiles []profilePriorityJSON `json:"profiles"`
	}
	if err != nil || json.Unmarshal([]byte(out), &got) != nil || len(got.Profiles) != 2 ||
		got.Profiles[0].Name != "casa" || !got.Profiles[0].ByPosition || got.Profiles[1].Priority != 7 {
		t.Fatalf("json: %v %q", err, out)
	}
}

func TestPolicyList_ShowsProfilePriority(t *testing.T) {
	prevEngine, prevCfg, prevJSON := engine, cfg, outputJSON
	t.Cleanup(func() { engine, cfg, outputJSON = prevEngine, prevCfg, prevJSON })
	t.Setenv("KONGTROL_LANG", "es")
	cfg = &config.Config{
		VPNs: map[string]config.VPNConfig{"oficina": {Type: "openvpn", Priority: 30}, "casa": {Type: "nordvpn"}},
		Policies: []config.PolicyRule{
			{Name: "intranet", Via: "oficina", Match: config.MatchSpec{Domains: []string{"intra.example.com"}}},
			{Name: "streaming", Via: "casa", Match: config.MatchSpec{Domains: []string{"tv.example.com"}}},
		},
	}
	e, err := policy.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	engine = e
	outputJSON = false
	out, err := runFlags(t, policyListCmd, nil)
	if err != nil || !strings.Contains(out, "Prioridad 30") || !strings.Contains(out, "por posición") || !strings.Contains(out, "kongtrol profile priority") {
		t.Fatalf("text: %v %q", err, out)
	}
	outputJSON = true
	out, err = runFlags(t, policyListCmd, nil)
	var rep mapReportJSON
	if err != nil || json.Unmarshal([]byte(out), &rep) != nil || len(rep.Rules) != 2 {
		t.Fatalf("json: %v %q", err, out)
	}
	for _, r := range rep.Rules {
		want := map[string]int{"oficina": 30, "casa": 0}[r.Via]
		if r.Priority != want {
			t.Fatalf("%s priority = %d", r.Name, r.Priority)
		}
	}
}
