package config

import "strings"

// ApplyDashboardPreferences overrides cfg.Monitor.Dashboard with any
// machine-local preference set via `kongtrol config dashboard`
// (~/.kongtrol/preferences.json). Every client of the daemon's local API
// (CLI, tray) must call it right after Load, before reading the dashboard
// bind/port, or it talks to the wrong address. A missing or unreadable
// preferences file leaves cfg unchanged.
func ApplyDashboardPreferences(cfg *Config) {
	if cfg == nil {
		return
	}
	p, err := LoadPreferences("")
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
