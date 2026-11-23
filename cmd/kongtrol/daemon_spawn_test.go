package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// stubUp isolates runUp/ensureDaemon from real processes, terminals and
// the user's ~/.kongtrol (HOME points at a temp dir).
func stubUp(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	oldCfg := cfg
	oldProbe, oldSpawn, oldConn, oldCancel, oldStatuses := daemonProbeFn, spawnDaemonFn, daemonConnectFn, daemonCancelConnectFn, daemonTunnelStatesFn
	oldViewer, oldTerm, oldPoll, oldGrace, oldTick, oldWait := attachViewerFn, upIsTerminalFn, upPollInterval, upNoProgressGrace, daemonProbeTick, daemonStartWaitFn
	t.Cleanup(func() {
		cfg = oldCfg
		daemonProbeFn, spawnDaemonFn, daemonConnectFn, daemonCancelConnectFn, daemonTunnelStatesFn = oldProbe, oldSpawn, oldConn, oldCancel, oldStatuses
		attachViewerFn, upIsTerminalFn, upPollInterval, upNoProgressGrace, daemonProbeTick, daemonStartWaitFn = oldViewer, oldTerm, oldPoll, oldGrace, oldTick, oldWait
	})
	cfg = &config.Config{}
	cfg.Monitor.Dashboard.Bind = "127.0.0.1"
	cfg.Monitor.Dashboard.Port = 1
	upIsTerminalFn = func() bool { return false }
	attachViewerFn = func(context.Context, context.CancelFunc, string) { t.Fatal("viewer must not open on a non-terminal") }
	upPollInterval = 5 * time.Millisecond
	upNoProgressGrace = 50 * time.Millisecond
	daemonProbeTick = 5 * time.Millisecond
	daemonStartWaitFn = 300 * time.Millisecond
	daemonCancelConnectFn = func(string, string) error { return nil }
}

func TestEnsureDaemon_RunningDaemonIsReused(t *testing.T) {
	stubUp(t)
	daemonProbeFn = func(string) bool { return true }
	spawnDaemonFn = func(string) (<-chan error, error) { t.Fatal("must not spawn when a daemon answers"); return nil, nil }
	started, err := ensureDaemon(context.Background(), "http://x")
	if err != nil || started {
		t.Fatalf("started=%v err=%v, want reuse", started, err)
	}
}

func TestEnsureDaemon_SpawnsAndWaitsForAPI(t *testing.T) {
	stubUp(t)
	var up atomic.Bool
	var spawns atomic.Int32
	daemonProbeFn = func(string) bool { return up.Load() }
	spawnDaemonFn = func(string) (<-chan error, error) {
		spawns.Add(1)
		go func() { time.Sleep(30 * time.Millisecond); up.Store(true) }()
		return make(chan error), nil
	}
	started, err := ensureDaemon(context.Background(), "http://x")
	if err != nil || !started || spawns.Load() != 1 {
		t.Fatalf("started=%v err=%v spawns=%d", started, err, spawns.Load())
	}
	if _, err := os.Stat(spawnLockPath()); !os.IsNotExist(err) {
		t.Fatal("spawn lock must be released after startup")
	}
}

func TestEnsureDaemon_DiedDuringStartupShowsLogTail(t *testing.T) {
	stubUp(t)
	daemonProbeFn = func(string) bool { return false }
	spawnDaemonFn = func(logPath string) (<-chan error, error) {
		_ = os.MkdirAll(filepath.Dir(logPath), 0o700)
		_ = os.WriteFile(logPath, []byte("boom: port in use\n"), 0o600)
		ch := make(chan error, 1)
		ch <- errors.New("exit status 1")
		return ch, nil
	}
	_, err := ensureDaemon(context.Background(), "http://x")
	var se *errDaemonStart
	if !errors.As(err, &se) {
		t.Fatalf("want errDaemonStart, got %v", err)
	}
	if len(se.LogTail) != 1 || !strings.Contains(se.LogTail[0], "port in use") {
		t.Fatalf("log tail = %v", se.LogTail)
	}
}

