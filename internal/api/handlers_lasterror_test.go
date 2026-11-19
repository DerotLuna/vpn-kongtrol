package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// newLastErrorServer wires a server whose async connect returns connectErr.
func newLastErrorServer(t *testing.T, connectErr func() error) (*Server, chan struct{}) {
	t.Helper()
	ad := newWSTestAdapter(vpn.StatusDisconnected)
	col := monitor.NewCollector(vpn.NewSet(map[string]vpn.VPNAdapter{"office": ad}))
	col.Start(time.Hour) // primes one snapshot; no further polls during the test
	t.Cleanup(col.Stop)
	done := make(chan struct{}, 4)
	s := &Server{
		adapters:      vpn.NewSet(map[string]vpn.VPNAdapter{"office": ad}),
		collector:     col,
		connectCancel: make(map[string]context.CancelFunc),
		connectProfile: func(context.Context, string) error {
			defer func() { done <- struct{}{} }()
			return connectErr()
		},
	}
	return s, done
}

func postConnect(s *Server) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnels/office/connect", nil)
	req.SetPathValue("name", "office")
	rec := httptest.NewRecorder()
	s.handleConnect(rec, req)
	return rec
}

func listTunnels(t *testing.T, s *Server) map[string]map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handleListTunnels(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tunnels", nil))
	var raw []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode tunnels: %v", err)
	}
	out := map[string]map[string]any{}
	for _, m := range raw {
		out[m["Name"].(string)] = m
	}
	return out
}

// waitPendingCleared waits until the async connect goroutine has finished
// recording its outcome (it clears the pending entry after recording).
func waitPendingCleared(t *testing.T, s *Server, done chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("connect was not invoked")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.connectMu.Lock()
		_, pending := s.connectCancel["office"]
		s.connectMu.Unlock()
		if !pending {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("pending connect never cleared")
}

func TestHandleConnect_FailureExposedAsLastError(t *testing.T) {
	fail := true
	s, done := newLastErrorServer(t, func() error {
		if fail {
			return errors.New("AUTH_FAILED: bad credentials")
		}
		return nil
	})

	if rec := postConnect(s); rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d, want 202", rec.Code)
	}
	waitPendingCleared(t, s, done)

	got := listTunnels(t, s)["office"]
	if got["last_error"] != "AUTH_FAILED: bad credentials" {
		t.Fatalf("last_error=%v", got["last_error"])
	}
	if at, _ := got["last_error_at"].(string); at == "" {
		t.Fatal("last_error_at missing")
	}
	// The WebSocket feed sends collector.Snapshot(), the same data.
	if snap := s.collector.Snapshot()["office"]; snap.LastError == "" {
		t.Fatal("snapshot must carry the last error")
	}

	// A new attempt clears the old error as soon as it starts, and a
	// successful one leaves it cleared.
	fail = false
	if rec := postConnect(s); rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d, want 202", rec.Code)
	}
	waitPendingCleared(t, s, done)
	got = listTunnels(t, s)["office"]
	if _, ok := got["last_error"]; ok {
		t.Fatalf("last_error must be cleared, got %v", got["last_error"])
	}
	if _, ok := got["last_error_at"]; ok {
		t.Fatal("last_error_at must be omitted when there is no error")
	}
}

func TestHandleConnect_CancelledConnectIsNotAnError(t *testing.T) {
	s, done := newLastErrorServer(t, func() error { return context.Canceled })
	postConnect(s)
	waitPendingCleared(t, s, done)
	if e := s.collector.Snapshot()["office"].LastError; e != "" {
		t.Fatalf("cancelled connect recorded as error %q", e)
	}
}
