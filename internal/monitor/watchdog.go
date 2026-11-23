package monitor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// DefaultRetryDelays is the reconnect policy (ADR-0098): one automatic
// attempt after each delay, then the watchdog gives up for this outage and
// tells the user once. Three attempts like the rule auto-connect
// (ADR-0095); the first comes fast for a blip, the last ~2.5 min later.
var DefaultRetryDelays = []time.Duration{2 * time.Second, 30 * time.Second, 2 * time.Minute}

// RetryDelay returns the wait before the next automatic attempt when made
// attempts were already made in this round, or false when the round's
// budget is spent (the watchdog gives up until a re-arm signal).
func RetryDelay(delays []time.Duration, made int) (time.Duration, bool) {
	if made < 0 || made >= len(delays) {
		return 0, false
	}
	return delays[made], true
}

const (
	watchdogPoll = 5 * time.Second
	// watchdogConnectTimeout bounds a single reconnect/failover attempt.
	// Without it, a hung adapter driver (e.g. a stuck subprocess or a
	// blocking syscall) would stall this profile's watcher goroutine
	// indefinitely — no further health checks or reconnect attempts for
	// that profile until the process restarts.
	watchdogConnectTimeout = 2 * time.Minute
)

// ConnectFunc is a callback that reconnects a named profile.
type ConnectFunc func(ctx context.Context, name string) error

// HealthResult is the outcome of a per-profile active health probe.
type HealthResult struct {
	Healthy   bool
	Reachable bool
	DNSOK     bool
	Latency   time.Duration
	CheckedAt time.Time
	Error     string
}

// HealthCheckFunc performs an active health probe for a profile.
type HealthCheckFunc func(ctx context.Context, name string, adapter vpn.VPNAdapter) HealthResult

// WatchdogEvent is one watchdog lifecycle event. Event values:
//
//   - reconnect_attempt, reconnect_failed, reconnected: Profile is the
//     watched profile.
//   - reconnect_gave_up: once per outage, every attempt of the round
//     failed and no declared backup took over; the watchdog stops
//     retrying Profile (it never connects another profile) until the
//     user connects it or the network changes (retry_rearmed).
//   - reconnect_blocked: once per outage, a reconnect failed with an
//     error only the user can resolve (SetWaitForUser, e.g. another VPN
//     carries all traffic, ADR-0099). Not an attempt; the watchdog waits
//     like after a give-up.
//   - retry_rearmed: after reconnect_gave_up the computer's network
//     changed (other addresses, or back online) or it woke from sleep: a
//     new round starts. The user is not told again.
//   - failover_started, failover_failed, failover_activated: Profile could
//     not reconnect and Target is a backup the user declared for it
//     (vpns.<Profile>.failover). After failover_activated the watchdog
//     stops retrying Profile until the user connects it again.
//   - health_degraded: the active health probe failed (reported only).
//   - manual_reconnect_required: adapter Capabilities().ManualReconnect;
//     notified once per drop, never reconnected.
//   - no_response: once per outage, the tunnel was up but its server
//     stopped answering (vpn.HealthNoResponse; the adapter reports
//     StatusError, so it is reconnected / failed over like a drop).
//   - responding: after no_response, the server answers again.
type WatchdogEvent struct {
	Profile string
	Event   string
	Attempt int
	Err     error
	Target  string
	// AfterGiveUp (reconnected): the user was told this outage gave up,
	// so the recovery deserves a notice too.
	AfterGiveUp bool
}

