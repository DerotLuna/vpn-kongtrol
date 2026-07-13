package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
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
	if s.hasPendingConnect(name) {
		writeError(w, http.StatusConflict, "connect already in progress for "+name)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	s.setPendingConnect(name, cancel)
	go func() {
		defer s.clearPendingConnect(name)
		// AdapterConfig is pre-loaded by the orchestrator; API-triggered connects
		// use the adapter's previously configured values.
		_ = adapter.Reconnect(ctx)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "connecting", "tunnel": name})
}

// POST /api/v1/tunnels/{name}/cancel_connect
func (s *Server) handleCancelConnect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	adapter, ok := s.adapters[name]
	if !ok {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}
	s.cancelPendingConnect(name)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	_ = adapter.Disconnect(ctx)

	writeJSON(w, http.StatusOK, map[string]string{"status": "connect_cancelled", "tunnel": name})
}

// POST /api/v1/tunnels/{name}/disconnect
func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	adapter, ok := s.adapters[name]
	if !ok {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}
	s.cancelPendingConnect(name)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := adapter.Disconnect(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected", "tunnel": name})
}

func (s *Server) hasPendingConnect(name string) bool {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	_, ok := s.connectCancel[name]
	return ok
}

func (s *Server) setPendingConnect(name string, cancel context.CancelFunc) {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	s.connectCancel[name] = cancel
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
	if s.policyEngine != nil {
		rules = s.policyEngine.Rules()
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
	type secStatus struct {
		KillSwitch           bool        `json:"kill_switch"`
		KillSwitchEnabled    bool        `json:"kill_switch_enabled"`
		DNSGuard             bool        `json:"dns_guard"`
		DNSGuardEnabled      bool        `json:"dns_guard_enabled"`
		LeakDetectionEnabled bool        `json:"leak_detection_enabled"`
		LeakCheck            interface{} `json:"leak_check"`
	}

	status := secStatus{
		KillSwitch:           s.ks != nil && s.ks.IsEnabled(),
		KillSwitchEnabled:    s.killSwitchOn,
		DNSGuard:             s.dnsMgr != nil && s.dnsMgr.IsActive(),
		DNSGuardEnabled:      s.dnsGuardOn,
		LeakDetectionEnabled: s.leakTest != nil,
	}

	if s.leakTest != nil {
		lr := s.leakTest.LastResult()
		if lr != nil {
			status.LeakCheck = map[string]interface{}{
				"has_leak":   lr.HasLeak,
				"public_ip":  lr.PublicIP,
				"reason":     lr.Reason,
				"checked_at": lr.CheckedAt,
			}
		}
	}

	writeJSON(w, http.StatusOK, status)
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
	if s.policyEngine != nil {
		for _, rule := range s.policyEngine.Rules() {
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
		Profiles []string `json:"profiles"`
	}
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	profiles := make([]string, 0, len(cfg.VPNs))
	for name := range cfg.VPNs {
		profiles = append(profiles, name)
	}
	sort.Strings(profiles)
	writeJSON(w, http.StatusOK, metaDTO{Profiles: profiles})
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
	if strings.TrimSpace(req.App) != "" {
		resp.Matched = rule.MatchesApp(req.App)
		if !resp.Matched {
			resp.Reason = "app did not match the rule"
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	if target == "" {
		writeError(w, http.StatusBadRequest, "target or app is required")
		return
	}
	if ip := net.ParseIP(target); ip != nil {
		resp.Matched = rule.MatchesIP(ip)
		if !resp.Matched {
			resp.Reason = "IP did not match the rule"
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp.Matched = rule.MatchesDomain(target)
	if !resp.Matched {
		resp.Reason = "domain did not match the rule"
	}
	writeJSON(w, http.StatusOK, resp)
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

	if s.policyEngine == nil {
		writeJSON(w, http.StatusOK, result)
		return
	}
	rules := s.policyEngine.Rules()

	if app != "" {
		if vpnName, matched := s.policyEngine.ResolveApp(app); matched {
			result.Via = vpnName
			result.Matched = true
			for _, rule := range rules {
				if rule.Via == vpnName && rule.MatchesApp(app) {
					result.Rule = rule.Name
					break
				}
			}
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
		if vpnName, matched := s.policyEngine.ResolveIP(ip); matched {
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
		if vpnName, matched := s.policyEngine.ResolveDomain(target); matched {
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
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(cfgPath, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	newEngine, err := policy.New(cfg)
	if err != nil {
		return fmt.Errorf("policy engine validation failed: %w", err)
	}
	s.policyEngine = newEngine
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
