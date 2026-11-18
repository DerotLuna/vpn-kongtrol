package main

import (
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// trayAttachLine renders a provider adapter's attach phase in the tray's
// language ("Esperando conexión en Surfshark…"), or "".
func trayAttachLine(st *vpn.AttachState) string {
	if st == nil {
		return ""
	}
	key, args := st.MessageKey()
	if key == "" {
		return ""
	}
	return trayF(key, args...)
}

// trayHealthLine renders a tunnel's health problem (server not answering,
// or not verifiable, ADR-0030) with its "what to do", or "".
func trayHealthLine(h *vpn.TunnelHealth) string {
	if h == nil || !h.Problem() {
		return ""
	}
	line := trayT(h.MessageKey())
	if h.State == vpn.HealthUnknown {
		fix := h.ReasonKey
		if fix == "" {
			fix = "health.unknown.fix"
		}
		line += " · " + trayT(fix)
	}
	return line
}

// trayWaitingTooltip returns the first message of a tunnel that needs the
// user, for the tray icon tooltip, or "" when none does.
func trayWaitingTooltip(states map[string]monitor.TunnelMetrics, names []string) string {
	for _, name := range names {
		m, ok := states[name]
		if ok && m.Attach != nil && m.Attach.Waiting() {
			if msg := trayAttachLine(m.Attach); msg != "" {
				return name + ": " + msg
			}
		}
		if ok {
			if msg := trayHealthLine(m.Health); msg != "" {
				return name + ": " + msg
			}
		}
	}
	return ""
}
