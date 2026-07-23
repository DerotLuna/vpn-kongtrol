package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"gopkg.in/yaml.v3"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// POST /api/v1/shutdown asks the daemon to terminate gracefully: it triggers
// the same root-context cancellation as Ctrl+C/SIGTERM, so the daemon's own
// deferred cleanup (kill switch teardown, DNS restore, history flush, PID
// file removal) runs before the process exits — unlike a caller sending
// SIGKILL/TerminateProcess directly, which skips all of it. This works
// identically on every OS, including Windows, where there is no reliable
// SIGTERM equivalent deliverable across processes.
func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if s.onShutdown == nil {
		writeError(w, http.StatusNotImplemented, "shutdown not supported by this daemon instance")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "shutting_down"})
	go s.onShutdown()
}

// GET /api/v1/metrics/history
func (s *Server) handleMetricsHistory(w http.ResponseWriter, r *http.Request) {
	if s.collector == nil {
		writeJSON(w, http.StatusOK, map[string]monitor.ProfileHistory{})
		return
	}
	writeJSON(w, http.StatusOK, s.collector.HistorySnapshot())
}

// GET /api/v1/tunnels
func (s *Server) handleListTunnels(w http.ResponseWriter, r *http.Request) {
	snapshot := s.collector.Snapshot()
	tunnels := make([]monitor.TunnelMetrics, 0, len(snapshot))
	for _, m := range snapshot {
		tunnels = append(tunnels, m)
	}
	writeJSON(w, http.StatusOK, tunnels)
}

// POST /api/v1/tunnels/{name}/connect
func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	adapter, ok := s.adapters[name]
	if !ok {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}
	if st := adapter.Status().Normalize(); st == vpn.StatusConnected {
		writeJSON(w, http.StatusOK, map[string]string{"status": "already_connected", "tunnel": name})
		return
	}
	if s.connectProfile == nil {
		writeError(w, http.StatusServiceUnavailable, "connection service unavailable")
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Minute)
	if !s.startPendingConnect(name, cancel) {
		cancel()
		writeError(w, http.StatusConflict, "connect already in progress for "+name)
		return
	}
	go func() {
		defer s.clearPendingConnect(name)
		_ = s.connectProfile(ctx, name)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "connecting", "tunnel": name})
}

// POST /api/v1/tunnels/{name}/cancel_connect
func (s *Server) handleCancelConnect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_, ok := s.adapters[name]
	if !ok {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}
	s.cancelPendingConnect(name)

	if s.disconnectProfile == nil {
		writeError(w, http.StatusServiceUnavailable, "connection service unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := s.disconnectProfile(ctx, name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "connect_cancelled", "tunnel": name})
}

// POST /api/v1/tunnels/{name}/disconnect
func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_, ok := s.adapters[name]
	if !ok {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}
	s.cancelPendingConnect(name)

	if s.disconnectProfile == nil {
		writeError(w, http.StatusServiceUnavailable, "connection service unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := s.disconnectProfile(ctx, name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected", "tunnel": name})
}

func (s *Server) startPendingConnect(name string, cancel context.CancelFunc) bool {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	if _, exists := s.connectCancel[name]; exists {
		return false
	}
	s.connectCancel[name] = cancel
	return true
}

func (s *Server) clearPendingConnect(name string) {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	delete(s.connectCancel, name)
}

func (s *Server) cancelPendingConnect(name string) {
	s.connectMu.Lock()
	cancel, ok := s.connectCancel[name]
	if ok {
		delete(s.connectCancel, name)
	}
	s.connectMu.Unlock()
	if ok {
		cancel()
	}
}

