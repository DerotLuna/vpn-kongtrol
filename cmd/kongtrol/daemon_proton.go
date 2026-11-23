package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
)

// A direct profile from a Proton account and Proton's own app clash
// (same account, same tunnel address), split tunnel or not (ADR-0103).
// When the user connects Proton's app while such a profile is up, the
// daemon disconnects the profile itself and says why, once.

// protonHandoverEvery is how often the daemon looks for Proton's app
// tunnel; the provider tunnels are listed only while a direct Proton
// profile is connected.
const protonHandoverEvery = 5 * time.Second

// runProtonHandover checks every interval until ctx ends.
func runProtonHandover(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			checkProtonHandover(ctx)
		}
	}
}

// handoverDisconnect is disconnectProfile (replaced in tests). It is a
// system disconnect: the profile stays in wanted.json (restore at start),
// and ProfileService marks it intended, so the watchdog does not reconnect
// it and the kill switch does not block it.
var handoverDisconnect = disconnectProfile

// checkProtonHandover disconnects every direct Proton profile while
// Proton's app tunnel is up, with one notice each.
func checkProtonHandover(ctx context.Context) {
	svc := profileSvc.Load()
	if svc == nil {
		return
	}
	for _, h := range svc.ProtonAppHandovers() {
		dctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := handoverDisconnect(dctx, h.Profile)
		cancel()
		if err != nil {
			msg := cf("cli.alert.proton_handover_failed", h.Profile, err)
			emitAlert("WARN", h.Profile, msg)
			logAudit("WARN", "vpn.proton_handover_failed", h.Profile, msg)
			continue
		}
		e := protonHandoverEvent(h)
		daemonEvents.Add(e)
		msg := cf(e.Key, h.Profile, h.AppProfile)
		emitAlert("WARN", h.Profile, msg)
		logAudit("WARN", "vpn.proton_handover", h.Profile, msg)
	}
}

// protonHandoverEvent is the notice for one handover: with a protonvpn
// profile it names it (the app's tunnel shows there); without one it
// offers to add it (Offer add_provider, last arg the provider type).
// Both offer "Reintentar" for the direct profile.
func protonHandoverEvent(h app.ProtonHandover) monitor.Event {
	e := monitor.Event{Kind: monitor.EventTunnelWatchdog, Level: "WARN", Profile: h.Profile, CanConnect: true}
	if h.AppProfile != "" {
		e.Key, e.Args = "watchdog.event.proton_handover", []string{h.Profile, h.AppProfile}
		return e
	}
	e.Key, e.Args, e.Offer = "watchdog.event.proton_handover_add", []string{h.Profile, "protonvpn"}, "add_provider"
	return e
}

// protonHinted remembers the profiles told, in this outage, that Proton's
// kill switch may block them (reset when they answer again).
var protonHinted sync.Map

// maybeProtonBlocksHint tells the user, once per outage, that a split
// profile that stopped answering may be blocked by Proton's "advanced
// kill switch" (not verified, ADR-0103).
func maybeProtonBlocksHint(profile string) {
	svc := profileSvc.Load()
	if svc == nil || !svc.ProtonAppMayBlock(profile) {
		return
	}
	if _, told := protonHinted.LoadOrStore(profile, true); told {
		return
	}
	watchdogNotice("WARN", "vpn.proton_blocks_other", "watchdog.event.proton_blocks_other", profile)
}

// reconnectBlockedNotice picks the notice for a reconnect only the user
// can unblock: the reason in err (ADR-0099, ADR-0103) and its args.
func reconnectBlockedNotice(profile string, err error) (string, []string) {
	var ce *app.FullTunnelConflictError
	if errors.As(err, &ce) && ce.ValidationError != nil {
		switch ce.Key {
		case "api.err.connect_proton_app":
			return "watchdog.event.reconnect_blocked_proton", []string{profile}
		case "api.err.connect_same_address":
			args := []string{profile}
			for _, a := range ce.Args[1:] {
				args = append(args, fmt.Sprint(a))
			}
			return "watchdog.event.reconnect_blocked_address", args
		}
	}
	return "watchdog.event.reconnect_blocked", []string{profile}
}
