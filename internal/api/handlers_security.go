package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

type leakCheckStatus struct {
	State     string    `json:"state"`
	HasLeak   bool      `json:"has_leak"`
	PublicIP  string    `json:"public_ip,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// leakCheckDTO renders a lookup result (nil when there is none).
func leakCheckDTO(lr *security.LeakResult) *leakCheckStatus {
	if lr == nil || lr.Disabled {
		return nil
	}
	state := "clean"
	if lr.HasLeak {
		state = "leak"
	} else if lr.PublicIP == "" && lr.Reason != "" {
		state = "error"
	}
	return &leakCheckStatus{
		State:     state,
		HasLeak:   lr.HasLeak,
		PublicIP:  lr.PublicIP,
		Reason:    lr.Reason,
		CheckedAt: lr.CheckedAt,
	}
}

// POST /api/v1/security/publicip  {"enabled": bool} — opt in/out of the
// external public IP lookup. Turning it off also clears the legacy
// leak_detection.enabled. The response carries the host so clients show
// "Usa un servicio externo (<host>) que ve tu IP pública".
func (s *Server) handleTogglePublicIP(w http.ResponseWriter, r *http.Request) {
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
	cfg.SetPublicIPCheck(req.Enabled)
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if s.onSecurityToggle != nil {
		s.onSecurityToggle(cfg)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":                  "updated",
		"public_ip_check_enabled": req.Enabled,
		"public_ip_check_host":    cfg.PublicIPHost(),
	})
}

// POST /api/v1/security/publicip/check — the user's "Revisar": one lookup
// now. 409 with a translated hint when the lookup is off (no request made).
func (s *Server) handleCheckPublicIP(w http.ResponseWriter, r *http.Request) {
	if !s.leakTest.Enabled() {
		writeLocalizedError(w, r, http.StatusConflict, &config.ValidationError{Key: "publicip.err.disabled"})
		return
	}
	res := s.leakTest.CheckNow()
	writeJSON(w, http.StatusOK, map[string]any{
		"public_ip_check_enabled": true,
		"public_ip_check_host":    s.leakTest.Host(),
		"leak_check":              leakCheckDTO(&res),
	})
}

// GET /api/v1/security/status
func (s *Server) handleSecurityStatus(w http.ResponseWriter, r *http.Request) {
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
		// PublicIPCheckEnabled: the user opted in to the external public IP
		// lookup (off by default). PublicIPCheckHost is the service it uses.
		PublicIPCheckEnabled bool   `json:"public_ip_check_enabled"`
		PublicIPCheckHost    string `json:"public_ip_check_host"`
		// KillSwitchStatus is the split-tunnel kill switch detail (which
		// tunnels it protects / blocks); render with its state key.
		KillSwitchStatus *security.KillSwitchStatus `json:"kill_switch_status,omitempty"`
	}

	sec := s.securityState()
	killActive, killEnabled, ksStatus := sec.killActive, sec.killEnabled, sec.ksStatus
	dnsActive, dnsEnabled := sec.dnsActive, sec.dnsEnabled
	status := secStatus{
		KillSwitch:           killActive,
		KillSwitchEnabled:    killEnabled,
		KillSwitchState:      configuredRuntimeState(killEnabled, killActive, "armed"),
		DNSGuard:             dnsActive,
		DNSGuardEnabled:      dnsEnabled,
		DNSGuardState:        configuredRuntimeState(dnsEnabled, dnsActive, "active"),
		LeakDetectionEnabled: s.leakTest.Enabled(),
		LeakCheckAvailable:   s.leakTest.Enabled(),
		PublicIPCheckEnabled: s.leakTest.Enabled(),
		PublicIPCheckHost:    s.leakTest.Host(),
		KillSwitchStatus:     ksStatus,
	}
	if ksStatus != nil {
		switch ksStatus.State {
		case security.KillSwitchActive:
			status.KillSwitchState = "active"
		case security.KillSwitchArmed:
			status.KillSwitchState = "armed"
		default:
			status.KillSwitchState = "disabled"
		}
	}
	if cfg, _, err := s.loadRuntimeConfig(); err == nil {
		status.LeakDetectionConfigured = cfg.PublicIPCheckEnabled()
	}

	status.LeakCheck = leakCheckDTO(s.leakTest.LastResult())

	writeJSON(w, http.StatusOK, status)
}

// securityRuntime is the daemon's live kill switch / DNS guard state, shared
// by GET /api/v1/security/status and GET /api/v1/doctor.
type securityRuntime struct {
	killActive, killEnabled bool
	ksStatus                *security.KillSwitchStatus
	dnsActive, dnsEnabled   bool
}

func (s *Server) securityState() securityRuntime {
	out := securityRuntime{
		killActive:  s.ks != nil && s.ks.IsEnabled(),
		killEnabled: s.killSwitchOn.Load(),
		dnsActive:   s.dnsMgr != nil && s.dnsMgr.IsActive(),
		dnsEnabled:  s.dnsGuardOn.Load(),
	}
	if s.ksStatus != nil {
		st := s.ksStatus()
		out.ksStatus = &st
		// active = blocking a dropped tunnel's destinations now.
		out.killActive = st.State == security.KillSwitchActive
		out.killEnabled = out.killEnabled || st.State != security.KillSwitchOff
	}
	return out
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

type groupDTO struct {
	Name     string   `json:"name"`
	Profiles []string `json:"profiles"`
}

// GET /api/v1/groups
