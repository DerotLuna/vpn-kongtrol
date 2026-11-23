package monitor

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

type fakeMatcher map[netip.Addr]PolicyMatch

func (f fakeMatcher) MatchAddr(a netip.Addr) (PolicyMatch, bool) { m, ok := f[a]; return m, ok }

func TestPolicyWatchDefersConflictingAutomaticConnect(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		addr := netip.MustParseAddr("192.0.2.42")
		conflict := "Proton VPN"
		attempts := 0
		w := &PolicyWatch{
			Matcher:         fakeMatcher{addr: {Policy: "rule", Via: "office", Action: config.OnTunnelDownConnect}},
			Status:          func(string) vpn.Status { return vpn.StatusDisconnected },
			ConnectConflict: func(string) string { return conflict },
			Connect:         func(context.Context, string) error { attempts++; return nil },
			Events:          NewEventLog(10),
			conns:           func() ([]netip.Addr, error) { return []netip.Addr{addr}, nil },
		}
		for range 5 {
			w.Check(context.Background())
		}
		synctest.Wait()
		events, _ := w.Events.After(0)
		if attempts != 0 || len(events) != 1 || events[0].Key != "policy.event.connect_conflict" || events[0].CanConnect {
			t.Fatalf("while conflict is active got %d attempts and events %+v", attempts, events)
		}
		conflict = ""
		w.Check(context.Background())
		synctest.Wait()
		if attempts != 1 {
			t.Fatalf("got %d attempts after conflict cleared, want 1", attempts)
		}
	})
}

func TestPolicyWatchRetryBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		git := netip.MustParseAddr("192.0.2.1")
		wiki := netip.MustParseAddr("192.0.2.2")
		other := netip.MustParseAddr("192.0.2.3")
		addrs := []netip.Addr{git, wiki}
		var attempts []string
		success := false
		w := &PolicyWatch{
			Matcher: fakeMatcher{
				git:   {Policy: "git", Via: "office", Action: config.OnTunnelDownConnect, Label: "git.corp"},
				wiki:  {Policy: "wiki", Via: "office", Action: config.OnTunnelDownConnect, Label: "wiki.corp"},
				other: {Policy: "db", Via: "other", Action: config.OnTunnelDownConnect, Label: "db.corp"},
			},
			Status: func(string) vpn.Status { return vpn.StatusDisconnected },
			Connect: func(_ context.Context, profile string) error {
				attempts = append(attempts, profile)
				if success {
					return nil
				}
				return errors.New("connect failed")
			},
			Events: NewEventLog(20),
			conns:  func() ([]netip.Addr, error) { return addrs, nil },
		}
		check := func(want int) {
			t.Helper()
			w.Check(ctx)
			synctest.Wait()
			if len(attempts) != want {
				t.Fatalf("got attempts %v, want %d", attempts, want)
			}
		}
		check(1)
		time.Sleep(3*time.Minute - time.Second)
		check(1)
		time.Sleep(time.Second)
		addrs = []netip.Addr{wiki, git}
		check(2)
		time.Sleep(6*time.Minute - time.Second)
		check(2)
		time.Sleep(time.Second)
		addrs = nil
		check(2) // eligibility alone cannot trigger a retry
		addrs = []netip.Addr{git, wiki}
		check(3)
		time.Sleep(time.Hour)
		check(3)

		events, _ := w.Events.After(0)
		if len(events) != 6 {
			t.Fatalf("got events %+v, want 6", events)
		}
		for _, e := range []Event{events[1], events[3]} {
			if e.Key != "policy.event.connect_failed" {
				t.Fatalf("intermediate failure = %+v", e)
			}
		}
		if e := events[5]; e.Key != "policy.event.connect_exhausted" || !e.CanConnect || e.Level != "WARN" || e.Profile != "office" || e.Action != config.OnTunnelDownConnect {
			t.Fatalf("exhaustion event = %+v", e)
		}

		addrs = append(addrs, other)
		check(4)
		if attempts[3] != "other" {
			t.Fatalf("exhausting office should still allow other: %v", attempts)
		}

		w.ResetConnectBudget("office") // a successful manual connect, even without observed traffic
		success = true
		check(5)
		check(6) // after an automatic success a new disconnect is immediately eligible
		if attempts[4] != "office" || attempts[5] != "office" {
			t.Fatalf("office was not re-armed: %v", attempts)
		}
	})
}

func TestPolicyWatchCoalescesInFlightAttempts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		addr := netip.MustParseAddr("192.0.2.1")
		status := vpn.StatusConnecting
		finish := make(chan struct{})
		var attempts atomic.Int32
		w := &PolicyWatch{
			Matcher: fakeMatcher{addr: {Policy: "git", Via: "office", Action: config.OnTunnelDownConnect}},
			Status:  func(string) vpn.Status { return status },
			Connect: func(context.Context, string) error {
				attempts.Add(1)
				<-finish
				return errors.New("connect failed")
			},
			Events: NewEventLog(10),
			conns:  func() ([]netip.Addr, error) { return []netip.Addr{addr}, nil },
		}
		w.Check(ctx)
		synctest.Wait()
		if got := attempts.Load(); got != 0 {
			t.Fatalf("got %d attempts while adapter was connecting", got)
		}

		status = vpn.StatusDisconnected
		var checks sync.WaitGroup
		for range 8 {
			checks.Go(func() { w.Check(ctx) })
		}
		checks.Wait()
		synctest.Wait()
		time.Sleep(time.Hour)
		status = vpn.StatusConnected
		w.Check(ctx) // a connection observed outside the daemon resets the budget
		status = vpn.StatusDisconnected
		w.Check(ctx)
		synctest.Wait()
		close(finish)
		synctest.Wait()
		if got := attempts.Load(); got != 1 {
			t.Fatalf("got %d attempts while one was in flight", got)
		}
		if events, _ := w.Events.After(0); len(events) != 1 {
			t.Fatalf("got events %+v, want only the initial attempt", events)
		}
		w.Check(ctx)
		synctest.Wait()
		if got := attempts.Load(); got != 2 {
			t.Fatalf("got %d attempts after resuming", got)
		}
	})
}

func TestPolicyWatchNoticeThrottle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		addr := netip.MustParseAddr("192.0.2.1")
		w := &PolicyWatch{
			Matcher: fakeMatcher{addr: {Policy: "ai", Via: "brainbox", Action: config.OnTunnelDownAllow, Label: "claude.ai"}},
			Status:  func(string) vpn.Status { return vpn.StatusDisconnected },
			Events:  NewEventLog(10),
			conns:   func() ([]netip.Addr, error) { return []netip.Addr{addr, addr}, nil },
		}
		w.Check(context.Background())
		time.Sleep(DefaultNoticeEvery - time.Second)
		w.Check(context.Background())
		if events, _ := w.Events.After(0); len(events) != 1 || events[0].Key != "policy.event.allow" || !events[0].CanConnect {
			t.Fatalf("got notices %+v, want one actionable allow notice", events)
		}
		time.Sleep(time.Second)
		w.Check(context.Background())
		if events, _ := w.Events.After(1); len(events) != 1 {
			t.Fatalf("got notices %+v, want another allow notice", events)
		}
	})
}
