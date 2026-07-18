package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func TestWaitForPIDFileGone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kongtrol.pid")
	if err := os.WriteFile(path, []byte("123"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Still present: must time out, not report success.
	if waitForPIDFileGone(path, 200*time.Millisecond) {
		t.Fatal("expected false while the PID file still exists")
	}

	// Removed shortly after starting to wait: must be detected before the
	// (much longer) deadline.
	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = os.Remove(path)
	}()
	start := time.Now()
	if !waitForPIDFileGone(path, 5*time.Second) {
		t.Fatal("expected true once the PID file was removed")
	}
	if elapsed := time.Since(start); elapsed >= 5*time.Second {
		t.Fatalf("waitForPIDFileGone took %v — expected it to detect removal promptly via polling, not wait out the full deadline", elapsed)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	out, _ := io.ReadAll(r)
	_ = r.Close()
	return string(out)
}

// startPlaceholderProcess starts a real, harmless long-running process and
// returns its PID. stopDaemon's PID lookup (os.FindProcess) behaves
// differently across OSes — on Unix it always succeeds regardless of
// whether the PID is real, but on Windows it actually opens the process
// and fails for a PID that doesn't exist — so a fake/made-up PID can't
// stand in for "a running daemon" portably; this needs a genuine process.
func startPlaceholderProcess(t *testing.T) int {
	t.Helper()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "ping -n 120 127.0.0.1 >nul")
	} else {
		cmd = exec.Command("sleep", "120")
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start placeholder process: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	return cmd.Process.Pid
}

// TestStopDaemon_PrefersGracefulShutdown is a real end-to-end check of the
// path `kongtrol down` takes to stop a running `up` daemon: a fake daemon
// HTTP server stands in for the real one (implementing /api/v1/tunnels for
// probeDaemonAPI and /api/v1/shutdown), and removes the PID file itself on
// receiving the shutdown request — exactly like the real daemon's own
// removePIDFile(), which only runs after its other cleanup defers. The PID
// file points at a real (but otherwise irrelevant) placeholder process, so
// if this test passes without a "cannot stop background daemon process"
// warning, stopDaemon took the graceful path and returned before ever
// falling back to Kill() on that process.
//
// pidFilePath() derives its path from the user's home directory, so this
// redirects HOME/USERPROFILE to an isolated temp dir for the duration of
// the test — it must never read or touch the real ~/.kongtrol/run PID file,
// which could belong to an actual running daemon on the machine running
// this test.
func TestStopDaemon_PrefersGracefulShutdown(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	realPID := startPlaceholderProcess(t)

	pidPath := pidFilePath()
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(realPID)), 0o600); err != nil {
		t.Fatal(err)
	}

	shutdownCalled := make(chan struct{}, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/tunnels", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/v1/shutdown", func(w http.ResponseWriter, r *http.Request) {
		_ = os.Remove(pidPath) // mirrors the real daemon's removePIDFile-after-cleanup
		w.WriteHeader(http.StatusAccepted)
		shutdownCalled <- struct{}{}
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	prevCfg := cfg
	defer func() { cfg = prevCfg }()
	cfg = &config.Config{}
	cfg.Monitor.Dashboard.Bind = host
	cfg.Monitor.Dashboard.Port = port

	out := captureStderr(t, stopDaemon)

	select {
	case <-shutdownCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("daemon's /api/v1/shutdown was never called")
	}

	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatalf("expected PID file to be gone after graceful shutdown, stat err=%v", err)
	}
	if strings.Contains(out, "cannot stop background daemon process") {
		t.Fatalf("stopDaemon fell back to a hard kill instead of taking the graceful path; stderr=%s", out)
	}
}
