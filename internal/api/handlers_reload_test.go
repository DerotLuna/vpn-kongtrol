package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

const baseTestYAML = `
vpns:
  office:
    type: openvpn
    auth:
      method: credentials
`

func writeTestConfig(t *testing.T, yamlBody string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	if err := os.WriteFile(path, []byte(yamlBody), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func newReloadTestServer(t *testing.T, yamlBody string, adapters map[string]vpn.VPNAdapter) (*Server, string) {
	t.Helper()
	path := writeTestConfig(t, yamlBody)
	s := &Server{
		configPath:    path,
		adapters:      adapters,
		connectCancel: make(map[string]context.CancelFunc),
	}
	initial, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := policy.New(initial)
	if err != nil {
		t.Fatal(err)
	}
	s.policyEngine.Store(engine)
	return s, path
}

// TestHandlePolicyReload_PicksUpHandEditedYAML proves that hand-editing
// kongtrol.yaml on disk (with no dashboard/CLI mutation) followed by a
// POST /api/v1/policies/reload results in the live policy engine reflecting
// the new file content — the exact gap CLAUDE.md's config-mutation pattern
// leaves open, since saveRuntimeConfig only ever re-marshals what's already
// in memory.
func TestHandlePolicyReload_PicksUpHandEditedYAML(t *testing.T) {
	s, path := newReloadTestServer(t, baseTestYAML, map[string]vpn.VPNAdapter{
		"office": newWSTestAdapter(vpn.StatusDisconnected),
	})

	if pe := s.policyEngine.Load(); len(pe.Rules()) != 0 {
		t.Fatalf("expected no rules before reload, got %d", len(pe.Rules()))
	}

	// Simulate a hand edit: append a policy rule directly to the file on
	// disk, bypassing every dashboard/CLI mutation path.
	edited := baseTestYAML + `
policies:
  - name: office-net
    via: office
    match:
      domains: ["example.com"]
`
	if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}

	var updateCalled atomic.Bool
	s.onPolicyUpdate = func(cfg *config.Config, engine *policy.Engine) {
		updateCalled.Store(true)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/reload", nil)
	rec := httptest.NewRecorder()
	s.handlePolicyReload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !updateCalled.Load() {
		t.Fatal("onPolicyUpdate was not invoked")
	}
	pe := s.policyEngine.Load()
	if len(pe.Rules()) != 1 || pe.Rules()[0].Name != "office-net" {
		t.Fatalf("policy engine not reloaded from disk: rules=%+v", pe.Rules())
	}
}

func TestHandlePolicyReload_InvalidYAMLReturnsBadRequest(t *testing.T) {
	s, path := newReloadTestServer(t, baseTestYAML, map[string]vpn.VPNAdapter{
		"office": newWSTestAdapter(vpn.StatusDisconnected),
	})
	if err := os.WriteFile(path, []byte("not: [valid yaml"), 0o600); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/reload", nil)
	rec := httptest.NewRecorder()
	s.handlePolicyReload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

// TestHandleReloadGroup_RestartsConnectedProfilesOnly proves group reload
// disconnects then reconnects only the currently-connected profiles in the
// group (via the shared connectProfile/disconnectProfile lifecycle, exactly
// like handleConnectGroup/handleDisconnectGroup), leaving already-disconnected
// profiles alone.
func TestHandleReloadGroup_RestartsConnectedProfilesOnly(t *testing.T) {
	yamlBody := baseTestYAML + `
  aws:
    type: openvpn
    auth:
      method: credentials
groups:
  work:
    profiles: ["office", "aws"]
`
	s, _ := newReloadTestServer(t, yamlBody, map[string]vpn.VPNAdapter{
		"office": newWSTestAdapter(vpn.StatusConnected),
		"aws":    newWSTestAdapter(vpn.StatusDisconnected),
	})

	var disconnected, connected []string
	s.disconnectProfile = func(_ context.Context, name string) error {
		disconnected = append(disconnected, name)
		return nil
	}
	connectDone := make(chan string, 2)
	s.connectProfile = func(_ context.Context, name string) error {
		connectDone <- name
		return nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/work/reload", nil)
	req.SetPathValue("name", "work")
	rec := httptest.NewRecorder()
	s.handleReloadGroup(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(disconnected) != 1 || disconnected[0] != "office" {
		t.Fatalf("disconnected=%v, want [office]", disconnected)
	}

	select {
	case name := <-connectDone:
		connected = append(connected, name)
	case <-time.After(2 * time.Second):
		t.Fatal("connectProfile was not invoked for office")
	}
	if len(connected) != 1 || connected[0] != "office" {
		t.Fatalf("connected=%v, want [office]", connected)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	skipped, _ := body["skipped"].([]any)
	if len(skipped) != 1 || skipped[0] != "aws" {
		t.Fatalf("skipped=%v, want [aws]", skipped)
	}
}

// TestHandleReloadGroup_MissingAdapterReturnsRestartRequired proves that a
// group referencing a profile the running daemon has no adapter for (a
// brand-new profile added by the hand edit) is reported back as
// restart_required instead of silently skipped or attempted — matching the
// same restart_required convention handleCreateVPN/handleUpdateVPN already
// use for the "adapters map isn't hot-registrable" limitation documented in
// CLAUDE.md.
func TestHandleReloadGroup_MissingAdapterReturnsRestartRequired(t *testing.T) {
	yamlBody := baseTestYAML + `
  newvpn:
    type: wireguard
    auth:
      method: certificate
groups:
  work:
    profiles: ["office", "newvpn"]
`
	s, _ := newReloadTestServer(t, yamlBody, map[string]vpn.VPNAdapter{
		"office": newWSTestAdapter(vpn.StatusConnected),
		// "newvpn" intentionally has no adapter — simulates a profile added
		// to kongtrol.yaml by hand after the daemon booted.
	})
	s.disconnectProfile = func(context.Context, string) error { return nil }
	s.connectProfile = func(context.Context, string) error { return nil }

	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/work/reload", nil)
	req.SetPathValue("name", "work")
	rec := httptest.NewRecorder()
	s.handleReloadGroup(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "restart_required" {
		t.Fatalf("status field=%v, want restart_required", body["status"])
	}
	missing, _ := body["missing_profiles"].([]any)
	if len(missing) != 1 || missing[0] != "newvpn" {
		t.Fatalf("missing_profiles=%v, want [newvpn]", missing)
	}
}

func TestHandleReloadGroup_NotFound(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, map[string]vpn.VPNAdapter{
		"office": newWSTestAdapter(vpn.StatusDisconnected),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/ghost/reload", nil)
	req.SetPathValue("name", "ghost")
	rec := httptest.NewRecorder()
	s.handleReloadGroup(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
