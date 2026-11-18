package main

import "github.com/vpn-kongtrol/kongtrol/internal/monitor"

// onWatchdogEvent turns watchdog events into alerts, audit entries and,
// for what the user must know about, a daemon notice (monitor.Event) that
// the tray, the dashboard and the live view render in their own language.
func onWatchdogEvent(e monitor.WatchdogEvent) {
	profile, attempt, err := e.Profile, e.Attempt, e.Err
	switch e.Event {
	case "reconnect_attempt":
		emitAlert("WARN", profile, cf("cli.alert.reconnect_attempt", profile, attempt))
		logAudit("WARN", "vpn.reconnect_attempt", profile, cf("cli.alert.reconnect_attempt", profile, attempt))
	case "reconnect_failed":
		emitAlert("ERROR", profile, cf("cli.alert.reconnect_failed", profile, attempt, err))
		logAudit("ERROR", "vpn.reconnect_failed", profile, cf("cli.alert.reconnect_failed", profile, attempt, err))
		if svc := profileSvc.Load(); svc != nil {
			svc.HandleReconnectError(profile, err)
		}
	case "reconnect_failing":
		// Once per outage: no backup declared (or none worked). Kongtrol
		// keeps retrying this same profile; it never picks another one.
		watchdogNotice("WARN", "vpn.reconnect_failing", "watchdog.event.reconnect_failing", profile)
	case "reconnected":
		if col != nil {
			col.RecordReconnect(profile)
		}
		emitAlert("INFO", profile, cf("cli.alert.reconnected", profile, attempt))
		logAudit("INFO", "vpn.reconnected", profile, cf("cli.alert.reconnected", profile, attempt))
		if attempt > 1 { // the user was told it was failing
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
		emitAlert("WARN", profile, cf("cli.alert.manual_reconnect_required", profile))
		logAudit("WARN", "vpn.manual_reconnect_required", profile, cf("cli.alert.manual_reconnect_required", profile))
	}
}

// watchdogNotice records a watchdog notice for every client and shows it
// in the daemon's own output and audit log.
func watchdogNotice(level, action, key string, args ...string) {
	daemonEvents.Add(monitor.Event{Kind: monitor.EventTunnelWatchdog, Level: level,
		Key: key, Args: args, Profile: args[0]})
	fmtArgs := make([]any, len(args))
	for i, a := range args {
		fmtArgs[i] = a
	}
	msg := cf(key, fmtArgs...)
	emitAlert(level, args[0], msg)
	logAudit(level, action, args[0], msg)
}
