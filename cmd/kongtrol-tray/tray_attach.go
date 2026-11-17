package main

import (
	"fmt"

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

// trayTunnelLabel decides one profile menu item's title and tooltip:
// a phase that needs the user (waiting / dropped) replaces the status in
// the title; a failed connect shows its error (the translated "what to do"
// line when the daemon sent its key); otherwise "name [status]" with the
// attach message (how it came up) as tooltip.
func trayTunnelLabel(name string, m monitor.TunnelMetrics) (title, tooltip string) {
	msg := trayAttachLine(m.Attach)
	switch {
	case m.Attach != nil && m.Attach.Waiting() && msg != "":
		return name + " · " + msg, msg
	case (m.LastError != "" || m.LastErrorKey != "") && m.Status != vpn.StatusConnected:
		if m.LastErrorKey != "" {
			return trayF("tray.tunnel_error", name), trayT(m.LastErrorKey)
		}
		return trayF("tray.tunnel_error", name), m.LastError
	default:
		return fmt.Sprintf("%s [%s]", name, m.Status), msg
	}
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
	}
	return ""
}