// GET /api/v1/routes
func (s *Server) handleListRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := s.routes.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type routeDTO struct {
		Destination string `json:"destination"`
		Gateway     string `json:"gateway"`
		Interface   string `json:"interface"`
		Metric      int    `json:"metric"`
		PolicyName  string `json:"policy_name,omitempty"`
		PolicyVia   string `json:"policy_via,omitempty"`
		IsDefault   bool   `json:"is_default"`
	}
	var rules []policy.Rule
	if pe := s.policyEngine.Load(); pe != nil {
		rules = pe.Rules()
	}
	ruleNameByVia := make(map[string]string, len(rules))
	for _, rule := range rules {
		if _, exists := ruleNameByVia[rule.Via]; !exists {
			ruleNameByVia[rule.Via] = rule.Name
		}
	}

	resolvedViaByCIDR := make(map[string]string)
	if s.policyResolver != nil {
		for _, snap := range s.policyResolver.Snapshot() {
			for _, cidr := range snap.ResolvedCIDRs {
				resolvedViaByCIDR[cidr] = snap.Name
			}
		}
	}

	out := make([]routeDTO, len(routes))
	for i, r := range routes {
		dest := r.Destination.String()
		dto := routeDTO{
			Destination: dest,
			Interface:   r.Interface,
			Metric:      r.Metric,
			IsDefault:   isDefaultRoute(r.Destination),
		}
		if r.Gateway != nil {
			dto.Gateway = r.Gateway.String()
		}
		if dto.IsDefault {
			dto.PolicyName = "default"
			dto.PolicyVia = "system"
		} else if via, ok := resolvedViaByCIDR[dest]; ok {
			dto.PolicyVia = via
			dto.PolicyName = ruleNameByVia[via]
		} else {
			dto.PolicyVia, dto.PolicyName = matchStaticPolicy(r.Destination, rules)
		}
		out[i] = dto
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDefault != out[j].IsDefault {
			return out[i].IsDefault
		}
		if (out[i].PolicyName != "") != (out[j].PolicyName != "") {
			return out[i].PolicyName != ""
		}
		if pI, pJ := cidrPrefixLen(out[i].Destination), cidrPrefixLen(out[j].Destination); pI != pJ {
			return pI > pJ // more specific first
		}
		if out[i].Metric != out[j].Metric {
			return out[i].Metric < out[j].Metric
		}
		if out[i].PolicyName != out[j].PolicyName {
			return out[i].PolicyName < out[j].PolicyName
		}
		return out[i].Destination < out[j].Destination
	})
	writeJSON(w, http.StatusOK, out)
}

// GET /api/v1/network/overview
func (s *Server) handleNetworkOverview(w http.ResponseWriter, r *http.Request) {
	type defaultRouteDTO struct {
		Destination string `json:"destination"`
		Gateway     string `json:"gateway"`
		Interface   string `json:"interface"`
		Metric      int    `json:"metric"`
	}
	type overviewDTO struct {
		ConnectedTunnels int               `json:"connected_tunnels"`
		DefaultRoutes    []defaultRouteDTO `json:"default_routes"`
		LocalIPs         []string          `json:"local_ips"`
		PublicIP         string            `json:"public_ip,omitempty"`
	}

	out := overviewDTO{}

	if s.collector != nil {
		for _, m := range s.collector.Snapshot() {
			if m.Status.Normalize() == vpn.StatusConnected {
				out.ConnectedTunnels++
			}
		}
	}

	if s.routes != nil {
		routes, err := s.routes.List()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, rt := range routes {
			if !isDefaultRoute(rt.Destination) {
				continue
			}
			row := defaultRouteDTO{
				Destination: rt.Destination.String(),
				Interface:   rt.Interface,
				Metric:      rt.Metric,
			}
			if rt.Gateway != nil {
				row.Gateway = rt.Gateway.String()
			}
			out.DefaultRoutes = append(out.DefaultRoutes, row)
		}
		sort.Slice(out.DefaultRoutes, func(i, j int) bool {
			return out.DefaultRoutes[i].Metric < out.DefaultRoutes[j].Metric
		})
	}

	if ifaces, err := net.Interfaces(); err == nil {
		ips := make(map[string]struct{})
		for _, iface := range ifaces {
			if (iface.Flags & net.FlagUp) == 0 {
				continue
			}
			if (iface.Flags & net.FlagLoopback) != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, a := range addrs {
				var ip net.IP
				switch v := a.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip == nil {
					continue
				}
				ip = ip.To4()
				if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
					continue
				}
				ips[ip.String()] = struct{}{}
			}
		}
		out.LocalIPs = make([]string, 0, len(ips))
		for ip := range ips {
			out.LocalIPs = append(out.LocalIPs, ip)
		}
		sort.Strings(out.LocalIPs)
	}
	if s.leakTest != nil {
		if lr := s.leakTest.LastResult(); lr != nil && lr.PublicIP != "" {
			out.PublicIP = lr.PublicIP
		}
	}
	if out.PublicIP == "" {
		out.PublicIP = fetchPublicIP()
	}

	writeJSON(w, http.StatusOK, out)
}

