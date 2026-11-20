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
//
// With ?events=1 (the panel and the tray flyout, ADR-0072) every frame is an
// envelope: {"type":"metrics","tunnels":{…}} for the snapshot and
// {"type":"changed","rev":N} after a successful settings change, so the
// page refetches what it shows. Without it (kongtrol up's live view) frames
// stay the bare snapshot map, as before.
// GET /api/v1/ws/metrics
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	events := r.URL.Query().Get("events") == "1"
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	changes, unsubscribe := s.collector.Subscribe()
	defer unsubscribe()
	// A nil channel never fires: clients without ?events=1 get no notices.
	var settings <-chan struct{}
	if events {
		ch, unsubscribeSettings := s.changes.subscribe()
		defer unsubscribeSettings()
		settings = ch
	}

	write := func(v any) bool {
		data, err := json.Marshal(v)
		if err != nil {
			return true
		}
		return conn.WriteMessage(1, data) == nil
	}
	send := func() bool {
		snapshot := s.collector.Snapshot()
		if events {
			return write(map[string]any{"type": "metrics", "tunnels": snapshot})
		}
		return write(snapshot)
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
		case <-settings:
			if !write(map[string]any{"type": "changed", "rev": s.changes.revision()}) {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}
