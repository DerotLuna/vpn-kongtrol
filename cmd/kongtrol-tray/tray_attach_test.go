package main

import (
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func TestTrayTunnelView(t *testing.T) {
	cases := []struct {
		name      string
		m         monitor.TunnelMetrics
		wantES    string
		wantEN    string
		wantTipEN string
		kind      stateKind
		checked   bool
	}{
		{"connected", monitor.TunnelMetrics{Status: vpn.StatusConnected},
			"surf · conectada", "surf · connected", "", kindUp, true},
		{"disconnected", monitor.TunnelMetrics{Status: vpn.StatusDisconnected},
			"surf · desconectada", "surf · disconnected", "", kindDown, false},
		{"zero status is disconnected", monitor.TunnelMetrics{},
			"surf · desconectada", "surf · disconnected", "", kindDown, false},
		{"connecting", monitor.TunnelMetrics{Status: vpn.StatusConnecting},
			"surf · conectando…", "surf · connecting…", "", kindConnecting, false},
		{"status error", monitor.TunnelMetrics{Status: vpn.StatusError},
			"surf · con error", "surf · error", "", kindError, false},
		{"waiting", monitor.TunnelMetrics{Status: vpn.StatusConnecting,
			Attach: &vpn.AttachState{Phase: vpn.PhaseWaitingUser, App: "Surfshark"}},
			"surf · Esperando conexión en Surfshark…", "surf · Waiting for connection in Surfshark…",
			"Waiting for connection in Surfshark…", kindConnecting, false},
		{"dropped beats last error", monitor.TunnelMetrics{Status: vpn.StatusDisconnected, LastError: "boom",
			Attach: &vpn.AttachState{Phase: vpn.PhaseDropped, App: "Surfshark"}},
			"surf · Surfshark se cayó · reconéctate en la app", "surf · Surfshark dropped · reconnect in the app",
			"Surfshark dropped · reconnect in the app", kindCaution, false},
		{"error", monitor.TunnelMetrics{Status: vpn.StatusDisconnected, LastError: "boom"},
			"surf · no conectó · abre el panel", "surf · failed to connect · open the panel", "boom", kindError, false},
		{"error key", monitor.TunnelMetrics{Status: vpn.StatusDisconnected, LastError: "boom",
			LastErrorKey: "service.error.not_installed"},
			"surf · no conectó · abre el panel", "surf · failed to connect · open the panel",
			"Install the helper once: kongtrol service install", kindError, false},
		{"attached", monitor.TunnelMetrics{Status: vpn.StatusConnected,
			Attach: &vpn.AttachState{Phase: vpn.PhaseAttached, App: "Surfshark"}},
			"surf · conectada", "surf · connected", "Surfshark already connected · kongtrol takes over", kindUp, true},
		{"no response", monitor.TunnelMetrics{Status: vpn.StatusConnected,
			Health: &vpn.TunnelHealth{State: vpn.HealthNoResponse}},
			"surf · sin respuesta", "surf · no answer",
			"No answer from the server · Kongtrol keeps retrying; if it persists, check your internet or tell whoever runs that VPN",
			kindCaution, true},
		{"helper outdated", monitor.TunnelMetrics{Status: vpn.StatusConnected,
			Health: &vpn.TunnelHealth{State: vpn.HealthUnknown, ReasonKey: "service.error.outdated"}},
			"surf · conectada · sin verificar", "surf · connected · not verified",
			"Can't check whether the server answers · The helper is out of date. Update it: kongtrol service install",
			kindUp, true},
		{"idle is not a problem", monitor.TunnelMetrics{Status: vpn.StatusConnected,
			Health: &vpn.TunnelHealth{State: vpn.HealthIdle}}, "surf · conectada", "surf · connected", "", kindUp, true},
	}
	for _, c := range cases {
		trayLang = i18n.ES
		es := trayTunnelView("surf", c.m)
		trayLang = i18n.EN
		en := trayTunnelView("surf", c.m)
		if es.Title != c.wantES || en.Title != c.wantEN || en.Tooltip != c.wantTipEN {
			t.Errorf("%s: got ES %q EN %q tip %q, want %q / %q / %q", c.name, es.Title, en.Title, en.Tooltip, c.wantES, c.wantEN, c.wantTipEN)
		}
		if en.Kind != c.kind || en.Checked != c.checked {
			t.Errorf("%s: kind/checked = %v/%v, want %v/%v", c.name, en.Kind, en.Checked, c.kind, c.checked)
		}
		// Never a raw English status in brackets (the rc.3 bug).
		if strings.Contains(es.Title, "[") || strings.Contains(en.Title, "[") {
			t.Errorf("%s: raw status in %q / %q", c.name, es.Title, en.Title)
		}
	}
	states := map[string]monitor.TunnelMetrics{
		"a": {Status: vpn.StatusConnected},
		"b": {Status: vpn.StatusConnected, Attach: &vpn.AttachState{Phase: vpn.PhaseWaitingDisconnect, App: "Surfshark"}},
	}
	trayLang = i18n.EN
	if tip := trayWaitingTooltip(states, []string{"a", "b"}); !strings.HasPrefix(tip, "b: Disconnect in Surfshark") {
		t.Errorf("tooltip = %q", tip)
	}
}