func fetchPublicIP() string {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

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

// GET /api/v1/scheduler/rules
func (s *Server) handleListScheduleRules(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]scheduleRuleDTO, 0, len(cfg.Monitor.Scheduler.Rules))
	for _, rule := range cfg.Monitor.Scheduler.Rules {
		out = append(out, scheduleRuleDTO{
			Name:     rule.Name,
			Profiles: rule.Profiles,
			Weekdays: rule.Weekdays,
			Start:    rule.Start,
			End:      rule.End,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// POST /api/v1/scheduler/rules
func (s *Server) handleCreateScheduleRule(w http.ResponseWriter, r *http.Request) {
	var req scheduleRuleDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "rule name is required")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, rule := range cfg.Monitor.Scheduler.Rules {
		if strings.EqualFold(rule.Name, req.Name) {
			writeError(w, http.StatusConflict, "scheduler rule already exists")
			return
		}
	}
	if err := s.saveScheduleRule(cfg, cfgPath, -1, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "rule": req.Name})
}

// PUT /api/v1/scheduler/rules/{name}
func (s *Server) handleUpdateScheduleRule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req scheduleRuleDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := -1
	for i, rule := range cfg.Monitor.Scheduler.Rules {
		if strings.EqualFold(rule.Name, name) {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeError(w, http.StatusNotFound, "scheduler rule not found")
		return
	}
	if err := s.saveScheduleRule(cfg, cfgPath, idx, req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "rule": name})
}

func (s *Server) saveScheduleRule(cfg *config.Config, cfgPath string, idx int, req scheduleRuleDTO) error {
	rule := config.ScheduleRule{
		Name:     strings.TrimSpace(req.Name),
		Profiles: req.Profiles,
		Weekdays: req.Weekdays,
		Start:    strings.TrimSpace(req.Start),
		End:      strings.TrimSpace(req.End),
	}

	trial := *cfg
	trialRules := make([]config.ScheduleRule, len(cfg.Monitor.Scheduler.Rules))
	copy(trialRules, cfg.Monitor.Scheduler.Rules)
	if idx >= 0 {
		trialRules[idx] = rule
	} else {
		trialRules = append(trialRules, rule)
	}
	trial.Monitor.Scheduler.Rules = trialRules
	if err := config.Validate(&trial); err != nil {
		return err
	}

	cfg.Monitor.Scheduler.Rules = trialRules
	return s.saveRuntimeConfig(cfgPath, cfg)
}

// DELETE /api/v1/scheduler/rules/{name}
func (s *Server) handleDeleteScheduleRule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := -1
	for i, rule := range cfg.Monitor.Scheduler.Rules {
		if strings.EqualFold(rule.Name, name) {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeError(w, http.StatusNotFound, "scheduler rule not found")
		return
	}
	cfg.Monitor.Scheduler.Rules = append(cfg.Monitor.Scheduler.Rules[:idx], cfg.Monitor.Scheduler.Rules[idx+1:]...)
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "rule": name})
}

func isDefaultRoute(dst net.IPNet) bool {
	ones, bits := dst.Mask.Size()
	return ones == 0 && (bits == 32 || bits == 128)
}

