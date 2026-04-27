package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
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
		LeakCheck  interface{} `json:"leak_check"`
	}

	status := secStatus{
		KillSwitch: s.ks != nil && s.ks.IsEnabled(),
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

// routeFromDTO converts a list of routing.Route to a safe DTO for the API.
// Kept here to avoid a circular import between api and routing packages.
type routeEntry routing.Route
