package monitor_test

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// The reconnect policy (ADR-0098): one attempt per delay, then stop.
func TestRetryDelay(t *testing.T) {
	delays := monitor.DefaultRetryDelays
	if len(delays) != 3 {
		t.Fatalf("DefaultRetryDelays has %d attempts, want 3 (like ADR-0095)", len(delays))
	}
	for made, want := range delays {
		got, ok := monitor.RetryDelay(delays, made)
		if !ok || got != want {
			t.Errorf("RetryDelay(%d) = %v, %v; want %v, true", made, got, ok, want)
		}
	}
	for _, made := range []int{3, 4, -1} {
		if _, ok := monitor.RetryDelay(delays, made); ok {
			t.Errorf("RetryDelay(%d) allowed another attempt", made)
		}
	}
	if _, ok := monitor.RetryDelay(nil, 0); ok {
		t.Error("an empty schedule allowed an attempt")
	}
}

// boundedWatchdog is a watchdog with fast delays over one profile "work"
// whose network fingerprint the test controls.
func boundedWatchdog(t *testing.T, connect monitor.ConnectFunc) (*monitor.Watchdog, *mockAdapter, *eventLog, *atomic.Value) {
	t.Helper()
	a := newMockAdapter(vpn.StatusConnected)
	wd := monitor.NewWatchdog(vpn.NewSet(map[string]vpn.VPNAdapter{"work": a}), connect, zapNop())
	wd.SetPollInterval(20 * time.Millisecond)
	wd.SetRetryDelays([]time.Duration{5 * time.Millisecond, 5 * time.Millisecond, 5 * time.Millisecond})
	var netID atomic.Value
	netID.Store("wifi-home")
	wd.SetNetworkFingerprint(func() string { return netID.Load().(string) })
	log := &eventLog{}
	wd.SetEventCallback(log.add)
	return wd, a, log, &netID
}

// Three attempts, one notice, then nothing until the network changes; a
// new round that fails again does not notify again; success after the
// notice is reported as AfterGiveUp.
func TestWatchdog_GivesUpThenRearmsOnNetworkChange(t *testing.T) {
	var tries atomic.Int32
	var succeed atomic.Bool
	var a *mockAdapter
	connect := func(_ context.Context, _ string) error {
		tries.Add(1)
		if succeed.Load() {
			a.setStatus(vpn.StatusConnected)
			return nil
		}
		return errors.New("server unreachable")
	}
	wd, adapter, log, netID := boundedWatchdog(t, connect)
	a = adapter
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("work")
	wd.Start(ctx)
	defer wd.Stop()
	a.setStatus(vpn.StatusError)

	if !waitFor(t, 5*time.Second, func() bool { return log.count("reconnect_gave_up") == 1 }) {
		t.Fatal("the watchdog never gave up")
	}
	time.Sleep(200 * time.Millisecond) // ten polls: a further attempt would show
	if got := tries.Load(); got != 3 {
		t.Fatalf("attempts before giving up = %d, want 3", got)
	}
	e, _ := log.first("reconnect_gave_up")
	if e.Profile != "work" || e.Attempt != 3 || e.Err == nil {
		t.Errorf("reconnect_gave_up event = %+v", e)
	}

	// Another network: one more round, silently.
	netID.Store("wifi-office")
	if !waitFor(t, 5*time.Second, func() bool { return tries.Load() == 6 }) {
		t.Fatalf("a network change did not start a new round (attempts %d)", tries.Load())
	}
	time.Sleep(200 * time.Millisecond)
	if got := tries.Load(); got != 6 {
		t.Errorf("second round made %d attempts, want 3", got-3)
	}
	if n := log.count("reconnect_gave_up"); n != 1 {
		t.Errorf("reconnect_gave_up = %d, want 1 per outage (no repeated notices)", n)
	}
	if n := log.count("retry_rearmed"); n != 1 {
		t.Errorf("retry_rearmed = %d, want 1", n)
	}

	// Back online for real: the third round reconnects and says so.
	succeed.Store(true)
	netID.Store("")
	if !waitFor(t, 5*time.Second, func() bool { return log.count("reconnected") == 1 }) {
		t.Fatal("the re-armed round did not reconnect")
	}
	if e, _ := log.first("reconnected"); !e.AfterGiveUp {
		t.Errorf("reconnected after a give-up notice: AfterGiveUp = false")
	}
}

// The user connecting it again (MarkActive + connected) ends the outage:
// a later drop gets a full round and a new notice.
func TestWatchdog_UserReconnectStartsFreshBudget(t *testing.T) {
	var tries atomic.Int32
	connect := func(context.Context, string) error {
		tries.Add(1)
		return errors.New("down")
	}
	wd, a, log, _ := boundedWatchdog(t, connect)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("work")
	wd.Start(ctx)
	defer wd.Stop()
	a.setStatus(vpn.StatusError)
	if !waitFor(t, 5*time.Second, func() bool { return log.count("reconnect_gave_up") == 1 }) {
		t.Fatal("the watchdog never gave up")
	}
	// "Reintentar": the user connects it.
	a.setStatus(vpn.StatusConnected)
	wd.MarkActive("work")
	time.Sleep(100 * time.Millisecond)
	a.setStatus(vpn.StatusError)
	if !waitFor(t, 5*time.Second, func() bool { return log.count("reconnect_gave_up") == 2 }) {
		t.Fatal("a new outage after the user reconnected was not handled")
	}
	if got := tries.Load(); got != 6 {
		t.Errorf("attempts = %d, want 3 per outage", got)
	}
}

