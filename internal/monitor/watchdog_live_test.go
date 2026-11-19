package monitor_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Profiles changed without a restart (ADR-0053): the watchdog follows an
// adapter replaced in the set, and Forget stops watching a removed one.

func TestWatchdog_FollowsReplacedAdapter(t *testing.T) {
	old := newMockAdapter(vpn.StatusConnected)
	set := vpn.NewSet(map[string]vpn.VPNAdapter{"office": old})
	repl := newMockAdapter(vpn.StatusDisconnected)
	var reconnects atomic.Int32
	wd := monitor.NewWatchdog(set, func(_ context.Context, name string) error {
		reconnects.Add(1)
		repl.setStatus(vpn.StatusConnected)
		return nil
	}, zapNop())
	wd.SetPollInterval(20 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("office")
	wd.Start(ctx)
	defer wd.Stop()

	set.Put("office", repl) // replaced while the old one still says connected
	if !waitFor(t, 3*time.Second, func() bool { return reconnects.Load() >= 1 }) {
		t.Fatal("watchdog kept polling the old adapter")
	}
}

func TestWatchdog_ForgetStopsWatching(t *testing.T) {
	a := newMockAdapter(vpn.StatusConnected)
	set := vpn.NewSet(map[string]vpn.VPNAdapter{"office": a, "backup": newMockAdapter(vpn.StatusDisconnected)})
	var reconnects atomic.Int32
	wd := monitor.NewWatchdog(set, func(context.Context, string) error {
		reconnects.Add(1)
		return nil
	}, zapNop())
	wd.SetPollInterval(20 * time.Millisecond)
	wd.ConfigureFailover(map[string][]string{"office": {"backup"}})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("office")
	wd.Start(ctx)
	defer wd.Stop()

	set.Delete("office")
	wd.Forget("office")
	a.setStatus(vpn.StatusDisconnected)
	time.Sleep(150 * time.Millisecond)
	if n := reconnects.Load(); n != 0 {
		t.Fatalf("a forgotten profile was reconnected %d times", n)
	}

	// Same name added again: a clean start, watched again after MarkActive.
	b := newMockAdapter(vpn.StatusConnected)
	set.Put("office", b)
	wd.MarkActive("office")
	b.setStatus(vpn.StatusDisconnected)
	if !waitFor(t, 3*time.Second, func() bool { return reconnects.Load() >= 1 }) {
		t.Fatal("re-added profile is not watched")
	}
}