// Watchdog monitors VPN adapters and reconnects on unexpected disconnects.
// It makes a bounded number of attempts per outage (DefaultRetryDelays,
// ADR-0098), then waits for the network to change, and only watches profiles
// that have been explicitly activated via MarkActive (profiles that were
// brought up). The only other profiles it ever connects are the backups
// the user declared for a profile (ConfigureFailover); nothing is
// inferred from names, adapter types or priority.
type Watchdog struct {
	mu       sync.Mutex
	adapters *vpn.Set
	// watchers holds the cancel func of each running per-profile watcher.
	watchers map[string]context.CancelFunc
	intended map[string]bool   // true = user requested disconnect; skip reconnect
	active   map[string]bool   // true = profile was connected; eligible for watchdog
	replaced map[string]string // profile → declared backup that took over
	connect  ConnectFunc
	log      *zap.Logger
	onEvent  func(WatchdogEvent)

	healthCheck    HealthCheckFunc
	healthInterval time.Duration
	healthTimeout  time.Duration
	lastHealthAt   map[string]time.Time
	lastHealth     map[string]HealthResult

	// failover holds the explicit, ordered backup lists from the config.
	failover map[string][]string

	// inflight holds the cancel funcs of the watchdog's own connect
	// attempts, under every profile they concern (the reconnected
	// profile, or a failover's source and target). MarkIntended cancels
	// them, so `down` stops a reconnect or failover at once.
	inflight map[string]map[uint64]context.CancelFunc
	nextID   uint64

	connectTimeout time.Duration
	poll           time.Duration
	retryDelays    []time.Duration
	// network returns a fingerprint of the computer's network; a change
	// re-arms a profile that gave up (ADR-0098).
	network func() string
	// waitForUser classifies a reconnect error that retrying cannot fix
	// and only the user can (another VPN carries all traffic, ADR-0099):
	// it does not count as an attempt; the watchdog waits (nil = none).
	waitForUser func(error) bool
	// seeded holds attempts spent outside the watchdog (the restore
	// connect at daemon start), taken by the profile's next outage.
	seeded map[string]int

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// SetEventCallback registers an optional callback for watchdog lifecycle
// events (see WatchdogEvent).
func (w *Watchdog) SetEventCallback(fn func(WatchdogEvent)) {
	w.mu.Lock()
	w.onEvent = fn
	w.mu.Unlock()
}

// NewWatchdog creates a watchdog.
// adapters is the daemon's live set: profiles added or removed without a
// restart (ADR-0053) are seen on the next poll.
func NewWatchdog(adapters *vpn.Set, connect ConnectFunc, log *zap.Logger) *Watchdog {
	return &Watchdog{
		adapters:       adapters,
		watchers:       make(map[string]context.CancelFunc),
		intended:       make(map[string]bool),
		active:         make(map[string]bool),
		replaced:       make(map[string]string),
		connect:        connect,
		log:            log,
		lastHealthAt:   make(map[string]time.Time),
		lastHealth:     make(map[string]HealthResult),
		failover:       make(map[string][]string),
		inflight:       make(map[string]map[uint64]context.CancelFunc),
		connectTimeout: watchdogConnectTimeout,
		poll:           watchdogPoll,
		retryDelays:    DefaultRetryDelays,
		network:        NetworkFingerprint,
		seeded:         make(map[string]int),
	}
}

// SetRetryDelays overrides DefaultRetryDelays (one attempt per delay). For
// tests; call before MarkActive/Start.
func (w *Watchdog) SetRetryDelays(delays []time.Duration) {
	w.mu.Lock()
	w.retryDelays = append([]time.Duration(nil), delays...)
	w.mu.Unlock()
}

// SetWaitForUser sets the classifier of reconnect errors only the user
// can resolve (e.g. another VPN already carries all traffic). Such a
// failure stops the round without using an attempt, is told once
// (reconnect_blocked) and is retried on the same signals as a give-up.
// Call before MarkActive/Start.
func (w *Watchdog) SetWaitForUser(fn func(error) bool) {
	w.mu.Lock()
	w.waitForUser = fn
	w.mu.Unlock()
}

// SetNetworkFingerprint overrides NetworkFingerprint. For tests; call
// before MarkActive/Start.
func (w *Watchdog) SetNetworkFingerprint(fn func() string) {
	w.mu.Lock()
	w.network = fn
	w.mu.Unlock()
}

// MarkActiveAfterFailedConnect watches name like MarkActive after a
// connect made outside the watchdog failed (the restore at daemon start):
// that attempt counts against the outage's budget, so the watchdog makes
// only the remaining ones before it gives up and tells the user.
func (w *Watchdog) MarkActiveAfterFailedConnect(name string) {
	w.mu.Lock()
	w.seeded[name] = 1
	w.mu.Unlock()
	w.MarkActive(name)
}

// takeSeeded returns and clears the attempts spent outside the watchdog.
func (w *Watchdog) takeSeeded(name string) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := w.seeded[name]
	delete(w.seeded, name)
	return n
}

