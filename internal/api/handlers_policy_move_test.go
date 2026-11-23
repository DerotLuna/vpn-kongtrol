package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const protonMoveConf = "[Interface]\n# NetShield = 1\n# VPN Accelerator = on\nPrivateKey = x\nAddress = 10.2.0.2/32\nDNS = 10.2.0.1\n\n[Peer]\nPublicKey = y\nEndpoint = 203.0.113.7:51820\n"

func moveTestYAML(t *testing.T) string {
	t.Helper()
	conf := filepath.Join(t.TempDir(), "proton-ch.conf")
	if err := os.WriteFile(conf, []byte(protonMoveConf), 0o600); err != nil {
		t.Fatal(err)
	}
	return `
vpns:
  proton-ch:
    type: wireguard
    config: "` + filepath.ToSlash(conf) + `"
    auth:
      method: certificate
  Proton:
    type: protonvpn
    auth:
      method: none
policies:
  - name: news
    via: proton-ch
    match:
      domains: ["news.example"]
  - name: shop
    via: proton-ch
    match:
      domains: ["shop.example"]
`
}

// GET /api/v1/vpns names the provider of a direct profile's file; the
// move endpoint points its rules at the app profile in one save (ADR-0104).
func TestMovePolicies_ProviderFieldAndMove(t *testing.T) {
	s, path := newReloadTestServer(t, moveTestYAML(t), nil)

	rec := doJSON(t, s, http.MethodGet, "/api/v1/vpns", "/api/v1/vpns", s.handleListVPNs, "")
	var list []struct {
		Name     string `json:"name"`
		Provider string `json:"provider"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, v := range list {
		got[v.Name] = v.Provider
	}
	if got["proton-ch"] != "protonvpn" || got["Proton"] != "" {
		t.Fatalf("provider = %v", got)
	}

	rec = doJSON(t, s, http.MethodPost, "/api/v1/policies/move", "/api/v1/policies/move", s.handleMovePolicies,
		`{"from":"proton-ch","to":"Proton"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		Moved []string `json:"moved"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Moved) != 2 {
		t.Fatalf("moved = %v", out.Moved)
	}
	for _, p := range reload(t, path).Policies {
		if p.Via != "Proton" {
			t.Fatalf("rule %s still via %s", p.Name, p.Via)
		}
	}

	rec = doJSONLang(t, s, http.MethodPost, "/api/v1/policies/move", "/api/v1/policies/move", s.handleMovePolicies,
		`{"from":"Proton","to":"Proton"}`, "es")
	var le localizedErr
	_ = json.Unmarshal(rec.Body.Bytes(), &le)
	if rec.Code != http.StatusBadRequest || le.ErrorKey != "api.err.policy_move_same" || le.Hint == "" {
		t.Fatalf("same profile: %d %s", rec.Code, rec.Body)
	}
}
