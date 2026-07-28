package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ── PID file helpers ──────────────────────────────────────────────────────────

// pidFilePath returns ~/.kongtrol/run/kongtrol.pid.
func pidFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kongtrol", "run", "kongtrol.pid")
}

// writePIDFile records the current process PID so that a concurrent
// `kongtrol down` invocation can stop this daemon.
func writePIDFile() {
	path := pidFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	_ = os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0600)
}

// removePIDFile removes the PID file only if it still holds our own PID
// (guards against a race where a new `kongtrol up` has already replaced it).
func removePIDFile() {
	path := pidFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && pid == os.Getpid() {
		_ = os.Remove(path)
	}
}

// stopDaemon reads the PID file and terminates the running `kongtrol up`
// daemon so it does not attempt to reconnect profiles that were just brought
// down.
//
// It prefers asking the daemon to shut down gracefully through its own API
// (POST /api/v1/shutdown) — that triggers the exact same cancellation path
// as Ctrl+C/SIGTERM, so kill-switch teardown, DNS restore, and history
// flush all run before the process exits. A bare kill skips every one of
// those, on every OS: Windows has no real cross-process SIGTERM equivalent
// (os.Process.Signal there only reliably implements Kill), so this was
// previously silent-but-unclean everywhere, not just on Windows.
// Falls back to a hard kill only if the API is unreachable (dashboard
// disabled) or the daemon doesn't exit within the grace period (wedged).
func stopDaemon() {
	path := pidFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return // no daemon running
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid == os.Getpid() {
		return // stale or own PID
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.warn.find", pid, err))
		return
	}

	base := daemonAPIBase()
	if probeDaemonAPI(base) {
		if err := daemonShutdown(base); err == nil {
			if waitForPIDFileGone(path, 10*time.Second) {
				fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.stopped", pid))
				return
			}
			fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.warn.graceful_timeout", pid))
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.warn.graceful_failed", pid, err))
		}
	}

	// Fall back to a hard kill (API unreachable or the daemon never
	// finished shutting down within the grace period).
	if err := proc.Kill(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.warn.stop", pid, err))
		return
	}
	_ = os.Remove(path)
	fmt.Fprintf(os.Stderr, "%s\n", cf("cli.daemon.stopped", pid))
}

// waitForPIDFileGone polls for the PID file to disappear, which the daemon
// does itself (removePIDFile) only after all of its deferred cleanup has
// already run — so its removal is a reliable, OS-agnostic signal that a
// graceful shutdown actually completed, without needing a platform-specific
// "is this PID still alive" check.
func waitForPIDFileGone(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return true
		}
		time.Sleep(150 * time.Millisecond)
	}
	return false
}
