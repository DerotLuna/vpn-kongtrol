package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

func setupStatusOf(t *testing.T, s *Server) SetupStatus {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handleSetupStatus(rec, httptest.NewRequest(http.MethodGet, "/api/v1/setup", nil))
	var st SetupStatus
	if err := json.NewDecoder(rec.Body).Decode(&st); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return st
}

// TestSetupMode_FirstProfileWritesConfig walks the first run: a daemon with
// no kongtrol.yaml lists no profiles, creates the first one from the
// dashboard request, writes the file, and then reports restart_pending.
func TestSetupMode_FirstProfileWritesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	var updated *config.Config
	s := &Server{
		configPath: path,
		onPolicyUpdate: func(c *config.Config, _ *policy.Engine) {
			updated = c
		},
	}
	s.SetSetupMode(true)

	if st := setupStatusOf(t, s); !st.NeedsSetup || st.RestartPending {
		t.Fatalf("before: %+v, want needs_setup and no restart", st)
	}

	rec := httptest.NewRecorder()
	s.handleListVPNs(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vpns", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s, want 200", rec.Code, rec.Body)
	}

	body := []byte(`{"name":"office","type":"wireguard","tunnel_name":"wg-office","config":"/tmp/wg-office.conf","auth_method":"cert"}`)
	rec = httptest.NewRecorder()
	s.handleCreateVPN(rec, httptest.NewRequest(http.MethodPost, "/api/v1/vpns", bytes.NewReader(body)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s, want 201", rec.Code, rec.Body)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if updated == nil || len(updated.VPNs) != 1 {
		t.Fatalf("onPolicyUpdate cfg=%v, want one profile", updated)
	}
	if st := setupStatusOf(t, s); st.NeedsSetup || !st.RestartPending {
		t.Fatalf("after: %+v, want restart_pending", st)
	}
}

// TestSetupStatus_NormalDaemon: a daemon started with a config is never in
// setup mode.
func TestSetupStatus_NormalDaemon(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kongtrol.yaml")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	st := setupStatusOf(t, &Server{configPath: path})
	if st.SetupMode || st.NeedsSetup || st.RestartPending {
		t.Fatalf("got %+v, want all false", st)
	}
}

func TestHandleRestart(t *testing.T) {
	t.Run("not wired", func(t *testing.T) {
		rec := httptest.NewRecorder()
		(&Server{}).handleRestart(rec, httptest.NewRequest(http.MethodPost, "/api/v1/restart", nil))
		if rec.Code != http.StatusNotImplemented {
			t.Fatalf("status=%d, want 501", rec.Code)
		}
	})

	t.Run("spawn fails: daemon keeps running", func(t *testing.T) {
		stopped := make(chan struct{}, 1)
		s := &Server{onShutdown: func() { stopped <- struct{}{} }}
		s.SetRestarter(func() error { return errors.New("no se pudo") })
		rec := httptest.NewRecorder()
		s.handleRestart(rec, httptest.NewRequest(http.MethodPost, "/api/v1/restart", nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d, want 500", rec.Code)
		}
		select {
		case <-stopped:
			t.Fatal("daemon stopped although the replacement did not start")
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("replacement started: daemon stops", func(t *testing.T) {
		stopped := make(chan struct{}, 1)
		s := &Server{onShutdown: func() { stopped <- struct{}{} }}
		s.SetRestarter(func() error { return nil })
		rec := httptest.NewRecorder()
		s.handleRestart(rec, httptest.NewRequest(http.MethodPost, "/api/v1/restart", nil))
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status=%d, want 202", rec.Code)
		}
		select {
		case <-stopped:
		case <-time.After(time.Second):
			t.Fatal("onShutdown not called")
		}
	})
}
