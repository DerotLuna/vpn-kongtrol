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

// vpnProfileReq is the create/update body. Every field is a pointer so a
// field absent from the JSON (nil) is distinguishable from one sent empty:
// on update only the fields present are applied on top of the existing
// VPNConfig, so YAML-only fields the dashboard form does not expose
// (binary_path, allow_insecure_cert, version, tunnel_name, ...) survive.
type vpnProfileReq struct {
	Name       *string `json:"name"`
	Type       *string `json:"type"`
	Host       *string `json:"host"`
	Port       *int    `json:"port"`
	Server     *string `json:"server"`
	Protocol   *string `json:"protocol"`
	ConfigFile *string `json:"config"`
	Priority   *int    `json:"priority"`
	// Failover: the backup profiles, in order; [] clears it.
	Failover   *[]string `json:"failover"`
	AuthMethod *string   `json:"auth_method"`
	Cert       *string   `json:"cert"`
	Key        *string   `json:"key"`
	Username   *string   `json:"username"`
	// Provider attach mode. AutoConnect true is the default and is stored
	// as "unset" so the YAML stays minimal.
	AutoConnect *bool   `json:"auto_connect"`
	Country     *string `json:"country"`
	City        *string `json:"city"`
	Group       *string `json:"group"`
	Interface   *string `json:"interface"`
	WaitTimeout *string `json:"wait_timeout"`
	AppPath     *string `json:"app_path"`
	// Password is plaintext in the request only; it is written to the OS
	// keychain and never persisted to the YAML config.
	Password string `json:"password"`
}

// strVal returns the trimmed value of p, or "" when absent.
func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// setTrimmed overwrites *dst with the trimmed *src when src is present.
func setTrimmed(dst, src *string) {
	if src != nil {
		*dst = strings.TrimSpace(*src)
	}
}

// applyTo overlays the fields present in req onto vc.
func (req vpnProfileReq) applyTo(vc *config.VPNConfig) {
	setTrimmed(&vc.Type, req.Type)
	setTrimmed(&vc.Host, req.Host)
	setTrimmed(&vc.Server, req.Server)
	setTrimmed(&vc.Protocol, req.Protocol)
	setTrimmed(&vc.ConfigFile, req.ConfigFile)
	setTrimmed(&vc.Auth.Method, req.AuthMethod)
	setTrimmed(&vc.Auth.Cert, req.Cert)
	setTrimmed(&vc.Auth.Key, req.Key)
	setTrimmed(&vc.Auth.Username, req.Username)
	setTrimmed(&vc.Country, req.Country)
	setTrimmed(&vc.City, req.City)
	setTrimmed(&vc.Group, req.Group)
	setTrimmed(&vc.Interface, req.Interface)
	setTrimmed(&vc.WaitTimeout, req.WaitTimeout)
	setTrimmed(&vc.AppPath, req.AppPath)
	if req.AutoConnect != nil {
		if *req.AutoConnect {
			vc.AutoConnect = nil
		} else {
			off := false
			vc.AutoConnect = &off
		}
	}
	if req.Port != nil {
		vc.Port = *req.Port
	}
	if req.Priority != nil {
		vc.Priority = *req.Priority
	}
	if req.Failover != nil {
		vc.Failover = nil
		for _, p := range *req.Failover {
			if p = strings.TrimSpace(p); p != "" {
				vc.Failover = append(vc.Failover, p)
			}
		}
	}
}

// POST /api/v1/vpns — create a new VPN profile. Writes config + OS keychain
// only; a newly created profile is not hot-registered with the running
// daemon (the shared adapters map has no safe runtime-mutation path — see
// cmd/kongtrol/main.go loadConfig), so the response flags restart_required.
func (s *Server) handleCreateVPN(w http.ResponseWriter, r *http.Request) {
	var req vpnProfileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, errInvalidJSON)
		return
	}
	name := strVal(req.Name)
	if name == "" {
		writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "api.err.name_required"})
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, exists := cfg.VPNs[name]; exists {
		writeLocalizedError(w, r, http.StatusConflict, &config.ValidationError{Key: "api.err.vpn_exists", Args: []any{name}})
		return
	}
	if err := s.saveVPNProfile(cfg, cfgPath, name, config.VPNConfig{}, req); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, err)
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
	var req vpnProfileReq
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
	if err := s.saveVPNProfile(cfg, cfgPath, name, existing, req); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "profile": name, "restart_required": true})
}

// saveVPNProfile overlays req onto base (zero value on create, the stored
// profile on update), validates the result against a trial copy of cfg
// before writing anything, then stores any provided credentials in the OS
// keychain and persists cfg via the existing saveRuntimeConfig pattern.
func (s *Server) saveVPNProfile(cfg *config.Config, cfgPath, name string, base config.VPNConfig, req vpnProfileReq) error {
	for field, value := range map[string]*string{
		"name": req.Name, "type": req.Type, "host": req.Host, "server": req.Server,
		"protocol": req.Protocol, "config": req.ConfigFile, "auth_method": req.AuthMethod,
		"cert": req.Cert, "key": req.Key, "username": req.Username,
		"country": req.Country, "city": req.City, "group": req.Group, "interface": req.Interface,
		"wait_timeout": req.WaitTimeout, "app_path": req.AppPath,
	} {
		if value != nil && containsEnvReference(*value) {
			return fmt.Errorf("%s must not contain environment-variable references", field)
		}
	}

	vc := base
	req.applyTo(&vc)
	vc = vc.WithDefaults()
	username := strVal(req.Username)
	if username != "" {
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
	if username != "" {
		backup := backupCredential(name, "username")
		if err := config.SetCredential(name, "username", username); err != nil {
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
