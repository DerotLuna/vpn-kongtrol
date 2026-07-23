package monitor

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

const (
	watchdogBaseDelay = 2 * time.Second
	watchdogMaxDelay  = 5 * time.Minute
	watchdogPoll      = 5 * time.Second
	failoverCooldown  = 30 * time.Second
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

// Watchdog monitors VPN adapters and reconnects on unexpected disconnects.
// It uses exponential backoff between retries and only watches profiles
// that have been explicitly activated via MarkActive.
type Watchdog struct {
	mu       sync.Mutex
	adapters map[string]vpn.VPNAdapter
	intended map[string]bool // true = user requested disconnect; skip reconnect
	active   map[string]bool // true = profile was connected; eligible for watchdog
	connect  ConnectFunc
	log      *zap.Logger
	onEvent  func(profile, event string, attempt int, err error)
	priority map[string]int

	healthCheck    HealthCheckFunc
	healthInterval time.Duration
	healthTimeout  time.Duration
	lastHealthAt   map[string]time.Time
	lastHealth     map[string]HealthResult

	failoverOrder  map[string][]string
	lastFailoverAt map[string]time.Time

	connectTimeout time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// SetEventCallback registers an optional callback for watchdog lifecycle events.
// event values:
// reconnect_attempt, reconnect_failed, reconnected,
// health_degraded, failover_started, failover_failed, failover_activated.
func (w *Watchdog) SetEventCallback(fn func(profile, event string, attempt int, err error)) {
	w.mu.Lock()
	w.onEvent = fn
	w.mu.Unlock()
}

// NewWatchdog creates a watchdog.
// adapters is a snapshot-safe map (the caller must not mutate it after passing).
func NewWatchdog(adapters map[string]vpn.VPNAdapter, connect ConnectFunc, log *zap.Logger) *Watchdog {
	return &Watchdog{
		adapters:       adapters,
		intended:       make(map[string]bool),
		active:         make(map[string]bool),
		connect:        connect,
		log:            log,
		priority:       make(map[string]int),
		lastHealthAt:   make(map[string]time.Time),
		lastHealth:     make(map[string]HealthResult),
		failoverOrder:  make(map[string][]string),
		lastFailoverAt: make(map[string]time.Time),
		connectTimeout: watchdogConnectTimeout,
	}
}

// SetConnectTimeout overrides the default per-attempt timeout bounding
// reconnect/failover connect calls. Mainly useful for tests; production
// callers can leave the default (watchdogConnectTimeout) in place.
func (w *Watchdog) SetConnectTimeout(d time.Duration) {
	w.mu.Lock()
	w.connectTimeout = d
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

// ConfigureFailover sets profile priorities and computes fallback order.
// Lower numeric priority means more preferred.
// Failover is enabled only for profiles with priority > 0.
func (w *Watchdog) ConfigureFailover(priority map[string]int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.priority = make(map[string]int, len(priority))
	for name, p := range priority {
		w.priority[name] = p
	}

	w.failoverOrder = make(map[string][]string, len(w.adapters))
	for source := range w.adapters {
		if w.priority[source] <= 0 {
			continue
		}
		candidates := make([]string, 0, len(w.adapters)-1)
		for target := range w.adapters {
			if source == target {
				continue
			}
			if w.priority[target] <= 0 {
				continue
			}
			candidates = append(candidates, target)
		}
		sort.Slice(candidates, func(i, j int) bool {
			pi := w.priority[candidates[i]]
			pj := w.priority[candidates[j]]
			if pi != pj {
				return pi < pj
			}
			return candidates[i] < candidates[j]
		})
		w.failoverOrder[source] = candidates
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
// attempt to reconnect.
func (w *Watchdog) MarkIntended(name string) {
	w.mu.Lock()
	w.intended[name] = true
	w.mu.Unlock()
}

// MarkActive clears the intended-disconnect flag and starts watching this
// profile for unexpected disconnects. Call AFTER a successful Connect.
func (w *Watchdog) MarkActive(name string) {
	w.mu.Lock()
	delete(w.intended, name)
	alreadyActive := w.active[name]
	w.active[name] = true
	ctx := w.ctx
	adapter, hasAdapter := w.adapters[name]
	if !alreadyActive && ctx != nil && hasAdapter {
		w.wg.Go(func() { w.watch(ctx, name, adapter) })
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
		if adapter, ok := w.adapters[name]; ok {
			name, adapter := name, adapter
			w.wg.Go(func() { w.watch(ctx, name, adapter) })
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
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	w.wg.Wait()
}

func (w *Watchdog) watch(ctx context.Context, name string, adapter vpn.VPNAdapter) {
	ticker := time.NewTicker(watchdogPoll)
	defer ticker.Stop()

	attempt := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			status := adapter.Status().Normalize()
			if status == vpn.StatusConnected {
				attempt = 0 // reset backoff on healthy state
				w.runHealthCheck(ctx, name, adapter)
				continue
			}
			if status == vpn.StatusConnecting {
				attempt = 0 // reset backoff on healthy state
				continue
			}

			w.mu.Lock()
			skip := w.intended[name]
			w.mu.Unlock()

			if skip {
				// User requested disconnect — do not reconnect.
				attempt = 0
				continue
			}

			delay := backoff(attempt)
			attempt++
			w.mu.Lock()
			cb := w.onEvent
			w.mu.Unlock()
			if cb != nil {
				cb(name, "reconnect_attempt", attempt, nil)
			}
			w.log.Warn("watchdog: unexpected disconnect, reconnecting",
				zap.String("profile", name),
				zap.String("status", string(status)),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", delay),
			)

			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			case <-timer.C:
			}

			w.mu.Lock()
			timeout := w.connectTimeout
			w.mu.Unlock()
			connectCtx, cancelConnect := context.WithTimeout(ctx, timeout)
			err := w.connect(connectCtx, name)
			cancelConnect()
			if err != nil {
				if cb != nil {
					cb(name, "reconnect_failed", attempt, err)
				}
				w.log.Error("watchdog: reconnect failed",
					zap.String("profile", name),
					zap.Int("attempt", attempt),
					zap.Error(err),
				)
				w.tryFailover(ctx, name, attempt, err)
			} else {
				if cb != nil {
					cb(name, "reconnected", attempt, nil)
				}
				w.log.Info("watchdog: reconnected", zap.String("profile", name))
				attempt = 0
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
	cb := w.onEvent
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
	if cb != nil {
		cb(name, "health_degraded", 0, err)
	}
	w.tryFailover(ctx, name, 0, err)
}

func (w *Watchdog) tryFailover(ctx context.Context, source string, attempt int, reason error) {
	w.mu.Lock()
	candidates := append([]string(nil), w.failoverOrder[source]...)
	lastAt := w.lastFailoverAt[source]
	cb := w.onEvent
	w.mu.Unlock()

	if len(candidates) == 0 {
		return
	}
	if !lastAt.IsZero() && time.Since(lastAt) < failoverCooldown {
		return
	}

	var target string
	for _, c := range candidates {
		adapter, ok := w.adapters[c]
		if !ok {
			continue
		}
		st := adapter.Status().Normalize()
		if st == vpn.StatusConnected || st == vpn.StatusConnecting {
			continue
		}
		target = c
		break
	}
	if target == "" {
		return
	}

	w.mu.Lock()
	w.lastFailoverAt[source] = time.Now()
	w.mu.Unlock()

	if cb != nil {
		cb(target, "failover_started", attempt, reason)
	}
	w.log.Warn("watchdog: starting failover",
		zap.String("from", source),
		zap.String("to", target),
		zap.Error(reason),
	)

	w.mu.Lock()
	timeout := w.connectTimeout
	w.mu.Unlock()
	connectCtx, cancelConnect := context.WithTimeout(ctx, timeout)
	err := w.connect(connectCtx, target)
	cancelConnect()
	if err != nil {
		if cb != nil {
			cb(target, "failover_failed", attempt, err)
		}
		w.log.Error("watchdog: failover failed",
			zap.String("from", source),
			zap.String("to", target),
			zap.Error(err),
		)
		return
	}

	w.MarkActive(target)
	if cb != nil {
		cb(target, "failover_activated", attempt, nil)
	}
	w.log.Info("watchdog: failover activated",
		zap.String("from", source),
		zap.String("to", target),
	)
}

// backoff returns exponential delay with a cap, no jitter needed for CLI.
func backoff(attempt int) time.Duration {
	exp := math.Pow(2, float64(attempt))
	d := time.Duration(exp) * watchdogBaseDelay
	if d > watchdogMaxDelay {
		d = watchdogMaxDelay
	}
	return d
}
