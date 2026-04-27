package monitor

import (
	"context"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

const (
	watchdogBaseDelay = 2 * time.Second
	watchdogMaxDelay  = 5 * time.Minute
	watchdogMaxJitter = 500 * time.Millisecond
)

// ConnectFunc is a callback that reconnects a named profile.
type ConnectFunc func(ctx context.Context, name string) error

// Watchdog monitors VPN adapters and reconnects on unexpected disconnects.
// It uses exponential backoff between retries and skips profiles that were
// intentionally disconnected via MarkIntended.
type Watchdog struct {
	mu       sync.Mutex
	adapters map[string]vpn.VPNAdapter
	intended map[string]bool // true = user requested disconnect; skip reconnect
	connect  ConnectFunc
	log      *zap.Logger

	cancel context.CancelFunc
}

// NewWatchdog creates a watchdog.
// adapters is a snapshot-safe map (the caller must not mutate it after passing).
func NewWatchdog(adapters map[string]vpn.VPNAdapter, connect ConnectFunc, log *zap.Logger) *Watchdog {
	return &Watchdog{
		adapters: adapters,
		intended: make(map[string]bool),
		connect:  connect,
		log:      log,
	}
}

// MarkIntended signals that a disconnect for profile name was user-requested.
// Call this BEFORE calling Disconnect so the watchdog does not immediately
// attempt to reconnect.
func (w *Watchdog) MarkIntended(name string) {
	w.mu.Lock()
	w.intended[name] = true
	w.mu.Unlock()
}

// MarkActive clears the intended-disconnect flag, re-enabling reconnect.
// Call this AFTER a successful Connect so future drops are treated as faults.
func (w *Watchdog) MarkActive(name string) {
	w.mu.Lock()
	delete(w.intended, name)
	w.mu.Unlock()
}

// Start begins the watchdog loop. It returns immediately; call Stop to end it.
func (w *Watchdog) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	for name, adapter := range w.adapters {
		go w.watch(ctx, name, adapter)
	}
}

// Stop shuts down all watchdog goroutines.
func (w *Watchdog) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *Watchdog) watch(ctx context.Context, name string, adapter vpn.VPNAdapter) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	attempt := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			status := adapter.Status()
			if status == vpn.StatusConnected || status == vpn.StatusConnecting {
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
			w.log.Warn("watchdog: unexpected disconnect, reconnecting",
				zap.String("profile", name),
				zap.String("status", string(status)),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", delay),
			)

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}

			if err := w.connect(ctx, name); err != nil {
				w.log.Error("watchdog: reconnect failed",
					zap.String("profile", name),
					zap.Int("attempt", attempt),
					zap.Error(err),
				)
			} else {
				w.log.Info("watchdog: reconnected", zap.String("profile", name))
				attempt = 0
			}
		}
	}
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
