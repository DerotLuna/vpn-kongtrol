package monitor_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// eventLog records watchdog events for assertions.
type eventLog struct {
	mu     sync.Mutex
	events []monitor.WatchdogEvent
}

func (l *eventLog) add(e monitor.WatchdogEvent) {
	l.mu.Lock()
	l.events = append(l.events, e)
	l.mu.Unlock()
}

func (l *eventLog) count(event string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, e := range l.events {
		if e.Event == event {
			n++
		}
	}
	return n
}

func (l *eventLog) first(event string) (monitor.WatchdogEvent, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range l.events {
		if e.Event == event {
			return e, true
		}
	}
	return monitor.WatchdogEvent{}, false
}

func waitFor(t *testing.T, d time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}

// Profiles reach different networks, so similar names, the same adapter
// family or the same priority never link them: with no `failover:`
// declared, a profile that cannot reconnect is retried on its own, the
// user is told once, and no other profile is ever connected — including
// one the user never brought up. (Real case: "ekiipago-ovpn" failed over
// to the unrelated FortiClient profile "ekiipago".)
func TestWatchdog_NoImplicitFailover(t *testing.T) {
	ovpn := newMockAdapter(vpn.StatusConnected)
	forti := newMockAdapter(vpn.StatusDisconnected)
	other := newMockAdapter(vpn.StatusDisconnected)
	adapters := map[string]vpn.VPNAdapter{
		"ekiipago-ovpn": ovpn,  // priority 1 in the user's config
		"ekiipago":      forti, // priority 1 too
		"us-content":    other,
	}
	var mu sync.Mutex
	connects := map[string]int{}
	connect := func(_ context.Context, name string) error {
		mu.Lock()
		connects[name]++
		mu.Unlock()
		return errors.New("openvpn: server unreachable")
	}

	wd := monitor.NewWatchdog(vpn.NewSet(adapters), connect, zapNop())
	wd.SetPollInterval(50 * time.Millisecond)
	wd.SetRetryDelays([]time.Duration{10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond})
	wd.SetNetworkFingerprint(func() string { return "home" })
	// Declared failover elsewhere must not leak to these profiles.
	wd.ConfigureFailover(map[string][]string{"us-content": {"ekiipago"}})
	var log eventLog
	wd.SetEventCallback(log.add)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("ekiipago-ovpn")
	wd.Start(ctx)
	ovpn.setStatus(vpn.StatusError)

	if !waitFor(t, 10*time.Second, func() bool { return log.count("reconnect_gave_up") == 1 }) {
		t.Fatal("the dropped profile was not retried until the watchdog gave up")
	}
	time.Sleep(200 * time.Millisecond) // a further attempt would show
	mu.Lock()
	defer mu.Unlock()
	for _, name := range []string{"ekiipago", "us-content"} {
		if connects[name] != 0 {
			t.Errorf("%s connected %d times without being declared as failover", name, connects[name])
		}
	}
	if n := log.count("failover_started"); n != 0 {
		t.Errorf("failover_started = %d, want 0", n)
	}
	if connects["ekiipago-ovpn"] != 3 {
		t.Errorf("ekiipago-ovpn attempts = %d, want 3 (then give up)", connects["ekiipago-ovpn"])
	}
	if n := log.count("reconnect_gave_up"); n != 1 {
		t.Errorf("reconnect_gave_up = %d, want exactly one notice per outage", n)
	}
}

// With `failover: [office-b]` declared on office-a, a failed reconnect
// connects office-b and the watchdog stops retrying office-a.
func TestWatchdog_FailoverToDeclaredBackup(t *testing.T) {
	primary := newMockAdapter(vpn.StatusConnected)
	backup := newMockAdapter(vpn.StatusDisconnected)
	unrelated := newMockAdapter(vpn.StatusDisconnected)
	adapters := map[string]vpn.VPNAdapter{"office-a": primary, "office-b": backup, "office-c": unrelated}

	var primaryTries, backupConnects, unrelatedConnects atomic.Int32
	connect := func(_ context.Context, name string) error {
		switch name {
		case "office-a":
			primaryTries.Add(1)
			return context.DeadlineExceeded
		case "office-b":
			backupConnects.Add(1)
			backup.setStatus(vpn.StatusConnected)
			return nil
		default:
			unrelatedConnects.Add(1)
			return nil
		}
	}

	wd := monitor.NewWatchdog(vpn.NewSet(adapters), connect, zapNop())
	wd.SetPollInterval(50 * time.Millisecond)
	wd.ConfigureFailover(map[string][]string{"office-a": {"office-b"}})
	var log eventLog
	wd.SetEventCallback(log.add)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("office-a")
	wd.Start(ctx)
	primary.setStatus(vpn.StatusDisconnected)

	if !waitFor(t, 10*time.Second, func() bool { return backupConnects.Load() >= 1 }) {
		t.Fatal("declared backup was not connected")
	}
	e, ok := log.first("failover_activated")
	if !ok || e.Profile != "office-a" || e.Target != "office-b" {
		t.Fatalf("failover_activated event = %+v, %v", e, ok)
	}
	tries := primaryTries.Load()
	time.Sleep(5 * time.Second) // longer than the next backoff (4s): a retry would show
	if got := primaryTries.Load(); got != tries {
		t.Errorf("office-a retried %d more times after its backup took over", got-tries)
	}
	if unrelatedConnects.Load() != 0 {
		t.Error("a profile not listed in failover was connected")
	}
	if n := log.count("reconnect_gave_up"); n != 0 {
		t.Errorf("reconnect_gave_up = %d, want 0 when a backup took over", n)
	}
}

// `down` (MarkIntended) cancels a reconnect the watchdog has in flight and
// returns only after cancelling it; the cancelled attempt is not reported
// as a failure.
func TestWatchdog_MarkIntendedCancelsInflightReconnect(t *testing.T) {
	adapter := newMockAdapter(vpn.StatusConnected)
	started := make(chan struct{}, 1)
	returned := make(chan error, 1)
	connect := func(ctx context.Context, _ string) error {
		started <- struct{}{}
		<-ctx.Done() // e.g. waiting for the user in the provider app
		returned <- ctx.Err()
		return ctx.Err()
	}
	wd := monitor.NewWatchdog(vpn.NewSet(map[string]vpn.VPNAdapter{"forti": adapter}), connect, zapNop())
	wd.SetPollInterval(50 * time.Millisecond)
	var log eventLog
	wd.SetEventCallback(log.add)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("forti")
	wd.Start(ctx)
	adapter.setStatus(vpn.StatusDisconnected)

	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("reconnect never started")
	}
	wd.MarkIntended("forti")
	select {
	case err := <-returned:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("connect ended with %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("MarkIntended did not cancel the in-flight reconnect")
	}
	time.Sleep(200 * time.Millisecond)
	if n := log.count("reconnect_failed") + log.count("reconnect_gave_up"); n != 0 {
		t.Errorf("a cancelled reconnect was reported as a failure (%d events)", n)
	}
}
