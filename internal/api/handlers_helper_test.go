package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// fakeHelper swaps the helper probes for a helper that answers only once
// installed is true.
func fakeHelper(t *testing.T, installed *bool) {
	t.Helper()
	q, p, e := helperQueryState, helperPing, helperIsElevated
	t.Cleanup(func() { helperQueryState, helperPing, helperIsElevated = q, p, e })
	helperIsElevated = func() bool { return false }
	helperQueryState = func() privhelper.State { return privhelper.State{Installed: *installed, Running: *installed} }
	helperPing = func(context.Context) (privops.PingResult, error) {
		if !*installed {
			return privops.PingResult{}, privops.ErrHelperNotInstalled
		}
		return privops.PingResult{Version: "1.2.3", Protocol: privops.ProtocolVersion}, nil
	}
}

func TestHelperStatus_NeedsInstall(t *testing.T) {
	installed := false
	fakeHelper(t, &installed)
	s := &Server{}
	s.SetHelperInstaller("1.2.3", nil)

	rec := httptest.NewRecorder()
	s.handleHelperStatus(rec, httptest.NewRequest(http.MethodGet, "/api/v1/helper/status", nil))
	var got HelperStatus
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := HelperStatus{CLIVersion: "1.2.3", NeedsInstall: true}
	if got != want {
		t.Fatalf("status = %+v, want %+v", got, want)
	}
}

func TestHelperInstall(t *testing.T) {
	cases := []struct {
		name       string
		installErr error
		wantCode   int
		wantKey    string
	}{
		{"ok", nil, http.StatusOK, ""},
		{"cancelled", fmt.Errorf("x: %w", privhelper.ErrElevationCancelled), http.StatusForbidden, "service.elevation.cancelled"},
		{"failed", errors.New("boom"), http.StatusInternalServerError, "service.install.failed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			installed := false
			fakeHelper(t, &installed)
			s := &Server{}
			s.SetHelperInstaller("1.2.3", func(context.Context) error {
				installed = c.installErr == nil
				return c.installErr
			})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/helper/install", nil)
			req.Header.Set("X-Kongtrol-Lang", "es")
			rec := httptest.NewRecorder()
			s.handleHelperInstall(rec, req)
			if rec.Code != c.wantCode {
				t.Fatalf("code = %d, want %d (%s)", rec.Code, c.wantCode, rec.Body)
			}
			var body map[string]any
			_ = json.NewDecoder(rec.Body).Decode(&body)
			if c.wantKey == "" {
				if body["available"] != true || body["needs_install"] != false {
					t.Fatalf("body = %v", body)
				}
				return
			}
			if body["error_key"] != c.wantKey || body["error"] != i18n.T(i18n.ES, c.wantKey) || body["hint"] == "" {
				t.Fatalf("body = %v", body)
			}
		})
	}
}

func TestHelperInstall_NoInstaller(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()
	s.handleHelperInstall(rec, httptest.NewRequest(http.MethodPost, "/api/v1/helper/install", nil))
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("code = %d", rec.Code)
	}
}
