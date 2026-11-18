package main

import (
	"net/url"
	"strings"
	"testing"
)

func TestFlyoutURL(t *testing.T) {
	ok := map[string]string{
		"http://127.0.0.1:9741":  "http://127.0.0.1:9741/app/tray",
		"http://[::1]:9741":      "http://[::1]:9741/app/tray",
		"http://localhost:8080/": "http://localhost:8080/app/tray",
	}
	for base, want := range ok {
		got, err := flyoutURL(base)
		if err != nil || got != want {
			t.Errorf("flyoutURL(%q) = %q, %v; want %q", base, got, err, want)
		}
	}
	for _, base := range []string{
		"https://127.0.0.1:9741",     // only plain http on loopback
		"http://192.168.1.10:9741",   // LAN bind: never in the flyout
		"http://kongtrol.app:80",     // remote
		"http://127.0.0.1",           // no port
		"http://user:pw@127.0.0.1:1", // credentials
		"file:///C:/x",
		"::bad",
	} {
		if got, err := flyoutURL(base); err == nil {
			t.Errorf("flyoutURL(%q) = %q, want error", base, got)
		}
	}
}

func TestFlyoutRequestAllowed(t *testing.T) {
	page := "http://127.0.0.1:9741/app/tray"
	allowed := []string{
		page,
		"http://127.0.0.1:9741/app/assets/index-abc.js",
		"http://127.0.0.1:9741/brand/fonts/inter.woff2",
		"http://127.0.0.1:9741/api/v1/tunnels",
		"ws://127.0.0.1:9741/api/v1/ws/metrics",
	}
	for _, u := range allowed {
		if !flyoutRequestAllowed(page, u) {
			t.Errorf("blocked %q", u)
		}
	}
	blocked := []string{
		"https://kongtrol.app/guia",
		"http://127.0.0.1:9742/app/tray", // another port = another program
		"http://localhost:9741/app/tray", // another origin for the cookie
		"http://evil.example/",
		"https://127.0.0.1:9741/app/tray",
		"http://user@127.0.0.1:9741/app/tray",
		"data:text/html,hi",
		"javascript:alert(1)",
		"file:///C:/Windows/win.ini",
	}
	for _, u := range blocked {
		if flyoutRequestAllowed(page, u) {
			t.Errorf("allowed %q", u)
		}
	}
}

// The address the window loads must stay on the flyout page and origin, so
// a bootstrap fragment (#k=<nonce>) added in flyoutLoadURL keeps working
// with the filter and the injected script; no token in path or query.
func TestFlyoutLoadURL(t *testing.T) {
	page := "http://127.0.0.1:9741/app/tray"
	got, err := flyoutLoadURL(page)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(got)
	if err != nil || u.Path != flyoutPath || u.RawQuery != "" || !flyoutRequestAllowed(page, got) {
		t.Errorf("flyoutLoadURL(%q) = %q", page, got)
	}
	if !flyoutRequestAllowed(page, page+"#k=abc") {
		t.Error("a bootstrap fragment must pass the filter")
	}
}

func TestFlyoutInitScript(t *testing.T) {
	s, err := flyoutInitScript("http://127.0.0.1:9741/app/tray", "n0nce")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`location.origin !== "http://127.0.0.1:9741"`,
		`location.pathname !== "/app/tray"`,
		`w.postMessage("kongtrol:off-page")`,
		`var nonce = "n0nce"`,
		`window.open = function () { return null; }`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("init script lacks %q:\n%s", want, s)
		}
	}
	// The nonce must only be reachable on the flyout page: it appears after
	// the off-page early return.
	if strings.Index(s, "n0nce") < strings.Index(s, "return;") {
		t.Error("nonce defined before the off-page check")
	}
	if _, err := flyoutInitScript("::bad", "n"); err == nil {
		t.Error("bad URL accepted")
	}
}

func TestParseFlyoutMessage(t *testing.T) {
	const nonce = "abc123"
	cases := map[string]flyoutAction{
		"abc123 ready":          flyoutReady,
		"abc123 close":          flyoutClose,
		"abc123 open-dashboard": flyoutOpenDashboard,
		"abc123 quit-tray":      flyoutQuitTray,
		"abc123 quit-all":       flyoutQuitAll,
		"abc123 rm -rf":         flyoutNone, // closed set
		"quit-all":              flyoutNone, // no nonce
		"wrong quit-all":        flyoutNone,
		"abc12 quit-all":        flyoutNone,
		"kongtrol:off-page":     flyoutNone,
		"":                      flyoutNone,
	}
	for msg, want := range cases {
		if got := parseFlyoutMessage(msg, nonce); got != want {
			t.Errorf("parseFlyoutMessage(%q) = %v, want %v", msg, got, want)
		}
	}
	if got := parseFlyoutMessage(" quit-all", ""); got != flyoutNone {
		t.Error("empty nonce must accept nothing")
	}
}

