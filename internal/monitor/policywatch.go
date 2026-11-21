package monitor

import (
	"context"
	"net/netip"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// PolicyWatch notices traffic for a policy whose tunnel is off (the user
// decides which tunnels are up; Kongtrol only tells them, or connects when
// the effective action is connect). It samples the machine's TCP table,
// which can miss short-lived or rejected sockets. The firewall runs
// independently of this observer.
//
// Notices are throttled to one per policy per NoticeEvery; automatic
// connects to one attempt per tunnel per ConnectEvery.
type PolicyWatch struct {
	Matcher interface {
		MatchAddr(a netip.Addr) (PolicyMatch, bool)
	}
	Status  func(profile string) vpn.Status
	Connect func(ctx context.Context, profile string) error
	Events  *EventLog
	// OnEvent is called for every new event (daemon log / audit).
	OnEvent func(Event)

	NoticeEvery  time.Duration
	ConnectEvery time.Duration

	conns func() ([]netip.Addr, error)
	now   func() time.Time

	mu          sync.Mutex
	lastNotice  map[string]time.Time // policy → last notice
	lastConnect map[string]time.Time // profile → last automatic connect
}

// Defaults for the throttles.
const (
	DefaultNoticeEvery  = 30 * time.Minute
	DefaultConnectEvery = 3 * time.Minute
)

func (w *PolicyWatch) init() {
	if w.lastNotice == nil {
		w.lastNotice = map[string]time.Time{}
		w.lastConnect = map[string]time.Time{}
	}
	if w.conns == nil {
		w.conns = activeRemoteAddrs
	}
	if w.now == nil {
		w.now = time.Now
	}
	if w.NoticeEvery <= 0 {
		w.NoticeEvery = DefaultNoticeEvery
	}
	if w.ConnectEvery <= 0 {
		w.ConnectEvery = DefaultConnectEvery
	}
}

// Run checks every interval until ctx ends.
func (w *PolicyWatch) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.Check(ctx)
		}
	}
}

// Check looks at the open connections once.
func (w *PolicyWatch) Check(ctx context.Context) {
	w.mu.Lock()
	w.init()
	w.mu.Unlock()
	if w.Matcher == nil || w.Status == nil {
		return
	}
	addrs, err := w.conns()
	if err != nil {
		return
	}
	done := map[string]bool{} // one decision per policy per check
	seen := map[netip.Addr]bool{}
	for _, a := range addrs {
		if seen[a] || !a.IsValid() || a.IsLoopback() || a.IsUnspecified() {
			continue
		}
		seen[a] = true
		m, ok := w.Matcher.MatchAddr(a)
		if !ok || done[m.Policy] {
			continue
		}
		done[m.Policy] = true
		if st := w.Status(m.Via).Normalize(); st == vpn.StatusConnected || st == vpn.StatusConnecting {
			continue // up, or coming up
		}
		w.handle(ctx, m)
	}
}

func (w *PolicyWatch) handle(ctx context.Context, m PolicyMatch) {
	now := w.now()
	w.mu.Lock()
	if m.Action == config.OnTunnelDownConnect {
		if last, ok := w.lastConnect[m.Via]; ok && now.Sub(last) < w.ConnectEvery {
			w.mu.Unlock()
			return
		}
		w.lastConnect[m.Via] = now
		w.mu.Unlock()
		w.emit(Event{Kind: EventPolicyTunnelDown, Level: "INFO", Key: "policy.event.connect",
			Args: []string{m.Label, m.Via}, Profile: m.Via, Policy: m.Policy, Action: m.Action})
		if w.Connect != nil {
			go func() {
				if err := w.Connect(ctx, m.Via); err != nil && ctx.Err() == nil {
					w.emit(Event{Kind: EventPolicyTunnelDown, Level: "WARN", Key: "policy.event.connect_failed",
						Args: []string{m.Label, m.Via}, Profile: m.Via, Policy: m.Policy, Action: m.Action, CanConnect: true})
				}
			}()
		}
		return
	}
	if last, ok := w.lastNotice[m.Policy]; ok && now.Sub(last) < w.NoticeEvery {
		w.mu.Unlock()
		return
	}
	w.lastNotice[m.Policy] = now
	w.mu.Unlock()
	key := "policy.event.allow"
	if m.Action == config.OnTunnelDownBlock {
		key = "policy.event.block"
	}
	w.emit(Event{Kind: EventPolicyTunnelDown, Level: "WARN", Key: key, Args: []string{m.Label, m.Via},
		Profile: m.Via, Policy: m.Policy, Action: m.Action, CanConnect: true})
}

func (w *PolicyWatch) emit(e Event) {
	if w.Events != nil {
		e = w.Events.Add(e)
	}
	if w.OnEvent != nil {
		w.OnEvent(e)
	}
}
