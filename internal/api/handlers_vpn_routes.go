package api

import (
	"net/http"

	"github.com/vpn-kongtrol/kongtrol/internal/app"
)

// vpnOwnRoutesResponse answers GET /api/v1/vpns/routes (ADR-0107).
type vpnOwnRoutesResponse struct {
	Profiles []app.ProfileOwnRoutes `json:"profiles"`
}

// GET /api/v1/vpns/routes[?profile=<name>] — the routes each VPN brings by
// itself (.ovpn route lines, server-pushed routes, WireGuard AllowedIPs),
// read-only, with the user's rules that overlap them.
func (s *Server) handleVPNOwnRoutes(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	only := r.URL.Query().Get("profile")
	if only != "" {
		if _, ok := cfg.VPNs[only]; !ok {
			writeError(w, http.StatusNotFound, "profile not found: "+only)
			return
		}
	}
	writeJSON(w, http.StatusOK, vpnOwnRoutesResponse{Profiles: app.OwnRoutes(cfg, s.adapters, only)})
}
