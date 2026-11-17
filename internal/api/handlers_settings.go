package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// GET /api/v1/settings
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settingsToDTO(cfg))
}

func settingsToDTO(cfg *config.Config) settingsDTO {
	return settingsDTO{
		DashboardBind:         cfg.Monitor.Dashboard.Bind,
		DashboardPort:         cfg.Monitor.Dashboard.Port,
		HealthCheckInterval:   cfg.Monitor.HealthCheck.Interval,
		HealthCheckTimeout:    cfg.Monitor.HealthCheck.Timeout,
		SchedulerEnabled:      cfg.Monitor.Scheduler.Enabled,
		SchedulerInterval:     cfg.Monitor.Scheduler.Interval,
		SplitDNSEnabled:       cfg.Monitor.SplitDNS.Enabled,
		SplitDNSInterval:      cfg.Monitor.SplitDNS.Interval,
		KillSwitchMode:        cfg.Security.KillSwitch.Mode,
		KillSwitchAllowLAN:    cfg.Security.KillSwitch.AllowLAN,
		DNSGuardFallbackDNS:   cfg.Security.DNSGuard.FallbackDNS,
		LeakDetectionEnabled:  cfg.PublicIPCheckEnabled(),
		PublicIPCheckEnabled:  cfg.PublicIPCheckEnabled(),
		PublicIPEndpoints:     cfg.PublicIPEndpoints(),
		LeakDetectionInterval: cfg.Security.LeakDetection.Interval,
		LeakDetectionAction:   cfg.Security.LeakDetection.Action,
		AuditLogPath:          cfg.Security.AuditLog.Path,
		AuditLogMaxSizeMB:     cfg.Security.AuditLog.MaxSizeMB,
		AuditLogSign:          cfg.Security.AuditLog.Sign,
	}
}

// PUT /api/v1/settings — everything except the dashboard bind/port, which is
// read-only here (see settingsDTO comment).
func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req settingsPatch
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	for _, p := range []*string{req.HealthCheckInterval, req.HealthCheckTimeout, req.SchedulerInterval, req.SplitDNSInterval, req.LeakDetectionInterval} {
		d := strVal(p)
		if d == "" {
			continue
		}
		if _, err := time.ParseDuration(d); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid duration %q: %v", d, err))
			return
		}
	}

	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	trial := *cfg
	req.applyTo(&trial)
	if err := config.Validate(&trial); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, err)
		return
	}

	*cfg = trial
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if s.onSecurityToggle != nil {
		s.onSecurityToggle(cfg)
	}
	writeJSON(w, http.StatusOK, settingsToDTO(cfg))
}

// PUT /api/v1/vpns/{name}/killswitch  {"override": "inherit"|"on"|"off"}
// Sets the per-profile kill switch override (config.VPNConfig.KillSwitch).
// "inherit" clears the override so the profile falls back to the global
// security.kill_switch.enabled setting (see KillSwitchService.profileKillSwitchEnabled).
func (s *Server) handleSetProfileKillSwitch(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		Override string `json:"override"`
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
	vc, ok := cfg.VPNs[name]
	if !ok {
		writeError(w, http.StatusNotFound, "vpn profile not found")
		return
	}
	switch strings.ToLower(strings.TrimSpace(req.Override)) {
	case "on":
		v := true
		vc.KillSwitch = &v
	case "off":
		v := false
		vc.KillSwitch = &v
	case "inherit", "":
		vc.KillSwitch = nil
	default:
		writeError(w, http.StatusBadRequest, "override must be one of: inherit, on, off")
		return
	}
	cfg.VPNs[name] = vc
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if s.onSecurityToggle != nil {
		s.onSecurityToggle(cfg)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "profile": name})
}

type scheduleRuleDTO struct {
	Name     string   `json:"name"`
	Profiles []string `json:"profiles"`
	Weekdays []string `json:"weekdays,omitempty"`
	Start    string   `json:"start,omitempty"`
	End      string   `json:"end,omitempty"`
}

