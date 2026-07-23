package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func TestHandleConnect_UsesSharedLifecycle(t *testing.T) {
	called := make(chan string, 1)
	s := &Server{
		adapters: map[string]vpn.VPNAdapter{
			"office": newWSTestAdapter(vpn.StatusDisconnected),
		},
		connectCancel: make(map[string]context.CancelFunc),
		connectProfile: func(_ context.Context, name string) error {
			called <- name
			return nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnels/office/connect", nil)
	req.SetPathValue("name", "office")
	rec := httptest.NewRecorder()

	s.handleConnect(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusAccepted)
	}
	select {
	case name := <-called:
		if name != "office" {
			t.Fatalf("profile=%q, want office", name)
		}
	case <-time.After(time.Second):
		t.Fatal("shared connect lifecycle was not invoked")
	}
}

func TestHandleDisconnect_UsesSharedLifecycle(t *testing.T) {
	called := ""
	s := &Server{
		adapters: map[string]vpn.VPNAdapter{
			"office": newWSTestAdapter(vpn.StatusConnected),
		},
		connectCancel: make(map[string]context.CancelFunc),
		disconnectProfile: func(_ context.Context, name string) error {
			called = name
			return nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnels/office/disconnect", nil)
	req.SetPathValue("name", "office")
	rec := httptest.NewRecorder()

	s.handleDisconnect(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusOK)
	}
	if called != "office" {
		t.Fatalf("profile=%q, want office", called)
	}
}
