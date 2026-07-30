package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// fakeDaemonPort/Host extracts the bind/port the daemon_client functions
// need from an httptest.Server URL, mirroring TestStopDaemon_PrefersGracefulShutdown's
// setup in commands_test.go.
func fakeDaemonBase(t *testing.T, ts *httptest.Server) (host string, port int) {
	t.Helper()
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	h, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatal(err)
	}
	p, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	return h, p
}

func withTestCfg(t *testing.T, host string, port int) {
	t.Helper()
	prevCfg := cfg
	t.Cleanup(func() { cfg = prevCfg })
	cfg = &config.Config{}
	cfg.Monitor.Dashboard.Bind = host
	cfg.Monitor.Dashboard.Port = port
}

func TestDaemonReloadPolicy(t *testing.T) {
	var gotMethod, gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/policies/reload", func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "reloaded", "policies": 2})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	if err := daemonReloadPolicy(ts.URL); err != nil {
		t.Fatalf("daemonReloadPolicy: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/policies/reload" {
		t.Fatalf("got %s %s, want POST /api/v1/policies/reload", gotMethod, gotPath)
	}
}

func TestDaemonReloadPolicy_PropagatesError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/policies/reload", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"config: validation failed"}`))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	err := daemonReloadPolicy(ts.URL)
	if err == nil {
		t.Fatal("expected an error for a 400 response")
	}
}

func TestDaemonGroupNames(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"name": "work", "profiles": []string{"office"}},
			{"name": "home", "profiles": []string{"aws"}},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	names, err := daemonGroupNames(ts.URL)
	if err != nil {
		t.Fatalf("daemonGroupNames: %v", err)
	}
	if len(names) != 2 || names[0] != "work" || names[1] != "home" {
		t.Fatalf("names=%v, want [work home]", names)
	}
}

func TestDaemonReloadGroup_Restarting(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/groups/work/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "restarting", "group": "work",
			"restarted": []string{"office"}, "skipped": []string{},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	result, err := daemonReloadGroup(ts.URL, "work")
	if err != nil {
		t.Fatalf("daemonReloadGroup: %v", err)
	}
	if result.RestartRequired {
		t.Fatal("expected RestartRequired=false for a 202 response")
	}
	if len(result.Restarted) != 1 || result.Restarted[0] != "office" {
		t.Fatalf("restarted=%v, want [office]", result.Restarted)
	}
}

// TestDaemonReloadGroup_RestartRequired proves the CLI-side decode correctly
// distinguishes the 409 restart_required shape (a profile the daemon has no
// adapter for) from a hard failure — the caller needs to tell these apart to
// print a warning instead of an error and keep processing other groups.
func TestDaemonReloadGroup_RestartRequired(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/groups/work/reload", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "restart_required", "group": "work",
			"missing_profiles": []string{"newvpn"},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	result, err := daemonReloadGroup(ts.URL, "work")
	if err != nil {
		t.Fatalf("daemonReloadGroup: %v", err)
	}
	if !result.RestartRequired {
		t.Fatal("expected RestartRequired=true for a 409 restart_required response")
	}
	if len(result.MissingProfiles) != 1 || result.MissingProfiles[0] != "newvpn" {
		t.Fatalf("missing_profiles=%v, want [newvpn]", result.MissingProfiles)
	}
}

// TestReloadCmd_NoDaemonRunning proves `kongtrol reload` fails clearly
// instead of silently reloading a throwaway in-process copy when no daemon
// is reachable at the well-known dashboard address — reload's entire point
// is to update the *running* daemon's in-memory state, so there is no safe
// local fallback (unlike up/down, whose adapters mostly proxy straight to
// OS-level VPN client state).
func TestReloadCmd_NoDaemonRunning(t *testing.T) {
	// An address nothing listens on (port 0 bound then closed) simulates
	// "no daemon running" without depending on a specific unused port.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().(*net.TCPAddr)
	_ = l.Close()

	withTestCfg(t, "127.0.0.1", addr.Port)

	err = reloadCmd.RunE(reloadCmd, nil)
	if err == nil {
		t.Fatal("expected an error when no daemon is reachable")
	}
}

// TestReloadCmd_PolicyOnlySkipsGroups proves --policy stops after reloading
// the policy engine and never touches the groups endpoints.
func TestReloadCmd_PolicyOnlySkipsGroups(t *testing.T) {
	groupsHit := false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tunnels", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/v1/policies/reload", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "reloaded", "policies": 0})
	})
	mux.HandleFunc("/api/v1/groups", func(w http.ResponseWriter, r *http.Request) {
		groupsHit = true
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	host, port := fakeDaemonBase(t, ts)
	withTestCfg(t, host, port)

	prevGroup, prevPolicyOnly := reloadGroup, reloadPolicyOnly
	reloadGroup = ""
	reloadPolicyOnly = true
	defer func() { reloadGroup, reloadPolicyOnly = prevGroup, prevPolicyOnly }()

	out, err := captureStdout(t, func() error { return reloadCmd.RunE(reloadCmd, nil) })
	if err != nil {
		t.Fatalf("RunE: %v; output=%s", err, out)
	}
	if groupsHit {
		t.Fatal("--policy must not call the groups endpoint")
	}
}
