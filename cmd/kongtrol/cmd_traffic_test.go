package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// `kongtrol reload` after a live apply (ADR-0053): only connected profiles
// whose settings changed are reconnected, each one asked (--yes answers).
func TestReload_ReconnectsOnlyChangedProfiles(t *testing.T) {
	for _, yes := range []bool{true, false} {
		var mu sync.Mutex
		var hits []string
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/tunnels", func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode([]any{})
		})
		mux.HandleFunc("/api/v1/policies/reload", func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "reloaded", "restart_required": false, "reconnect_required": []string{"office"},
			})
		})
		mux.HandleFunc("/api/v1/tunnels/{name}/reload", func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			hits = append(hits, r.PathValue("name"))
			mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "restarting", "restarted": []string{r.PathValue("name")}})
		})
		ts := httptest.NewServer(mux)
		host, port := fakeDaemonBase(t, ts)
		withTestCfg(t, host, port)
		prevYes := reloadYes
		reloadYes = yes

		out, err := captureStdout(t, func() error { return reloadCmd.RunE(reloadCmd, nil) })
		reloadYes = prevYes
		ts.Close()
		if err != nil {
			t.Fatalf("yes=%v: %v; %s", yes, err, out)
		}
		mu.Lock()
		got := strings.Join(hits, ",")
		mu.Unlock()
		if yes && got != "office" {
			t.Fatalf("--yes must reconnect only office, got %q", got)
		}
		if !yes {
			// No terminal in tests: never asked, nothing restarted, hint shown.
			if got != "" || !strings.Contains(out, "kongtrol reload office") {
				t.Fatalf("without --yes: hits=%q out=%s", got, out)
			}
		}
	}
}

func fakeRecentDaemon(t *testing.T, body any) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tunnels", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	})
	mux.HandleFunc("/api/v1/connections/recent", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "2" {
			t.Errorf("limit not sent: %q", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(body)
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	host, port := fakeDaemonBase(t, ts)
	withTestCfg(t, host, port)
}

func TestTrafficRecent_TableAndJSON(t *testing.T) {
	now := time.Now()
	fakeRecentDaemon(t, map[string]any{
		"supported": true, "enabled": true, "max": 500, "ttl_seconds": 3600,
		"connections": []monitor.RecentConnection{
			{Host: "intranet.example", RemoteIP: "10.0.0.5", Port: 22, Via: "office", Interface: "wg0", FirstSeen: now, LastSeen: now, Count: 2},
			{RemoteIP: "93.184.216.34", Port: 443, Process: "chrome.exe", FirstSeen: now, LastSeen: now, Count: 1},
		},
	})
	prevLimit := recentLimit
	recentLimit = 2
	t.Cleanup(func() { recentLimit = prevLimit })

	out, err := captureStdout(t, func() error { return trafficRecentCmd.RunE(trafficRecentCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"intranet.example (10.0.0.5)", "office · wg0", "chrome.exe", ct("cli.recent.direct")} {
		if !strings.Contains(out, want) {
			t.Fatalf("table lacks %q:\n%s", want, out)
		}
	}

	outputJSON = true
	t.Cleanup(func() { outputJSON = false })
	out, err = captureStdout(t, func() error { return trafficRecentCmd.RunE(trafficRecentCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Supported   bool                       `json:"supported"`
		Connections []monitor.RecentConnection `json:"connections"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || !got.Supported || len(got.Connections) != 2 {
		t.Fatalf("json %v: %s", err, out)
	}
}

func TestTrafficRecent_OffAndUnsupported(t *testing.T) {
	prevLimit := recentLimit
	recentLimit = 2
	t.Cleanup(func() { recentLimit = prevLimit })

	fakeRecentDaemon(t, map[string]any{"supported": true, "enabled": false, "connections": []any{}})
	out, _ := captureStdout(t, func() error { return trafficRecentCmd.RunE(trafficRecentCmd, nil) })
	if !strings.Contains(out, "kongtrol config recent-connections on") {
		t.Fatalf("off must say how to turn it on: %s", out)
	}
	fakeRecentDaemon(t, map[string]any{"supported": false, "enabled": true, "connections": []any{}})
	out, _ = captureStdout(t, func() error { return trafficRecentCmd.RunE(trafficRecentCmd, nil) })
	if !strings.Contains(out, ct("cli.recent.unsupported")) {
		t.Fatalf("unsupported: %s", out)
	}
}

func TestTrafficRecent_ErrorsHaveHint(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	withTestCfg(t, "127.0.0.1", port)
	err = trafficRecentCmd.RunE(trafficRecentCmd, nil)
	if err == nil || !strings.Contains(err.Error(), ct("cli.recent.no_daemon_do")) {
		t.Fatalf("no daemon: %v", err)
	}
	prevLimit := recentLimit
	recentLimit = -1
	t.Cleanup(func() { recentLimit = prevLimit })
	err = trafficRecentCmd.RunE(trafficRecentCmd, nil)
	if err == nil || !strings.Contains(err.Error(), ct("api.err.recent_limit.fix")) {
		t.Fatalf("bad limit: %v", err)
	}
}

func TestSetRecentConnectionsInFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	src := "# my comment\nvpns:\n  office:\n    type: surfshark\n"
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := setRecentConnectionsInFile(path, false); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.RecentConnectionsEnabled() {
		t.Fatal("off not written")
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "# my comment") {
		t.Fatal("comments must survive")
	}
	if err := setRecentConnectionsInFile(path, true); err != nil {
		t.Fatal(err)
	}
	if c, _ := config.Load(path); !c.RecentConnectionsEnabled() {
		t.Fatal("on not written")
	}
}
