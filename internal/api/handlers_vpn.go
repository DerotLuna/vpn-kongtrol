package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
)

// GET /api/v1/vpns — list VPN profiles. Credentials are never returned, only
// whether a username/password is stored in the OS keychain for the profile.
// Certificate/key *paths* are returned (they are file locations, not secrets)
// so the editor can show them and send them back unchanged on update.
func (s *Server) handleListVPNs(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type vpnDTO struct {
		Name               string   `json:"name"`
		Type               string   `json:"type"`
		Host               string   `json:"host,omitempty"`
		Port               int      `json:"port,omitempty"`
		Server             string   `json:"server,omitempty"`
		Protocol           string   `json:"protocol,omitempty"`
		ConfigFile         string   `json:"config,omitempty"`
		Priority           int      `json:"priority"`
		Failover           []string `json:"failover"`
		AuthMethod         string   `json:"auth_method,omitempty"`
		Cert               string   `json:"cert,omitempty"`
		Key                string   `json:"key,omitempty"`
		Username           string   `json:"username,omitempty"`
		HasUsernameCred    bool     `json:"has_username_credential"`
		HasPasswordCred    bool     `json:"has_password_credential"`
		KillSwitchOverride string   `json:"kill_switch_override"` // "inherit" | "on" | "off"
		// Provider attach mode (forticlient, protonvpn, nordvpn, surfshark).
		AutoConnect bool   `json:"auto_connect"`
		Country     string `json:"country,omitempty"`
		City        string `json:"city,omitempty"`
		Group       string `json:"group,omitempty"`
		Interface   string `json:"interface,omitempty"`
		WaitTimeout string `json:"wait_timeout,omitempty"`
		AppPath     string `json:"app_path,omitempty"`
	}
	out := make([]vpnDTO, 0, len(cfg.VPNs))
	for name, v := range cfg.VPNs {
		override := "inherit"
		if v.KillSwitch != nil {
			if *v.KillSwitch {
				override = "on"
			} else {
				override = "off"
			}
		}
		out = append(out, vpnDTO{
			Name:               name,
			Type:               v.Type,
			Host:               v.Host,
			Port:               v.Port,
			Server:             v.Server,
			Protocol:           v.Protocol,
			ConfigFile:         v.ConfigFile,
			Priority:           v.Priority,
			Failover:           append([]string{}, v.Failover...),
			AuthMethod:         v.Auth.Method,
			Cert:               v.Auth.Cert,
			Key:                v.Auth.Key,
			Username:           v.Auth.Username,
			HasUsernameCred:    v.Auth.UsernameKeychain != "",
			HasPasswordCred:    v.Auth.PasswordKeychain != "",
			KillSwitchOverride: override,
			AutoConnect:        v.AutoConnect == nil || *v.AutoConnect,
			Country:            v.Country,
			City:               v.City,
			Group:              v.Group,
			Interface:          v.Interface,
			WaitTimeout:        v.WaitTimeout,
			AppPath:            v.AppPath,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, http.StatusOK, out)
}

// strVal returns the trimmed value of p, or "" when absent.
func strVal(p *string) string { return configedit.Str(p) }

// setTrimmed overwrites *dst with the trimmed *src when src is present.
func setTrimmed(dst, src *string) {
	if src != nil {
		*dst = strings.TrimSpace(*src)
	}
}

// POST /api/v1/vpns — create a new VPN profile. Writes config + OS keychain
// only; a newly created profile is not hot-registered with the running
// daemon (the shared adapters map has no safe runtime-mutation path — see
// cmd/kongtrol/main.go loadConfig), so the response flags restart_required.
func (s *Server) handleCreateVPN(w http.ResponseWriter, r *http.Request) {
	var req configedit.VPNProfile
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, errInvalidJSON)
		return
	}
	if strVal(req.Name) == "" {
		writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "api.err.name_required"})
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Same path as `kongtrol profile add` with no daemon.
	name, err := configedit.AddVPNProfile(cfg, req, s.saver(cfgPath))
	if err != nil {
		status := http.StatusBadRequest
		var ve *config.ValidationError
		if errors.As(err, &ve) && ve.Key == configedit.ErrKeyVPNExists {
			status = http.StatusConflict
		}
		writeLocalizedError(w, r, status, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "created", "profile": name, "restart_required": true})
}

