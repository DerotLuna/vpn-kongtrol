package monitor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type ScheduleConnectFunc func(ctx context.Context, profile string) error
type ScheduleDisconnectFunc func(ctx context.Context, profile string) error
type ScheduleStatusFunc func(profile string) vpn.Status

// Scheduler enforces profile schedules (time windows/weekdays). Its rules,
// interval and on/off follow the config at runtime (Reconfigure): a
// dashboard or CLI edit applies without restarting the daemon.
type Scheduler struct {
	connect    ScheduleConnectFunc
	disconnect ScheduleDisconnectFunc
	status     ScheduleStatusFunc
	log        *zap.Logger

	mu       sync.Mutex
	rules    []config.ScheduleRule // enabled rules only
	interval time.Duration
	enabled  bool
	base     context.Context // from Start; nil until started
	cancel   context.CancelFunc
}

func NewScheduler(
	cfg *config.Config,
	interval time.Duration,
	connect ScheduleConnectFunc,
	disconnect ScheduleDisconnectFunc,
	status ScheduleStatusFunc,
	log *zap.Logger,
) *Scheduler {
	s := &Scheduler{connect: connect, disconnect: disconnect, status: status, log: log}
	s.set(cfg, interval)
	return s
}

// set copies what the scheduler needs from cfg. Called with s.mu held or
// before the scheduler is shared.
func (s *Scheduler) set(cfg *config.Config, interval time.Duration) {
	s.rules, s.enabled, s.interval = nil, false, interval
	if cfg == nil {
		return
	}
	s.enabled = cfg.Monitor.Scheduler.Enabled
	for _, r := range cfg.Monitor.Scheduler.Rules {
		if r.IsEnabled() {
			r.Profiles = append([]string(nil), r.Profiles...)
			r.Weekdays = append([]string(nil), r.Weekdays...)
			s.rules = append(s.rules, r)
		}
	}
}

// Start runs the scheduler under ctx while it is enabled and has rules.
// Safe to call when disabled: a later Reconfigure can turn it on.
func (s *Scheduler) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.base = ctx
	s.restartLocked()
}

// Reconfigure applies new rules, interval and on/off now: the running loop
// stops and, when still enabled with rules, starts again with them (it
// applies them at once, then every interval).
func (s *Scheduler) Reconfigure(cfg *config.Config, interval time.Duration) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.set(cfg, interval)
	if s.base != nil {
		s.restartLocked()
	}
}

// Running reports whether the loop is active (tests, status).
func (s *Scheduler) Running() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancel != nil
}

func (s *Scheduler) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
	s.base = nil
}

// stopLocked cancels the running loop without waiting for it: the loop
// takes mu to read the rules, so waiting here could deadlock. A cancelled
// loop does no further connect/disconnect (apply checks ctx).
func (s *Scheduler) stopLocked() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

func (s *Scheduler) restartLocked() {
	s.stopLocked()
	if !s.enabled || len(s.rules) == 0 || s.interval <= 0 || s.base == nil || s.base.Err() != nil {
		return
	}
	ctx, cancel := context.WithCancel(s.base)
	s.cancel = cancel
	go s.loop(ctx, s.interval)
}

func (s *Scheduler) loop(ctx context.Context, interval time.Duration) {
	s.apply(ctx, time.Now())
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.apply(ctx, now)
		}
	}
}

func (s *Scheduler) snapshot() []config.ScheduleRule {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rules
}

func (s *Scheduler) apply(ctx context.Context, now time.Time) {
	desired := map[string]bool{}
	managed := map[string]bool{}
	for _, r := range s.snapshot() {
		for _, p := range r.Profiles {
			if strings.TrimSpace(p) != "" {
				managed[p] = true
			}
		}
		if !ruleActiveNow(r, now) {
			continue
		}
		for _, p := range r.Profiles {
			p = strings.TrimSpace(p)
			if p != "" {
				desired[p] = true
			}
		}
	}

	for profile := range desired {
		if ctx.Err() != nil {
			return // reconfigured or stopping
		}
		st := s.status(profile).Normalize()
		if st == vpn.StatusConnected || st == vpn.StatusConnecting {
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		err := s.connect(cctx, profile)
		cancel()
		if err != nil {
			s.log.Warn("scheduler: connect failed", zap.String("profile", profile), zap.Error(err))
		}
	}

	for profile := range managed {
		if desired[profile] {
			continue
		}
		if ctx.Err() != nil {
			return // reconfigured or stopping
		}
		st := s.status(profile).Normalize()
		if st != vpn.StatusConnected && st != vpn.StatusConnecting {
			continue
		}
		dctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		err := s.disconnect(dctx, profile)
		cancel()
		if err != nil {
			s.log.Warn("scheduler: disconnect failed", zap.String("profile", profile), zap.Error(err))
		}
	}
}

func ruleActiveNow(r config.ScheduleRule, now time.Time) bool {
	if !weekdayMatch(r.Weekdays, now.Weekday()) {
		return false
	}
	return timeWindowMatch(r.Start, r.End, now)
}

func weekdayMatch(days []string, wd time.Weekday) bool {
	if len(days) == 0 {
		return true
	}
	want := weekdayToken(wd)
	for _, d := range days {
		if strings.EqualFold(strings.TrimSpace(d), want) {
			return true
		}
	}
	return false
}

func weekdayToken(wd time.Weekday) string {
	switch wd {
	case time.Monday:
		return "mon"
	case time.Tuesday:
		return "tue"
	case time.Wednesday:
		return "wed"
	case time.Thursday:
		return "thu"
	case time.Friday:
		return "fri"
	case time.Saturday:
		return "sat"
	default:
		return "sun"
	}
}

func timeWindowMatch(start, end string, now time.Time) bool {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if start == "" || end == "" {
		return true
	}
	sm, err := parseHHMM(start)
	if err != nil {
		return false
	}
	em, err := parseHHMM(end)
	if err != nil {
		return false
	}
	nm := now.Hour()*60 + now.Minute()
	if sm <= em {
		return nm >= sm && nm < em
	}
	// Overnight window: 22:00-06:00
	return nm >= sm || nm < em
}

func parseHHMM(v string) (int, error) {
	t, err := time.Parse("15:04", v)
	if err != nil {
		return 0, fmt.Errorf("invalid time %q", v)
	}
	return t.Hour()*60 + t.Minute(), nil
}
