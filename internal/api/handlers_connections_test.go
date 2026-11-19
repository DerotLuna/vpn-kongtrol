package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

func TestRecentConnectionsHandler(t *testing.T) {
	// No list wired: an explicit, empty, unsupported answer.
	rec := httptest.NewRecorder()
	(&Server{}).handleRecentConnections(rec, httptest.NewRequest(http.MethodGet, "/api/v1/connections/recent", nil))
	var got RecentConnectionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || got.Supported || got.Connections == nil || got.Max != monitor.RecentMax {
		t.Fatalf("got %d %+v", rec.Code, got)
	}

	s := &Server{}
	r := monitor.NewRecentConnections(false)
	s.SetRecentConnections(r)
	rec = httptest.NewRecorder()
	s.handleRecentConnections(rec, httptest.NewRequest(http.MethodGet, "/api/v1/connections/recent?limit=5", nil))
	got = RecentConnectionsResponse{}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Enabled || len(got.Connections) != 0 || got.Supported != r.Supported() {
		t.Fatalf("off list: %+v", got)
	}

	rec = doJSONLang(t, s, http.MethodGet, "/api/v1/connections/recent?limit=-1", "/api/v1/connections/recent", s.handleRecentConnections, "", "es")
	var e localizedErr
	_ = json.Unmarshal(rec.Body.Bytes(), &e)
	if rec.Code != http.StatusBadRequest || e.ErrorKey != "api.err.recent_limit" || e.Hint == "" {
		t.Fatalf("bad limit: %d %+v", rec.Code, e)
	}
}

func TestSettings_RecentConnectionsToggle(t *testing.T) {
	s, path := newReloadTestServer(t, baseTestYAML, nil)
	rec := doJSON(t, s, http.MethodGet, "/api/v1/settings", "/api/v1/settings", s.handleGetSettings, "")
	var dto settingsDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &dto)
	if !dto.RecentConnectionsEnabled {
		t.Fatal("recent connections must default to on")
	}
	rec = doJSON(t, s, http.MethodPut, "/api/v1/settings", "/api/v1/settings", s.handleUpdateSettings, `{"recent_connections_enabled":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if reload(t, path).RecentConnectionsEnabled() {
		t.Fatal("off not saved")
	}
}

func TestDetectVPNs_ActiveTunnel(t *testing.T) {
	origC, origA := detectClients, activeTunnel
	t.Cleanup(func() { detectClients, activeTunnel = origC, origA })
	detectClients = func() []vpndetect.Client {
		return []vpndetect.Client{
			{Label: "NordVPN", AdapterType: "nordvpn"},
			{Label: "Surfshark", AdapterType: "surfshark"},
			{Label: "WireGuard", AdapterType: "wireguard"},
			{Label: "Other"},
		}
	}
	activeTunnel = func(typ string) (appattach.Tunnel, bool, bool) {
		switch typ {
		case "nordvpn":
			return appattach.Tunnel{Name: "NordLynx"}, true, true
		case "surfshark":
			return appattach.Tunnel{}, false, true
		}
		return appattach.Tunnel{}, false, false
	}
	rec := httptest.NewRecorder()
	(&Server{}).handleDetectVPNs(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vpns/detect", nil))
	var raw []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if raw[0]["active"] != true || raw[0]["interface"] != "NordLynx" {
		t.Fatalf("nord: %v", raw[0])
	}
	if raw[1]["active"] != false || raw[1]["interface"] != nil {
		t.Fatalf("surfshark: %v", raw[1])
	}
	// Kongtrol cannot tell: explicit null, never a guess.
	if v, ok := raw[2]["active"]; !ok || v != nil {
		t.Fatalf("wireguard: %v", raw[2])
	}
	if v, ok := raw[3]["active"]; !ok || v != nil {
		t.Fatalf("other: %v", raw[3])
	}
}
