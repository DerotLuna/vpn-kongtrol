package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// GET /api/v1/vpns — list VPN profiles. Credentials are never returned, only
// whether a username/password is stored in the OS keychain for the profile.
func (s *Server) handleListVPNs(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type vpnDTO struct {
		Name               string `json:"name"`
		Type               string `json:"type"`
		Host               string `json:"host,omitempty"`
		Port               int    `json:"port,omitempty"`
		Server             string `json:"server,omitempty"`
		Protocol           string `json:"protocol,omitempty"`
		ConfigFile         string `json:"config,omitempty"`
		Priority           int    `json:"priority"`
		AuthMethod         string `json:"auth_method,omitempty"`
		Username           string `json:"username,omitempty"`
		HasUsernameCred    bool   `json:"has_username_credential"`
		HasPasswordCred    bool   `json:"has_password_credential"`
		KillSwitchOverride string `json:"kill_switch_override"` // "inherit" | "on" | "off"
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
			AuthMethod:         v.Auth.Method,
			Username:           v.Auth.Username,
			HasUsernameCred:    v.Auth.UsernameKeychain != "",
			HasPasswordCred:    v.Auth.PasswordKeychain != "",
			KillSwitchOverride: override,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, http.StatusOK, out)
}

type vpnProfileReq struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Server     string `json:"server"`
	Protocol   string `json:"protocol"`
	ConfigFile string `json:"config"`
	Priority   int    `json:"priority"`
	AuthMethod string `json:"auth_method"`
	Cert       string `json:"cert"`
	Key        string `json:"key"`
	Username   string `json:"username"`
	// Password is plaintext in the request only; it is written to the OS
	// keychain and never persisted to the YAML config.
	Password string `json:"password"`
}

// POST /api/v1/vpns — create a new VPN profile. Writes config + OS keychain
// only; a newly created profile is not hot-registered with the running
// daemon (the shared adapters map has no safe runtime-mutation path — see
// cmd/kongtrol/main.go loadConfig), so the response flags restart_required.
func (s *Server) handleCreateVPN(w http.ResponseWriter, r *http.Request) {
	var req vpnProfileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "profile name is required")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, exists := cfg.VPNs[req.Name]; exists {
		writeError(w, http.StatusConflict, "vpn profile already exists")
		return
	}
	if err := s.saveVPNProfile(cfg, cfgPath, req.Name, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "created", "profile": req.Name, "restart_required": true})
}

// PUT /api/v1/vpns/{name} — update an existing VPN profile. Same
// restart-required caveat as create.
func (s *Server) handleUpdateVPN(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing profile name")
		return
	}
	var req vpnProfileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, exists := cfg.VPNs[name]; !exists {
		writeError(w, http.StatusNotFound, "vpn profile not found")
		return
	}
	if err := s.saveVPNProfile(cfg, cfgPath, name, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "profile": name, "restart_required": true})
}

