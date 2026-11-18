package main

import (
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// attachLine renders a provider adapter's attach state (how the tunnel came
// up, or what the user must do in the app) as one translated line in this
// CLI's language, or "" when there is nothing to say.
func attachLine(st *vpn.AttachState) string {
	if st == nil {
		return ""
	}
	key, args := st.MessageKey()
	if key == "" {
		return ""
	}
	return cf(key, args...)
}

// healthLine renders a tunnel's health (does its server answer, ADR-0030)
// as one translated line, or "" when there is nothing to say. "unknown"
// adds what to do: the helper guidance (ReasonKey) or a generic hint.
func healthLine(h *vpn.TunnelHealth) string {
	if h == nil {
		return ""
	}
	key := h.MessageKey()
	if key == "" {
		return ""
	}
	line := ct(key)
	if h.State == vpn.HealthUnknown {
		fix := h.ReasonKey
		if fix == "" {
			fix = "health.unknown.fix"
		}
		line += " · " + ct(fix)
	}
	return line
}

// attachSpinnerLabel is the `up` / `down` spinner text while a provider
// waits for the user: "<profile>  <message> (Ctrl+C ...)". It returns ""
// when st is not a waiting phase, so the caller keeps its own label.
// hintKey is attach.hint.cancel (up: Ctrl+C cancels the connect) or
// attach.hint.stop_waiting (down: the tunnel stays, the CLI stops waiting).
func attachSpinnerLabel(name string, st *vpn.AttachState, hintKey string) string {
	if st == nil || !st.Waiting() {
		return ""
	}
	return name + "  " + attachLine(st) + " " + ct(hintKey)
}

// attachDoneLine is the `up` result text for a provider tunnel that came
// up ("Conectado con NordVPN (United States)"), or "" for other adapters /
// phases so the caller prints its generic "connected in" line.
func attachDoneLine(st *vpn.AttachState) string {
	if st == nil {
		return ""
	}
	switch st.Phase {
	case vpn.PhaseAttached, vpn.PhaseAuto, vpn.PhaseUser:
		return attachLine(st)
	}
	return ""
}

// attachWaitingUser reports whether `up` should stop applying its own
// connect timeout: the adapter waits for the user and bounds that wait
// with the profile's wait_timeout itself.
func attachWaitingUser(st *vpn.AttachState) bool {
	return st != nil && st.Phase == vpn.PhaseWaitingUser
}

// renderAttachRowLines is renderAttachLines for the live view's rows.
func renderAttachRowLines(rows []tunnelRow) []string {
	tunnels := make([]monitor.TunnelMetrics, 0, len(rows))
	for _, r := range rows {
		if r.Attach != nil || r.Health != nil {
			tunnels = append(tunnels, monitor.TunnelMetrics{Name: r.Name, Status: r.Status, Attach: r.Attach, Health: r.Health})
		}
	}
	return renderAttachLines(tunnels)
}
