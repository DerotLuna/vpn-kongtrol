package monitor

import (
	"sync"
	"time"
)

// Event is a daemon notice for the user (e.g. "claude.ai is going over the
// normal internet · brainbox is off"). The daemon never renders it: Key is
// an internal/i18n key and Args its arguments, so each client (tray,
// dashboard, up/status --watch) shows it in its own language.
type Event struct {
	Seq     int64     `json:"seq"`
	Time    time.Time `json:"time"`
	Kind    string    `json:"kind"`  // EventPolicyTunnelDown, …
	Level   string    `json:"level"` // INFO | WARN
	Key     string    `json:"key"`
	Args    []string  `json:"args,omitempty"`
	Profile string    `json:"profile,omitempty"` // tunnel the notice is about
	Policy  string    `json:"policy,omitempty"`
	Action  string    `json:"action,omitempty"` // on_tunnel_down: allow | block | connect
	// CanConnect: clients offer a "Connect <Profile>" button.
	CanConnect bool `json:"can_connect,omitempty"`
}

// EventPolicyTunnelDown: traffic for a policy whose tunnel is off.
const EventPolicyTunnelDown = "policy_tunnel_down"

// EventLog keeps the latest events in memory for clients to poll
// (GET /api/v1/events?after=<seq>).
type EventLog struct {
	mu     sync.Mutex
	seq    int64
	events []Event
	max    int
}

// NewEventLog keeps up to max events.
func NewEventLog(max int) *EventLog {
	if max <= 0 {
		max = 100
	}
	return &EventLog{max: max}
}

// Add stores e (Seq and Time are set here) and returns it.
func (l *EventLog) Add(e Event) Event {
	l.mu.Lock()
	l.seq++
	e.Seq = l.seq
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	l.events = append(l.events, e)
	if len(l.events) > l.max {
		l.events = l.events[len(l.events)-l.max:]
	}
	l.mu.Unlock()
	return e
}

// After returns the events with Seq > after and the latest Seq.
func (l *EventLog) After(after int64) ([]Event, int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []Event
	for _, e := range l.events {
		if e.Seq > after {
			out = append(out, e)
		}
	}
	return out, l.seq
}
