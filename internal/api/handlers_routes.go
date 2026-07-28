package api

import (
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/policy"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

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
