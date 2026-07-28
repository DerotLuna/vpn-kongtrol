package api

import (
	"context"
	"net/http"
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
	adapter, ok := s.adapters[name]
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

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Minute)
	if !s.startPendingConnect(name, cancel) {
		cancel()
		writeError(w, http.StatusConflict, "connect already in progress for "+name)
		return
	}
	go func() {
		defer s.clearPendingConnect(name)
		_ = s.connectProfile(ctx, name)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "connecting", "tunnel": name})
}

// POST /api/v1/tunnels/{name}/cancel_connect
func (s *Server) handleCancelConnect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_, ok := s.adapters[name]
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
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "connect_cancelled", "tunnel": name})
}

// POST /api/v1/tunnels/{name}/disconnect
func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_, ok := s.adapters[name]
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
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected", "tunnel": name})
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
