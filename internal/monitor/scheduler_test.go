package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type schedProbe struct {
	mu        sync.Mutex
	connected map[string]bool
}

func (p *schedProbe) connect(_ context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connected[name] = true
	return nil
}

func (p *schedProbe) disconnect(_ context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.connected, name)
	return nil
}

func (p *schedProbe) status(name string) vpn.Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.connected[name] {
		return vpn.StatusConnected
	}
	return vpn.StatusDisconnected
}

func (p *schedProbe) is(name string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connected[name]
}

func TestTimeWindowMatch_DayAndOvernight(t *testing.T) {
	now := time.Date(2026, 7, 14, 10, 30, 0, 0, time.UTC)
	if !timeWindowMatch("09:00", "18:00", now) {
		t.Fatal("expected daytime window to match")
	}
	if timeWindowMatch("18:00", "09:00", now) {
		t.Fatal("expected overnight window not to match at 10:30")
	}
	night := time.Date(2026, 7, 14, 23, 30, 0, 0, time.UTC)
	if !timeWindowMatch("22:00", "06:00", night) {
		t.Fatal("expected overnight window to match at 23:30")
	}
}

func TestRuleActiveNow_WeekdayAndWindow(t *testing.T) {
	r := config.ScheduleRule{
		Weekdays: []string{"mon", "tue", "wed", "thu", "fri"},
		Start:    "09:00",
		End:      "18:00",
	}
	ts := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC) // Tue
	if !ruleActiveNow(r, ts) {
		t.Fatal("expected business-hours rule active on Tuesday 10:00")
	}
	off := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC) // Sun
	if ruleActiveNow(r, off) {
		t.Fatal("expected rule inactive on Sunday")
	}
}

func schedCfg(enabled bool, rules ...config.ScheduleRule) *config.Config {
	c := &config.Config{}
	c.Monitor.Scheduler.Enabled = enabled
	c.Monitor.Scheduler.Rules = rules
	return c
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Rules, on/off and per-rule enabled apply at runtime, without a restart.
func TestScheduler_ReconfigureHotReload(t *testing.T) {
	p := &schedProbe{connected: map[string]bool{}}
	always := config.ScheduleRule{Name: "always", Profiles: []string{"office"}}
	s := NewScheduler(schedCfg(false, always), 10*time.Millisecond, p.connect, p.disconnect, p.status, zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()
	if s.Running() {
		t.Fatal("disabled scheduler is running")
	}

	// Turned on from the dashboard: connects at once.
	s.Reconfigure(schedCfg(true, always), 10*time.Millisecond)
	waitFor(t, "office connected", func() bool { return p.is("office") })

	// The rule moves to another profile: office is no longer wanted.
	home := config.ScheduleRule{Name: "always", Profiles: []string{"home", "office"}}
	off := false
	home.Enabled = &off
	other := config.ScheduleRule{Name: "home", Profiles: []string{"home"}}
	s.Reconfigure(schedCfg(true, home, other), 10*time.Millisecond)
	waitFor(t, "home connected", func() bool { return p.is("home") })
	// The paused rule no longer manages office, so office is left alone.
	if !p.is("office") {
		t.Fatal("a paused rule must not disconnect its profiles")
	}

	// Scheduler off: the loop stops.
	s.Reconfigure(schedCfg(false, other), 10*time.Millisecond)
	if s.Running() {
		t.Fatal("still running after scheduler.enabled=false")
	}
}