// A restore connect that failed at daemon start counts as the first
// attempt: the watchdog makes the remaining two.
func TestWatchdog_MarkActiveAfterFailedConnect(t *testing.T) {
	var tries atomic.Int32
	connect := func(context.Context, string) error {
		tries.Add(1)
		return errors.New("down")
	}
	wd, a, log, _ := boundedWatchdog(t, connect)
	a.setStatus(vpn.StatusDisconnected)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.Start(ctx)
	defer wd.Stop()
	wd.MarkActiveAfterFailedConnect("work")
	if !waitFor(t, 5*time.Second, func() bool { return log.count("reconnect_gave_up") == 1 }) {
		t.Fatal("the watchdog never gave up")
	}
	if got := tries.Load(); got != 2 {
		t.Errorf("watchdog attempts after a failed restore = %d, want 2", got)
	}
}

// `down` after the watchdog gave up: no notice, no attempts on a network
// change.
func TestWatchdog_GaveUpThenUserDisconnects(t *testing.T) {
	var tries atomic.Int32
	connect := func(context.Context, string) error {
		tries.Add(1)
		return errors.New("down")
	}
	wd, a, log, netID := boundedWatchdog(t, connect)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("work")
	wd.Start(ctx)
	defer wd.Stop()
	a.setStatus(vpn.StatusError)
	if !waitFor(t, 5*time.Second, func() bool { return log.count("reconnect_gave_up") == 1 }) {
		t.Fatal("the watchdog never gave up")
	}
	wd.MarkIntended("work")
	netID.Store("other")
	time.Sleep(200 * time.Millisecond)
	if got := tries.Load(); got != 3 {
		t.Errorf("attempts after the user turned it off = %d, want none", got-3)
	}
}

// A reconnect refused because another VPN carries all traffic (ADR-0099)
// is not an attempt: one notice, then wait; a network change retries.
func TestWatchdog_WaitForUserIsNotAnAttempt(t *testing.T) {
	errConflict := errors.New("another VPN carries all traffic")
	var tries atomic.Int32
	connect := func(context.Context, string) error {
		tries.Add(1)
		return errConflict
	}
	wd, a, log, netID := boundedWatchdog(t, connect)
	wd.SetWaitForUser(func(err error) bool { return errors.Is(err, errConflict) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wd.MarkActive("work")
	wd.Start(ctx)
	defer wd.Stop()
	a.setStatus(vpn.StatusError)
	if !waitFor(t, 5*time.Second, func() bool { return log.count("reconnect_blocked") == 1 }) {
		t.Fatal("no reconnect_blocked notice")
	}
	time.Sleep(200 * time.Millisecond)
	if got := tries.Load(); got != 1 {
		t.Fatalf("tries while waiting for the user = %d, want 1", got)
	}
	if n := log.count("reconnect_failed") + log.count("reconnect_gave_up"); n != 0 {
		t.Errorf("a conflict was counted as a failure (%d events)", n)
	}
	netID.Store("cable")
	if !waitFor(t, 5*time.Second, func() bool { return tries.Load() == 2 }) {
		t.Fatal("a network change did not retry")
	}
	time.Sleep(200 * time.Millisecond)
	if n := log.count("reconnect_blocked"); n != 1 {
		t.Errorf("reconnect_blocked = %d, want 1 per outage", n)
	}
}

func TestFingerprintOf(t *testing.T) {
	ip := net.ParseIP
	a := monitor.FingerprintOf([]monitor.InterfaceAddrs{
		{Name: "wlan0", IPs: []net.IP{ip("192.168.1.20"), ip("fe80::1"), ip("2001:db8::aaaa")}},
		{Name: "eth0", IPs: []net.IP{ip("169.254.3.4")}},
	})
	// Same network: other order, a rotated IPv6 temporary address.
	b := monitor.FingerprintOf([]monitor.InterfaceAddrs{
		{Name: "eth0", IPs: []net.IP{ip("169.254.3.4")}},
		{Name: "wlan0", IPs: []net.IP{ip("2001:db8::bbbb"), ip("192.168.1.20")}},
	})
	if a != b || a != "wlan0=192.168.1.20,wlan0=v6" {
		t.Errorf("same network: %q vs %q", a, b)
	}
	other := monitor.FingerprintOf([]monitor.InterfaceAddrs{{Name: "wlan0", IPs: []net.IP{ip("10.0.0.7")}}})
	if other == a {
		t.Error("another IPv4 address gave the same fingerprint")
	}
	if got := monitor.FingerprintOf(nil); got != "" {
		t.Errorf("offline = %q, want empty", got)
	}
}

func TestWokeFromSleep(t *testing.T) {
	for _, c := range []struct {
		mono, wall time.Duration
		want       bool
	}{
		{5 * time.Second, 5 * time.Second, false},
		{5 * time.Second, 7 * time.Second, false}, // clock adjusted
		{5 * time.Second, 2 * time.Hour, true},    // a night asleep
		{5 * time.Second, -time.Hour, false},      // clock set back
		{2 * time.Minute, 2 * time.Minute, false}, // a long connect attempt
	} {
		if got := monitor.WokeFromSleep(c.mono, c.wall); got != c.want {
			t.Errorf("WokeFromSleep(%v, %v) = %v, want %v", c.mono, c.wall, got, c.want)
		}
	}
}
