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
		LeakDetectionEnabled:  cfg.Security.LeakDetection.Enabled,
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
	var req settingsDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	for _, d := range []string{req.HealthCheckInterval, req.HealthCheckTimeout, req.SchedulerInterval, req.SplitDNSInterval, req.LeakDetectionInterval} {
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
	trial.Monitor.HealthCheck.Interval = req.HealthCheckInterval
	trial.Monitor.HealthCheck.Timeout = req.HealthCheckTimeout
	trial.Monitor.Scheduler.Enabled = req.SchedulerEnabled
	trial.Monitor.Scheduler.Interval = req.SchedulerInterval
	trial.Monitor.SplitDNS.Enabled = req.SplitDNSEnabled
	trial.Monitor.SplitDNS.Interval = req.SplitDNSInterval
	trial.Security.KillSwitch.Mode = req.KillSwitchMode
	trial.Security.KillSwitch.AllowLAN = req.KillSwitchAllowLAN
	trial.Security.DNSGuard.FallbackDNS = req.DNSGuardFallbackDNS
	trial.Security.LeakDetection.Enabled = req.LeakDetectionEnabled
	trial.Security.LeakDetection.Interval = req.LeakDetectionInterval
	trial.Security.LeakDetection.Action = req.LeakDetectionAction
	trial.Security.AuditLog.Path = req.AuditLogPath
	trial.Security.AuditLog.MaxSizeMB = req.AuditLogMaxSizeMB
	trial.Security.AuditLog.Sign = req.AuditLogSign
	if err := config.Validate(&trial); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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
