package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleWebSocket pushes live metrics to browser clients every second.
// GET /api/v1/ws/metrics
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			snapshot := s.collector.Snapshot()
			data, err := json.Marshal(snapshot)
			if err != nil {
				continue
			}
			if err := conn.WriteMessage(1, data); err != nil {
				// Client disconnected.
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}
