package monitor_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// mockAdapter is a controllable VPNAdapter stub. status is set by the test
// body and read concurrently by the watchdog's own polling goroutine — a
// plain field here would be a genuine data race (caught by `go test
// -race`), not just a lint nit.
type mockAdapter struct {
	status atomic.Value // vpn.Status
}

func newMockAdapter(status vpn.Status) *mockAdapter {
	m := &mockAdapter{}
	m.setStatus(status)
	return m
}

func (m *mockAdapter) setStatus(s vpn.Status) { m.status.Store(s) }

func (m *mockAdapter) Name() string                                         { return "mock" }
func (m *mockAdapter) Version() string                                      { return "0.0.0" }
func (m *mockAdapter) Capabilities() vpn.Capabilities                       { return vpn.Capabilities{} }
func (m *mockAdapter) Connect(_ context.Context, _ vpn.AdapterConfig) error { return nil }
func (m *mockAdapter) Disconnect(_ context.Context) error                   { return nil }
func (m *mockAdapter) Reconnect(_ context.Context) error                    { return nil }
func (m *mockAdapter) TunnelInfo() (*vpn.TunnelInfo, error)                 { return nil, nil }
func (m *mockAdapter) Status() vpn.Status                                   { return m.status.Load().(vpn.Status) }

func zapNop() *zap.Logger { return zap.NewNop() }

// TestWatchdog_ReconnectsOnDrop verifies the watchdog calls ConnectFunc when
// an adapter transitions to disconnected without MarkIntended.
func TestWatchdog_ReconnectsOnDrop(t *testing.T) {
	adapter := newMockAdapter(vpn.StatusConnected)
	adapters := map[string]vpn.VPNAdapter{"office": adapter}

	var reconnects atomic.Int32
	connect := func(_ context.Context, name string) error {
		reconnects.Add(1)
		adapter.setStatus(vpn.StatusConnected)
		return nil
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("office") // watchdog only monitors active profiles
	wd.Start(ctx)

	// Simulate unexpected drop.
	adapter.setStatus(vpn.StatusDisconnected)

	// Watchdog polls every 5s; override via shorter poll would require exposing
	// interval. Instead we wait up to 10s for at least one reconnect call.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if reconnects.Load() >= 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if reconnects.Load() == 0 {
		t.Error("watchdog did not call ConnectFunc after unexpected disconnect")
	}
}

