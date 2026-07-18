package api

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestHandleShutdown_InvokesCallback proves POST /api/v1/shutdown responds
// promptly and then triggers the daemon's own shutdown callback — this is
// what lets `kongtrol down` ask a daemon to exit gracefully (running its
// deferred cleanup) from a separate process, cross-platform, instead of
// relying on an OS-level kill/signal.
func TestHandleShutdown_InvokesCallback(t *testing.T) {
	var called atomic.Bool
	srv := &Server{onShutdown: func() { called.Store(true) }}

	ts := httptest.NewServer(http.HandlerFunc(srv.handleShutdown))
	defer ts.Close()

	resp, err := http.Post(ts.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if called.Load() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("onShutdown was never called")
}

// TestHandleShutdown_NotConfigured proves a server with no shutdown
// callback wired (e.g. the tray, or a viewer-only instance) reports 501
// instead of silently doing nothing or panicking on a nil call.
func TestHandleShutdown_NotConfigured(t *testing.T) {
	srv := &Server{}
	ts := httptest.NewServer(http.HandlerFunc(srv.handleShutdown))
	defer ts.Close()

	resp, err := http.Post(ts.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusNotImplemented)
	}
}
