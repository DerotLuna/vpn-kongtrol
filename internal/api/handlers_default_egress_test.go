package api

import (
	"net/http"
	"runtime"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

const optedInWireGuard = `vpns:
  wg:
    type: wireguard
    default_egress: true
    auth:
      method: certificate
`

func TestProfileDefaultEgressCheckAndDisable(t *testing.T) {
	s, path := newLiveServer(t, optedInWireGuard, nil)
	const endpoint = "/api/v1/vpns/wg/default-egress"
	const pattern = "/api/v1/vpns/{name}/default-egress"
	check := doJSON(t, s, http.MethodGet, endpoint, pattern, s.handleDefaultEgress, "")
	wantReason := "unsupported_platform"
	if runtime.GOOS == "linux" {
		wantReason = "config_unavailable"
	}
	if body := decodeBody(t, check); check.Code != http.StatusOK || body["enabled"] != true || body["reason"] != wantReason {
		t.Fatalf("status %d: %v", check.Code, body)
	}
	set := doJSON(t, s, http.MethodPut, endpoint, pattern, s.handleSetDefaultEgress, `{"enabled":false}`)
	if body := decodeBody(t, set); set.Code != http.StatusOK || body["enabled"] != false {
		t.Fatalf("status %d: %v", set.Code, body)
	}
	cfg, err := config.Load(path)
	if err != nil || cfg.VPNs["wg"].DefaultEgress {
		t.Fatalf("default egress remained enabled: %v", err)
	}
	set = doJSON(t, s, http.MethodPut, endpoint, pattern, s.handleSetDefaultEgress, `{"enabled":true}`)
	if set.Code != http.StatusBadRequest {
		t.Fatalf("incompatible profile enable returned %d: %s", set.Code, set.Body)
	}
}

func TestProfileDefaultEgressRefusesChangeWhileConnected(t *testing.T) {
	s, _ := newLiveServer(t, optedInWireGuard, map[string]vpn.VPNAdapter{
		"wg": newWSTestAdapter(vpn.StatusConnected),
	})
	rec := doJSON(t, s, http.MethodPut, "/api/v1/vpns/wg/default-egress", "/api/v1/vpns/{name}/default-egress",
		s.handleSetDefaultEgress, `{"enabled":false}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("connected toggle returned %d: %s", rec.Code, rec.Body)
	}
}
