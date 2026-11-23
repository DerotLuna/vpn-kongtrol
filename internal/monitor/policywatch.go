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
// Notices are throttled to one per policy per NoticeEvery. Each profile
// gets at most three traffic-triggered attempts between successful connections.
type PolicyWatch struct {
	Matcher interface {
		MatchAddr(a netip.Addr) (PolicyMatch, bool)
	}
	Status  func(profile string) vpn.Status
	Connect func(ctx context.Context, profile string) error
	// ConnectConflict reports an active VPN that should defer this automatic start.
	ConnectConflict func(profile string) string
	Events          *EventLog
	// OnEvent is called for every new event (daemon log / audit).
	OnEvent func(Event)

	NoticeEvery time.Duration

	conns func() ([]netip.Addr, error)

	mu         sync.Mutex
	lastNotice map[string]time.Time
	retries    map[string]policyConnectRetry
}

type policyConnectRetry struct {
	attempts     int
	inFlight     bool
	retryAt      time.Time
	lastConflict time.Time
}

const (
	DefaultNoticeEvery   = 30 * time.Minute
	policyConnectLimit   = 3
	policyConnectBackoff = 3 * time.Minute
)

func (w *PolicyWatch) init() {
	if w.lastNotice == nil {
		w.lastNotice = map[string]time.Time{}
		w.retries = map[string]policyConnectRetry{}
	}
	if w.conns == nil {
		w.conns = activeRemoteAddrs
	}
	if w.NoticeEvery <= 0 {
		w.NoticeEvery = DefaultNoticeEvery
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
		switch w.Status(m.Via).Normalize() {
		case vpn.StatusConnected:
			w.ResetConnectBudget(m.Via)
		case vpn.StatusConnecting:
			continue
		default:
			w.handle(ctx, m)
		}
	}
}

func (w *PolicyWatch) handle(ctx context.Context, m PolicyMatch) {
	if m.Action == config.OnTunnelDownConnect {
		w.startConnect(ctx, m)
		return
	}
	now := time.Now()
	w.mu.Lock()
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

// ResetConnectBudget re-arms traffic-triggered connection after a tunnel comes up.
func (w *PolicyWatch) ResetConnectBudget(profile string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.retries[profile].inFlight {
		// Keep the reservation until the running attempt finishes. Its outcome
		// belongs to the budget that was just reset.
		w.retries[profile] = policyConnectRetry{inFlight: true}
		return
	}
	delete(w.retries, profile)
}

func (w *PolicyWatch) startConnect(ctx context.Context, m PolicyMatch) {
	if w.Connect == nil || ctx.Err() != nil {
		return
	}
	var conflict string
	if w.ConnectConflict != nil {
		conflict = w.ConnectConflict(m.Via)
	}
	now := time.Now()
	w.mu.Lock()
	retry := w.retries[m.Via]
	if retry.inFlight || retry.attempts >= policyConnectLimit || now.Before(retry.retryAt) {
		w.mu.Unlock()
		return
	}
	if conflict != "" {
		notify := retry.lastConflict.IsZero() || now.Sub(retry.lastConflict) >= w.NoticeEvery
		if notify {
			retry.lastConflict = now
			w.retries[m.Via] = retry
		}
		w.mu.Unlock()
		if notify {
			w.emit(Event{Kind: EventPolicyTunnelDown, Level: "WARN", Key: "policy.event.connect_conflict",
				Args: []string{m.Label, m.Via, conflict}, Profile: m.Via, Policy: m.Policy, Action: m.Action})
		}
		return
	}
	retry.attempts++
	retry.inFlight = true
	w.retries[m.Via] = retry
	w.mu.Unlock()

	e := Event{Kind: EventPolicyTunnelDown, Level: "INFO", Key: "policy.event.connect",
		Args: []string{m.Label, m.Via}, Profile: m.Via, Policy: m.Policy, Action: m.Action}
	w.emit(e)
	go func() {
		if key := w.finishConnect(m.Via, w.Connect(ctx, m.Via)); key != "" && ctx.Err() == nil {
			e.Level, e.Key, e.CanConnect = "WARN", key, true
			w.emit(e)
		}
	}()
}

func (w *PolicyWatch) finishConnect(profile string, err error) string {
	w.mu.Lock()
	defer w.mu.Unlock()
	retry := w.retries[profile]
	if err == nil || retry.attempts == 0 {
		delete(w.retries, profile)
		return ""
	}
	retry.inFlight = false
	key := "policy.event.connect_exhausted"
	if retry.attempts < policyConnectLimit {
		retry.retryAt = time.Now().Add(policyConnectBackoff << (retry.attempts - 1))
		key = "policy.event.connect_failed"
	}
	w.retries[profile] = retry
	return key
}

func (w *PolicyWatch) emit(e Event) {
	if w.Events != nil {
		e = w.Events.Add(e)
	}
	if w.OnEvent != nil {
		w.OnEvent(e)
	}
}
