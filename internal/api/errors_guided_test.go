package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

func TestWriteGuidedError(t *testing.T) {
	// A typed helper denial becomes one translated line + its key.
	err := fmt.Errorf("wireguard: disconnect: %w", &privops.DeniedError{Reason: privops.DenyWGNotManaged, Detail: `tunnel "rluna" was not started by Kongtrol`})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/tunnels/brainbox/disconnect", nil)
	r.Header.Set("X-Kongtrol-Lang", "es")
	w := httptest.NewRecorder()
	writeGuidedError(w, r, http.StatusInternalServerError, err)
	var body map[string]string
	if e := json.Unmarshal(w.Body.Bytes(), &body); e != nil {
		t.Fatal(e)
	}
	if body["error_key"] != "privops.denied.wg_not_managed" || !strings.Contains(body["error"], "WireGuard") ||
		strings.Contains(body["error"], "privops:") || !strings.Contains(body["detail"], "rluna") {
		t.Fatalf("body %v", body)
	}
	// Anything else keeps the raw message.
	w = httptest.NewRecorder()
	writeGuidedError(w, r, http.StatusInternalServerError, errors.New("boom"))
	if !strings.Contains(w.Body.String(), "boom") || strings.Contains(w.Body.String(), "error_key") {
		t.Fatalf("plain body %s", w.Body.String())
	}
}
