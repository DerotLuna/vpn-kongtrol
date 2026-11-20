package api

import (
	"net/http"
	"sync"
)

// changeHub tells live clients that the daemon's settings changed (ADR-0072):
// every successful mutating API request (profiles, groups, rules, security
// toggles, settings, preferences, reload) bumps a revision and wakes the
// WebSocket connections that asked for events, so another tab of the panel
// and the tray flyout refresh at once instead of on their next show. It
// carries no data: clients refetch through the API they already use.
type changeHub struct {
	mu   sync.Mutex
	rev  uint64
	subs map[chan struct{}]struct{}
}

func (h *changeHub) subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	if h.subs == nil {
		h.subs = make(map[chan struct{}]struct{})
	}
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

// bump records one change and wakes every subscriber; a subscriber that
// has not read the previous wake-up keeps one pending (changes coalesce).
func (h *changeHub) bump() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rev++
	for ch := range h.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (h *changeHub) revision() uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.rev
}

// changeRecorder sees the status a mutating handler answered with, so only
// a request that succeeded announces a change.
type changeRecorder struct {
	http.ResponseWriter
	status int
}

func (w *changeRecorder) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *changeRecorder) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// serveAnnounced serves a mutating request and bumps the change hub when
// it succeeded (2xx). Reads never announce anything.
func (s *Server) serveAnnounced(next http.Handler, w http.ResponseWriter, r *http.Request) {
	if !isMutation(r.Method) {
		s.serveAudited(next, w, r)
		return
	}
	rec := &changeRecorder{ResponseWriter: w}
	s.serveAudited(next, rec, r)
	status := rec.status
	if status == 0 {
		status = http.StatusOK
	}
	if status >= 200 && status < 300 {
		s.changes.bump()
	}
}