// SetConnectTimeout overrides the default per-attempt timeout bounding
// reconnect/failover connect calls. Mainly useful for tests; production
// callers can leave the default (watchdogConnectTimeout) in place.
func (w *Watchdog) SetConnectTimeout(d time.Duration) {
	w.mu.Lock()
	w.connectTimeout = d
	w.mu.Unlock()
}

// SetPollInterval overrides how often each watched profile is checked
// (default 5s). For tests; call before MarkActive/Start.
func (w *Watchdog) SetPollInterval(d time.Duration) {
	w.mu.Lock()
	w.poll = d
	w.mu.Unlock()
}

// ConfigureHealthCheck enables active per-profile health probing.
func (w *Watchdog) ConfigureHealthCheck(interval, timeout time.Duration, check HealthCheckFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.healthInterval = interval
	w.healthTimeout = timeout
	w.healthCheck = check
}

// ConfigureFailover sets the backup profiles the user declared per profile
// (vpns.<name>.failover), in the order to try them. A profile without a
// list never fails over: the watchdog notifies the user and keeps
// retrying that same profile. Names that are not adapters, and a profile
// listed as its own backup, are dropped (config validation rejects both).
func (w *Watchdog) ConfigureFailover(failover map[string][]string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.failover = make(map[string][]string, len(failover))
	for source, targets := range failover {
		if !w.adapters.Has(source) {
			continue
		}
		var list []string
		for _, t := range targets {
			if w.adapters.Has(t) && t != source {
				list = append(list, t)
			}
		}
		if len(list) > 0 {
			w.failover[source] = list
		}
	}
}

// HealthSnapshot returns the latest active health probe result for a profile.
func (w *Watchdog) HealthSnapshot(profile string) (HealthResult, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	v, ok := w.lastHealth[profile]
	return v, ok
}

// MarkIntended signals that a disconnect for profile name was user-requested.
// Call this BEFORE calling Disconnect so the watchdog does not immediately
// attempt to reconnect. Every reconnect or failover the watchdog has in
// flight for this profile (as the profile reconnected, a failover source
// or a failover target) is cancelled before MarkIntended returns.
func (w *Watchdog) MarkIntended(name string) {
	w.mu.Lock()
	w.intended[name] = true
	delete(w.replaced, name)
	cancels := make([]context.CancelFunc, 0, len(w.inflight[name]))
	for _, c := range w.inflight[name] {
		cancels = append(cancels, c)
	}
	w.mu.Unlock()
	for _, c := range cancels {
		c()
	}
}

// MarkActive clears the intended-disconnect flag and starts watching this
// profile for unexpected disconnects. Call AFTER a successful Connect.
func (w *Watchdog) MarkActive(name string) {
	w.mu.Lock()
	delete(w.intended, name)
	delete(w.replaced, name)
	alreadyActive := w.active[name]
	w.active[name] = true
	if !alreadyActive && w.ctx != nil && w.adapters.Has(name) {
		w.spawnLocked(name)
	}
	w.mu.Unlock()
}

// Start begins the watchdog. It returns immediately; call Stop to end it.
// Watcher goroutines are spawned lazily when MarkActive is called for a profile.
func (w *Watchdog) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	w.mu.Lock()
	w.ctx = ctx
	w.cancel = cancel

	// Spawn watchers for any profiles already marked active before Start.
	for name := range w.active {
		if w.adapters.Has(name) {
			w.spawnLocked(name)
		}
	}
	w.mu.Unlock()
}

// Stop shuts down all watchdog goroutines.
func (w *Watchdog) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	w.cancel = nil
	w.ctx = nil
	clear(w.watchers)
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	w.wg.Wait()
}

// paused reports whether the watchdog must leave name alone: the user
// turned it off, or a declared backup took over after a failover.
func (w *Watchdog) paused(name string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.intended[name] || w.replaced[name] != ""
}

func (w *Watchdog) emit(e WatchdogEvent) {
	w.mu.Lock()
	cb := w.onEvent
	w.mu.Unlock()
	if cb != nil {
		cb(e)
	}
}

