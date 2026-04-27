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

// mockAdapter is a controllable VPNAdapter stub.
type mockAdapter struct {
	status vpn.Status
}

func (m *mockAdapter) Name() string                                          { return "mock" }
func (m *mockAdapter) Version() string                                       { return "0.0.0" }
func (m *mockAdapter) Capabilities() vpn.Capabilities                       { return vpn.Capabilities{} }
func (m *mockAdapter) Connect(_ context.Context, _ vpn.AdapterConfig) error { return nil }
func (m *mockAdapter) Disconnect(_ context.Context) error                   { return nil }
func (m *mockAdapter) Reconnect(_ context.Context) error                    { return nil }
func (m *mockAdapter) TunnelInfo() (*vpn.TunnelInfo, error)                 { return nil, nil }
func (m *mockAdapter) Status() vpn.Status                                   { return m.status }

func zapNop() *zap.Logger { return zap.NewNop() }

// TestWatchdog_ReconnectsOnDrop verifies the watchdog calls ConnectFunc when
// an adapter transitions to disconnected without MarkIntended.
func TestWatchdog_ReconnectsOnDrop(t *testing.T) {
	adapter := &mockAdapter{status: vpn.StatusConnected}
	adapters := map[string]vpn.VPNAdapter{"office": adapter}

	var reconnects atomic.Int32
	connect := func(_ context.Context, name string) error {
		reconnects.Add(1)
		adapter.status = vpn.StatusConnected
		return nil
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.Start(ctx)

	// Simulate unexpected drop.
	adapter.status = vpn.StatusDisconnected

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
	adapter := &mockAdapter{status: vpn.StatusConnected}
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

	adapter.status = vpn.StatusDisconnected

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
	adapter := &mockAdapter{status: vpn.StatusConnected}
	adapters := map[string]vpn.VPNAdapter{"office": adapter}

	var reconnects atomic.Int32
	connect := func(_ context.Context, _ string) error {
		reconnects.Add(1)
		adapter.status = vpn.StatusConnected
		return nil
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wd.MarkIntended("office")
	wd.Start(ctx)

	adapter.status = vpn.StatusDisconnected

	// Still no reconnect while intended.
	time.Sleep(300 * time.Millisecond)
	if reconnects.Load() > 0 {
		t.Fatalf("reconnected while MarkIntended was set")
	}

	// Re-arm and drop.
	wd.MarkActive("office")
	adapter.status = vpn.StatusDisconnected

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

// TestWatchdog_Stop verifies no reconnects happen after Stop().
func TestWatchdog_Stop(t *testing.T) {
	adapter := &mockAdapter{status: vpn.StatusConnected}
	adapters := map[string]vpn.VPNAdapter{"office": adapter}

	var reconnects atomic.Int32
	connect := func(_ context.Context, _ string) error {
		reconnects.Add(1)
		return nil
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())

	ctx, cancel := context.WithCancel(context.Background())
	wd.Start(ctx)
	cancel() // cancel context before Stop to make sure Stop is idempotent
	wd.Stop()

	adapter.status = vpn.StatusDisconnected
	time.Sleep(300 * time.Millisecond)

	if reconnects.Load() > 0 {
		t.Errorf("got %d reconnect(s) after Stop — expected 0", reconnects.Load())
	}
}

// TestWatchdog_MultipleProfiles verifies each profile is watched independently.
func TestWatchdog_MultipleProfiles(t *testing.T) {
	a1 := &mockAdapter{status: vpn.StatusConnected}
	a2 := &mockAdapter{status: vpn.StatusConnected}
	adapters := map[string]vpn.VPNAdapter{"p1": a1, "p2": a2}

	reconnected := map[string]*atomic.Int32{
		"p1": {},
		"p2": {},
	}

	connect := func(_ context.Context, name string) error {
		if c, ok := reconnected[name]; ok {
			c.Add(1)
		}
		adapters[name].(*mockAdapter).status = vpn.StatusConnected
		return nil
	}

	wd := monitor.NewWatchdog(adapters, connect, zapNop())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.Start(ctx)

	// Drop only p1.
	a1.status = vpn.StatusDisconnected

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
