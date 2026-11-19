package api

import (
	"net/http"
	"strconv"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// SetRecentConnections enables GET /api/v1/connections/recent.
func (s *Server) SetRecentConnections(r *monitor.RecentConnections) { s.recent = r }

// RecentConnectionsResponse is the body of GET /api/v1/connections/recent.
type RecentConnectionsResponse struct {
	// Supported is false on an OS where Kongtrol cannot list connections
	// without admin rights (macOS today).
	Supported bool `json:"supported"`
	// Enabled mirrors monitor.recent_connections.enabled.
	Enabled     bool                       `json:"enabled"`
	Max         int                        `json:"max"`
	TTLSeconds  int                        `json:"ttl_seconds"`
	Connections []monitor.RecentConnection `json:"connections"`
}

// GET /api/v1/connections/recent[?limit=n] — the destinations this
// computer connected to lately (ADR-0054): kept only in the daemon's
// memory, served only here (authenticated, loopback), never logged.
func (s *Server) handleRecentConnections(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "api.err.recent_limit"})
			return
		}
		limit = n
	}
	out := RecentConnectionsResponse{
		Max:         monitor.RecentMax,
		TTLSeconds:  int(monitor.RecentTTL.Seconds()),
		Connections: []monitor.RecentConnection{},
	}
	if s.recent != nil {
		out.Supported = s.recent.Supported()
		out.Enabled = s.recent.Enabled()
		out.Connections = s.recent.List(limit)
	}
	writeJSON(w, http.StatusOK, out)
}
