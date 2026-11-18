package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/update"
)

func lifecycleMux(s *Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/updates", s.handleGetUpdates)
	mux.HandleFunc("POST /api/v1/updates", s.handleSetUpdates)
	mux.HandleFunc("POST /api/v1/data/purge", s.handlePurgeData)
	return s.secureHandler(mux)
}

func lifecycleReq(method, path, body, token string) *http.Request {
	req := httptest.NewRequest(method, "http://127.0.0.1:9741"+path, strings.NewReader(body))
	req.Host = "127.0.0.1:9741"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("X-Kongtrol-Token", token)
	}
	return req
}

func TestPurgeEndpoint_RequiresAuthAndConfirm(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	s.bind, s.port, s.apiToken = "127.0.0.1", 9741, "tok"
	calls := 0
	s.SetDataPurger(func() error { calls++; return nil })
	h := lifecycleMux(s)

	cases := []struct {
		name, token, body string
		want              int
	}{
		{"no token", "", `{"confirm":true}`, http.StatusUnauthorized},
		{"wrong token", "nope", `{"confirm":true}`, http.StatusUnauthorized},
		{"no confirm", "tok", `{}`, http.StatusBadRequest},
		{"confirm false", "tok", `{"confirm":false}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, lifecycleReq(http.MethodPost, "/api/v1/data/purge", tc.body, tc.token))
		if rec.Code != tc.want {
			t.Errorf("%s: status %d, want %d", tc.name, rec.Code, tc.want)
		}
	}
	// Cross-site form post (text/plain) is refused before the handler.
	req := lifecycleReq(http.MethodPost, "/api/v1/data/purge", `{"confirm":true}`, "tok")
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("text/plain: status %d", rec.Code)
	}
	if calls != 0 {
		t.Fatalf("purge ran %d times without auth+confirm", calls)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, lifecycleReq(http.MethodPost, "/api/v1/data/purge", `{"confirm":true}`, "tok"))
	if rec.Code != http.StatusOK || calls != 1 {
		t.Fatalf("authorized+confirmed purge: status %d calls %d", rec.Code, calls)
	}
}

func TestUpdatesEndpoint_ToggleSavesConfig(t *testing.T) {
	s, path := newReloadTestServer(t, baseTestYAML, nil)
	s.bind, s.port, s.apiToken = "127.0.0.1", 9741, "tok"
	var toggled []bool
	s.SetUpdateChecker(func() update.Result {
		return update.Result{Current: "v1.0.0", Latest: "v1.1.0", Newer: true}
	}, func(on bool) { toggled = append(toggled, on) })
	h := lifecycleMux(s)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, lifecycleReq(http.MethodGet, "/api/v1/updates", "", "tok"))
	var st UpdateStatus
	_ = json.Unmarshal(rec.Body.Bytes(), &st)
	if st.Enabled || st.Newer {
		t.Fatalf("default must be off and show no notice: %+v", st)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, lifecycleReq(http.MethodPost, "/api/v1/updates", `{"enabled":true}`, "tok"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	cfg, err := config.Load(path)
	if err != nil || !cfg.UpdateCheckEnabled() {
		t.Fatalf("updates.check_enabled not saved: %v", err)
	}
	if len(toggled) != 1 || !toggled[0] {
		t.Fatalf("onToggle not called: %v", toggled)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, lifecycleReq(http.MethodPost, "/api/v1/updates", `{}`, "tok"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing enabled: status %d", rec.Code)
	}
}