func TestEnsureDaemon_SecondUpWaitsForFirstSpawner(t *testing.T) {
	stubUp(t)
	// Simulate a racing `up` that holds a fresh spawn lock.
	_ = os.MkdirAll(filepath.Dir(spawnLockPath()), 0o700)
	if err := os.WriteFile(spawnLockPath(), []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	var probes atomic.Int32
	daemonProbeFn = func(string) bool { return probes.Add(1) > 3 }
	spawnDaemonFn = func(string) (<-chan error, error) { t.Fatal("second up must not spawn"); return nil, nil }
	started, err := ensureDaemon(context.Background(), "http://x")
	if err != nil || started {
		t.Fatalf("started=%v err=%v, want to attach to the other spawner's daemon", started, err)
	}
}

func TestAcquireSpawnLock_TakesOverStaleLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spawn.lock")
	_ = os.WriteFile(path, []byte("1"), 0o600)
	old := time.Now().Add(-2 * spawnLockStale)
	_ = os.Chtimes(path, old, old)
	release, owner := acquireSpawnLock(path)
	if !owner {
		t.Fatal("stale lock must be taken over")
	}
	release()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("release must remove the lock")
	}
}

func TestRunUp_ConnectsThroughDaemonAndFollowsState(t *testing.T) {
	stubUp(t)
	daemonProbeFn = func(string) bool { return true }
	var connected []string
	daemonConnectFn = func(_ string, name string) (bool, error) {
		connected = append(connected, name)
		return name == "b", nil // b already connected
	}
	daemonTunnelStatesFn = stubStates(map[string]vpn.Status{"a": vpn.StatusConnected, "b": vpn.StatusConnected})
	if err := runUp(context.Background(), []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(connected, []string{"a", "b"}) {
		t.Fatalf("connected %v", connected)
	}
}

func TestConnectViaDaemon_Failures(t *testing.T) {
	stubUp(t)
	daemonConnectFn = func(string, string) (bool, error) { return false, nil }

	daemonTunnelStatesFn = stubStates(map[string]vpn.Status{"a": vpn.StatusError})
	if _, err := connectViaDaemon(context.Background(), "http://x", "a"); err == nil {
		t.Fatal("error status must fail the connect")
	}

	// Never leaves "disconnected": fails after the no-progress grace.
	daemonTunnelStatesFn = stubStates(map[string]vpn.Status{"a": vpn.StatusDisconnected})
	if _, err := connectViaDaemon(context.Background(), "http://x", "a"); err == nil {
		t.Fatal("no progress must fail the connect")
	}

	// Cancel (Ctrl+C) aborts the daemon-side connect.
	var cancelled atomic.Bool
	daemonCancelConnectFn = func(string, string) error { cancelled.Store(true); return nil }
	daemonTunnelStatesFn = stubStates(map[string]vpn.Status{"a": vpn.StatusConnecting})
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	if _, err := connectViaDaemon(ctx, "http://x", "a"); !errors.Is(err, context.Canceled) || !cancelled.Load() {
		t.Fatalf("err=%v cancelled=%v", err, cancelled.Load())
	}
}

func TestRunUp_AttachesViewerOnTerminal(t *testing.T) {
	stubUp(t)
	daemonProbeFn = func(string) bool { return true }
	upIsTerminalFn = func() bool { return true }
	var base string
	attachViewerFn = func(_ context.Context, _ context.CancelFunc, b string) { base = b }
	if err := runUp(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if base != "http://127.0.0.1:1" {
		t.Fatalf("viewer attached to %q", base)
	}
}

func TestDaemonAPIBase_WildcardBindDialsLoopback(t *testing.T) {
	old := cfg
	t.Cleanup(func() { cfg = old })
	cfg = &config.Config{}
	cfg.Monitor.Dashboard.Bind = "0.0.0.0"
	cfg.Monitor.Dashboard.Port = 9741
	if got := daemonAPIBase(); got != "http://127.0.0.1:9741" {
		t.Fatalf("got %s", got)
	}
}

func TestDaemonAPIBaseWithoutLoadedConfig(t *testing.T) {
	oldCfg, oldPath := cfg, cfgPath
	t.Cleanup(func() { cfg, cfgPath = oldCfg, oldPath })
	cfg = nil
	cfgPath = filepath.Join(t.TempDir(), "missing.yaml")
	defaults := config.NewEmpty()
	config.ApplyDashboardPreferences(defaults)
	dashboard := defaults.Monitor.Dashboard
	want := daemonctl.APIBase(dashboard.Bind, dashboard.Port)
	if got := daemonAPIBase(); got != want {
		t.Fatalf("missing config: got %s, want %s", got, want)
	}
	if err := os.WriteFile(cfgPath, []byte("vpns: ["), 0600); err != nil {
		t.Fatal(err)
	}
	if got := daemonAPIBase(); got != want {
		t.Fatalf("unreadable config: got %s, want %s", got, want)
	}
}

// stubStates serves fixed statuses as the daemon's tunnel states.
func stubStates(m map[string]vpn.Status) func(string) (map[string]monitor.TunnelMetrics, error) {
	return func(string) (map[string]monitor.TunnelMetrics, error) {
		out := make(map[string]monitor.TunnelMetrics, len(m))
		for name, st := range m {
			out[name] = monitor.TunnelMetrics{Name: name, Status: st}
		}
		return out, nil
	}
}

func TestUpConnectDecision(t *testing.T) {
	old := upNoProgressGrace
	t.Cleanup(func() { upNoProgressGrace = old })
	upNoProgressGrace = time.Minute
	failed := monitor.TunnelMetrics{Status: vpn.StatusDisconnected, LastError: "auth failed: bad password"}
	cases := []struct {
		name     string
		state    monitor.TunnelMetrics
		known    bool
		progress bool
		elapsed  time.Duration
		done     bool
		wantErr  string // substring; "" means no error
	}{
		{"unknown keeps waiting", monitor.TunnelMetrics{}, false, false, 0, false, ""},
		{"connected succeeds", monitor.TunnelMetrics{Status: vpn.StatusConnected}, true, false, 0, true, ""},
		{"connecting keeps waiting", monitor.TunnelMetrics{Status: vpn.StatusConnecting}, true, true, 0, false, ""},
		{"disconnected within grace waits", monitor.TunnelMetrics{Status: vpn.StatusDisconnected}, true, false, time.Second, false, ""},
		{"last error fails at once with its message", failed, true, false, time.Second, true, "bad password"},
		{"error status shows last error", monitor.TunnelMetrics{Status: vpn.StatusError, LastError: "tun busy"}, true, true, 0, true, "tun busy"},
		{"dropped after progress without message", monitor.TunnelMetrics{Status: vpn.StatusDisconnected}, true, true, 0, true, "logs --daemon"},
		{"no progress past grace", monitor.TunnelMetrics{Status: vpn.StatusDisconnected}, true, false, 2 * time.Minute, true, "logs --daemon"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			done, err := upConnectDecision(tc.state, tc.known, tc.progress, tc.elapsed)
			if done != tc.done {
				t.Fatalf("done=%v, want %v", done, tc.done)
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestConnectViaDaemon_ReportsDaemonLastError(t *testing.T) {
	stubUp(t)
	daemonConnectFn = func(string, string) (bool, error) { return false, nil }
	daemonTunnelStatesFn = func(string) (map[string]monitor.TunnelMetrics, error) {
		return map[string]monitor.TunnelMetrics{"a": {
			Name: "a", Status: vpn.StatusDisconnected,
			LastError: "openvpn: AUTH_FAILED", LastErrorAt: time.Now(),
		}}, nil
	}
	_, err := connectViaDaemon(context.Background(), "http://x", "a")
	if err == nil || !strings.Contains(err.Error(), "AUTH_FAILED") {
		t.Fatalf("err=%v, want the daemon's last error", err)
	}
}
