package main

import (
	"reflect"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// A full-tunnel drop with allow_lan off also cuts the local network, so the
// notice says so and names the switch that keeps it (ADR-0063).
func TestKillSwitchBlockKey(t *testing.T) {
	for _, c := range []struct {
		full, lan bool
		want      string
	}{
		{false, false, "killswitch.event.blocking"},
		{false, true, "killswitch.event.blocking"},
		{true, true, "killswitch.event.blocking_full"},
		{true, false, "killswitch.event.blocking_full_lan"},
	} {
		if got := killSwitchBlockKey(c.full, c.lan); got != c.want {
			t.Errorf("killSwitchBlockKey(%v, %v) = %q, want %q", c.full, c.lan, got, c.want)
		}
		for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
			if i18n.T(lang, c.want) == c.want {
				t.Errorf("%v: missing %s", lang, c.want)
			}
		}
	}
}

// The user's real config: similar names and priorities, no failover
// declared. Priority must never become failover (ADR-0018).
func TestProfileFailover_OnlyExplicitDeclarations(t *testing.T) {
	old := cfg
	t.Cleanup(func() { cfg = old })
	cfg = &config.Config{VPNs: map[string]config.VPNConfig{
		"ekiipago":      {Type: "forticlient", Priority: 1},
		"us-content":    {Type: "protonvpn", Priority: 1},
		"brainbox":      {Type: "wireguard", Priority: 15},
		"ekiipago-ovpn": {Type: "openvpn", Priority: 20},
		"brainbox-ovpn": {Type: "openvpn", Priority: 20},
	}}
	if got := profileFailover(cfg); len(got) != 0 {
		t.Fatalf("failover derived from priority/names: %v", got)
	}

	v := cfg.VPNs["brainbox"]
	v.Failover = []string{"brainbox-ovpn"}
	cfg.VPNs["brainbox"] = v
	want := map[string][]string{"brainbox": {"brainbox-ovpn"}}
	if got := profileFailover(cfg); !reflect.DeepEqual(got, want) {
		t.Fatalf("profileFailover(cfg) = %v, want %v", got, want)
	}
}

// A provider tunnel Kongtrol cannot reconnect on its own must reach every
// client as a notice (tray, dashboards, live view), not only the daemon log.
func TestOnWatchdogEvent_ManualReconnectIsANotice(t *testing.T) {
	old := daemonEvents
	t.Cleanup(func() { daemonEvents = old })
	daemonEvents = monitor.NewEventLog(10)

	onWatchdogEvent(monitor.WatchdogEvent{Profile: "proton", Event: "manual_reconnect_required"})

	evs, _ := daemonEvents.After(0)
	if len(evs) != 1 {
		t.Fatalf("events = %d, want 1", len(evs))
	}
	e := evs[0]
	if e.Kind != monitor.EventTunnelWatchdog || e.Key != "watchdog.event.manual_reconnect" ||
		e.Profile != "proton" || !reflect.DeepEqual(e.Args, []string{"proton"}) || e.Level != "WARN" {
		t.Fatalf("event = %+v", e)
	}
}
