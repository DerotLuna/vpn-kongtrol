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

// POST /api/v1/groups/{name}/reload — re-reads kongtrol.yaml from disk
// (hot-swapping the policy engine along the way, same as
// handlePolicyReload), then restarts every currently-connected profile in
// the group in place: disconnect then reconnect through the existing
// adapters, so routes/DNS/kill-switch settings edited by hand for the
// group's profiles take effect without a full daemon restart. Disconnected
// profiles in the group are left alone (nothing to restart). Profiles whose
// VPN type isn't already registered in the running daemon's adapters map —
// e.g. a brand-new profile added by the hand edit — can't be restarted this
// way (see CLAUDE.md's restart_required note on VPN CRUD: the adapters map
// is built once at boot and shared, unsynchronized, with the collector and
// watchdog goroutines) and are reported back as restart_required instead of
// silently skipped.
func (s *Server) handleReloadGroup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, err := s.reloadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	g, ok := cfg.Groups[name]
	if !ok {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}

	restarted, skipped, missing, errs := s.restartProfilesInPlace(r, g.Profiles)
	if len(missing) > 0 {
		writeJSON(w, http.StatusConflict, map[string]any{
			"status":           "restart_required",
			"group":            name,
			"missing_profiles": missing,
			"message":          "these profiles are not registered with the running daemon (likely added by a hand edit) — a full process restart ('kongtrol down' then 'kongtrol up') is required before they can connect",
		})
		return
	}
	if errs != nil {
		writeError(w, http.StatusInternalServerError, strings.Join(errs, "; "))
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":    "restarting",
		"group":     name,
		"restarted": restarted,
		"skipped":   skipped,
	})
}

// restartProfilesInPlace disconnects then reconnects every currently
// connected profile in the given list through the daemon's real
// connectProfile/disconnectProfile closures, so Watchdog.MarkIntended/
// MarkActive and DNSManager.OnConnect/OnDisconnect stay correctly balanced.
// Profiles not registered in s.adapters (e.g. a brand-new VPN type added by
// a hand edit — the adapters map is built once at boot, per CLAUDE.md) are
// returned as missing instead of silently skipped; already-disconnected
// profiles are returned as skipped. Shared by handleReloadGroup (whole
// group) and handleReloadTunnel (single profile).
func (s *Server) restartProfilesInPlace(r *http.Request, profiles []string) (restarted, skipped, missing []string, errs []string) {
	for _, p := range profiles {
		if _, ok := s.adapters[p]; !ok {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		return nil, nil, missing, nil
	}

	if s.connectProfile == nil || s.disconnectProfile == nil {
		return nil, nil, nil, []string{"connection service unavailable"}
	}

	restarted = make([]string, 0, len(profiles))
	skipped = make([]string, 0, len(profiles))
	for _, profile := range profiles {
		adapter, ok := s.adapters[profile]
		if !ok || adapter.Status().Normalize() != vpn.StatusConnected {
			skipped = append(skipped, profile)
			continue
		}

		s.cancelPendingConnect(profile)
		disconnectCtx, disconnectCancel := context.WithTimeout(r.Context(), 30*time.Second)
		disconnectErr := s.disconnectProfile(disconnectCtx, profile)
		disconnectCancel()
		if disconnectErr != nil {
			errs = append(errs, fmt.Sprintf("%s: disconnect: %v", profile, disconnectErr))
			continue
		}

		connectCtx, connectCancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Minute)
		if !s.startPendingConnect(profile, connectCancel) {
			connectCancel()
			errs = append(errs, fmt.Sprintf("%s: connect already in progress", profile))
			continue
		}
		restarted = append(restarted, profile)
		go func(p string, ctx context.Context) {
			defer s.clearPendingConnect(p)
			_ = s.connectProfile(ctx, p)
		}(profile, connectCtx)
	}
	return restarted, skipped, nil, errs
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