// settingsPatch is the PUT /api/v1/settings body: the editable subset of
// settingsDTO with pointer fields, so a field absent from the request keeps
// its stored value instead of being reset to the zero value. The dashboard
// bind/port are intentionally not part of it (read-only, see settingsDTO).
type settingsPatch struct {
	HealthCheckInterval   *string   `json:"health_check_interval"`
	HealthCheckTimeout    *string   `json:"health_check_timeout"`
	SchedulerEnabled      *bool     `json:"scheduler_enabled"`
	SchedulerInterval     *string   `json:"scheduler_interval"`
	SplitDNSEnabled       *bool     `json:"split_dns_enabled"`
	SplitDNSInterval      *string   `json:"split_dns_interval"`
	KillSwitchMode        *string   `json:"kill_switch_mode"`
	KillSwitchAllowLAN    *bool     `json:"kill_switch_allow_lan"`
	DNSGuardFallbackDNS   *string   `json:"dns_guard_fallback_dns"`
	LeakDetectionEnabled  *bool     `json:"leak_detection_enabled"`
	LeakDetectionInterval *string   `json:"leak_detection_interval"`
	LeakDetectionAction   *string   `json:"leak_detection_action"`
	PublicIPCheckEnabled  *bool     `json:"public_ip_check_enabled"`
	PublicIPEndpoints     *[]string `json:"public_ip_endpoints"`
	AuditLogPath          *string   `json:"audit_log_path"`
	AuditLogMaxSizeMB     *int      `json:"audit_log_max_size_mb"`
	AuditLogSign          *bool     `json:"audit_log_sign"`
}

func setBool(dst *bool, src *bool) {
	if src != nil {
		*dst = *src
	}
}

// applyTo overlays the fields present in p onto cfg.
func (p settingsPatch) applyTo(cfg *config.Config) {
	setTrimmed(&cfg.Monitor.HealthCheck.Interval, p.HealthCheckInterval)
	setTrimmed(&cfg.Monitor.HealthCheck.Timeout, p.HealthCheckTimeout)
	setBool(&cfg.Monitor.Scheduler.Enabled, p.SchedulerEnabled)
	setTrimmed(&cfg.Monitor.Scheduler.Interval, p.SchedulerInterval)
	setBool(&cfg.Monitor.SplitDNS.Enabled, p.SplitDNSEnabled)
	setTrimmed(&cfg.Monitor.SplitDNS.Interval, p.SplitDNSInterval)
	setTrimmed(&cfg.Security.KillSwitch.Mode, p.KillSwitchMode)
	setBool(&cfg.Security.KillSwitch.AllowLAN, p.KillSwitchAllowLAN)
	setTrimmed(&cfg.Security.DNSGuard.FallbackDNS, p.DNSGuardFallbackDNS)
	// leak_detection_enabled is the legacy name of the public IP opt-in.
	if p.LeakDetectionEnabled != nil {
		cfg.SetPublicIPCheck(*p.LeakDetectionEnabled)
	}
	if p.PublicIPCheckEnabled != nil {
		cfg.SetPublicIPCheck(*p.PublicIPCheckEnabled)
	}
	if p.PublicIPEndpoints != nil {
		var eps []string
		for _, ep := range *p.PublicIPEndpoints {
			if ep = strings.TrimSpace(ep); ep != "" {
				eps = append(eps, ep)
			}
		}
		if len(eps) == 1 && eps[0] == config.DefaultPublicIPEndpoint {
			eps = nil // the default needs no entry
		}
		cfg.Security.PublicIPCheck.Endpoints = eps
	}
	setTrimmed(&cfg.Security.LeakDetection.Interval, p.LeakDetectionInterval)
	setTrimmed(&cfg.Security.LeakDetection.Action, p.LeakDetectionAction)
	setTrimmed(&cfg.Security.AuditLog.Path, p.AuditLogPath)
	if p.AuditLogMaxSizeMB != nil {
		cfg.Security.AuditLog.MaxSizeMB = *p.AuditLogMaxSizeMB
	}
	setBool(&cfg.Security.AuditLog.Sign, p.AuditLogSign)
}
