package monitor_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// healthAdapter also reports tunnel health (like the WireGuard adapter).
type healthAdapter struct {
	*mockAdapter
	health atomic.Value // vpn.HealthState
}

func (h *healthAdapter) TunnelHealth() *vpn.TunnelHealth {
	return &vpn.TunnelHealth{State: h.health.Load().(vpn.HealthState)}
}

// A tunnel that is up but stopped answering (adapter: StatusError +
// HealthNoResponse) is notified once and reconnected like a drop; once
// handshakes resume the watchdog reports it responds again.
func TestWatchdog_NoResponseThenResponding(t *testing.T) {
	a := &healthAdapter{mockAdapter: newMockAdapter(vpn.StatusConnected)}
	a.health.Store(vpn.HealthOK)

	var reconnects atomic.Int32
	connect := func(context.Context, string) error {
		reconnects.Add(1)
		a.setStatus(vpn.StatusConnected)
		a.health.Store(vpn.HealthIdle) // not proven yet
		return nil
	}
	var mu sync.Mutex
	var events []string
	wd := monitor.NewWatchdog(vpn.NewSet(map[string]vpn.VPNAdapter{"brainbox": a}), connect, zapNop())
	wd.SetPollInterval(20 * time.Millisecond)
	wd.SetEventCallback(func(e monitor.WatchdogEvent) {
		mu.Lock()
		events = append(events, e.Event)
		mu.Unlock()
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("brainbox")
	wd.Start(ctx)
	defer wd.Stop()

	a.health.Store(vpn.HealthNoResponse)
	a.setStatus(vpn.StatusError)

	waitFor(t, 10*time.Second, func() bool { return reconnects.Load() >= 1 })
	time.Sleep(100 * time.Millisecond) // several polls while unproven
	a.health.Store(vpn.HealthOK)
	waitFor(t, 10*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return count(events, "responding") == 1
	})
	mu.Lock()
	defer mu.Unlock()
	if n := count(events, "no_response"); n != 1 {
		t.Fatalf("no_response events = %d, want 1 (%v)", n, events)
	}
}

func count(s []string, v string) int {
	n := 0
	for _, x := range s {
		if x == v {
			n++
		}
	}
	return n
}
