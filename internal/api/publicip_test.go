package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// countingTransport fails the test path open: it counts every outbound
// request and answers a fixed IP without touching the network.
type countingTransport struct{ hits atomic.Int32 }

func (c *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.hits.Add(1)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioNopCloser("203.0.113.7"),
		Header:     http.Header{},
		Request:    r,
	}, nil
}

type nopBody struct{ *strings.Reader }

func (nopBody) Close() error { return nil }

func ioNopCloser(s string) nopBody { return nopBody{strings.NewReader(s)} }

const securityOnYAML = baseTestYAML + `
security:
  kill_switch:
    enabled: true
  dns_guard:
    enabled: true
`

func newPublicIPServer(t *testing.T, yamlBody string) (*Server, *countingTransport) {
	t.Helper()
	s, path := newReloadTestServer(t, yamlBody, nil)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	tr := &countingTransport{}
	s.leakTest = security.NewLeakTester(security.LeakTesterOptions{
		Enabled:     cfg.PublicIPCheckEnabled(),
		Endpoints:   cfg.PublicIPEndpoints(),
		Client:      &http.Client{Transport: tr},
		SettleDelay: -1,
	})
	return s, tr
}

// Kill switch + DNS guard on, public IP lookup off (the default): status,
// overview and "Revisar" make zero outbound requests.
func TestPublicIP_DecoupledFromKillSwitchAndDNSGuard(t *testing.T) {
	s, tr := newPublicIPServer(t, securityOnYAML)

	rec := httptest.NewRecorder()
	s.handleSecurityStatus(rec, httptest.NewRequest(http.MethodGet, "/api/v1/security/status", nil))
	var st map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if st["public_ip_check_enabled"] != false || st["leak_check_available"] != false {
		t.Errorf("lookup should be off with only kill switch / DNS guard on: %v", st)
	}

	rec = httptest.NewRecorder()
	s.handleNetworkOverview(rec, httptest.NewRequest(http.MethodGet, "/api/v1/network/overview", nil))
	if strings.Contains(rec.Body.String(), `"public_ip":`) {
		t.Errorf("overview shows a public IP while the lookup is off: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	s.handleCheckPublicIP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/security/publicip/check", nil))
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "publicip.err.disabled") {
		t.Errorf("check while off = %d %s, want 409 publicip.err.disabled", rec.Code, rec.Body.String())
	}

	s.leakTest.Trigger()
	if n := tr.hits.Load(); n != 0 {
		t.Fatalf("made %d outbound requests with the lookup off", n)
	}
}

func TestPublicIP_ToggleAndCheckOnDemand(t *testing.T) {
	s, tr := newPublicIPServer(t, securityOnYAML)
	s.onSecurityToggle = func(c *config.Config) {
		s.leakTest.Configure(c.PublicIPCheckEnabled(), c.PublicIPEndpoints(), nil)
	}

	rec := httptest.NewRecorder()
	s.handleTogglePublicIP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/security/publicip", strings.NewReader(`{"enabled":true}`)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"public_ip_check_host":"api.ipify.org"`) {
		t.Fatalf("toggle on = %d %s", rec.Code, rec.Body.String())
	}
	if tr.hits.Load() != 0 {
		t.Fatal("turning it on must not look up by itself in the handler")
	}

	rec = httptest.NewRecorder()
	s.handleCheckPublicIP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/security/publicip/check", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "203.0.113.7") || tr.hits.Load() != 1 {
		t.Fatalf("check = %d %s (hits %d)", rec.Code, rec.Body.String(), tr.hits.Load())
	}

	// Turning it off persists and forgets the IP.
	rec = httptest.NewRecorder()
	s.handleTogglePublicIP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/security/publicip", strings.NewReader(`{"enabled":false}`)))
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicIPCheckEnabled() || s.leakTest.LastResult() != nil {
		t.Error("lookup should be off and the IP forgotten")
	}
}

// Legacy leak_detection.enabled still opts in (backward compatibility).
func TestPublicIP_LegacyLeakDetectionOptsIn(t *testing.T) {
	s, _ := newPublicIPServer(t, baseTestYAML+"\nsecurity:\n  leak_detection:\n    enabled: true\n")
	if !s.leakTest.Enabled() {
		t.Error("leak_detection.enabled: true should keep the lookup on")
	}
}
