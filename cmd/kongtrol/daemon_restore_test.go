package main

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

func TestRestoreTargets(t *testing.T) {
	exists := func(n string) bool { return n != "deleted" }
	got := restoreTargets([]string{"home", "deleted", "work"}, []string{"work"}, exists)
	if !slices.Equal(got, []string{"home"}) {
		t.Fatalf("restoreTargets = %v, want [home]", got)
	}
}

// The kill switch keeps blocking a profile that gave up (security first);
// the notice says how to get traffic back.
func TestGaveUpKey(t *testing.T) {
	for _, c := range []struct {
		ks   security.KillSwitchStatus
		want string
	}{
		{security.KillSwitchStatus{}, "watchdog.event.gave_up"},
		{security.KillSwitchStatus{Blocking: []string{"other"}}, "watchdog.event.gave_up"},
		{security.KillSwitchStatus{Blocking: []string{"work"}}, "watchdog.event.gave_up_blocking"},
		{security.KillSwitchStatus{Blocking: []string{"work"}, FullTunnel: []string{"work"}}, "watchdog.event.gave_up_blocking_full"},
	} {
		if got := gaveUpKey("work", c.ks); got != c.want {
			t.Errorf("gaveUpKey(%+v) = %q, want %q", c.ks, got, c.want)
		}
		for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
			if i18n.T(lang, c.want) == c.want {
				t.Errorf("%v: missing %s", lang, c.want)
			}
		}
	}
}

// One notice with a Retry action; the recovery after it is told too.
func TestOnWatchdogEvent_GaveUpOffersRetry(t *testing.T) {
	old := daemonEvents
	t.Cleanup(func() { daemonEvents = old })
	daemonEvents = monitor.NewEventLog(10)

	onWatchdogEvent(monitor.WatchdogEvent{Profile: "work", Event: "reconnect_gave_up", Attempt: 3, Err: errors.New("x")})
	onWatchdogEvent(monitor.WatchdogEvent{Profile: "work", Event: "retry_rearmed"})
	onWatchdogEvent(monitor.WatchdogEvent{Profile: "work", Event: "reconnected", Attempt: 1, AfterGiveUp: true})
	onWatchdogEvent(monitor.WatchdogEvent{Profile: "home", Event: "reconnected", Attempt: 2})

	evs, _ := daemonEvents.After(0)
	if len(evs) != 2 {
		t.Fatalf("events = %+v, want the give-up and one recovery", evs)
	}
	if e := evs[0]; e.Key != "watchdog.event.gave_up" || !e.CanConnect || e.Profile != "work" ||
		!slices.Equal(e.Args, []string{"work", "3"}) || e.Level != "WARN" {
		t.Errorf("give-up = %+v", e)
	}
	if e := evs[1]; e.Key != "watchdog.event.reconnected" || e.CanConnect || e.Profile != "work" {
		t.Errorf("recovery = %+v", e)
	}
	if !offersRetry("watchdog.event.reconnect_blocked") || offersRetry("watchdog.event.no_response") {
		t.Error("offersRetry")
	}
}

// What the user turns on is remembered; what the user turns off, or
// "stop everything", is forgotten.
func TestUserConnectDisconnectRemember(t *testing.T) {
	oldW := wantedTunnels
	t.Cleanup(func() { wantedTunnels = oldW })
	wantedTunnels = daemonctl.NewWanted(filepath.Join(t.TempDir(), "wanted.json"))

	// No profile service loaded: the connect fails and nothing is saved.
	oldSvc := profileSvc.Load()
	profileSvc.Store(nil)
	t.Cleanup(func() { profileSvc.Store(oldSvc) })
	if err := userConnectProfile(context.Background(), "work"); err == nil {
		t.Fatal("connect without a service succeeded")
	}
	if got := wantedTunnels.List(); len(got) != 0 {
		t.Fatalf("a failed connect was remembered: %v", got)
	}
	_ = wantedTunnels.Add("work")
	_ = wantedTunnels.Add("home")
	_ = userDisconnectProfile(context.Background(), "work")
	if got := wantedTunnels.List(); !slices.Equal(got, []string{"home"}) {
		t.Fatalf("after the user disconnected work: %v", got)
	}
	forgetWantedTunnels()
	if got := wantedTunnels.List(); len(got) != 0 {
		t.Fatalf("after stop-all: %v", got)
	}
}