func matchStaticPolicy(route net.IPNet, rules []policy.Rule) (via string, name string) {
	bestPrefix := -1
	for _, rule := range rules {
		for _, cidr := range rule.Match.IPRanges {
			if cidrOverlaps(route, *cidr) {
				ones, _ := cidr.Mask.Size()
				if ones > bestPrefix {
					bestPrefix = ones
					via = rule.Via
					name = rule.Name
				}
			}
		}
	}
	return via, name
}

func cidrOverlaps(a net.IPNet, b net.IPNet) bool {
	return a.Contains(b.IP) || b.Contains(a.IP)
}

func cidrPrefixLen(cidr string) int {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil || n == nil {
		return -1
	}
	ones, _ := n.Mask.Size()
	return ones
}

// GET /api/v1/policies — active policies with resolved IPs from PolicyResolver.
func (s *Server) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	type policyDTO struct {
		Name          string   `json:"name"`
		Via           string   `json:"via"`
		Domains       []string `json:"domains"`
		IPRanges      []string `json:"ip_ranges"`
		Apps          []string `json:"apps"`
		ResolvedCIDRs []string `json:"resolved_cidrs"`
	}

	// Start from the policy engine rules (static config).
	var out []policyDTO
	if pe := s.policyEngine.Load(); pe != nil {
		for _, rule := range pe.Rules() {
			dto := policyDTO{
				Name: rule.Name,
				Via:  rule.Via,
			}
			dto.Domains = rule.Match.Domains
			dto.Apps = rule.Match.Apps
			for _, ipnet := range rule.Match.IPRanges {
				dto.IPRanges = append(dto.IPRanges, ipnet.String())
			}
			out = append(out, dto)
		}
	}

	// Enrich with resolved CIDRs from PolicyResolver.
	if s.policyResolver != nil {
		snapshots := s.policyResolver.Snapshot()
		// Index snapshots by profile name for fast lookup.
		byProfile := make(map[string]monitor.ProfileSnapshot, len(snapshots))
		for _, snap := range snapshots {
			byProfile[snap.Name] = snap
		}
		for i := range out {
			if snap, ok := byProfile[out[i].Via]; ok {
				out[i].ResolvedCIDRs = snap.ResolvedCIDRs
			}
		}
	}

	if out == nil {
		out = []policyDTO{}
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /api/v1/policies/meta
func (s *Server) handlePoliciesMeta(w http.ResponseWriter, r *http.Request) {
	type metaDTO struct {
		Profiles   []string `json:"profiles"`
		ConfigPath string   `json:"config_path"`
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	profiles := make([]string, 0, len(cfg.VPNs))
	for name := range cfg.VPNs {
		profiles = append(profiles, name)
	}
	sort.Strings(profiles)
	writeJSON(w, http.StatusOK, metaDTO{Profiles: profiles, ConfigPath: cfgPath})
}

// POST /api/v1/policies
func (s *Server) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req config.PolicyRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	req = normalizePolicyRule(req)
	if err := validatePolicyRule(req, cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	for _, p := range cfg.Policies {
		if strings.EqualFold(p.Name, req.Name) {
			writeError(w, http.StatusConflict, "policy name already exists")
			return
		}
	}
	cfg.Policies = append(cfg.Policies, req)
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "policy": req.Name})
}

// PUT /api/v1/policies/{name}
func (s *Server) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing policy name")
		return
	}
	var req config.PolicyRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	req = normalizePolicyRule(req)
	if req.Name == "" {
		req.Name = name
	}
	if !strings.EqualFold(req.Name, name) {
		for _, p := range cfg.Policies {
			if strings.EqualFold(p.Name, req.Name) {
				writeError(w, http.StatusConflict, "policy name already exists")
				return
			}
		}
	}
	if err := validatePolicyRule(req, cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated := false
	for i := range cfg.Policies {
		if strings.EqualFold(cfg.Policies[i].Name, name) {
			cfg.Policies[i] = req
			updated = true
			break
		}
	}
	if !updated {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "policy": req.Name})
}

