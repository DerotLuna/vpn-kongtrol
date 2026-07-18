package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type ScheduleConnectFunc func(ctx context.Context, profile string) error
type ScheduleDisconnectFunc func(ctx context.Context, profile string) error
type ScheduleStatusFunc func(profile string) vpn.Status

// Scheduler enforces profile schedules (time windows/weekdays).
type Scheduler struct {
	cfg        *config.Config
	interval   time.Duration
	connect    ScheduleConnectFunc
	disconnect ScheduleDisconnectFunc
	status     ScheduleStatusFunc
	log        *zap.Logger
	cancel     context.CancelFunc
}

func NewScheduler(
	cfg *config.Config,
	interval time.Duration,
	connect ScheduleConnectFunc,
	disconnect ScheduleDisconnectFunc,
	status ScheduleStatusFunc,
	log *zap.Logger,
) *Scheduler {
	return &Scheduler{
		cfg:        cfg,
		interval:   interval,
		connect:    connect,
		disconnect: disconnect,
		status:     status,
		log:        log,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	if s == nil || s.cfg == nil || len(s.cfg.Monitor.Scheduler.Rules) == 0 || s.interval <= 0 {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	go s.loop(ctx)
}

func (s *Scheduler) Stop() {
	if s != nil && s.cancel != nil {
		s.cancel()
	}
}

func (s *Scheduler) loop(ctx context.Context) {
	s.apply(ctx, time.Now())
	t := time.NewTicker(s.interval)
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

func (s *Scheduler) apply(ctx context.Context, now time.Time) {
	desired := map[string]bool{}
	managed := map[string]bool{}
	for _, r := range s.cfg.Monitor.Scheduler.Rules {
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
