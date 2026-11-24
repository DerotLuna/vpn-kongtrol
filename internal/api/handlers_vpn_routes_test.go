package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GET /api/v1/vpns/routes lists each VPN's own routes with their origin,
// and 404s an unknown ?profile= (ADR-0107).
func TestHandleVPNOwnRoutes(t *testing.T) {
	ovpn := filepath.Join(t.TempDir(), "office.ovpn")
	if err := os.WriteFile(ovpn, []byte("client\nroute 10.20.0.0 255.255.0.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	yamlBody := strings.Replace(baseTestYAML, "type: openvpn", "type: openvpn\n    config: "+filepath.ToSlash(ovpn), 1)
	s, _ := newReloadTestServer(t, yamlBody, nil)

	rec := httptest.NewRecorder()
	s.handleVPNOwnRoutes(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vpns/routes", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Profiles []struct {
			Name          string `json:"name"`
			Source        string `json:"source"`
			ServerPending bool   `json:"server_pending"`
			Routes        []struct {
				Destination string `json:"destination"`
				Origin      string `json:"origin"`
			} `json:"routes"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Profiles) != 1 {
		t.Fatalf("profiles = %+v", body.Profiles)
	}
	p := body.Profiles[0]
	if p.Name != "office" || p.Source != "config" || !p.ServerPending || len(p.Routes) != 1 ||
		p.Routes[0].Destination != "10.20.0.0/16" || p.Routes[0].Origin != "file" {
		t.Errorf("office = %+v", p)
	}

	rec = httptest.NewRecorder()
	s.handleVPNOwnRoutes(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vpns/routes?profile=nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown profile: status %d", rec.Code)
	}
}
