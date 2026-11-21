package api

import (
	"encoding/json"
	"net/http"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// GET /api/v1/vpns/{name}/default-egress
func (s *Server) handleDefaultEgress(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := s.loadRuntimeConfig()
	if err != nil {
		writeLocalizedError(w, r, http.StatusInternalServerError, err)
		return
	}
	name := r.PathValue("name")
	profile, ok := cfg.VPNs[name]
	if !ok {
		writeLocalizedError(w, r, http.StatusNotFound, &config.ValidationError{Key: "api.err.vpn_not_found", Args: []any{name}})
		return
	}
	writeJSON(w, http.StatusOK, profile.DefaultEgressStatus())
}

// PUT /api/v1/vpns/{name}/default-egress
func (s *Server) handleSetDefaultEgress(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Enabled == nil {
		writeLocalizedError(w, r, http.StatusBadRequest, errInvalidJSON)
		return
	}
	cfg, path, err := s.loadRuntimeConfig()
	if err != nil {
		writeLocalizedError(w, r, http.StatusInternalServerError, err)
		return
	}
	name := r.PathValue("name")
	profile, ok := cfg.VPNs[name]
	if !ok {
		writeLocalizedError(w, r, http.StatusNotFound, &config.ValidationError{Key: "api.err.vpn_not_found", Args: []any{name}})
		return
	}
	if profile.DefaultEgress != *req.Enabled {
		s.connectMu.Lock()
		_, connecting := s.connectCancel[name]
		s.connectMu.Unlock()
		a := s.adapterOf(name)
		if connecting || (a != nil && busyDefaultEgress(a.Status().Normalize())) {
			writeLocalizedError(w, r, http.StatusConflict, &config.ValidationError{Key: "config.err.default_egress_busy"})
			return
		}
	}
	if err := configedit.SetDefaultEgress(cfg, name, *req.Enabled, s.saver(path)); err != nil {
		writeLocalizedError(w, r, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg.VPNs[name].DefaultEgressStatus())
}

func busyDefaultEgress(status vpn.Status) bool {
	return status == vpn.StatusConnected || status == vpn.StatusConnecting
}