func TestTrayLeftClick(t *testing.T) {
	cases := []struct {
		goos    string
		webview bool
		flyout  bool
	}{
		{"windows", true, true},
		{"windows", false, false},
		{"darwin", true, false},
		{"linux", true, false},
	}
	for _, c := range cases {
		got, reason := trayLeftClick(c.goos, c.webview)
		if got != c.flyout || (!got && reason == "") {
			t.Errorf("trayLeftClick(%s, %v) = %v, %q", c.goos, c.webview, got, reason)
		}
	}
}

func TestPlaceFlyout(t *testing.T) {
	monitor := rect{0, 0, 1920, 1080}
	const w, h, gap = 360, 520, 12
	cases := []struct {
		name         string
		icon, work   rect
		monitor      rect
		wantX, wantY int
	}{
		{
			name: "bottom taskbar, icon near the right corner: clamped inside",
			icon: rect{1800, 1040, 1824, 1064}, work: rect{0, 0, 1920, 1032}, monitor: monitor,
			wantX: 1920 - gap - w, wantY: 1032 - gap - h,
		},
		{
			name: "bottom taskbar, icon mid-screen: centered on the icon",
			icon: rect{900, 1040, 924, 1064}, work: rect{0, 0, 1920, 1032}, monitor: monitor,
			wantX: 912 - w/2, wantY: 1032 - gap - h,
		},
		{
			name: "top taskbar",
			icon: rect{1800, 10, 1824, 34}, work: rect{0, 48, 1920, 1080}, monitor: monitor,
			wantX: 1920 - gap - w, wantY: 48 + gap,
		},
		{
			name: "left taskbar",
			icon: rect{10, 1000, 34, 1024}, work: rect{48, 0, 1920, 1080}, monitor: monitor,
			wantX: 48 + gap, wantY: 1080 - gap - h,
		},
		{
			name: "right taskbar, icon mid-height",
			icon: rect{1880, 500, 1904, 524}, work: rect{0, 0, 1872, 1080}, monitor: monitor,
			wantX: 1872 - gap - w, wantY: 512 - h/2,
		},
		{
			name: "auto-hide taskbar at the bottom: edge from the icon",
			icon: rect{1800, 1070, 1801, 1071}, work: monitor, monitor: monitor,
			wantX: 1920 - gap - w, wantY: 1080 - gap - h,
		},
		{
			name: "second monitor left of the primary (negative coordinates)",
			icon: rect{-900, 1040, -876, 1064}, work: rect{-1920, 0, 0, 1032}, monitor: rect{-1920, 0, 0, 1080},
			wantX: -888 - w/2, wantY: 1032 - gap - h,
		},
		{
			name: "work area smaller than the flyout: pinned to its top-left",
			icon: rect{100, 300, 124, 324}, work: rect{0, 0, 300, 400}, monitor: rect{0, 0, 300, 440},
			wantX: gap, wantY: gap,
		},
	}
	for _, c := range cases {
		x, y := placeFlyout(c.icon, c.work, c.monitor, w, h, gap)
		if x != c.wantX || y != c.wantY {
			t.Errorf("%s: got (%d,%d), want (%d,%d)", c.name, x, y, c.wantX, c.wantY)
		}
	}
}

func TestScaleDIP(t *testing.T) {
	for dpi, want := range map[uint32]int{0: 360, 96: 360, 120: 450, 144: 540, 192: 720, 168: 630} {
		if got := scaleDIP(360, dpi); got != want {
			t.Errorf("scaleDIP(360, %d) = %d, want %d", dpi, got, want)
		}
	}
}

func TestRunFlyoutActionClose(t *testing.T) {
	hidden := 0
	runFlyoutAction(flyoutClose, func() { hidden++ })
	runFlyoutAction(flyoutNone, func() { hidden++ })
	runFlyoutAction(flyoutReady, func() { hidden++ })
	if hidden != 1 {
		t.Errorf("hide called %d times, want 1 (only for close)", hidden)
	}
}