// DELETE /api/v1/policies/{name}
func (s *Server) handleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var out []config.PolicyRule
	deleted := false
	for _, p := range cfg.Policies {
		if strings.EqualFold(p.Name, name) {
			deleted = true
			continue
		}
		out = append(out, p)
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}
	cfg.Policies = out
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "policy": name})
}

// POST /api/v1/policies/test
func (s *Server) handleTestPolicy(w http.ResponseWriter, r *http.Request) {
	type testReq struct {
		Rule   config.PolicyRule `json:"rule"`
		Target string            `json:"target"`
		App    string            `json:"app"`
	}
	type testResp struct {
		Matched bool   `json:"matched"`
		Via     string `json:"via"`
		Rule    string `json:"rule"`
		Reason  string `json:"reason,omitempty"`
	}
	var req testReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Rule = normalizePolicyRule(req.Rule)
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := validatePolicyRule(req.Rule, cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rule, err := policy.ParseRule(req.Rule.Name, req.Rule.Via, req.Rule.Match.IPRanges, req.Rule.Match.Domains, req.Rule.Match.Apps, 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	target := normalizeResolveTarget(req.Target)
	resp := testResp{Via: req.Rule.Via, Rule: req.Rule.Name}
	if strings.TrimSpace(req.App) != "" || target != "" {
		resp.Matched = rule.MatchesFlow(target, req.App)
		if !resp.Matched {
			resp.Reason = "flow did not match the rule"
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	writeError(w, http.StatusBadRequest, "target or app is required")
}

// GET /api/v1/resolve?target=<ip-or-domain>&app=<exe-or-path> — which VPN handles this match.
func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	target := normalizeResolveTarget(r.URL.Query().Get("target"))
	app := r.URL.Query().Get("app")
	if target == "" && app == "" {
		writeError(w, http.StatusBadRequest, "missing query parameter: provide 'target' or 'app'")
		return
	}

	type resolveDTO struct {
		Target  string `json:"target"`
		App     string `json:"app,omitempty"`
		Via     string `json:"via"`
		Rule    string `json:"rule"`
		Matched bool   `json:"matched"`
	}

	result := resolveDTO{Target: target, App: app}

	pe := s.policyEngine.Load()
	if pe == nil {
		writeJSON(w, http.StatusOK, result)
		return
	}
	rules := pe.Rules()

	if app != "" || target != "" {
		if vpnName, ruleName, matched := pe.ResolveFlow(target, app); matched {
			result.Via = vpnName
			result.Rule = ruleName
			result.Matched = true
			writeJSON(w, http.StatusOK, result)
			return
		}
	}
	if target != "" {
		if via, ruleName, ok := matchPolicyOrProfileToken(target, rules); ok {
			result.Via = via
			result.Rule = ruleName
			result.Matched = true
			writeJSON(w, http.StatusOK, result)
			return
		}
	}

	// Try target as IP first, then as domain.
	if ip := net.ParseIP(target); ip != nil {
		if vpnName, matched := pe.ResolveIP(ip); matched {
			result.Via = vpnName
			result.Matched = true
			// Find matching rule name.
			for _, rule := range rules {
				if rule.Via == vpnName && rule.MatchesIP(ip) {
					result.Rule = rule.Name
					break
				}
			}
		}
	} else {
		if vpnName, matched := pe.ResolveDomain(target); matched {
			result.Via = vpnName
			result.Matched = true
			for _, rule := range rules {
				if rule.Via == vpnName && rule.MatchesDomain(target) {
					result.Rule = rule.Name
					break
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, result)
}

func normalizeResolveTarget(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			s = u.Hostname()
		}
	}
	if strings.Contains(s, "/") {
		s = strings.SplitN(s, "/", 2)[0]
	}
	if h, _, err := net.SplitHostPort(s); err == nil {
		s = h
	}
	s = strings.Trim(s, "[]")
	s = strings.TrimSuffix(s, ".")
	return strings.ToLower(strings.TrimSpace(s))
}

// GET /api/v1/dns/resolve?domain=<fqdn>&via=<profile>
func (s *Server) handleDNSResolve(w http.ResponseWriter, r *http.Request) {
	domain := strings.TrimSpace(r.URL.Query().Get("domain"))
	if domain == "" {
		writeError(w, http.StatusBadRequest, "missing query parameter: domain")
		return
	}
	via := strings.TrimSpace(r.URL.Query().Get("via"))
	if via == "" {
		if pe := s.policyEngine.Load(); pe != nil {
			if resolvedVia, matched := pe.ResolveDomain(domain); matched {
				via = resolvedVia
			}
		}
	}
	if via == "" {
		writeError(w, http.StatusBadRequest, "missing query parameter: via (or no policy matched domain)")
		return
	}
	if s.policyResolver == nil {
		writeError(w, http.StatusServiceUnavailable, "policy resolver unavailable")
		return
	}
	ips, err := s.policyResolver.ResolveDomainViaProfile(via, domain)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	sort.Strings(out)
	writeJSON(w, http.StatusOK, map[string]any{
		"domain": domain,
		"via":    via,
		"ips":    out,
	})
}

func normalizePolicyRule(r config.PolicyRule) config.PolicyRule {
	r.Name = strings.TrimSpace(r.Name)
	r.Via = strings.TrimSpace(r.Via)
	for i := range r.Match.Domains {
		r.Match.Domains[i] = strings.TrimSpace(r.Match.Domains[i])
	}
	for i := range r.Match.IPRanges {
		r.Match.IPRanges[i] = strings.TrimSpace(r.Match.IPRanges[i])
	}
	for i := range r.Match.Apps {
		r.Match.Apps[i] = strings.TrimSpace(r.Match.Apps[i])
	}
	r.Match.Domains = filterNonEmpty(r.Match.Domains)
	r.Match.IPRanges = filterNonEmpty(r.Match.IPRanges)
	r.Match.Apps = filterNonEmpty(r.Match.Apps)
	return r
}

func filterNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}

func validatePolicyRule(rule config.PolicyRule, cfg *config.Config) error {
	if rule.Name == "" {
		return fmt.Errorf("policy name is required")
	}
	if rule.Via == "" {
		return fmt.Errorf("policy via profile is required")
	}
	if _, ok := cfg.VPNs[rule.Via]; !ok {
		return fmt.Errorf("via profile %q not found in vpns", rule.Via)
	}
	if len(rule.Match.Domains) == 0 && len(rule.Match.IPRanges) == 0 && len(rule.Match.Apps) == 0 {
		return fmt.Errorf("policy must define at least one domain, ip_range, or app")
	}
	_, err := policy.ParseRule(rule.Name, rule.Via, rule.Match.IPRanges, rule.Match.Domains, rule.Match.Apps, 1)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) loadRuntimeConfig() (*config.Config, string, error) {
	cfgPath := s.configPath
	if cfgPath == "" {
		for _, candidate := range config.DefaultPaths() {
			if _, err := os.Stat(candidate); err == nil {
				cfgPath = candidate
				break
			}
		}
	}
	if cfgPath == "" {
		return nil, "", fmt.Errorf("config path not found")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, "", err
	}
	return cfg, cfgPath, nil
}

func (s *Server) saveRuntimeConfig(cfgPath string, cfg *config.Config) error {
	if err := config.Validate(cfg); err != nil {
		return err
	}
	newEngine, err := policy.New(cfg)
	if err != nil {
		return fmt.Errorf("policy engine validation failed: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := config.WriteFileAtomic(cfgPath, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	s.policyEngine.Store(newEngine)
	if s.onPolicyUpdate != nil {
		s.onPolicyUpdate(cfg, newEngine)
	}
	return nil
}

func matchPolicyOrProfileToken(token string, rules []policy.Rule) (via string, rule string, ok bool) {
	if token == "" {
		return "", "", false
	}
	for _, r := range rules {
		if strings.EqualFold(r.Name, token) || strings.EqualFold(r.Via, token) {
			return r.Via, r.Name, true
		}
	}
	return "", "", false
}
