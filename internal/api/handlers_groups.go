package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// GET /api/v1/groups
func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]groupDTO, 0, len(cfg.Groups))
	for name, g := range cfg.Groups {
		out = append(out, groupDTO{Name: name, Profiles: g.Profiles})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, http.StatusOK, out)
}

type groupReq struct {
	Name     string   `json:"name"`
	Profiles []string `json:"profiles"`
}

// POST /api/v1/groups
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req groupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "group name is required")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, exists := cfg.Groups[req.Name]; exists {
		writeError(w, http.StatusConflict, "group already exists")
		return
	}
	if err := s.saveGroup(cfg, cfgPath, req.Name, req.Profiles); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "group": req.Name})
}

// PUT /api/v1/groups/{name}
func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing group name")
		return
	}
	var req groupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, exists := cfg.Groups[name]; !exists {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}
	if err := s.saveGroup(cfg, cfgPath, name, req.Profiles); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "group": name})
}

func (s *Server) saveGroup(cfg *config.Config, cfgPath, name string, profiles []string) error {
	clean := make([]string, 0, len(profiles))
	for _, p := range profiles {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := cfg.VPNs[p]; !ok {
			return fmt.Errorf("group references unknown profile %q", p)
		}
		clean = append(clean, p)
	}
	if len(clean) == 0 {
		return fmt.Errorf("group must include at least one profile")
	}
	if cfg.Groups == nil {
		cfg.Groups = make(map[string]config.GroupConfig)
	}
	cfg.Groups[name] = config.GroupConfig{Profiles: clean}
	return s.saveRuntimeConfig(cfgPath, cfg)
}

// DELETE /api/v1/groups/{name}
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, ok := cfg.Groups[name]; !ok {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}
	delete(cfg.Groups, name)
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "group": name})
}

// POST /api/v1/groups/{name}/connect — starts a connect for every profile in
// the group that isn't already connected/connecting, mirroring handleConnect
// per-profile through the shared orchestrator lifecycle.
func (s *Server) handleConnectGroup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	g, ok := cfg.Groups[name]
	if !ok {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}

	started := make([]string, 0, len(g.Profiles))
	alreadyConnected := make([]string, 0)
	for _, profile := range g.Profiles {
		adapter, ok := s.adapters[profile]
		if !ok {
			continue
		}
		if adapter.Status().Normalize() == vpn.StatusConnected {
			alreadyConnected = append(alreadyConnected, profile)
			continue
		}
		if s.connectProfile == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Minute)
		if !s.startPendingConnect(profile, cancel) {
			cancel()
			continue
		}
		started = append(started, profile)
		go func(p string, ctx context.Context) {
			defer s.clearPendingConnect(p)
			_ = s.connectProfile(ctx, p)
		}(profile, ctx)
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":            "connecting",
		"group":             name,
		"started":           started,
		"already_connected": alreadyConnected,
	})
}

// POST /api/v1/groups/{name}/disconnect — disconnects every connected
// profile in the group, collecting per-profile errors instead of stopping
// at the first failure.
func (s *Server) handleDisconnectGroup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	g, ok := cfg.Groups[name]
	if !ok {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}

	var errs []string
	if s.disconnectProfile == nil {
		writeError(w, http.StatusServiceUnavailable, "connection service unavailable")
		return
	}
	for _, profile := range g.Profiles {
		if _, ok := s.adapters[profile]; !ok {
			continue
		}
		s.cancelPendingConnect(profile)
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		disconnectErr := s.disconnectProfile(ctx, profile)
		cancel()
		if disconnectErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", profile, disconnectErr))
		}
	}
	if len(errs) > 0 {
		writeError(w, http.StatusInternalServerError, strings.Join(errs, "; "))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected", "group": name})
}

type settingsDTO struct {
	// Read-only — the dashboard's own bind/port. Changing it from the page
	// serving this request would cut the connection mid-response, so it's
	// only changeable via `kongtrol config dashboard set-port/set-bind`
	// (see cmd/kongtrol preferences.go); shown here for visibility only.
	DashboardBind string `json:"dashboard_bind"`
	DashboardPort int    `json:"dashboard_port"`

	HealthCheckInterval string `json:"health_check_interval"`
	HealthCheckTimeout  string `json:"health_check_timeout"`

	SchedulerEnabled  bool   `json:"scheduler_enabled"`
	SchedulerInterval string `json:"scheduler_interval"`

	SplitDNSEnabled  bool   `json:"split_dns_enabled"`
	SplitDNSInterval string `json:"split_dns_interval"`

	KillSwitchMode     string `json:"kill_switch_mode"`
	KillSwitchAllowLAN bool   `json:"kill_switch_allow_lan"`

	DNSGuardFallbackDNS string `json:"dns_guard_fallback_dns"`

	LeakDetectionEnabled  bool   `json:"leak_detection_enabled"`
	LeakDetectionInterval string `json:"leak_detection_interval"`
	LeakDetectionAction   string `json:"leak_detection_action"`

	AuditLogPath      string `json:"audit_log_path"`
	AuditLogMaxSizeMB int    `json:"audit_log_max_size_mb"`
	AuditLogSign      bool   `json:"audit_log_sign"`
}
