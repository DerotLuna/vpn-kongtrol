package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/autostart"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func TestStartupSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	if err := config.SavePreferences(path, &config.Preferences{Language: "es"}); err != nil {
		t.Fatal(err)
	}
	login := autostart.State{Supported: true, Enabled: true}
	stopped := false
	s := &Server{preferencesPath: path}
	s.SetStartup(StartupHooks{
		Autostart: func() autostart.State { return login },
		SetAutostart: func(on bool) error {
			login.Enabled = on
			return nil
		},
		OnStopAll: func() { stopped = true },
	})

	rec := doJSON(t, s, http.MethodGet, "/api/v1/startup", "/api/v1/startup", s.handleGetStartup, "")
	var got startupDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != http.StatusOK || !got.StartAtLogin || !got.StartAtLoginSupported || !got.RestoreTunnels {
		t.Fatalf("defaults: %d %+v (restore is on by default)", rec.Code, got)
	}

	rec = doJSON(t, s, http.MethodPut, "/api/v1/startup", "/api/v1/startup", s.handleSetStartup,
		`{"start_at_login":false,"restore_tunnels":false}`)
	got = startupDTO{}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != http.StatusOK || got.StartAtLogin || got.RestoreTunnels {
		t.Fatalf("after off: %d %s", rec.Code, rec.Body.String())
	}
	prefs, _ := config.LoadPreferences(path)
	if prefs.RestoreTunnelsEnabled() || prefs.Language != "es" {
		t.Fatalf("preferences: %+v", prefs)
	}

	rec = doJSON(t, s, http.MethodPut, "/api/v1/startup", "/api/v1/startup", s.handleSetStartup, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty body = %d, want 400", rec.Code)
	}
	if stopped {
		t.Error("changing a setting called the stop-all hook")
	}
}

func TestStartupNoTrayIsGuided(t *testing.T) {
	s := &Server{preferencesPath: filepath.Join(t.TempDir(), "p.json")}
	s.SetStartup(StartupHooks{
		Autostart:    func() autostart.State { return autostart.State{} },
		SetAutostart: func(bool) error { return autostart.ErrNoTray },
	})
	rec := doJSON(t, s, http.MethodPut, "/api/v1/startup", "/api/v1/startup", s.handleSetStartup, `{"start_at_login":true}`)
	var body struct {
		ErrorKey string `json:"error_key"`
		Hint     string `json:"hint"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if rec.Code != http.StatusConflict || body.ErrorKey != "startup.err.no_tray" || body.Hint == "" {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

// "Salir y desconectar todo" / `kongtrol down`: nothing is restored later.
func TestShutdownCallsStopAll(t *testing.T) {
	stopped, shut := false, make(chan struct{})
	s := &Server{onShutdown: func() { close(shut) }}
	s.SetStartup(StartupHooks{OnStopAll: func() { stopped = true }})
	rec := doJSON(t, s, http.MethodPost, "/api/v1/shutdown", "/api/v1/shutdown", s.handleShutdown, "")
	<-shut
	if rec.Code != http.StatusAccepted || !stopped {
		t.Fatalf("%d stopped=%v", rec.Code, stopped)
	}
}
