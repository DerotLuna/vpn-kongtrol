package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// attachPhases covers every phase a provider adapter reports.
var attachPhases = []vpn.AttachState{
	{Phase: vpn.PhaseAttached, App: "NordVPN"},
	{Phase: vpn.PhaseAuto, App: "NordVPN", Location: "United States"},
	{Phase: vpn.PhaseAuto, App: "NordVPN"},
	{Phase: vpn.PhaseUser, App: "Surfshark"},
	{Phase: vpn.PhaseWaitingUser, App: "Surfshark"},
	{Phase: vpn.PhaseWaitingUser, App: "FortiClient", Reason: "gui layout changed"},
	{Phase: vpn.PhaseWaitingDisconnect, App: "Surfshark"},
	{Phase: vpn.PhaseDropped, App: "Surfshark"},
}

// TestAttachMessagesTranslated checks every phase renders as one short line
// in both languages, naming the app, with no missing key or arg.
func TestAttachMessagesTranslated(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		for _, st := range attachPhases {
			key, args := st.MessageKey()
			msg := i18n.F(lang, key, args...)
			if msg == key || strings.Contains(msg, "%!") || strings.Contains(msg, "\n") {
				t.Errorf("%s %s: bad message %q", lang, st.Phase, msg)
			}
			if !strings.Contains(msg, st.App) {
				t.Errorf("%s %s: %q does not name %s", lang, st.Phase, msg, st.App)
			}
			if len([]rune(msg)) > 80 {
				t.Errorf("%s %s: message too long (%d): %q", lang, st.Phase, len([]rune(msg)), msg)
			}
		}
		for _, k := range []string{"attach.hint.cancel", "attach.hint.stop_waiting", "cli.attach.wait_timeout", "cli.attach.disconnect_timeout", "cli.alert.manual_reconnect_required"} {
			if i18n.T(lang, k) == k {
				t.Errorf("%s: missing key %s", lang, k)
			}
		}
	}
}

func TestAttachRenderingDecisions(t *testing.T) {
	t.Setenv("KONGTROL_LANG", "en")
	for _, st := range attachPhases {
		st := st
		waiting := st.Waiting()
		if got := attachSpinnerLabel("p", &st, "attach.hint.cancel"); (got != "") != waiting {
			t.Errorf("%s: spinner label %q, waiting=%v", st.Phase, got, waiting)
		}
		done := st.Phase == vpn.PhaseAttached || st.Phase == vpn.PhaseAuto || st.Phase == vpn.PhaseUser
		if got := attachDoneLine(&st); (got != "") != done {
			t.Errorf("%s: done line %q, want set=%v", st.Phase, got, done)
		}
		if got := attachWaitingUser(&st); got != (st.Phase == vpn.PhaseWaitingUser) {
			t.Errorf("%s: attachWaitingUser = %v", st.Phase, got)
		}
	}
	if attachLine(nil) != "" || attachSpinnerLabel("p", nil, "attach.hint.cancel") != "" || attachDoneLine(nil) != "" {
		t.Error("nil attach state must render nothing")
	}
	label := attachSpinnerLabel("surf", &vpn.AttachState{Phase: vpn.PhaseWaitingUser, App: "Surfshark"}, "attach.hint.cancel")
	if label != "surf  Waiting for connection in Surfshark… (Ctrl+C cancels)" {
		t.Errorf("spinner label = %q", label)
	}
	lines := renderAttachLines([]monitor.TunnelMetrics{
		{Name: "a"},
		{Name: "b", Attach: &vpn.AttachState{Phase: vpn.PhaseDropped, App: "Surfshark"}},
	})
	if len(lines) != 1 || !strings.Contains(lines[0], "Surfshark dropped") {
		t.Errorf("renderAttachLines = %q", lines)
	}
}

// Issue #33: the status table's health line under a WireGuard tunnel.
func TestHealthLine(t *testing.T) {
	t.Setenv("KONGTROL_LANG", "es")
	cases := []struct {
		h    *vpn.TunnelHealth
		want string
	}{
		{nil, ""},
		{&vpn.TunnelHealth{State: vpn.HealthOK}, ""},
		{&vpn.TunnelHealth{State: vpn.HealthIdle}, "Sin tráfico reciente · normal si no la estás usando"},
		{&vpn.TunnelHealth{State: vpn.HealthNoResponse}, "Sin respuesta del servidor · Kongtrol reintenta solo; si sigue, revisa tu internet o avisa a quien administra esa VPN"},
		{&vpn.TunnelHealth{State: vpn.HealthUnknown, ReasonKey: "service.error.outdated"},
			"No se puede comprobar si el servidor responde · El ayudante está desactualizado. Actualízalo: kongtrol service install"},
		{&vpn.TunnelHealth{State: vpn.HealthUnknown},
			"No se puede comprobar si el servidor responde · si algo no carga, desconéctala y vuelve a conectarla"},
	}
	for _, c := range cases {
		if got := healthLine(c.h); got != c.want {
			t.Errorf("healthLine(%+v) = %q, want %q", c.h, got, c.want)
		}
	}
	lines := renderAttachLines([]monitor.TunnelMetrics{{Name: "brainbox", Status: vpn.StatusError,
		Health: &vpn.TunnelHealth{State: vpn.HealthNoResponse}}})
	if len(lines) != 1 || !strings.Contains(lines[0], "brainbox") || !strings.Contains(lines[0], "Sin respuesta del servidor") {
		t.Errorf("renderAttachLines = %q", lines)
	}
}

func TestWaitManualDown(t *testing.T) {
	old := downPollInterval
	downPollInterval = 5 * time.Millisecond
	defer func() { downPollInterval = old }()

	waiting := &vpn.AttachState{Phase: vpn.PhaseWaitingDisconnect, App: "Surfshark"}
	reads := 0
	read := func(string) (monitor.TunnelMetrics, bool) {
		reads++
		if reads < 3 {
			return monitor.TunnelMetrics{Status: vpn.StatusConnected, Attach: waiting}, true
		}
		return monitor.TunnelMetrics{Status: vpn.StatusDisconnected}, true
	}
	if err := waitManualDown(context.Background(), "surf", read, nil); err != nil {
		t.Fatalf("user disconnected: err = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stillUp := func(string) (monitor.TunnelMetrics, bool) {
		return monitor.TunnelMetrics{Status: vpn.StatusConnected, Attach: waiting}, true
	}
	if err := waitManualDown(ctx, "surf", stillUp, nil); !errors.Is(err, vpn.ErrManualDisconnect) {
		t.Fatalf("stopped waiting: err = %v, want ErrManualDisconnect", err)
	}
}