// saveVPNProfile validates the merged profile against a trial copy of cfg
// before writing anything, then stores any provided credentials in the OS
// keychain and persists cfg via the existing saveRuntimeConfig pattern.
func (s *Server) saveVPNProfile(cfg *config.Config, cfgPath, name string, req vpnProfileReq) error {
	for field, value := range map[string]string{
		"name": req.Name, "type": req.Type, "host": req.Host, "server": req.Server,
		"protocol": req.Protocol, "config": req.ConfigFile, "auth_method": req.AuthMethod,
		"cert": req.Cert, "key": req.Key, "username": req.Username,
	} {
		if containsEnvReference(value) {
			return fmt.Errorf("%s must not contain environment-variable references", field)
		}
	}

	vc := config.VPNConfig{
		Type:       strings.TrimSpace(req.Type),
		Host:       strings.TrimSpace(req.Host),
		Port:       req.Port,
		Server:     strings.TrimSpace(req.Server),
		Protocol:   strings.TrimSpace(req.Protocol),
		ConfigFile: strings.TrimSpace(req.ConfigFile),
		Priority:   req.Priority,
		Auth: config.AuthConfig{
			Method:   strings.TrimSpace(req.AuthMethod),
			Cert:     strings.TrimSpace(req.Cert),
			Key:      strings.TrimSpace(req.Key),
			Username: strings.TrimSpace(req.Username),
		},
	}
	if existing, ok := cfg.VPNs[name]; ok {
		vc.Auth.UsernameKeychain = existing.Auth.UsernameKeychain
		vc.Auth.PasswordKeychain = existing.Auth.PasswordKeychain
		vc.KillSwitch = existing.KillSwitch
	}
	if strings.TrimSpace(req.Username) != "" {
		vc.Auth.UsernameKeychain = name + ".username"
	}
	if req.Password != "" {
		vc.Auth.PasswordKeychain = name + ".password"
	}

	trial := *cfg
	trialVPNs := make(map[string]config.VPNConfig, len(cfg.VPNs)+1)
	for k, v := range cfg.VPNs {
		trialVPNs[k] = v
	}
	trialVPNs[name] = vc
	trial.VPNs = trialVPNs
	if err := config.Validate(&trial); err != nil {
		return err
	}

	cfg.VPNs[name] = vc
	backups := make([]credentialBackup, 0, 2)
	if strings.TrimSpace(req.Username) != "" {
		backup := backupCredential(name, "username")
		if err := config.SetCredential(name, "username", req.Username); err != nil {
			return fmt.Errorf("store username credential: %w", err)
		}
		backups = append(backups, backup)
	}
	if req.Password != "" {
		backup := backupCredential(name, "password")
		if err := config.SetCredential(name, "password", req.Password); err != nil {
			return errors.Join(
				fmt.Errorf("store password credential: %w", err),
				restoreCredentials(name, backups),
			)
		}
		backups = append(backups, backup)
	}
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		return errors.Join(err, restoreCredentials(name, backups))
	}
	return nil
}

type credentialBackup struct {
	key     string
	value   string
	existed bool
}

func backupCredential(profile, key string) credentialBackup {
	value, err := config.GetCredential(profile, key)
	return credentialBackup{key: key, value: value, existed: err == nil}
}

func restoreCredentials(profile string, backups []credentialBackup) error {
	var errs []error
	for i := len(backups) - 1; i >= 0; i-- {
		backup := backups[i]
		var err error
		if backup.existed {
			err = config.SetCredential(profile, backup.key, backup.value)
		} else {
			err = config.DeleteCredential(profile, backup.key)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("restore %s credential: %w", backup.key, err))
		}
	}
	return errors.Join(errs...)
}

func containsEnvReference(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] != '$' || i+1 >= len(value) {
			continue
		}
		next := value[i+1]
		if next == '{' || next == '_' || next >= 'A' && next <= 'Z' || next >= 'a' && next <= 'z' {
			return true
		}
	}
	return false
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
		writeError(w, http.StatusNotFound, "vpn profile not found")
		return
	}
	if len(cfg.VPNs) <= 1 {
		writeError(w, http.StatusConflict, "cannot delete the last remaining VPN profile")
		return
	}
	for _, p := range cfg.Policies {
		if strings.EqualFold(p.Via, name) {
			writeError(w, http.StatusConflict, fmt.Sprintf("profile %q is referenced by policy %q", name, p.Name))
			return
		}
	}
	for groupName, g := range cfg.Groups {
		for _, prof := range g.Profiles {
			if strings.EqualFold(prof, name) {
				writeError(w, http.StatusConflict, fmt.Sprintf("profile %q is referenced by group %q", name, groupName))
				return
			}
		}
	}

	delete(cfg.VPNs, name)
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var cleanupErrs []error
	if existing.Auth.UsernameKeychain != "" {
		if err := config.DeleteCredential(name, "username"); err != nil {
			cleanupErrs = append(cleanupErrs, err)
		}
	}
	if existing.Auth.PasswordKeychain != "" {
		if err := config.DeleteCredential(name, "password"); err != nil {
			cleanupErrs = append(cleanupErrs, err)
		}
	}
	if err := errors.Join(cleanupErrs...); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("profile deleted but credential cleanup failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "profile": name, "restart_required": true})
}
