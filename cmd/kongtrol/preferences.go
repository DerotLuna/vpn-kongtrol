package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

type preferences struct {
	Favorites    []string `json:"favorites"`
	DefaultGroup string   `json:"default_group"`
	Language     string   `json:"language,omitempty"` // "es" or "en"
	// DashboardPort/DashboardBind are a machine-local override for the
	// embedded dashboard's listen address — set via `kongtrol config
	// dashboard set-port/set-bind`, not editable from the dashboard itself
	// (changing the port from the page serving that request would cut the
	// connection mid-response). Takes precedence over kongtrol.yaml's
	// monitor.dashboard settings when non-zero/non-empty.
	DashboardPort int    `json:"dashboard_port,omitempty"`
	DashboardBind string `json:"dashboard_bind,omitempty"`
}

func preferencesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kongtrol", "preferences.json")
}

func loadPreferences() (*preferences, error) {
	path := preferencesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &preferences{}, nil
		}
		return nil, fmt.Errorf("preferences: read: %w", err)
	}
	var p preferences
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("preferences: decode: %w", err)
	}
	return &p, nil
}

func savePreferences(p *preferences) error {
	path := preferencesPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("preferences: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("preferences: encode: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("preferences: write: %w", err)
	}
	return nil
}

// applyDashboardPreferences overrides cfg.Monitor.Dashboard with any
// machine-local preference set via `kongtrol config dashboard`. Called right
// after config.Load in loadConfig(), before anything reads the dashboard
// bind/port.
func applyDashboardPreferences(cfg *config.Config) {
	p, err := loadPreferences()
	if err != nil {
		return
	}
	if p.DashboardPort > 0 {
		cfg.Monitor.Dashboard.Port = p.DashboardPort
	}
	if strings.TrimSpace(p.DashboardBind) != "" {
		cfg.Monitor.Dashboard.Bind = p.DashboardBind
	}
}

func addFavorite(name string) error {
	p, err := loadPreferences()
	if err != nil {
		return err
	}
	if !slices.Contains(p.Favorites, name) {
		p.Favorites = append(p.Favorites, name)
	}
	return savePreferences(p)
}

func removeFavorite(name string) error {
	p, err := loadPreferences()
	if err != nil {
		return err
	}
	out := make([]string, 0, len(p.Favorites))
	for _, n := range p.Favorites {
		if n != name {
			out = append(out, n)
		}
	}
	p.Favorites = out
	return savePreferences(p)
}
