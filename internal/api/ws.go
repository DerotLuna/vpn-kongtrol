package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleWebSocket pushes live metrics to browser clients: immediately on
// any real tunnel-state change (via the collector's change broadcast), and
// otherwise on a heartbeat matched to the collector's own poll interval —
// there's no point waking faster than the source of truth updates, and the
// heartbeat still carries in-flight byte counters, which change
// continuously during an active transfer without counting as a "state
// change" on their own.
// GET /api/v1/ws/metrics
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	changes, unsubscribe := s.collector.Subscribe()
	defer unsubscribe()

	send := func() bool {
		snapshot := s.collector.Snapshot()
		data, err := json.Marshal(snapshot)
		if err != nil {
			return true
		}
		return conn.WriteMessage(1, data) == nil
	}

	if !send() {
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !send() {
				return
			}
		case <-changes:
			if !send() {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}
