package main

import (
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func TestTrayTunnelLabel(t *testing.T) {
	trayLang = i18n.EN
	cases := []struct {
		name      string
		m         monitor.TunnelMetrics
		wantTitle string
		wantTip   string
	}{
		{"waiting", monitor.TunnelMetrics{Status: vpn.StatusConnecting,
			Attach: &vpn.AttachState{Phase: vpn.PhaseWaitingUser, App: "Surfshark"}},
			"surf · Waiting for connection in Surfshark…", "Waiting for connection in Surfshark…"},
		{"dropped beats last error", monitor.TunnelMetrics{Status: vpn.StatusDisconnected, LastError: "boom",
			Attach: &vpn.AttachState{Phase: vpn.PhaseDropped, App: "Surfshark"}},
			"surf · Surfshark dropped · reconnect in the app", "Surfshark dropped · reconnect in the app"},
		{"error", monitor.TunnelMetrics{Status: vpn.StatusDisconnected, LastError: "boom"},
			"surf · failed to connect · open the panel", "boom"},
		{"error key", monitor.TunnelMetrics{Status: vpn.StatusDisconnected, LastError: "boom",
			LastErrorKey: "service.error.not_installed"},
			"surf · failed to connect · open the panel", "Install the helper once: kongtrol service install"},
		{"attached", monitor.TunnelMetrics{Status: vpn.StatusConnected,
			Attach: &vpn.AttachState{Phase: vpn.PhaseAttached, App: "Surfshark"}},
			"surf [connected]", "Surfshark already connected · kongtrol takes over"},
		{"plain", monitor.TunnelMetrics{Status: vpn.StatusConnected}, "surf [connected]", ""},
		{"no response", monitor.TunnelMetrics{Status: vpn.StatusError,
			Health: &vpn.TunnelHealth{State: vpn.HealthNoResponse}},
			"surf · no answer from the server",
			"No answer from the server · Kongtrol keeps retrying; if it persists, check your internet or tell whoever runs that VPN"},
		{"helper outdated", monitor.TunnelMetrics{Status: vpn.StatusConnected,
			Health: &vpn.TunnelHealth{State: vpn.HealthUnknown, ReasonKey: "service.error.outdated"}},
			"surf · answer not verified",
			"Can't check whether the server answers · The helper is out of date. Update it: kongtrol service install"},
		{"idle is not a problem", monitor.TunnelMetrics{Status: vpn.StatusConnected,
			Health: &vpn.TunnelHealth{State: vpn.HealthIdle}}, "surf [connected]", ""},
	}
	for _, c := range cases {
		title, tip := trayTunnelLabel("surf", c.m)
		if title != c.wantTitle || tip != c.wantTip {
			t.Errorf("%s: got (%q, %q), want (%q, %q)", c.name, title, tip, c.wantTitle, c.wantTip)
		}
	}
	states := map[string]monitor.TunnelMetrics{
		"a": {Status: vpn.StatusConnected},
		"b": {Status: vpn.StatusConnected, Attach: &vpn.AttachState{Phase: vpn.PhaseWaitingDisconnect, App: "Surfshark"}},
	}
	if tip := trayWaitingTooltip(states, []string{"a", "b"}); !strings.HasPrefix(tip, "b: Disconnect in Surfshark") {
		t.Errorf("tooltip = %q", tip)
	}
}
