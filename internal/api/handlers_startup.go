package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/vpn-kongtrol/kongtrol/internal/autostart"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// StartupHooks back /api/v1/startup (ADR-0098): "Abrir Kongtrol al
// iniciar sesión" (the OS's per-user autostart entry) and "Al iniciar,
// volver a conectar las VPN que tenías conectadas" (preferences.json).
// OnStopAll runs when a client asks to stop everything
// (POST /api/v1/shutdown: `kongtrol down`, the tray's "Salir y
// desconectar todo"), before the daemon stops; a restart does not call it.
type StartupHooks struct {
	Autostart    func() autostart.State
	SetAutostart func(bool) error
	OnStopAll    func()
}

// SetStartup wires the startup settings and the stop-all hook.
func (s *Server) SetStartup(h StartupHooks) { s.startup = h }

type startupDTO struct {
	StartAtLogin          bool `json:"start_at_login"`
	StartAtLoginSupported bool `json:"start_at_login_supported"`
	RestoreTunnels        bool `json:"restore_tunnels"`
}

func (s *Server) startupStatus() (startupDTO, error) {
	var out startupDTO
	if s.startup.Autostart != nil {
		st := s.startup.Autostart()
		out.StartAtLogin, out.StartAtLoginSupported = st.Enabled, st.Supported
	}
	prefs, err := config.LoadPreferences(s.preferencesPath)
	if err != nil {
		return out, err
	}
	out.RestoreTunnels = prefs.RestoreTunnelsEnabled()
	return out, nil
}

// GET /api/v1/startup
func (s *Server) handleGetStartup(w http.ResponseWriter, r *http.Request) {
	st, err := s.startupStatus()
	if err != nil {
		writeLocalizedError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// PUT /api/v1/startup {"start_at_login"?: bool, "restore_tunnels"?: bool}
func (s *Server) handleSetStartup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StartAtLogin   *bool `json:"start_at_login"`
		RestoreTunnels *bool `json:"restore_tunnels"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil || (req.StartAtLogin == nil && req.RestoreTunnels == nil) {
		writeError(w, http.StatusBadRequest, `invalid JSON body: want {"start_at_login": bool} and/or {"restore_tunnels": bool}`)
		return
	}
	if req.RestoreTunnels != nil {
		on := *req.RestoreTunnels
		if _, err := config.UpdatePreferences(s.preferencesPath, func(p *config.Preferences) error {
			p.RestoreTunnels = &on
			return nil
		}); err != nil {
			writeLocalizedError(w, r, http.StatusInternalServerError, err)
			return
		}
	}
	if req.StartAtLogin != nil {
		if s.startup.SetAutostart == nil {
			writeError(w, http.StatusNotImplemented, "autostart not supported by this daemon instance")
			return
		}
		if err := s.startup.SetAutostart(*req.StartAtLogin); err != nil {
			writeLocalizedError(w, r, http.StatusConflict, AutostartError(err))
			return
		}
	}
	s.changes.bump()
	st, err := s.startupStatus()
	if err != nil {
		writeLocalizedError(w, r, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// AutostartError turns an autostart failure into a guided error (shared
// with `kongtrol config autostart`).
func AutostartError(err error) error {
	if errors.Is(err, autostart.ErrNoTray) {
		return &config.ValidationError{Key: "startup.err.no_tray"}
	}
	return &config.ValidationError{Key: "startup.err.autostart", Args: []any{err.Error()}}
}
