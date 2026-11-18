package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/doctor"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func authedGet(s *Server, path, lang string, token bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9741"+path, nil)
	req.Host = "127.0.0.1:9741"
	if token {
		req.Header.Set("X-Kongtrol-Token", s.apiToken)
	}
	if lang != "" {
		req.Header.Set("X-Kongtrol-Lang", lang)
	}
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/doctor", s.handleDoctor)
	mux.HandleFunc("GET /api/v1/config/export", s.handleConfigExport)
	s.secureHandler(mux).ServeHTTP(rec, req)
	return rec
}

func stubDoctor(t *testing.T, got *doctor.Options) {
	t.Helper()
	prev := runDoctor
	t.Cleanup(func() { runDoctor = prev })
	runDoctor = func(opt doctor.Options) doctor.Report {
		*got = opt
		return doctor.Report{Warnings: 1, Checks: []doctor.Check{{
			Label: "helper", Status: doctor.StatusWarn, Detail: "x", Hint: "y", Fix: doctor.FixHelperInstall,
		}}}
	}
}

func TestDoctorEndpoint(t *testing.T) {
	var opt doctor.Options
	stubDoctor(t, &opt)
	s := &Server{bind: "127.0.0.1", port: 9741, apiToken: "tok", configPath: "/x/kongtrol.yaml"}
	s.dnsGuardOn.Store(true)

	if rec := authedGet(s, "/api/v1/doctor", "", false); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", rec.Code)
	}

	// No installer in this daemon: the fix is not offered.
	rec := authedGet(s, "/api/v1/doctor", "es", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d: %s", rec.Code, rec.Body)
	}
	var rep doctor.Report
	if err := json.NewDecoder(rec.Body).Decode(&rep); err != nil {
		t.Fatal(err)
	}
	if rep.Checks[0].Fix != "" || rep.Checks[0].Hint != "y" || rep.Warnings != 1 {
		t.Fatalf("report = %+v", rep)
	}
	if opt.Lang != i18n.ES || opt.ConfigPath != "/x/kongtrol.yaml" || opt.Daemon == nil {
		t.Fatalf("options = %+v", opt)
	}

	// With an installer the GUI may offer "Reparar".
	s.SetHelperInstaller("v9", func(context.Context) error { return nil })
	rec = authedGet(s, "/api/v1/doctor", "en", true)
	_ = json.NewDecoder(rec.Body).Decode(&rep)
	if rep.Checks[0].Fix != doctor.FixHelperInstall || opt.Version != "v9" || opt.Lang != i18n.EN {
		t.Fatalf("report = %+v, options = %+v", rep, opt)
	}
}

func TestConfigExportEndpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kongtrol.yaml")
	s := &Server{bind: "127.0.0.1", port: 9741, apiToken: "tok", configPath: path}

	if rec := authedGet(s, "/api/v1/config/export", "", false); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", rec.Code)
	}

	// No config yet (first run): 404 with a translated hint.
	s.setupMode.Store(true)
	rec := authedGet(s, "/api/v1/config/export", "es", true)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("no config: %d %s", rec.Code, rec.Body)
	}
	var body map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body["error_key"] != "export.err.no_config" || body["hint"] != i18n.T(i18n.ES, "export.err.no_config.fix") {
		t.Fatalf("body = %v", body)
	}

	cfgYAML := "vpns:\n  office:\n    type: wireguard\n    config: /etc/wg.conf\n    auth:\n      method: none\npolicies:\n  - name: corp\n    match:\n      ip_ranges: [10.0.0.0/8]\n      domains: [\"*.corp.com\"]\n    via: office\n"
	if err := os.WriteFile(path, []byte(cfgYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	rec = authedGet(s, "/api/v1/config/export", "en", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d: %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/yaml") {
		t.Fatalf("content-type %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="kongtrol-export.yaml"` {
		t.Fatalf("disposition %q", cd)
	}
	out := filepath.Join(dir, "export.yaml")
	if err := os.WriteFile(out, rec.Body.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(out); err != nil {
		t.Fatalf("export does not load: %v\n%s", err, rec.Body)
	}
}
