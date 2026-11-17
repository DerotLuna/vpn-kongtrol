package api

import (
	"net/http"
	"strconv"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// SetKillSwitchStatus sets the source of the split-tunnel kill switch state
// (app.KillSwitchService.Status) reported by /api/v1/security/status.
func (s *Server) SetKillSwitchStatus(fn func() security.KillSwitchStatus) { s.ksStatus = fn }

// SetEventLog sets the daemon notices served by /api/v1/events.
func (s *Server) SetEventLog(l *monitor.EventLog) { s.events = l }

// EventsResponse is the body of GET /api/v1/events.
type EventsResponse struct {
	Events []monitor.Event `json:"events"`
	// Seq is the latest event number: pass it as ?after= on the next poll.
	Seq int64 `json:"seq"`
}

// GET /api/v1/events?after=<seq> — notices newer than seq (keys + args,
// rendered by each client in its own language). A client starting fresh
// passes after=-1 to get only the latest Seq and no backlog.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	resp := EventsResponse{Events: []monitor.Event{}}
	if s.events == nil {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	after, err := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	if err != nil {
		after = 0
	}
	evs, seq := s.events.After(max(after, 0))
	resp.Seq = seq
	if after >= 0 && evs != nil {
		resp.Events = evs
	}
	writeJSON(w, http.StatusOK, resp)
}
