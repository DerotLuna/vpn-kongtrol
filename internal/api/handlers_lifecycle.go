package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/update"
)

// UpdateStatus is the body of GET/POST /api/v1/updates. Reading it never
// contacts GitHub: Latest/Newer come from the last remembered check.
type UpdateStatus struct {
	Enabled   bool       `json:"check_enabled"`
	Host      string     `json:"host"`
	Current   string     `json:"current"`
	Latest    string     `json:"latest,omitempty"`
	URL       string     `json:"url,omitempty"`
	Newer     bool       `json:"newer"`
	CheckedAt *time.Time `json:"checked_at,omitempty"`
}

// SetUpdateChecker enables /api/v1/updates. last returns the remembered
// result (no request); onToggle runs after the setting was saved.
func (s *Server) SetUpdateChecker(last func() update.Result, onToggle func(enabled bool)) {
	s.updateLast = last
	s.onUpdateToggle = onToggle
}

// SetDataPurger enables POST /api/v1/data/purge. fn disconnects every
// tunnel and deletes the user's data and Kongtrol's keychain entries,
// keeping only what this running daemon needs; it returns an error naming
// what could not be deleted.
func (s *Server) SetDataPurger(fn func() error) { s.purgeData = fn }

func (s *Server) updateStatus(enabled bool) UpdateStatus {
	st := UpdateStatus{Enabled: enabled, Host: "api.github.com"}
	if s.updateLast != nil {
		r := s.updateLast()
		st.Current, st.Latest, st.URL = r.Current, r.Latest, r.URL
		st.Newer = enabled && r.Newer
		if !r.CheckedAt.IsZero() {
			t := r.CheckedAt
			st.CheckedAt = &t
		}
	}
	return st
}

// GET /api/v1/updates
func (s *Server) handleGetUpdates(w http.ResponseWriter, _ *http.Request) {
	enabled := false
	if cfg, _, err := s.loadRuntimeConfig(); err == nil {
		enabled = cfg.UpdateCheckEnabled()
	}
	writeJSON(w, http.StatusOK, s.updateStatus(enabled))
}

// POST /api/v1/updates {"enabled": bool} — opt in/out of the update notice.
// With no kongtrol.yaml yet (setup mode) it answers 409: there is nowhere to
// save the choice until the first VPN exists.
func (s *Server) handleSetUpdates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Enabled == nil {
		writeError(w, http.StatusBadRequest, `invalid JSON body: want {"enabled": true|false}`)
		return
	}
	cfg, cfgPath, err := s.loadRuntimeConfig()
	if err != nil {
		writeLocalizedError(w, r, http.StatusConflict, &config.ValidationError{Key: "cli.update.no_config"})
		return
	}
	cfg.Updates.CheckEnabled = *req.Enabled
	if err := s.saveRuntimeConfig(cfgPath, cfg); err != nil {
		writeLocalizedError(w, r, http.StatusInternalServerError, err)
		return
	}
	if s.onUpdateToggle != nil {
		s.onUpdateToggle(*req.Enabled)
	}
	writeJSON(w, http.StatusOK, s.updateStatus(*req.Enabled))
}

// POST /api/v1/data/purge {"confirm": true} — the dashboard's "Borrar
// todos mis datos": disconnect everything, delete settings, profiles and
// saved passwords, then restart into first-run setup. The app stays
// installed. Same auth as every /api/ route (token or control cookie, same
// origin, JSON body only), plus the explicit confirm field.
func (s *Server) handlePurgeData(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !req.Confirm {
		writeLocalizedError(w, r, http.StatusBadRequest, &config.ValidationError{Key: "purge.err.confirm"})
		return
	}
	if s.purgeData == nil {
		writeError(w, http.StatusNotImplemented, "data purge not supported by this daemon instance")
		return
	}
	err := s.purgeData()
	if err != nil {
		writeLocalizedError(w, r, http.StatusInternalServerError,
			&config.ValidationError{Key: "purge.err.failed", Args: []any{err.Error()}})
		return
	}
	restarting := s.restart != nil && s.onShutdown != nil && s.restart() == nil
	writeJSON(w, http.StatusOK, map[string]any{"status": "purged", "restarting": restarting})
	if restarting {
		go s.onShutdown()
	}
}
