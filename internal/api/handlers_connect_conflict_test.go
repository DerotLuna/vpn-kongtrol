package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// A connect that would start a second full tunnel is refused at once with
// 409 and the translated reason + "what to do" (ADR-0099); the async
// connect never starts.
func TestHandleConnect_RefusedByConnectCheck(t *testing.T) {
	s, done := newLastErrorServer(t, func() error { return nil })
	s.SetConnectCheck(func(name string) error {
		return &config.ValidationError{Key: "api.err.connect_full_conflict_app", Args: []any{name, "Proton VPN"}}
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnels/office/connect", nil)
	req.SetPathValue("name", "office")
	req.Header.Set("X-Kongtrol-Lang", "es")
	rec := httptest.NewRecorder()
	s.handleConnect(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error_key"] != "api.err.connect_full_conflict_app" || body["hint"] != "Desconecta Proton VPN en su app y vuelve a conectar office." {
		t.Fatalf("body = %v", body)
	}
	select {
	case <-done:
		t.Fatal("connect ran despite the refusal")
	default:
	}
}
