package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Preferences contains settings that belong to one machine/user rather than
// the portable kongtrol.yaml configuration.
type Preferences struct {
	Favorites     []string `json:"favorites,omitempty"`
	DefaultGroup  string   `json:"default_group,omitempty"`
	Language      string   `json:"language,omitempty"`
	Theme         string   `json:"theme,omitempty"`
	DashboardPort int      `json:"dashboard_port,omitempty"`
	DashboardBind string   `json:"dashboard_bind,omitempty"`
}

var preferencesMu sync.Mutex

// PreferencesPath returns path when supplied, or the current user's default
// preferences file otherwise.
func PreferencesPath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".kongtrol", "preferences.json"), nil
}

// LoadPreferences loads the machine-local preferences. A missing file is an
// empty preference set.
func LoadPreferences(path string) (*Preferences, error) {
	preferencesMu.Lock()
	defer preferencesMu.Unlock()
	return loadPreferences(path)
}

func loadPreferences(path string) (*Preferences, error) {
	resolved, err := PreferencesPath(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(resolved)
	if errors.Is(err, os.ErrNotExist) {
		return &Preferences{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read preferences: %w", err)
	}
	var prefs Preferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, fmt.Errorf("parse preferences: %w", err)
	}
	if err := ValidatePreferences(&prefs); err != nil {
		return nil, err
	}
	return &prefs, nil
}

// SavePreferences validates and atomically persists machine-local preferences.
func SavePreferences(path string, prefs *Preferences) error {
	preferencesMu.Lock()
	defer preferencesMu.Unlock()
	return savePreferences(path, prefs)
}

func savePreferences(path string, prefs *Preferences) error {
	if err := ValidatePreferences(prefs); err != nil {
		return err
	}
	resolved, err := PreferencesPath(path)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return fmt.Errorf("encode preferences: %w", err)
	}
	data = append(data, '\n')
	return WriteFileAtomic(resolved, data, 0o600)
}

// UpdatePreferences performs a serialized read-modify-write operation.
func UpdatePreferences(path string, update func(*Preferences) error) (*Preferences, error) {
	preferencesMu.Lock()
	defer preferencesMu.Unlock()
	prefs, err := loadPreferences(path)
	if err != nil {
		return nil, err
	}
	if err := update(prefs); err != nil {
		return nil, err
	}
	if err := savePreferences(path, prefs); err != nil {
		return nil, err
	}
	return prefs, nil
}

// ValidatePreferences rejects values that no Kongtrol frontend understands.
func ValidatePreferences(prefs *Preferences) error {
	if prefs == nil {
		return errors.New("preferences are nil")
	}
	if prefs.Language != "" && prefs.Language != "es" && prefs.Language != "en" {
		return fmt.Errorf("unsupported language %q", prefs.Language)
	}
	if prefs.Theme != "" && prefs.Theme != "light" && prefs.Theme != "dark" {
		return fmt.Errorf("unsupported theme %q", prefs.Theme)
	}
	if prefs.DashboardPort < 0 || prefs.DashboardPort > 65535 {
		return fmt.Errorf("dashboard port must be between 1 and 65535")
	}
	return nil
}