// connectFor runs one watchdog connect of target on behalf of source
// (the same profile for a plain reconnect). ran is false when the user
// turned source off, or cancelled the attempt with MarkIntended on source
// or target while it ran: neither is a failure to report. The intent
// check and the registration share one lock, so a `down` either sees the
// attempt (and cancels it) or the attempt sees the `down` (and never
// starts).
func (w *Watchdog) connectFor(ctx context.Context, source, target string) (ran bool, err error) {
	w.mu.Lock()
	if w.intended[source] {
		w.mu.Unlock()
		return false, nil
	}
	cctx, cancel := context.WithTimeout(ctx, w.connectTimeout)
	w.nextID++
	id := w.nextID
	names := []string{source}
	if target != source {
		names = append(names, target)
	}
	for _, n := range names {
		if w.inflight[n] == nil {
			w.inflight[n] = make(map[uint64]context.CancelFunc)
		}
		w.inflight[n][id] = cancel
	}
	w.mu.Unlock()

	err = w.connect(cctx, target)

	w.mu.Lock()
	for _, n := range names {
		delete(w.inflight[n], id)
		if len(w.inflight[n]) == 0 {
			delete(w.inflight, n)
		}
	}
	w.mu.Unlock()
	cancelledByUser := errors.Is(cctx.Err(), context.Canceled) && ctx.Err() == nil
	cancel()
	if err != nil && cancelledByUser {
		return false, nil
	}
	return true, err
}

// spawnLocked starts the watcher of name. w.mu must be held and w.ctx set.
func (w *Watchdog) spawnLocked(name string) {
	if cancel, ok := w.watchers[name]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(w.ctx)
	w.watchers[name] = cancel
	w.wg.Go(func() { w.watch(ctx, name) })
}

// Forget drops everything the watchdog knows about name and stops its
// watcher: the profile was removed or renamed without a restart
// (ADR-0053). A profile with the same name added later starts clean.
func (w *Watchdog) Forget(name string) {
	w.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(w.inflight[name])+1)
	for _, c := range w.inflight[name] {
		cancels = append(cancels, c)
	}
	if c, ok := w.watchers[name]; ok {
		cancels = append(cancels, c)
		delete(w.watchers, name)
	}
	delete(w.active, name)
	delete(w.intended, name)
	delete(w.replaced, name)
	delete(w.lastHealth, name)
	delete(w.lastHealthAt, name)
	delete(w.seeded, name)
	delete(w.failover, name)
	for src, list := range w.failover {
		kept := list[:0:0]
		for _, t := range list {
			if t != name {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(w.failover, src)
		} else {
			w.failover[src] = kept
		}
	}
	w.mu.Unlock()
	for _, c := range cancels {
		c()
	}
}

