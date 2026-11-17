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
