package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// offerTestSetup points the offer at a temporary preferences.json and a
// fresh event log; protects says whether the kill switch guards a profile.
func offerTestSetup(t *testing.T, protects bool) string {
	t.Helper()
	oldEv, oldPrefs, oldProt, oldDone := daemonEvents, killSwitchOfferPrefs, killSwitchProtects, killSwitchOfferDone
	t.Cleanup(func() {
		daemonEvents, killSwitchOfferPrefs, killSwitchProtects, killSwitchOfferDone = oldEv, oldPrefs, oldProt, oldDone
	})
	daemonEvents = monitor.NewEventLog(10)
	killSwitchOfferPrefs = filepath.Join(t.TempDir(), "preferences.json")
	killSwitchProtects = func(string) bool { return protects }
	killSwitchOfferDone = false
	return killSwitchOfferPrefs
}

// The first unexpected drop with the kill switch off offers it once; no
// answer is needed for it not to come back (ADR-0102).
func TestOfferKillSwitch_OnceAndPersisted(t *testing.T) {
	prefs := offerTestSetup(t, false)

	onWatchdogEvent(monitor.WatchdogEvent{Profile: "work", Event: "reconnect_attempt", Attempt: 1})
	onWatchdogEvent(monitor.WatchdogEvent{Profile: "work", Event: "reconnect_attempt", Attempt: 2})
	onWatchdogEvent(monitor.WatchdogEvent{Profile: "home", Event: "reconnect_attempt", Attempt: 1})

	evs, _ := daemonEvents.After(0)
	if len(evs) != 1 {
		t.Fatalf("events = %+v, want one offer", evs)
	}
	if e := evs[0]; e.Key != "killswitch.event.offer" || e.Offer != offerEnableKillSwitch || e.Profile != "work" || e.CanConnect {
		t.Errorf("offer = %+v", e)
	}
	p, err := config.LoadPreferences(prefs)
	if err != nil || !p.KillSwitchOffered {
		t.Fatalf("preferences = %+v, %v; want killswitch_offered", p, err)
	}

	// A new daemon run reads it back: no second offer.
	killSwitchOfferDone = false
	daemonEvents = monitor.NewEventLog(10)
	onWatchdogEvent(monitor.WatchdogEvent{Profile: "work", Event: "reconnect_attempt", Attempt: 1})
	if evs, _ := daemonEvents.After(0); len(evs) != 0 {
		t.Fatalf("offered again after a restart: %+v", evs)
	}
}

// With the kill switch on for the profile there is nothing to offer, and
// the offer is not spent.
func TestOfferKillSwitch_NotWhenProtected(t *testing.T) {
	prefs := offerTestSetup(t, true)
	onWatchdogEvent(monitor.WatchdogEvent{Profile: "work", Event: "reconnect_attempt", Attempt: 1})
	if evs, _ := daemonEvents.After(0); len(evs) != 0 {
		t.Fatalf("events = %+v, want none", evs)
	}
	if p, _ := config.LoadPreferences(prefs); p.KillSwitchOffered {
		t.Fatal("offer marked as asked without asking")
	}
}

// Every kill switch block notice carries "Recuperar internet".
func TestBlockingNoticesOfferReleaseBlock(t *testing.T) {
	old := daemonEvents
	t.Cleanup(func() { daemonEvents = old })
	daemonEvents = monitor.NewEventLog(20)

	for _, key := range []string{"killswitch.event.blocking", "killswitch.event.blocking_full", "killswitch.event.blocking_full_lan"} {
		watchdogNotice("WARN", "killswitch.blocking", key, "work")
	}
	watchdogNotice("WARN", "vpn.reconnect_gave_up", "watchdog.event.gave_up_blocking", "work", "3")
	watchdogNotice("WARN", "vpn.reconnect_gave_up", "watchdog.event.gave_up_blocking_full", "work", "3")
	watchdogNotice("WARN", "vpn.reconnect_gave_up", "watchdog.event.gave_up", "work", "3")
	watchdogNotice("WARN", "killswitch.superseded", "killswitch.event.superseded", "work", "other")

	evs, _ := daemonEvents.After(0)
	if len(evs) != 7 {
		t.Fatalf("events = %d, want 7", len(evs))
	}
	for i, e := range evs {
		want := offerReleaseBlock
		if i >= 5 {
			want = ""
		}
		if e.Offer != want {
			t.Errorf("%s: offer %q, want %q", e.Key, e.Offer, want)
		}
	}
}

func TestOfferHintCLI(t *testing.T) {
	now := time.Now()
	for _, c := range []struct {
		e    monitor.Event
		want string // substring; "" = no hint
	}{
		{monitor.Event{Key: "killswitch.event.offer", Args: []string{"work"}, Profile: "work", Offer: offerEnableKillSwitch}, "kongtrol config killswitch on"},
		{monitor.Event{Key: "killswitch.event.blocking_full", Args: []string{"work"}, Profile: "work", Offer: offerReleaseBlock}, "kongtrol down work"},
		{monitor.Event{Key: "x", Args: []string{"work", "protonvpn"}, Profile: "work", Offer: offerAddProvider}, "kongtrol profile add protonvpn"},
		{monitor.Event{Key: "x", Profile: "work"}, ""},
	} {
		got := offerHintCLI(&c.e)
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("offerHintCLI(%+v) = %q, want %q", c.e, got, c.want)
		}
		c.e.Time = now
		if c.want != "" && !strings.Contains(policyNoticeLine(&c.e, now), c.want) {
			t.Errorf("live view line misses %q", c.want)
		}
	}
	// gave_up_blocking already names `kongtrol down`: not twice.
	e := monitor.Event{Time: now, Kind: monitor.EventTunnelWatchdog, Key: "watchdog.event.gave_up_blocking",
		Args: []string{"work", "3"}, Profile: "work", Offer: offerReleaseBlock, CanConnect: true}
	if offerHintCLI(&e) != "" {
		t.Error("release hint repeated a command the notice already names")
	}
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		if i18n.T(lang, "killswitch.event.offer") == "killswitch.event.offer" {
			t.Errorf("%v: missing killswitch.event.offer", lang)
		}
	}
}
