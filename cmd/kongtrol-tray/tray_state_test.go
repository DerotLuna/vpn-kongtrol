package main

import (
	"errors"
	"net/url"
	"reflect"
	"syscall"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/assets"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

func TestTrayTunnelViews_ConnectedFirst(t *testing.T) {
	trayLang = i18n.ES
	states := map[string]monitor.TunnelMetrics{
		"brainbox":      {Status: vpn.StatusConnected},
		"ekiipago":      {Status: vpn.StatusDisconnected},
		"ekiipago-ovpn": {Status: vpn.StatusConnected},
		"us-content":    {Status: vpn.StatusConnecting},
		// rluna-ovpn: not reported by the daemon → disconnected.
	}
	views := trayTunnelViews([]string{"brainbox", "ekiipago", "ekiipago-ovpn", "rluna-ovpn", "us-content"}, states)
	var got []string
	for _, v := range views {
		got = append(got, v.Title)
	}
	want := []string{
		"brainbox · conectada",
		"ekiipago-ovpn · conectada",
		"ekiipago · desconectada",
		"rluna-ovpn · desconectada",
		"us-content · conectando…",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("menu order:\n got %q\nwant %q", got, want)
	}
}

func TestGroupView(t *testing.T) {
	states := map[string]monitor.TunnelMetrics{"a": {Status: vpn.StatusConnected}, "b": {}}
	if title, checked := groupView("work", []string{"a", "b"}, states); title != "work (1/2)" || checked {
		t.Fatalf("got %q %v", title, checked)
	}
	if _, checked := groupView("work", []string{"a"}, states); !checked {
		t.Fatal("all connected must be checked")
	}
	if _, checked := groupView("empty", nil, states); checked {
		t.Fatal("an empty group is never checked")
	}
}

func TestClassifyDaemon(t *testing.T) {
	refused := &url.Error{Op: "Get", URL: "http://127.0.0.1:7777/api/v1/tunnels", Err: syscall.ECONNREFUSED}
	timeout := &url.Error{Op: "Get", URL: "x", Err: timeoutErr{}}
	cases := []struct {
		name     string
		err      error
		starting bool
		want     daemonState
	}{
		{"answers", nil, false, daemonRunning},
		{"refused = stopped (kongtrol down)", refused, false, daemonStopped},
		{"refused while starting", refused, true, daemonStarting},
		{"timeout = not responding", timeout, false, daemonUnresponsive},
		{"HTTP error from a listening daemon", errors.New("500 internal"), false, daemonUnresponsive},
	}
	for _, c := range cases {
		if got := classifyDaemon(c.err, c.starting); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

// TestTrayHeaderTransitions walks the daemon lifecycle the tray must
// follow without restarting: running → `kongtrol down` (stopped, start item)
// → "Iniciar Kongtrol" (starting) → running again.
func TestTrayHeaderTransitions(t *testing.T) {
	up := []tunnelView{{Checked: true, Kind: kindUp}, {Kind: kindDown}}
	steps := []struct {
		d       daemonState
		views   []tunnelView
		wantES  string
		wantEN  string
		badge   assets.TrayBadge
		startOn bool
	}{
		{daemonRunning, up, "1 VPN conectada", "1 VPN connected", assets.BadgeLive, false},
		{daemonStopped, nil, "Kongtrol detenido", "Kongtrol stopped", assets.BadgeStopped, true},
		{daemonStarting, nil, "Iniciando Kongtrol…", "Starting Kongtrol…", assets.BadgeNone, false},
		{daemonRunning, []tunnelView{{Checked: true, Kind: kindUp}, {Checked: true, Kind: kindUp}}, "2 VPN conectadas", "2 VPNs connected", assets.BadgeLive, false},
		{daemonRunning, []tunnelView{{Kind: kindDown}}, "Ninguna VPN conectada", "No VPN connected", assets.BadgeNone, false},
		{daemonRunning, []tunnelView{{Kind: kindError}, {Kind: kindCaution}}, "Ninguna VPN conectada", "No VPN connected", assets.BadgeError, false},
		{daemonRunning, []tunnelView{{Checked: true, Kind: kindCaution}}, "1 VPN conectada", "1 VPN connected", assets.BadgeCaution, false},
		{daemonUnresponsive, nil, "Kongtrol no responde · espera o abre el panel", "Kongtrol is not responding · wait or open the panel", assets.BadgeCaution, false},
	}
	for i, s := range steps {
		trayLang = i18n.ES
		es, badge := trayHeader(s.d, s.views)
		trayLang = i18n.EN
		en, _ := trayHeader(s.d, s.views)
		if es != s.wantES || en != s.wantEN || badge != s.badge {
			t.Errorf("step %d: got %q / %q / badge %d, want %q / %q / %d", i, es, en, badge, s.wantES, s.wantEN, s.badge)
		}
		// The start item is shown exactly in the stopped state (renderDaemon).
		if (s.d == daemonStopped) != s.startOn {
			t.Errorf("step %d: start item visibility mismatch", i)
		}
	}
}

func TestTrayVersionLabel(t *testing.T) {
	trayLang = i18n.ES
	cases := []struct{ daemon, tray, want string }{
		{"v0.6.0", "v0.5.0", "Kongtrol v0.6.0"},
		{"0.6.0-rc.3", "", "Kongtrol v0.6.0-rc.3"}, // goreleaser drops the v
		{"", "v0.6.0", "Kongtrol v0.6.0"},          // daemon stopped
		{"", "", ""},
	}
	for _, c := range cases {
		if got := trayVersionLabel(c.daemon, c.tray); got != c.want {
			t.Errorf("(%q,%q) = %q, want %q", c.daemon, c.tray, got, c.want)
		}
	}
}

func TestStateWordsTranslated(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		trayLang = lang
		for _, s := range []vpn.Status{vpn.StatusConnected, vpn.StatusConnecting, vpn.StatusDisconnected, vpn.StatusError, ""} {
			if w := trayStateWord(s); w == "" || w[:5] == "tray." {
				t.Errorf("%v %q: untranslated %q", lang, s, w)
			}
		}
	}
}