// watch polls name until ctx ends. The adapter is looked up on every
// poll, so an adapter replaced without a restart (ADR-0053) is followed;
// a profile no longer in the set ends the watcher.
func (w *Watchdog) watch(ctx context.Context, name string) {
	w.mu.Lock()
	poll := w.poll
	delays := w.retryDelays
	network := w.network
	waitForUser := w.waitForUser
	w.mu.Unlock()
	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	attempt := 0 // attempts made in this round
	manualNotified := false
	failoverTried := false
	// gaveUp: every attempt of this round failed; nothing more is tried
	// until the network differs from gaveUpNet (ADR-0098).
	gaveUp := false
	gaveUpNet := ""
	// gaveUpNotified: the user was told, once for the whole outage.
	gaveUpNotified := false
	// blockedNotified: told once that only the user can unblock it.
	blockedNotified := false
	seededTaken := false
	// noResponse: this outage began with a tunnel that was up but whose
	// server stopped answering (vpn.HealthNoResponse, ADR-0030).
	noResponse := false
	resetOutage := func() {
		attempt = 0
		failoverTried = false
		gaveUp = false
		gaveUpNotified = false
		blockedNotified = false
		seededTaken = false
	}
	// woke: the computer slept since the last give-up (a re-arm signal).
	woke := false
	var lastTick time.Time
	giveUp := func(err error) {
		gaveUp = true
		woke = false
		if network != nil {
			gaveUpNet = network()
		}
		if gaveUpNotified || w.paused(name) {
			return
		}
		gaveUpNotified = true
		w.emit(WatchdogEvent{Profile: name, Event: "reconnect_gave_up", Attempt: attempt, Err: err})
		w.log.Warn("watchdog: giving up reconnecting until the network changes",
			zap.String("profile", name), zap.Int("attempts", attempt), zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			if !lastTick.IsZero() && WokeFromSleep(now.Sub(lastTick), now.Round(0).Sub(lastTick.Round(0))) {
				woke = true
			}
			lastTick = now
			adapter, ok := w.adapters.Get(name)
			if !ok {
				return // removed without a restart; Forget cleans up
			}
			status := adapter.Status().Normalize()
			health := tunnelHealth(adapter)
			if status == vpn.StatusConnected {
				if noResponse {
					if health == nil || health.State != vpn.HealthOK {
						// Up again but the server has not answered yet:
						// the outage (attempts, one failover) goes on.
						continue
					}
					noResponse = false
					w.emit(WatchdogEvent{Profile: name, Event: "responding"})
					w.log.Info("watchdog: tunnel responds again", zap.String("profile", name))
				}
				resetOutage() // a healthy tunnel ends the outage
				manualNotified = false
				w.runHealthCheck(ctx, name, adapter)
				continue
			}
			if status == vpn.StatusConnecting {
				// Someone is connecting it: wait. The round is not reset,
				// so a tunnel that keeps failing stays bounded.
				continue
			}

			if w.paused(name) {
				// User requested disconnect, or a declared backup took
				// over — do not reconnect.
				resetOutage()
				noResponse = false
				continue
			}

			// Once per outage: the tunnel is up but its server stopped
			// answering. It is then handled like any drop below.
			if !noResponse && health != nil && health.State == vpn.HealthNoResponse {
				noResponse = true
				w.emit(WatchdogEvent{Profile: name, Event: "no_response"})
				w.log.Warn("watchdog: tunnel up but the server does not answer",
					zap.String("profile", name), zap.Time("last_handshake", health.LastHandshake))
			}

			// Provider app with no automatic connect path (no CLI, or
			// auto_connect: false): only the user can bring it back, so
			// notify once per drop instead of reconnecting.
			if adapter.Capabilities().ManualReconnect {
				if !manualNotified {
					manualNotified = true
					w.emit(WatchdogEvent{Profile: name, Event: "manual_reconnect_required"})
					w.log.Warn("watchdog: tunnel dropped; reconnect required in the provider app",
						zap.String("profile", name), zap.String("status", string(status)))
				}
				continue
			}

			if gaveUp {
				// Retry only on a clear signal: the network changed
				// (other addresses, or back online), or the computer
				// woke from sleep.
				changed := network != nil && network() != gaveUpNet
				if !changed && !woke {
					continue
				}
				gaveUp, woke = false, false
				attempt = 0
				w.emit(WatchdogEvent{Profile: name, Event: "retry_rearmed"})
				w.log.Info("watchdog: network changed, trying again", zap.String("profile", name))
			}
			if !seededTaken {
				seededTaken = true
				attempt += w.takeSeeded(name)
			}

			delay, more := RetryDelay(delays, attempt)
			if !more {
				giveUp(nil)
				continue
			}
			attempt++
			w.emit(WatchdogEvent{Profile: name, Event: "reconnect_attempt", Attempt: attempt})
			w.log.Warn("watchdog: unexpected disconnect, reconnecting",
				zap.String("profile", name),
				zap.String("status", string(status)),
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay),
			)

			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}

			ran, err := w.connectFor(ctx, name, name)
			if !ran {
				// `down` during the wait or the attempt.
				resetOutage()
				continue
			}
			if err == nil {
				w.emit(WatchdogEvent{Profile: name, Event: "reconnected", Attempt: attempt, AfterGiveUp: gaveUpNotified || blockedNotified})
				w.log.Info("watchdog: reconnected", zap.String("profile", name))
				resetOutage()
				continue
			}
			if waitForUser != nil && waitForUser(err) {
				// Only the user can resolve it (another VPN carries all
				// traffic): not an attempt, no further retries now.
				attempt--
				gaveUp, woke = true, false
				if network != nil {
					gaveUpNet = network()
				}
				if !blockedNotified && !w.paused(name) {
					blockedNotified = true
					w.emit(WatchdogEvent{Profile: name, Event: "reconnect_blocked", Attempt: attempt, Err: err})
					w.log.Warn("watchdog: reconnect needs the user", zap.String("profile", name), zap.Error(err))
				}
				continue
			}
			w.emit(WatchdogEvent{Profile: name, Event: "reconnect_failed", Attempt: attempt, Err: err})
			w.log.Error("watchdog: reconnect failed",
				zap.String("profile", name),
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			if !failoverTried {
				failoverTried = true
				if w.tryFailover(ctx, name, attempt, err) {
					continue
				}
			}
			if _, more := RetryDelay(delays, attempt); !more {
				giveUp(err)
			}
		}
	}
}

