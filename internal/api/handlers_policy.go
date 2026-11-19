package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

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
		OnTunnelDown  string   `json:"on_tunnel_down"`
		ResolvedCIDRs []string `json:"resolved_cidrs"`
		// Resolved: this policy's own resolved IPs with their expiry
		// (resolved_cidrs above stays every IP of the profile, as before).
		Resolved []monitor.ResolvedIP `json:"resolved"`
	}

	// Start from the policy engine rules (static config).
	var out []policyDTO
	onDown := map[string]string{} // policy → on_tunnel_down (not in the engine)
	if cfg, _, err := s.loadRuntimeConfig(); err == nil {
		for _, p := range cfg.Policies {
			onDown[p.Name] = p.TunnelDownAction()
		}
	}
	if pe := s.policyEngine.Load(); pe != nil {
		for _, rule := range pe.Rules() {
			dto := policyDTO{
				Name:         rule.Name,
				Via:          rule.Via,
				OnTunnelDown: onDown[rule.Name],
			}
			if dto.OnTunnelDown == "" {
				dto.OnTunnelDown = config.OnTunnelDownAllow
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
			out[i].Resolved = s.policyResolver.ResolvedIPs(out[i].Name)
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

// POST /api/v1/policies/reload — re-reads kongtrol.yaml from disk and
// hot-swaps the policy engine from it, picking up IP/domain/app routing
// rules that were hand-edited directly in the YAML file (bypassing the
// dashboard/CLI) without requiring a full daemon restart. This is
// deliberately a distinct code path from saveRuntimeConfig (used by every
// other policy CRUD handler in this file): those mutate an in-memory copy
// and persist it; this only re-reads and re-applies what's already on disk.
func (s *Server) handlePolicyReload(w http.ResponseWriter, r *http.Request) {
	cfg, applied, err := s.reloadRuntimeConfigApply()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Profiles applied live (ADR-0053); the ones whose tunnel is up and
	// whose settings changed keep the old ones until reconnected.
	reconnect := append(append([]string{}, applied.res.Reconnect...), applied.res.Kept...)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":             "reloaded",
		"policies":           len(cfg.Policies),
		"restart_required":   !applied.live || len(applied.res.Failed) > 0,
		"reconnect_required": reconnect,
	})
}

// POST /api/v1/policies
func (s *Server) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var patch policyPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	req := normalizePolicyRule(patch.applyTo(config.PolicyRule{}))
	if err := validatePolicyRule(req, cfg); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, err)
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
	var patch policyPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := -1
	for i := range cfg.Policies {
		if strings.EqualFold(cfg.Policies[i].Name, name) {
			idx = i
			break
		}
	}
	if idx == -1 {
		writeError(w, http.StatusNotFound, "policy not found")
		return
	}
	req := normalizePolicyRule(patch.applyTo(clonePolicyRule(cfg.Policies[idx])))
	if req.Name == "" {
		req.Name = cfg.Policies[idx].Name
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
		writeLocalizedError(w, r, http.StatusBadRequest, err)
		return
	}
	cfg.Policies[idx] = req
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
		Rule   policyPatch `json:"rule"`
		Target string      `json:"target"`
		App    string      `json:"app"`
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
	pr := normalizePolicyRule(req.Rule.applyTo(config.PolicyRule{}))
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := validatePolicyRule(pr, cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rule, err := policy.ParseRule(pr.Name, pr.Via, pr.Match.IPRanges, pr.Match.Domains, pr.Match.Apps, 1)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	target := normalizeResolveTarget(req.Target)
	resp := testResp{Via: pr.Via, Rule: pr.Name}
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

// GET /api/v1/policies/explain?target=<ip|domain|url|app:exe> — which policy
// and VPN a destination uses and why: the same answer and JSON shape as
// `kongtrol policy show <target> --json` (policy.ExplainResult; the reason
// also comes as ReasonCode + ReasonArgs so clients translate it). Read-only.
func (s *Server) handleExplainPolicy(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("target"))
	if raw == "" {
		writeError(w, http.StatusBadRequest, "missing query parameter: target")
		return
	}
	target := raw
	if !strings.HasPrefix(strings.ToLower(raw), "app:") {
		target = normalizeResolveTarget(raw)
	}
	pe := s.policyEngine.Load()
	if pe == nil {
		writeError(w, http.StatusServiceUnavailable, "policy engine not loaded")
		return
	}
	writeJSON(w, http.StatusOK, pe.ExplainTarget(target))
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
	r.OnTunnelDown = strings.ToLower(strings.TrimSpace(r.OnTunnelDown))
	if r.OnTunnelDown == config.OnTunnelDownAllow {
		r.OnTunnelDown = "" // the default stays out of kongtrol.yaml
	}
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
	if err := config.ValidateOnTunnelDown(rule); err != nil {
		return err
	}
	_, err := policy.ParseRule(rule.Name, rule.Via, rule.Match.IPRanges, rule.Match.Domains, rule.Match.Apps, 1)
	if err != nil {
		return err
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

// policyPatch is the JSON body for policy create/update/test. config.PolicyRule
// carries only yaml tags, so decoding JSON straight into it silently dropped
// "ip_ranges"; pointer fields also let an update apply only what was sent.
type policyPatch struct {
	Name *string `json:"name"`
	Via  *string `json:"via"`
	// OnTunnelDown: allow (default) | block | connect.
	OnTunnelDown *string `json:"on_tunnel_down"`
	Match        *struct {
		IPRanges *[]string `json:"ip_ranges"`
		Domains  *[]string `json:"domains"`
		Apps     *[]string `json:"apps"`
	} `json:"match"`
}

// applyTo overlays the fields present in p onto base and returns the result.
func (p policyPatch) applyTo(base config.PolicyRule) config.PolicyRule {
	if p.Name != nil {
		base.Name = *p.Name
	}
	if p.OnTunnelDown != nil {
		base.OnTunnelDown = *p.OnTunnelDown
	}
	if p.Via != nil {
		base.Via = *p.Via
	}
	if p.Match != nil {
		if p.Match.IPRanges != nil {
			base.Match.IPRanges = *p.Match.IPRanges
		}
		if p.Match.Domains != nil {
			base.Match.Domains = *p.Match.Domains
		}
		if p.Match.Apps != nil {
			base.Match.Apps = *p.Match.Apps
		}
	}
	return base
}

// clonePolicyRule deep-copies r so normalizing the copy never mutates the
// slices still referenced by the loaded config.
func clonePolicyRule(r config.PolicyRule) config.PolicyRule {
	r.Match.IPRanges = append([]string(nil), r.Match.IPRanges...)
	r.Match.Domains = append([]string(nil), r.Match.Domains...)
	r.Match.Apps = append([]string(nil), r.Match.Apps...)
	return r
}
