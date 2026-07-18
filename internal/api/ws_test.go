package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// wsTestAdapter's status is set by the test body and read concurrently by
// the Collector's own polling goroutine — a plain field here would be a
// genuine data race (caught by `go test -race`), not just a lint nit.
type wsTestAdapter struct {
	status atomic.Value // vpn.Status
}

func newWSTestAdapter(status vpn.Status) *wsTestAdapter {
	a := &wsTestAdapter{}
	a.setStatus(status)
	return a
}

func (a *wsTestAdapter) setStatus(s vpn.Status) { a.status.Store(s) }

func (a *wsTestAdapter) Connect(context.Context, vpn.AdapterConfig) error { return nil }
func (a *wsTestAdapter) Disconnect(context.Context) error                 { return nil }
func (a *wsTestAdapter) Reconnect(context.Context) error                  { return nil }
func (a *wsTestAdapter) Name() string                                     { return "mock" }
func (a *wsTestAdapter) Version() string                                  { return "v0" }
func (a *wsTestAdapter) Capabilities() vpn.Capabilities                   { return vpn.Capabilities{} }
func (a *wsTestAdapter) Status() vpn.Status                               { return a.status.Load().(vpn.Status) }
func (a *wsTestAdapter) TunnelInfo() (*vpn.TunnelInfo, error)             { return nil, nil }

// TestHandleWebSocket_PushesOnChangeNotHeartbeat proves the dashboard feed
// delivers a real state change promptly via the collector's Subscribe
// channel, rather than waiting for the (much slower) heartbeat ticker.
func TestHandleWebSocket_PushesOnChangeNotHeartbeat(t *testing.T) {
	ad := newWSTestAdapter(vpn.StatusDisconnected)
	col := monitor.NewCollector(map[string]vpn.VPNAdapter{"office": ad})
	// Fast poll — this test only cares that a real state change is pushed
	// well under the server's 5s heartbeat, not the collector's own cadence.
	col.Start(30 * time.Millisecond)
	defer col.Stop()

	srv := &Server{collector: col}
	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Initial snapshot, sent immediately on connect.
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read initial snapshot: %v", err)
	}

	ad.setStatus(vpn.StatusConnected)

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	start := time.Now()
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read change notification: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed >= 5*time.Second {
		t.Fatalf("update took %v — expected it to arrive via the change channel, well under the 5s heartbeat", elapsed)
	}
}

// TestHandleWebSocket_PayloadShape locks in the wire contract remote
// consumers (status --watch's streamRemoteTunnels in cmd/kongtrol) depend
// on: a JSON object keyed by profile name, not an array — unlike the REST
// /api/v1/tunnels handler, which does convert to an array.
func TestHandleWebSocket_PayloadShape(t *testing.T) {
	ad := newWSTestAdapter(vpn.StatusConnected)
	col := monitor.NewCollector(map[string]vpn.VPNAdapter{"office": ad})
	col.Start(time.Hour)
	defer col.Stop()

	srv := &Server{collector: col}
	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var snapshot map[string]monitor.TunnelMetrics
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("payload is not a map[string]TunnelMetrics: %v\nraw: %s", err, data)
	}
	tm, ok := snapshot["office"]
	if !ok {
		t.Fatalf("expected key %q in snapshot, got %+v", "office", snapshot)
	}
	if tm.Status.Normalize() != vpn.StatusConnected {
		t.Fatalf("Status=%v, want connected", tm.Status)
	}
}
