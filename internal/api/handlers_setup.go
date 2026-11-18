package api

import (
	"errors"
	"net/http"
	"os"
)

// SetupStatus is the body of GET /api/v1/setup. A daemon started with no
// kongtrol.yaml runs in setup mode: the dashboard shows the "create your
// first VPN" guide while NeedsSetup is true, and restarts the daemon once
// RestartPending says the first profile was saved (the adapters map is
// built only at boot, see AGENTS.md "Dashboard REST API").
type SetupStatus struct {
	SetupMode      bool   `json:"setup_mode"`
	NeedsSetup     bool   `json:"needs_setup"`
	RestartPending bool   `json:"restart_pending"`
	ConfigPath     string `json:"config_path"`
}

// SetSetupMode marks this daemon as started with no config file.
func (s *Server) SetSetupMode(on bool) { s.setupMode.Store(on) }

// SetRestarter enables POST /api/v1/restart. fn must start the replacement
// daemon (which waits for this one to stop) and return an error, already
// translated, when it could not; the server then stops this daemon through
// onShutdown, exactly like POST /api/v1/shutdown.
func (s *Server) SetRestarter(fn func() error) { s.restart = fn }

func (s *Server) configFileExists() bool {
	if s.configPath == "" {
		return false
	}
	_, err := os.Stat(s.configPath)
	return err == nil
}

// GET /api/v1/setup
func (s *Server) handleSetupStatus(w http.ResponseWriter, _ *http.Request) {
	setup := s.setupMode.Load()
	has := s.configFileExists()
	writeJSON(w, http.StatusOK, SetupStatus{
		SetupMode:      setup,
		NeedsSetup:     setup && !has,
		RestartPending: setup && has,
		ConfigPath:     s.configPath,
	})
}

// POST /api/v1/restart — stop this daemon (tunnels disconnect like on
// `kongtrol down`) and start a fresh one that reads kongtrol.yaml again and
// reconnects the tunnels that were up. Used after VPN profile changes
// (restart_required) and to leave setup mode.
func (s *Server) handleRestart(w http.ResponseWriter, _ *http.Request) {
	if s.restart == nil || s.onShutdown == nil {
		writeError(w, http.StatusNotImplemented, "restart not supported by this daemon instance")
		return
	}
	if err := s.restart(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "restarting"})
	go s.onShutdown()
}

// errSetupNoConfig is returned by loadRuntimeConfig outside setup mode when
// no config file exists.
var errSetupNoConfig = errors.New("config path not found")
