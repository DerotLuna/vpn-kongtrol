package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// POST /api/v1/shutdown asks the daemon to terminate gracefully: it triggers
// the same root-context cancellation as Ctrl+C/SIGTERM, so the daemon's own
// deferred cleanup (kill switch teardown, DNS restore, history flush, PID
// file removal) runs before the process exits — unlike a caller sending
// SIGKILL/TerminateProcess directly, which skips all of it. This works
// identically on every OS, including Windows, where there is no reliable
// SIGTERM equivalent deliverable across processes.
func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if s.onShutdown == nil {
		writeError(w, http.StatusNotImplemented, "shutdown not supported by this daemon instance")
		return
	}
	// The user stops everything: nothing is reconnected at the next start
	// (ADR-0098). A restart does not come through here.
	if s.startup.OnStopAll != nil {
		s.startup.OnStopAll()
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "shutting_down"})
	go s.onShutdown()
}

// GET /api/v1/metrics/history
func (s *Server) handleMetricsHistory(w http.ResponseWriter, r *http.Request) {
	if s.collector == nil {
		writeJSON(w, http.StatusOK, map[string]monitor.ProfileHistory{})
		return
	}
	writeJSON(w, http.StatusOK, s.collector.HistorySnapshot())
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
	adapter, ok := s.adapters.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}
	if st := adapter.Status().Normalize(); st == vpn.StatusConnected {
		writeJSON(w, http.StatusOK, map[string]string{"status": "already_connected", "tunnel": name})
		return
	}
	if s.connectProfile == nil {
		writeError(w, http.StatusServiceUnavailable, "connection service unavailable")
		return
	}
	// A connect that would start a second full tunnel is refused at once
	// (409 + translated "what to do"), not after the 202 (ADR-0099).
	if s.connectCheck != nil {
		if err := s.connectCheck(name); err != nil {
			writeLocalizedError(w, r, http.StatusConflict, err)
			return
		}
	}

	// Generous: provider adapters (attach mode) may wait for the user to
	// connect in the app; they bound that wait with the profile's
	// wait_timeout themselves.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 30*time.Minute)
	if !s.startPendingConnect(name, cancel) {
		cancel()
		writeError(w, http.StatusConflict, "connect already in progress for "+name)
		return
	}
	// A new attempt supersedes the previous failure; this one's outcome is
	// recorded when it ends so clients (`up`, dashboard, tray, `status`)
	// can show why an accepted (202) connect did not come up.
	s.collector.ClearConnectError(name)
	go func() {
		defer s.clearPendingConnect(name)
		s.collector.RecordConnectError(name, connectOutcome(s.connectProfile(ctx, name)))
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "connecting", "tunnel": name})
}

// SetConnectCheck sets the synchronous check POST .../connect runs before
// accepting a connect (ProfileService.CheckFullTunnelConflict).
func (s *Server) SetConnectCheck(fn func(name string) error) { s.connectCheck = fn }

// POST /api/v1/tunnels/{name}/cancel_connect
func (s *Server) handleCancelConnect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_, ok := s.adapters.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}
	s.cancelPendingConnect(name)

	if s.disconnectProfile == nil {
		writeError(w, http.StatusServiceUnavailable, "connection service unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := s.disconnectProfile(ctx, name); err != nil {
		writeGuidedError(w, r, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "connect_cancelled", "tunnel": name})
}

// POST /api/v1/tunnels/{name}/disconnect
func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_, ok := s.adapters.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, "tunnel not found: "+name)
		return
	}
	s.cancelPendingConnect(name)

	if s.disconnectProfile == nil {
		writeError(w, http.StatusServiceUnavailable, "connection service unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := s.disconnectProfile(ctx, name); err != nil {
		if errors.Is(err, vpn.ErrManualDisconnect) {
			// Provider app without a CLI: the user disconnects in the app;
			// the tunnel's attach phase is waiting_disconnect until it drops.
			writeJSON(w, http.StatusAccepted, map[string]string{"status": "waiting_user", "tunnel": name})
			return
		}
		writeGuidedError(w, r, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected", "tunnel": name})
}

// POST /api/v1/tunnels/{name}/reload — re-reads kongtrol.yaml from disk
// (hot-swapping the policy engine along the way, same as
// handlePolicyReload/handleReloadGroup), then restarts this single profile
// in place if it's currently connected: disconnect then reconnect through
// the existing adapter, so routes/DNS/kill-switch settings edited by hand
// for this one tunnel take effect without restarting its whole group or the
// daemon. This is the single-tunnel counterpart to handleReloadGroup, for
// picking up a hand edit scoped to one connection instead of a whole group.
func (s *Server) handleReloadTunnel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.reloadRuntimeConfig(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	restarted, skipped, missing, errs := s.restartProfilesInPlace(r, []string{name})
	if len(missing) > 0 {
		writeJSON(w, http.StatusConflict, map[string]any{
			"status":  "restart_required",
			"tunnel":  name,
			"message": "this profile is not registered with the running daemon (likely added by a hand edit) — a full process restart ('kongtrol down' then 'kongtrol up') is required before it can connect",
		})
		return
	}
	if errs != nil {
		writeError(w, http.StatusInternalServerError, strings.Join(errs, "; "))
		return
	}

	status := "restarting"
	if len(skipped) > 0 {
		status = "skipped_not_connected"
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":    status,
		"tunnel":    name,
		"restarted": restarted,
		"skipped":   skipped,
	})
}

// connectOutcome maps an async connect's result to the error worth keeping
// as the tunnel's last error: a cancelled connect (cancel_connect,
// disconnect, shutdown) was a user decision, not a failure.
func connectOutcome(err error) error {
	if err == nil || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func (s *Server) startPendingConnect(name string, cancel context.CancelFunc) bool {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	if _, exists := s.connectCancel[name]; exists {
		return false
	}
	s.connectCancel[name] = cancel
	return true
}

func (s *Server) clearPendingConnect(name string) {
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	delete(s.connectCancel, name)
}

func (s *Server) cancelPendingConnect(name string) {
	s.connectMu.Lock()
	cancel, ok := s.connectCancel[name]
	if ok {
		delete(s.connectCancel, name)
	}
	s.connectMu.Unlock()
	if ok {
		cancel()
	}
}
