package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
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

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	// AdapterConfig is pre-loaded by the orchestrator; for API-triggered
	// connects we use the last known config from the adapter's state.
	// Detailed credential injection is handled by the CLI's connect flow.
	if err := adapter.Reconnect(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "connected", "tunnel": name})
}

// POST /api/v1/tunnels/{name}/disconnect
func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	adapter, ok := s.adapters[name]
	if !ok {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := adapter.Disconnect(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected", "tunnel": name})
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
	}
	out := make([]routeDTO, len(routes))
	for i, r := range routes {
		dto := routeDTO{
			Destination: r.Destination.String(),
			Interface:   r.Interface,
			Metric:      r.Metric,
		}
		if r.Gateway != nil {
			dto.Gateway = r.Gateway.String()
		}
		out[i] = dto
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /api/v1/security/status
func (s *Server) handleSecurityStatus(w http.ResponseWriter, r *http.Request) {
	type secStatus struct {
		KillSwitch bool        `json:"kill_switch"`
		DNSGuard   bool        `json:"dns_guard"`
		LeakCheck  interface{} `json:"leak_check"`
	}

	status := secStatus{
		KillSwitch: s.ks != nil && s.ks.IsEnabled(),
		DNSGuard:   s.dnsMgr != nil && s.dnsMgr.IsActive(),
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

// GET /api/v1/resolve?target=<ip-or-domain>&app=<exe-or-path> — which VPN handles this match.
func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
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

	if app != "" {
		if vpnName, matched := s.policyEngine.ResolveApp(app); matched {
			result.Via = vpnName
			result.Matched = true
			for _, rule := range s.policyEngine.Rules() {
				if rule.Via == vpnName && rule.MatchesApp(app) {
					result.Rule = rule.Name
					break
				}
			}
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
			for _, rule := range s.policyEngine.Rules() {
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
			for _, rule := range s.policyEngine.Rules() {
				if rule.Via == vpnName && rule.MatchesDomain(target) {
					result.Rule = rule.Name
					break
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, result)
}