// PUT /api/v1/vpns/{name} — partial update of an existing VPN profile: the
// stored VPNConfig is the base and only fields present in the body change.
// Same restart-required caveat as create.
func (s *Server) handleUpdateVPN(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "api.err.name_required"})
		return
	}
	var req configedit.VPNProfile
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, errInvalidJSON)
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	existing, exists := cfg.VPNs[name]
	if !exists {
		writeLocalizedError(w, r, http.StatusNotFound, &config.ValidationError{Key: "api.err.vpn_not_found", Args: []any{name}})
		return
	}
	// A different "name" in the body renames the profile (ADR-0042).
	if newName := strVal(req.Name); newName != "" && newName != name {
		if status, err := s.renameVPNProfile(cfg, cfgPath, name, newName, req); err != nil {
			writeLocalizedError(w, r, status, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "renamed", "profile": newName, "previous": name, "restart_required": true})
		return
	}
	if err := configedit.SaveVPNProfile(cfg, name, existing, req, s.saver(cfgPath)); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "profile": name, "restart_required": true})
}

// saver persists a config with saveRuntimeConfig (write + hot swap), for
// configedit's profile edits.
func (s *Server) saver(cfgPath string) func(*config.Config) error {
	return func(cfg *config.Config) error { return s.saveRuntimeConfig(cfgPath, cfg) }
}

// DELETE /api/v1/vpns/{name} — reject if the profile is still referenced by
// a policy or group, or if it's the last remaining profile.
func (s *Server) handleDeleteVPN(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	existing, ok := cfg.VPNs[name]
	if !ok {
		writeLocalizedError(w, r, http.StatusNotFound, &config.ValidationError{Key: "api.err.vpn_not_found", Args: []any{name}})
		return
	}
	if len(cfg.VPNs) <= 1 {
		writeLocalizedError(w, r, http.StatusConflict, &config.ValidationError{Key: "api.err.vpn_last", Args: []any{name}})
		return
	}
	// Every reference is listed at once: policies, groups, other profiles'
	// failover, scheduler rules (ADR-0042).
	if refs := findProfileRefs(cfg, name); !refs.empty() {
		writeLocalizedError(w, r, http.StatusConflict, inUseError(name, refs))
		return
	}

	// Trial-then-commit: the expected IP of the profile goes with it; the
	// rest must still validate.
	trial := *cfg
	trial.VPNs = make(map[string]config.VPNConfig, len(cfg.VPNs))
	for k, v := range cfg.VPNs {
		if k != name {
			trial.VPNs[k] = v
		}
	}
	if ips := cfg.Security.IntegrityCheck.ExpectedIPs; ips[name] != "" {
		m := make(map[string]string, len(ips))
		for k, v := range ips {
			if k != name {
				m[k] = v
			}
		}
		trial.Security.IntegrityCheck.ExpectedIPs = m
	}
	if err := config.Validate(&trial); err != nil {
		writeLocalizedError(w, r, http.StatusConflict, err)
		return
	}
	*cfg = trial
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var cleanupErrs []error
	for key, named := range map[string]bool{
		"username": existing.Auth.UsernameKeychain != "",
		"password": existing.Auth.PasswordKeychain != "",
	} {
		if !named {
			continue
		}
		// An entry that was never stored is already gone, not a failure.
		if err := config.DeleteCredential(name, key); err != nil && !errors.Is(err, config.ErrCredentialNotFound) {
			cleanupErrs = append(cleanupErrs, err)
		}
	}
	if err := errors.Join(cleanupErrs...); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("profile deleted but credential cleanup failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "profile": name, "restart_required": true})
}
