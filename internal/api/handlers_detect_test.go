package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

func TestDetectVPNs(t *testing.T) {
	orig, origActive := detectClients, activeTunnel
	t.Cleanup(func() { detectClients, activeTunnel = orig, origActive })
	// Tunnel state is covered by TestDetectVPNs_ActiveTunnel.
	activeTunnel = func(string) (appattach.Tunnel, bool, bool) { return appattach.Tunnel{}, false, false }

	// Nothing installed: an empty list, never null.
	detectClients = func() []vpndetect.Client { return nil }
	rec := httptest.NewRecorder()
	(&Server{}).handleDetectVPNs(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vpns/detect", nil))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("none found: got %d %q, want 200 []", rec.Code, rec.Body.String())
	}

	detectClients = func() []vpndetect.Client {
		return []vpndetect.Client{
			{Label: "NordVPN", AdapterType: "nordvpn", Path: `C:\Program Files\NordVPN\NordVPN.exe`, VersionArgs: []string{"--version"}},
			{Label: "Other VPN", Path: "/usr/bin/other"},
		}
	}
	rec = httptest.NewRecorder()
	(&Server{}).handleDetectVPNs(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vpns/detect", nil))
	var got []DetectedClient
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := []DetectedClient{
		{Label: "NordVPN", AdapterType: "nordvpn", Path: `C:\Program Files\NordVPN\NordVPN.exe`},
		{Label: "Other VPN", Path: "/usr/bin/other"},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
