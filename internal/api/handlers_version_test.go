package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

func TestHelperStatus_Outdated(t *testing.T) {
	q, p, e := helperQueryState, helperPing, helperIsElevated
	t.Cleanup(func() { helperQueryState, helperPing, helperIsElevated = q, p, e })
	helperIsElevated = func() bool { return false }
	helperQueryState = func() privhelper.State { return privhelper.State{Installed: true, Running: true} }
	helperPing = func(context.Context) (privops.PingResult, error) {
		return privops.PingResult{Version: "0.5.0", Protocol: privops.ProtocolVersion - 1}, nil
	}
	s := &Server{}
	s.SetHelperInstaller("0.6.0", nil)
	got := s.helperStatus(context.Background())
	if !got.Outdated || !got.NeedsInstall || got.Available {
		t.Fatalf("status = %+v, want outdated + needs_install", got)
	}
	helperIsElevated = func() bool { return true }
	if got := s.helperStatus(context.Background()); got.Outdated || got.NeedsInstall {
		t.Fatalf("elevated status = %+v, want nothing to do", got)
	}
}

func TestVersionEndpoint(t *testing.T) {
	s := &Server{}
	s.SetHelperInstaller("v0.6.0-rc.3", nil)
	rec := httptest.NewRecorder()
	s.handleVersion(rec, httptest.NewRequest(http.MethodGet, "/api/v1/version", nil))
	var got VersionInfo
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || got.Version != "v0.6.0-rc.3" {
		t.Fatalf("got %d %+v", rec.Code, got)
	}
}