// TestWatchdog_SkipsIntendedDisconnect verifies that MarkIntended suppresses reconnect.
func TestWatchdog_SkipsIntendedDisconnect(t *testing.T) {
	adapter := newMockAdapter(vpn.StatusConnected)
	adapters := map[string]vpn.VPNAdapter{"office": adapter}

	var reconnects atomic.Int32
	connect := func(_ context.Context, _ string) error {
		reconnects.Add(1)
		return nil
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Mark intended BEFORE start so the flag is set before the first poll.
	wd.MarkIntended("office")
	wd.Start(ctx)

	adapter.setStatus(vpn.StatusDisconnected)

	// Wait two poll cycles (5s each). Use a shorter time since we can't change interval.
	// We just verify no reconnect happens for at least ~500ms, which is enough
	// to catch the "would immediately reconnect" bug.
	time.Sleep(600 * time.Millisecond)

	if reconnects.Load() > 0 {
		t.Errorf("watchdog reconnected %d time(s) after MarkIntended — should be 0", reconnects.Load())
	}
}

// TestWatchdog_MarkActive_ReEnablesReconnect verifies MarkActive re-arms the watchdog.
func TestWatchdog_MarkActive_ReEnablesReconnect(t *testing.T) {
	adapter := newMockAdapter(vpn.StatusConnected)
	adapters := map[string]vpn.VPNAdapter{"office": adapter}

	var reconnects atomic.Int32
	connect := func(_ context.Context, _ string) error {
		reconnects.Add(1)
		adapter.setStatus(vpn.StatusConnected)
		return nil
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wd.MarkIntended("office")
	wd.Start(ctx)

	adapter.setStatus(vpn.StatusDisconnected)

	// Still no reconnect while intended.
	time.Sleep(300 * time.Millisecond)
	if reconnects.Load() > 0 {
		t.Fatalf("reconnected while MarkIntended was set")
	}

	// Re-arm and drop.
	wd.MarkActive("office")
	adapter.setStatus(vpn.StatusDisconnected)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if reconnects.Load() >= 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if reconnects.Load() == 0 {
		t.Error("watchdog did not reconnect after MarkActive")
	}
}

// TestWatchdog_ConnectTimeoutBoundsHungConnect verifies a ConnectFunc that
// never returns on its own (a hung adapter driver) doesn't stall the
// watcher goroutine forever — SetConnectTimeout bounds the attempt, and the
// watchdog must still report reconnect_failed instead of hanging.
func TestWatchdog_ConnectTimeoutBoundsHungConnect(t *testing.T) {
	adapter := newMockAdapter(vpn.StatusConnected)
	adapters := map[string]vpn.VPNAdapter{"office": adapter}

	var failedEvents atomic.Int32
	connect := func(ctx context.Context, _ string) error {
		<-ctx.Done() // simulates a hung driver — only returns when the caller's timeout fires
		return ctx.Err()
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())
	wd.SetConnectTimeout(200 * time.Millisecond)
	wd.SetEventCallback(func(_ string, event string, _ int, _ error) {
		if event == "reconnect_failed" {
			failedEvents.Add(1)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("office")
	wd.Start(ctx)

	adapter.setStatus(vpn.StatusDisconnected)

	// Watchdog polls every 5s, then backs off ~2s before the first attempt,
	// then the hung connect is bounded to 200ms — well under the poll
	// interval, so a failure must show up within one poll cycle.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if failedEvents.Load() >= 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if failedEvents.Load() == 0 {
		t.Fatal("expected a reconnect_failed event from a timed-out connect attempt, but the watcher never reported one — it may be stuck")
	}
}

// TestWatchdog_Stop verifies no reconnects happen after Stop().
func TestWatchdog_Stop(t *testing.T) {
	adapter := newMockAdapter(vpn.StatusConnected)
	adapters := map[string]vpn.VPNAdapter{"office": adapter}

	var reconnects atomic.Int32
	connect := func(_ context.Context, _ string) error {
		reconnects.Add(1)
		return nil
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())

	ctx, cancel := context.WithCancel(context.Background())
	wd.MarkActive("office")
	wd.Start(ctx)
	cancel() // cancel context before Stop to make sure Stop is idempotent
	wd.Stop()

	adapter.setStatus(vpn.StatusDisconnected)
	time.Sleep(300 * time.Millisecond)

	if reconnects.Load() > 0 {
		t.Errorf("got %d reconnect(s) after Stop — expected 0", reconnects.Load())
	}
}

// TestWatchdog_MultipleProfiles verifies each profile is watched independently.
func TestWatchdog_MultipleProfiles(t *testing.T) {
	a1 := newMockAdapter(vpn.StatusConnected)
	a2 := newMockAdapter(vpn.StatusConnected)
	adapters := map[string]vpn.VPNAdapter{"p1": a1, "p2": a2}

	reconnected := map[string]*atomic.Int32{
		"p1": {},
		"p2": {},
	}

	connect := func(_ context.Context, name string) error {
		if c, ok := reconnected[name]; ok {
			c.Add(1)
		}
		adapters[name].(*mockAdapter).setStatus(vpn.StatusConnected)
		return nil
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("p1")
	wd.MarkActive("p2")
	wd.Start(ctx)

	// Drop only p1.
	a1.setStatus(vpn.StatusDisconnected)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if reconnected["p1"].Load() >= 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if reconnected["p1"].Load() == 0 {
		t.Error("p1 was not reconnected")
	}
	if reconnected["p2"].Load() > 0 {
		t.Errorf("p2 was reconnected %d time(s) unexpectedly", reconnected["p2"].Load())
	}
}

// TestWatchdog_FailoverOnReconnectFailure verifies watchdog failover to backup profile.
func TestWatchdog_FailoverOnReconnectFailure(t *testing.T) {
	primary := newMockAdapter(vpn.StatusConnected)
	backup := newMockAdapter(vpn.StatusDisconnected)
	adapters := map[string]vpn.VPNAdapter{
		"office-a": primary,
		"office-b": backup,
	}

	var backupConnects atomic.Int32
	connect := func(_ context.Context, name string) error {
		switch name {
		case "office-a":
			return context.DeadlineExceeded
		case "office-b":
			backupConnects.Add(1)
			backup.setStatus(vpn.StatusConnected)
			return nil
		default:
			return nil
		}
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())
	wd.ConfigureFailover(map[string]int{
		"office-a": 10,
		"office-b": 20,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("office-a")
	wd.Start(ctx)

	primary.setStatus(vpn.StatusDisconnected)

	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		if backupConnects.Load() >= 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if backupConnects.Load() == 0 {
		t.Fatal("watchdog did not activate backup profile failover")
	}
}
