package config

import (
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"
)

func TestPreferencesRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	want := &Preferences{
		Favorites:     []string{"work"},
		DefaultGroup:  "office",
		Language:      "es",
		Theme:         "dark",
		DashboardPort: 9741,
		DashboardBind: "127.0.0.1",
	}
	if err := SavePreferences(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Language != want.Language || got.Theme != want.Theme ||
		got.DefaultGroup != want.DefaultGroup || !slices.Equal(got.Favorites, want.Favorites) {
		t.Fatalf("round trip mismatch: got %#v, want %#v", got, want)
	}
}

func TestUpdatePreferencesSerializesReadModifyWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	const updates = 20
	var wg sync.WaitGroup
	for i := range updates {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := UpdatePreferences(path, func(prefs *Preferences) error {
				prefs.Favorites = append(prefs.Favorites, fmt.Sprintf("profile-%d", i))
				return nil
			})
			if err != nil {
				t.Errorf("update preferences: %v", err)
			}
		}()
	}
	wg.Wait()

	prefs, err := LoadPreferences(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(prefs.Favorites) != updates {
		t.Fatalf("got %d favorites, want %d", len(prefs.Favorites), updates)
	}
}

func TestValidatePreferences(t *testing.T) {
	for _, prefs := range []*Preferences{
		{Language: "fr"},
		{Theme: "system"},
		{DashboardPort: 70000},
	} {
		if err := ValidatePreferences(prefs); err == nil {
			t.Fatalf("ValidatePreferences(%#v) succeeded, want error", prefs)
		}
	}
}