func (w *Watchdog) runHealthCheck(ctx context.Context, name string, adapter vpn.VPNAdapter) {
	w.mu.Lock()
	interval := w.healthInterval
	timeout := w.healthTimeout
	check := w.healthCheck
	lastAt := w.lastHealthAt[name]
	w.mu.Unlock()

	if check == nil || interval <= 0 {
		return
	}
	if !lastAt.IsZero() && time.Since(lastAt) < interval {
		return
	}

	probeCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		probeCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	result := check(probeCtx, name, adapter)
	if result.CheckedAt.IsZero() {
		result.CheckedAt = time.Now()
	}

	w.mu.Lock()
	w.lastHealthAt[name] = result.CheckedAt
	w.lastHealth[name] = result
	w.mu.Unlock()

	if result.Healthy {
		return
	}

	err := fmt.Errorf("%s", result.Error)
	if result.Error == "" {
		err = fmt.Errorf("health probe failed")
	}
	w.log.Warn("watchdog: health degraded",
		zap.String("profile", name),
		zap.Bool("reachable", result.Reachable),
		zap.Bool("dns_ok", result.DNSOK),
		zap.Duration("latency", result.Latency),
		zap.String("error", result.Error),
	)
	// Reported only: bringing up a backup next to a tunnel that is still
	// up would fight it over routes and DNS.
	w.emit(WatchdogEvent{Profile: name, Event: "health_degraded", Err: err})
}

// tryFailover tries the backups the user declared for source, in order
// (called once per outage). A backup that is already connected counts as
// taken over; one that is connecting is skipped. It returns true when a
// backup took over: the watchdog then stops retrying source until the user
// connects it (MarkActive) or turns it off (MarkIntended).
func (w *Watchdog) tryFailover(ctx context.Context, source string, attempt int, reason error) bool {
	w.mu.Lock()
	targets := append([]string(nil), w.failover[source]...)
	w.mu.Unlock()

	for _, target := range targets {
		adapter, ok := w.adapters.Get(target)
		if !ok {
			continue
		}
		switch adapter.Status().Normalize() {
		case vpn.StatusConnected:
			w.takeOver(source, target, attempt)
			return true
		case vpn.StatusConnecting:
			continue
		}

		w.emit(WatchdogEvent{Profile: source, Event: "failover_started", Attempt: attempt, Err: reason, Target: target})
		w.log.Warn("watchdog: starting failover to a declared backup",
			zap.String("from", source),
			zap.String("to", target),
			zap.Error(reason),
		)
		ran, err := w.connectFor(ctx, source, target)
		if !ran {
			return false
		}
		if err != nil {
			w.emit(WatchdogEvent{Profile: source, Event: "failover_failed", Attempt: attempt, Err: err, Target: target})
			w.log.Error("watchdog: failover failed",
				zap.String("from", source),
				zap.String("to", target),
				zap.Error(err),
			)
			continue
		}
		w.MarkActive(target)
		w.takeOver(source, target, attempt)
		return true
	}
	return false
}

func (w *Watchdog) takeOver(source, target string, attempt int) {
	w.mu.Lock()
	w.replaced[source] = target
	w.mu.Unlock()
	w.emit(WatchdogEvent{Profile: source, Event: "failover_activated", Attempt: attempt, Target: target})
	w.log.Info("watchdog: failover activated",
		zap.String("from", source),
		zap.String("to", target),
	)
}

// tunnelHealth returns adapter's tunnel health, or nil when it does not
// report one.
func tunnelHealth(adapter vpn.VPNAdapter) *vpn.TunnelHealth {
	if r, ok := adapter.(vpn.HealthReporter); ok {
		return r.TunnelHealth()
	}
	return nil
}
