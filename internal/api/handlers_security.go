package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// GET /api/v1/security/status
func (s *Server) handleSecurityStatus(w http.ResponseWriter, r *http.Request) {
	type leakCheckStatus struct {
		State     string    `json:"state"`
		HasLeak   bool      `json:"has_leak"`
		PublicIP  string    `json:"public_ip,omitempty"`
		Reason    string    `json:"reason,omitempty"`
		CheckedAt time.Time `json:"checked_at"`
	}
	type secStatus struct {
		KillSwitch              bool             `json:"kill_switch"`
		KillSwitchEnabled       bool             `json:"kill_switch_enabled"`
		KillSwitchState         string           `json:"kill_switch_state"`
		DNSGuard                bool             `json:"dns_guard"`
		DNSGuardEnabled         bool             `json:"dns_guard_enabled"`
		DNSGuardState           string           `json:"dns_guard_state"`
		LeakDetectionEnabled    bool             `json:"leak_detection_enabled"`
		LeakDetectionConfigured bool             `json:"leak_detection_configured"`
		LeakCheckAvailable      bool             `json:"leak_check_available"`
		LeakCheck               *leakCheckStatus `json:"leak_check,omitempty"`
	}

	killActive := s.ks != nil && s.ks.IsEnabled()
	killEnabled := s.killSwitchOn.Load()
	dnsActive := s.dnsMgr != nil && s.dnsMgr.IsActive()
	dnsEnabled := s.dnsGuardOn.Load()
	status := secStatus{
		KillSwitch:           killActive,
		KillSwitchEnabled:    killEnabled,
		KillSwitchState:      configuredRuntimeState(killEnabled, killActive, "armed"),
		DNSGuard:             dnsActive,
		DNSGuardEnabled:      dnsEnabled,
		DNSGuardState:        configuredRuntimeState(dnsEnabled, dnsActive, "active"),
		LeakDetectionEnabled: s.leakTest != nil,
		LeakCheckAvailable:   s.leakTest != nil,
	}
	if cfg, _, err := s.loadRuntimeConfig(); err == nil {
		status.LeakDetectionConfigured = cfg.Security.LeakDetection.Enabled
	}

	if s.leakTest != nil {
		lr := s.leakTest.LastResult()
		if lr != nil {
			state := "clean"
			if lr.HasLeak {
				state = "leak"
			} else if lr.PublicIP == "" && lr.Reason != "" {
				state = "error"
			}
			status.LeakCheck = &leakCheckStatus{
				State:     state,
				HasLeak:   lr.HasLeak,
				PublicIP:  lr.PublicIP,
				Reason:    lr.Reason,
				CheckedAt: lr.CheckedAt,
			}
		}
	}

	writeJSON(w, http.StatusOK, status)
}

func configuredRuntimeState(configured, active bool, activeState string) string {
	if !configured {
		return "disabled"
	}
	if active {
		return activeState
	}
	return "idle"
}

type preferenceDTO struct {
	Language string `json:"language"`
	Theme    string `json:"theme"`
}

// GET /api/v1/preferences
func (s *Server) handleGetPreferences(w http.ResponseWriter, _ *http.Request) {
	prefs, err := config.LoadPreferences(s.preferencesPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preferenceDTO{Language: prefs.Language, Theme: prefs.Theme})
}

// PUT /api/v1/preferences — dashboard-owned fields only. Other machine-local
// preferences are preserved by the serialized read-modify-write operation.
func (s *Server) handleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Language *string `json:"language"`
		Theme    *string `json:"theme"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	prefs, err := config.UpdatePreferences(s.preferencesPath, func(prefs *config.Preferences) error {
		if req.Language != nil {
			prefs.Language = *req.Language
		}
		if req.Theme != nil {
			prefs.Theme = *req.Theme
		}
		return config.ValidatePreferences(prefs)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preferenceDTO{Language: prefs.Language, Theme: prefs.Theme})
}

// POST /api/v1/security/killswitch  {"enabled": bool}
func (s *Server) handleToggleKillSwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.Security.KillSwitch.Enabled = req.Enabled
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.killSwitchOn.Store(req.Enabled)
	if s.onSecurityToggle != nil {
		s.onSecurityToggle(cfg)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "kill_switch_enabled": req.Enabled})
}

// POST /api/v1/security/dnsguard  {"enabled": bool}
func (s *Server) handleToggleDNSGuard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.Security.DNSGuard.Enabled = req.Enabled
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.dnsGuardOn.Store(req.Enabled)
	if s.onSecurityToggle != nil {
		s.onSecurityToggle(cfg)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "dns_guard_enabled": req.Enabled})
}

// GET /api/v1/vpns — list VPN profiles. Credentials are never returned, only
// whether a username/password is stored in the OS keychain for the profile.

// GET /api/v1/audit?limit=200&profile=&level= — recent audit log events,
// newest first. Reuses the same reader the CLI `logs` command uses
// (internal/security.ReadRecentAuditEvents) so both stay in sync.
func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cfg.Security.AuditLog.Path == "" {
		writeJSON(w, http.StatusOK, []security.AuditEvent{})
		return
	}

	events, err := security.ReadRecentAuditEvents(cfg.Security.AuditLog.Path, 2)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	profile := strings.TrimSpace(r.URL.Query().Get("profile"))
	level := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("level")))
	filtered := make([]security.AuditEvent, 0, len(events))
	for _, ev := range events {
		if profile != "" && !strings.EqualFold(ev.Profile, profile) {
			continue
		}
		if level != "" && strings.ToUpper(ev.Level) != level {
			continue
		}
		filtered = append(filtered, ev)
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Timestamp.After(filtered[j].Timestamp) })

	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	writeJSON(w, http.StatusOK, filtered)
}

type groupDTO struct {
	Name     string   `json:"name"`
	Profiles []string `json:"profiles"`
}

// GET /api/v1/groups
