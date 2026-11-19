package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Profile CRUD applied without a restart (ADR-0053).

// newLiveServer wires the server to a real app.AdapterSync, like the
// daemon does, with test adapters instead of real VPN clients.
func newLiveServer(t *testing.T, yamlBody string, adapters map[string]vpn.VPNAdapter) (*Server, string) {
	t.Helper()
	s, path := newReloadTestServer(t, yamlBody, adapters)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	sync := app.NewAdapterSync(s.adapters, cfg, nil)
	sync.SetBuilder(func(config.VPNConfig) (vpn.VPNAdapter, error) {
		return newWSTestAdapter(vpn.StatusDisconnected), nil
	})
	s.SetProfileSync(func(c *config.Config) ProfileSyncResult {
		r := sync.Sync(c)
		out := ProfileSyncResult{Reconnect: r.Reconnect, Kept: r.Kept}
		for n := range r.Failed {
			out.Failed = append(out.Failed, n)
		}
		return out
	})
	return s, path
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("body %q: %v", rec.Body.String(), err)
	}
	return m
}

func TestProfileCRUD_AppliedLive(t *testing.T) {
	s, _ := newLiveServer(t, providerTestYAML, map[string]vpn.VPNAdapter{
		"nord":  newWSTestAdapter(vpn.StatusDisconnected),
		"shark": newWSTestAdapter(vpn.StatusDisconnected),
	})

	rec := doJSON(t, s, http.MethodPost, "/api/v1/vpns", "/api/v1/vpns", s.handleCreateVPN,
		`{"name":"proton","type":"protonvpn","country":"CH"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %d: %s", rec.Code, rec.Body)
	}
	if b := decodeBody(t, rec); b["restart_required"] != false || b["reconnect_required"] != false {
		t.Fatalf("create body %v", b)
	}
	if !s.adapters.Has("proton") {
		t.Fatal("new profile not registered live")
	}

	before, _ := s.adapters.Get("shark")
	rec = doJSON(t, s, http.MethodPut, "/api/v1/vpns/shark", "/api/v1/vpns/{name}", s.handleUpdateVPN,
		`{"interface":"surf*"}`)
	if b := decodeBody(t, rec); rec.Code != http.StatusOK || b["restart_required"] != false {
		t.Fatalf("update %d %v", rec.Code, b)
	}
	if after, _ := s.adapters.Get("shark"); after == before {
		t.Fatal("edited disconnected profile keeps its old adapter")
	}

	rec = doJSON(t, s, http.MethodPut, "/api/v1/vpns/proton", "/api/v1/vpns/{name}", s.handleUpdateVPN,
		`{"name":"proton-ch"}`)
	if b := decodeBody(t, rec); rec.Code != http.StatusOK || b["restart_required"] != false || b["previous"] != "proton" {
		t.Fatalf("rename %d %v", rec.Code, b)
	}
	if s.adapters.Has("proton") || !s.adapters.Has("proton-ch") {
		t.Fatalf("rename not applied: %v", s.adapters.Names())
	}

	rec = doJSON(t, s, http.MethodDelete, "/api/v1/vpns/proton-ch", "/api/v1/vpns/{name}", s.handleDeleteVPN, "")
	if b := decodeBody(t, rec); rec.Code != http.StatusOK || b["restart_required"] != false {
		t.Fatalf("delete %d %v", rec.Code, b)
	}
	if s.adapters.Has("proton-ch") {
		t.Fatal("deleted profile still registered")
	}
}

func TestProfileCRUD_WithoutLiveApplyNeedsRestart(t *testing.T) {
	s, _ := newReloadTestServer(t, providerTestYAML, nil)
	rec := doJSON(t, s, http.MethodPost, "/api/v1/vpns", "/api/v1/vpns", s.handleCreateVPN,
		`{"name":"proton","type":"protonvpn"}`)
	if b := decodeBody(t, rec); b["restart_required"] != true {
		t.Fatalf("no live apply must answer restart_required: %v", b)
	}
	// Setup mode keeps its restart too.
	s2, _ := newLiveServer(t, providerTestYAML, nil)
	s2.SetSetupMode(true)
	rec = doJSON(t, s2, http.MethodPost, "/api/v1/vpns", "/api/v1/vpns", s2.handleCreateVPN,
		`{"name":"proton","type":"protonvpn"}`)
	if b := decodeBody(t, rec); b["restart_required"] != true {
		t.Fatalf("setup mode must answer restart_required: %v", b)
	}
}

func TestUpdateConnectedProfile_ReconnectRequiredOrDone(t *testing.T) {
	nord := newWSTestAdapter(vpn.StatusConnected)
	other := newWSTestAdapter(vpn.StatusConnected)
	s, _ := newLiveServer(t, providerTestYAML, map[string]vpn.VPNAdapter{"nord": nord, "shark": other})
	var mu sync.Mutex
	var calls []string
	s.disconnectProfile = func(_ context.Context, n string) error {
		mu.Lock()
		calls = append(calls, "down:"+n)
		mu.Unlock()
		return nil
	}
	connected := make(chan string, 2)
	s.connectProfile = func(_ context.Context, n string) error { connected <- n; return nil }

	// Priority is read when used: no reconnect needed.
	rec := doJSON(t, s, http.MethodPut, "/api/v1/vpns/nord", "/api/v1/vpns/{name}", s.handleUpdateVPN, `{"priority":5}`)
	if b := decodeBody(t, rec); b["reconnect_required"] != false {
		t.Fatalf("priority edit: %v", b)
	}

	// A tunnel setting: the running tunnel keeps the old one, said so.
	rec = doJSON(t, s, http.MethodPut, "/api/v1/vpns/nord", "/api/v1/vpns/{name}", s.handleUpdateVPN, `{"city":"Berlin"}`)
	b := decodeBody(t, rec)
	if b["reconnect_required"] != true || b["restart_required"] != false || b["reconnected"] != nil {
		t.Fatalf("connected edit: %v", b)
	}
	if cur, _ := s.adapters.Get("nord"); cur != nord {
		t.Fatal("an up tunnel's adapter was replaced")
	}
	if len(calls) != 0 {
		t.Fatalf("no reconnect without asking: %v", calls)
	}

	// ?reconnect=true: only this tunnel goes down and up again.
	rec = doJSON(t, s, http.MethodPut, "/api/v1/vpns/nord?reconnect=true", "/api/v1/vpns/{name}", s.handleUpdateVPN, `{"city":"Hamburg"}`)
	b = decodeBody(t, rec)
	if b["reconnected"] != true || b["reconnect_required"] != false {
		t.Fatalf("save and reconnect: %v", b)
	}
	if got := <-connected; got != "nord" {
		t.Fatalf("connected %q", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 1 || calls[0] != "down:nord" {
		t.Fatalf("other tunnels must never be touched: %v", calls)
	}
}

func TestDeleteConnectedProfile_Refused(t *testing.T) {
	s, path := newLiveServer(t, providerTestYAML, map[string]vpn.VPNAdapter{
		"nord":  newWSTestAdapter(vpn.StatusConnected),
		"shark": newWSTestAdapter(vpn.StatusDisconnected),
	})
	rec := doJSONLang(t, s, http.MethodDelete, "/api/v1/vpns/nord", "/api/v1/vpns/{name}", s.handleDeleteVPN, "", "en")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var e localizedErr
	_ = json.Unmarshal(rec.Body.Bytes(), &e)
	if e.ErrorKey != "api.err.vpn_delete_busy" || e.Hint == "" {
		t.Fatalf("error %+v", e)
	}
	if _, ok := reload(t, path).VPNs["nord"]; !ok || !s.adapters.Has("nord") {
		t.Fatal("refused delete changed something")
	}
}

func TestPolicyReload_HandEditedProfilesApplyLive(t *testing.T) {
	nord := newWSTestAdapter(vpn.StatusConnected)
	s, path := newLiveServer(t, providerTestYAML, map[string]vpn.VPNAdapter{
		"nord": nord, "shark": newWSTestAdapter(vpn.StatusDisconnected),
	})
	edited := strings.Replace(providerTestYAML, "country: Germany", "country: Spain", 1) +
		"  proton:\n    type: protonvpn\n"
	if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	s.handlePolicyReload(rec, httptest.NewRequest(http.MethodPost, "/api/v1/policies/reload", nil))
	b := decodeBody(t, rec)
	if rec.Code != http.StatusOK || b["restart_required"] != false {
		t.Fatalf("reload %d %v", rec.Code, b)
	}
	if rr, _ := b["reconnect_required"].([]any); len(rr) != 1 || rr[0] != "nord" {
		t.Fatalf("reconnect_required = %v", b["reconnect_required"])
	}
	if !s.adapters.Has("proton") {
		t.Fatal("hand-added profile not registered")
	}
}

func TestTunnelSettingsChanged(t *testing.T) {
	base := config.VPNConfig{Type: "nordvpn", Country: "DE", Priority: 1}
	p := base
	p.Priority = 9
	p.Failover = []string{"x"}
	on := true
	p.KillSwitch = &on
	if tunnelSettingsChanged(base, p) {
		t.Fatal("priority/failover/kill switch are read when used")
	}
	p.Country = "ES"
	if !tunnelSettingsChanged(base, p) {
		t.Fatal("country is a tunnel setting")
	}
}
