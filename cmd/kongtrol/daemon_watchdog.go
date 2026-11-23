package main

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/app"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
)

// onWatchdogEvent turns watchdog events into alerts, audit entries and,
// for what the user must know about, a daemon notice (monitor.Event) that
// the tray, the dashboard and the live view render in their own language.
func onWatchdogEvent(e monitor.WatchdogEvent) {
	profile, attempt, err := e.Profile, e.Attempt, e.Err
	switch e.Event {
	case "reconnect_attempt":
		emitAlert("WARN", profile, cf("cli.alert.reconnect_attempt", profile, attempt))
		logAudit("WARN", "vpn.reconnect_attempt", profile, cf("cli.alert.reconnect_attempt", profile, attempt))
		releaseDroppedDNS(profile)
	case "reconnect_failed":
		emitAlert("ERROR", profile, cf("cli.alert.reconnect_failed", profile, attempt, err))
		logAudit("ERROR", "vpn.reconnect_failed", profile, cf("cli.alert.reconnect_failed", profile, attempt, err))
		if svc := profileSvc.Load(); svc != nil {
			svc.HandleReconnectError(profile, err)
		}
	case "reconnect_gave_up":
		// Once per outage: every attempt failed and no declared backup
		// took over (ADR-0098). Kongtrol stops trying until the network
		// changes; the notice offers "Reintentar" and, while the kill
		// switch blocks for it, how to get internet back.
		key := gaveUpKey(profile, killSwitchSvc.Status())
		watchdogNotice("WARN", "vpn.reconnect_gave_up", key, profile, strconv.Itoa(attempt))
	case "reconnect_blocked":
		// Once per outage: only the user can resolve it (another VPN
		// carries all traffic now, ADR-0099). Not an attempt.
		watchdogNotice("WARN", "vpn.reconnect_blocked", "watchdog.event.reconnect_blocked", profile)
	case "retry_rearmed":
		emitAlert("INFO", profile, cf("cli.alert.retry_rearmed", profile))
		logAudit("INFO", "vpn.retry_rearmed", profile, cf("cli.alert.retry_rearmed", profile))
	case "reconnected":
		if col != nil {
			col.RecordReconnect(profile)
		}
		emitAlert("INFO", profile, cf("cli.alert.reconnected", profile, attempt))
		logAudit("INFO", "vpn.reconnected", profile, cf("cli.alert.reconnected", profile, attempt))
		if e.AfterGiveUp { // the user was told it gave up
			daemonEvents.Add(monitor.Event{Kind: monitor.EventTunnelWatchdog, Level: "INFO",
				Key: "watchdog.event.reconnected", Args: []string{profile}, Profile: profile})
		}
	case "health_degraded":
		emitAlert("WARN", profile, cf("cli.alert.health_degraded", profile, err))
		logAudit("WARN", "vpn.health_degraded", profile, cf("cli.alert.health_degraded", profile, err))
	case "failover_started":
		emitAlert("WARN", profile, cf("cli.alert.failover_started", profile, e.Target))
		logAudit("WARN", "vpn.failover_started", profile, cf("cli.alert.failover_started", profile, e.Target))
	case "failover_failed":
		emitAlert("ERROR", profile, cf("cli.alert.failover_failed", profile, e.Target, err))
		logAudit("ERROR", "vpn.failover_failed", profile, cf("cli.alert.failover_failed", profile, e.Target, err))
	case "failover_activated":
		watchdogNotice("WARN", "vpn.failover_activated", "watchdog.event.failover", profile, e.Target)
	case "no_response":
		// Once per outage: interface up, server silent (no WireGuard
		// handshake). The watchdog reconnects it like a drop (ADR-0030).
		watchdogNotice("WARN", "vpn.no_response", "watchdog.event.no_response", profile)
	case "responding":
		watchdogNotice("INFO", "vpn.responding", "watchdog.event.responding", profile)
	case "manual_reconnect_required":
		// Once per drop: no automatic path (no CLI, or auto_connect off),
		// so the user has to reconnect in the provider's app.
		watchdogNotice("WARN", "vpn.manual_reconnect_required", "watchdog.event.manual_reconnect", profile)
		releaseDroppedDNS(profile)
	}
}

// gaveUpKey picks the give-up notice: plain, or with how to get traffic
// back while the kill switch keeps blocking for the profile (its
// destinations, or everything for a full tunnel). The block stays on
// purpose (security first); disconnecting the profile lifts it.
func gaveUpKey(profile string, ks security.KillSwitchStatus) string {
	if slices.Contains(ks.FullTunnel, profile) {
		return "watchdog.event.gave_up_blocking_full"
	}
	if slices.Contains(ks.Blocking, profile) {
		return "watchdog.event.gave_up_blocking"
	}
	return "watchdog.event.gave_up"
}

// offersRetry: the notices after which only the user connects the profile
// again, so clients offer "Reintentar".
func offersRetry(key string) bool {
	return strings.HasPrefix(key, "watchdog.event.gave_up") || key == "watchdog.event.reconnect_blocked"
}

// reconnectWaitsForUser classifies reconnect errors only the user can
// resolve (SetWaitForUser): another VPN already carries all traffic
// (ADR-0099), so retrying on a timer cannot help.
var reconnectWaitsForUser = func(err error) bool { return errors.Is(err, app.ErrFullTunnelConflict) }

// watchdogNotice records a watchdog notice for every client and shows it
// in the daemon's own output and audit log. A give-up notice offers to
// connect the profile again (CanConnect: "Reintentar" in the panel and
// the tray, `kongtrol up <profile>` in the CLI).
func watchdogNotice(level, action, key string, args ...string) {
	daemonEvents.Add(monitor.Event{Kind: monitor.EventTunnelWatchdog, Level: level,
		Key: key, Args: args, Profile: args[0], CanConnect: offersRetry(key)})
	fmtArgs := make([]any, len(args))
	for i, a := range args {
		fmtArgs[i] = a
	}
	msg := cf(key, fmtArgs...)
	emitAlert(level, args[0], msg)
	logAudit(level, action, args[0], msg)
}

// releaseDroppedDNS gives a tunnel's DNS servers back when it dropped on
// its own: they are only reachable through the dead tunnel, so keeping
// them makes every name fail, even for sites that never used that VPN
// (#37). A full tunnel the kill switch holds keeps them (no lookups
// outside the tunnel while everything else is blocked). The reconnect
// (ProfileService.Connect) applies them again (ADR-0058).
func releaseDroppedDNS(profile string) {
	if dnsMgr == nil || killSwitchSvc.HoldsAllTraffic(profile) {
		return
	}
	dnsMgr.Release(profile)
}

// onKillSwitchBlock tells every client, once per outage, that the kill
// switch started blocking for a dropped tunnel and what that means: only
// its destinations, or everything (a full tunnel) with what to do; with
// allow_lan off a full tunnel also cuts the local network, and says so.
func onKillSwitchBlock(profile string, fullTunnel bool) {
	watchdogNotice("WARN", "killswitch.blocking", killSwitchBlockKey(fullTunnel, killSwitchSvc.LANAllowed()), profile)
}

// killSwitchBlockKey picks the notice for onKillSwitchBlock.
func killSwitchBlockKey(fullTunnel, lanAllowed bool) string {
	switch {
	case !fullTunnel:
		return "killswitch.event.blocking"
	case lanAllowed:
		return "killswitch.event.blocking_full"
	default:
		return "killswitch.event.blocking_full_lan"
	}
}
