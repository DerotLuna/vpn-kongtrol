package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

func TestUpdatePreferencesPreservesCLIFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	if err := config.SavePreferences(path, &config.Preferences{
		Favorites:     []string{"office"},
		DefaultGroup:  "work",
		DashboardPort: 9800,
	}); err != nil {
		t.Fatal(err)
	}
	s := &Server{preferencesPath: path}
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/preferences",
		strings.NewReader(`{"language":"en","theme":"dark"}`),
	)
	rec := httptest.NewRecorder()
	s.handleUpdatePreferences(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	prefs, err := config.LoadPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	if prefs.Language != "en" || prefs.Theme != "dark" ||
		prefs.DefaultGroup != "work" || len(prefs.Favorites) != 1 ||
		prefs.DashboardPort != 9800 {
		t.Fatalf("unexpected preferences: %#v", prefs)
	}
}

func TestUpdatePreferencesRejectsUnsupportedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	s := &Server{preferencesPath: path}
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/preferences",
		strings.NewReader(`{"language":"fr"}`),
	)
	rec := httptest.NewRecorder()
	s.handleUpdatePreferences(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetPreferencesOnlyExposesDashboardFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	if err := config.SavePreferences(path, &config.Preferences{
		Favorites: []string{"private-profile"},
		Language:  "es",
		Theme:     "light",
	}); err != nil {
		t.Fatal(err)
	}
	s := &Server{preferencesPath: path}
	rec := httptest.NewRecorder()
	s.handleGetPreferences(rec, httptest.NewRequest(http.MethodGet, "/api/v1/preferences", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["language"] != "es" || got["theme"] != "light" || len(got) != 2 {
		t.Fatalf("unexpected response: %#v", got)
	}
}
