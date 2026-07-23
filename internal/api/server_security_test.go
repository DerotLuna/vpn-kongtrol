package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecureHandler_RequiresTokenAndSameOrigin(t *testing.T) {
	s := &Server{bind: "127.0.0.1", port: 9741, apiToken: "test-control-token"}
	handler := s.secureHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name       string
		token      string
		origin     string
		wantStatus int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "hostile origin", token: s.apiToken, origin: "https://evil.example", wantStatus: http.StatusForbidden},
		{name: "same origin", token: s.apiToken, origin: "http://127.0.0.1:9741", wantStatus: http.StatusNoContent},
		{name: "native client", token: s.apiToken, wantStatus: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9741/api/v1/tunnels", nil)
			req.Host = "127.0.0.1:9741"
			if tt.token != "" {
				req.Header.Set("X-Kongtrol-Token", tt.token)
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status=%d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestSecureHandler_RejectsSimpleJSONCSRF(t *testing.T) {
	s := &Server{bind: "127.0.0.1", port: 9741, apiToken: "test-control-token"}
	handler := s.secureHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(
		http.MethodPost,
		"http://127.0.0.1:9741/api/v1/security/killswitch",
		strings.NewReader(`{"enabled":false}`),
	)
	req.Host = "127.0.0.1:9741"
	req.Header.Set("X-Kongtrol-Token", s.apiToken)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusUnsupportedMediaType)
	}
}
