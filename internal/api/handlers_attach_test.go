package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// attachTestAdapter is a provider adapter (attach mode) stub.
type attachTestAdapter struct {
	*wsTestAdapter
	state *vpn.AttachState
}

func (a attachTestAdapter) AttachState() *vpn.AttachState { return a.state }

// TestDisconnectManualIsAccepted: a provider app without a CLI answers 202
// waiting_user instead of a 500, so clients show "disconnect in <App>".
func TestDisconnectManualIsAccepted(t *testing.T) {
	s := &Server{
		adapters:      vpn.NewSet(map[string]vpn.VPNAdapter{"surf": newWSTestAdapter(vpn.StatusConnected)}),
		connectCancel: make(map[string]context.CancelFunc),
		disconnectProfile: func(context.Context, string) error {
			return fmt.Errorf("Surfshark: %w", vpn.ErrManualDisconnect)
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnels/surf/disconnect", nil)
	req.SetPathValue("name", "surf")
	rec := httptest.NewRecorder()
	s.handleDisconnect(rec, req)
	var body map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusAccepted || body["status"] != "waiting_user" {
		t.Fatalf("got %d %v, want 202 waiting_user", rec.Code, body)
	}
}

// TestTunnelsExposeAttach: /tunnels (and so the WebSocket snapshot, which
// serializes the same TunnelMetrics) carries the attach phase.
func TestTunnelsExposeAttach(t *testing.T) {
	ad := attachTestAdapter{newWSTestAdapter(vpn.StatusConnecting),
		&vpn.AttachState{Phase: vpn.PhaseWaitingUser, App: "Surfshark", TimeoutSeconds: 300}}
	col := monitor.NewCollector(vpn.NewSet(map[string]vpn.VPNAdapter{"surf": ad}))
	col.Start(time.Hour)
	t.Cleanup(col.Stop)
	s := &Server{adapters: vpn.NewSet(map[string]vpn.VPNAdapter{"surf": ad}), collector: col}
	attach, ok := listTunnels(t, s)["surf"]["attach"].(map[string]any)
	if !ok || attach["phase"] != "waiting_user" || attach["app"] != "Surfshark" {
		t.Fatalf("attach = %v", attach)
	}
}
