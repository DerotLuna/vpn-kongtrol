package main

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestFlyoutURL(t *testing.T) {
	ok := map[string]string{
		"http://127.0.0.1:9741":  "http://127.0.0.1:9741/tray",
		"http://[::1]:9741":      "http://[::1]:9741/tray",
		"http://localhost:8080/": "http://localhost:8080/tray",
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
	page := "http://127.0.0.1:9741/tray"
	allowed := []string{
		page,
		"http://127.0.0.1:9741/assets/index-abc.js",
		"http://127.0.0.1:9741/assets/inter-abc.woff2",
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
		"http://127.0.0.1:9742/tray", // another port = another program
		"http://localhost:9741/tray", // another origin for the cookie
		"http://evil.example/",
		"https://127.0.0.1:9741/tray",
		"http://user@127.0.0.1:9741/tray",
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
	page := "http://127.0.0.1:9741/tray"
	old := flyoutPanelURL
	t.Cleanup(func() { flyoutPanelURL = old })
	flyoutPanelURL = func(base, path string) (string, error) { return base + path + "#k=n0nce", nil }
	got, err := flyoutLoadURL(page)
	if err != nil {
		t.Fatal(err)
	}
	if got != page+"#k=n0nce" {
		t.Errorf("flyoutLoadURL = %q, want the page with the sign-in nonce", got)
	}
	// A PanelURL answer that leaves the daemon's page is never loaded.
	flyoutPanelURL = func(string, string) (string, error) { return "http://evil.example/tray#k=x", nil }
	if got, _ := flyoutLoadURL(page); got != page {
		t.Errorf("foreign URL used: %q", got)
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
	s, err := flyoutInitScript("http://127.0.0.1:9741/tray", "n0nce")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"origin":"http://127.0.0.1:9741"`,
		`"path":"/tray"`,
		`"offPage":"kongtrol:off-page"`,
		`"nonce":"n0nce"`,
		`location.origin !== cfg.origin || location.pathname !== cfg.path`,
		`w.postMessage(cfg.offPage)`,
		`window.open = function () { return null; }`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("init script lacks %q:\n%s", want, s)
		}
	}
	// The nonce is only handed to the page (kongtrolTray.post) after the
	// off-page early return; before it, it is a closure local only.
	if strings.Index(s, "var nonce = cfg.nonce") < strings.Index(s, "return;") {
		t.Error("nonce used before the off-page check")
	}
	if _, err := flyoutInitScript("::bad", "n"); err == nil {
		t.Error("bad URL accepted")
	}
}

// CodeQL go/unsafe-quoting: hostile values stay data. The script is the
// fixed code around exactly one JSON object holding them.
func TestFlyoutInitScriptQuoting(t *testing.T) {
	nonce := "a\"b'c</script><script>alert(1)</script>\u2028\u2029\\`${x}"
	page := "http://127.0.0.1:9741/tr%22ay%3C/script%3E"
	s, err := flyoutInitScript(page, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s, flyoutScriptHead) || !strings.HasSuffix(s, flyoutScriptBody) {
		t.Fatalf("script is not head + JSON + body:\n%s", s)
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(s, flyoutScriptHead), flyoutScriptBody)
	var cfg struct{ Origin, Path, Nonce, OffPage string }
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("config is not one JSON value: %v\n%s", err, raw)
	}
	if cfg.Nonce != nonce || cfg.Path != "/tr%22ay%3C/script%3E" || cfg.Origin != "http://127.0.0.1:9741" {
		t.Fatalf("round trip: %+v", cfg)
	}
	for _, bad := range []string{"</script>", "\u2028", "\u2029", `a"b`} {
		if strings.Contains(raw, bad) {
			t.Errorf("unescaped %q in the embedded JSON", bad)
		}
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
		"abc123 open-add":       flyoutOpenAdd,
		"abc123 reauth":         flyoutReauth,
		"reauth":                flyoutNone, // no nonce
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

func TestFlyoutWatch(t *testing.T) {
	var w flyoutWatch
	first := w.show()
	if !w.unanswered(first) {
		t.Fatal("a fresh show must wait for the page")
	}
	w.ack()
	if w.unanswered(first) {
		t.Fatal("ready answered the show")
	}
	second := w.show()
	if !w.unanswered(second) {
		t.Fatal("an older ready must not answer a new show")
	}
	third := w.show()
	if w.unanswered(second) {
		t.Error("a superseded show must not trigger the fallback")
	}
	if !w.unanswered(third) {
		t.Error("the latest show still waits")
	}
	w.ack()
	if w.unanswered(third) {
		t.Error("ready answers the latest show")
	}
}

func TestFlyoutNeedsDaemonCheck(t *testing.T) {
	t.Cleanup(func() { daemonAnsweredAt.Store(0) })
	now := time.Now()

	daemonAnsweredAt.Store(0)
	if !flyoutNeedsDaemonCheck(true, now) {
		t.Error("no poll ever answered: probe the daemon")
	}
	daemonAnsweredAt.Store(now.Add(-time.Second).UnixNano())
	if flyoutNeedsDaemonCheck(true, now) {
		t.Error("loaded page and a fresh poll: show at once")
	}
	if !flyoutNeedsDaemonCheck(false, now) {
		t.Error("page not loaded yet: the load needs the daemon")
	}
	daemonAnsweredAt.Store(now.Add(-flyoutDaemonFresh - time.Second).UnixNano())
	if !flyoutNeedsDaemonCheck(true, now) {
		t.Error("stale poll (daemon may be stopped): probe it")
	}
	daemonAnsweredAt.Store(now.Add(time.Minute).UnixNano())
	if !flyoutNeedsDaemonCheck(true, now) {
		t.Error("a poll time in the future (clock change) is not fresh")
	}
}

func TestFlyoutReauthAllowed(t *testing.T) {
	t0 := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	if !flyoutReauthAllowed(time.Time{}, t0) {
		t.Error("the first reauth is allowed")
	}
	if flyoutReauthAllowed(t0, t0.Add(flyoutReauthEvery-time.Millisecond)) {
		t.Error("a second reauth inside the window must wait")
	}
	if !flyoutReauthAllowed(t0, t0.Add(flyoutReauthEvery)) {
		t.Error("a reauth after the window is allowed")
	}
	if !flyoutReauthAllowed(t0, t0.Add(-time.Hour)) {
		t.Error("a clock that went back must not block reauth forever")
	}
}

func TestRunFlyoutActionReauthIsNotGeneric(t *testing.T) {
	// reauth is handled by the window (reload with a fresh nonce), never by
	// runFlyoutAction: it must not hide the flyout.
	hidden := 0
	runFlyoutAction(flyoutReauth, func() { hidden++ })
	if hidden != 0 {
		t.Error("reauth must not hide the flyout")
	}
}

func TestTokenTag(t *testing.T) {
	if tokenTag("") != "" {
		t.Error("empty token must give an empty tag")
	}
	a, b := tokenTag("token-a"), tokenTag("token-b")
	if a == "" || a == b || a != tokenTag("token-a") {
		t.Errorf("tags must be stable and differ per token: %q %q", a, b)
	}
	if strings.Contains(a, "token") {
		t.Error("the tag must not contain the token")
	}
}

func TestFlyoutOnShow(t *testing.T) {
	cases := []struct {
		name          string
		loaded        bool
		loadedTag     string
		currentTag    string
		sinceNavigate time.Duration
		want          flyoutShowStep
	}{
		{"warm, same daemon", true, "a", "a", time.Hour, showRefresh},
		{"warm, daemon restarted", true, "a", "b", time.Hour, showNavigate},
		{"warm, token unreadable now", true, "a", "", time.Hour, showRefresh},
		{"warm, no token at load", true, "", "b", time.Hour, showRefresh},
		{"still loading", false, "a", "a", time.Second, showWait},
		{"load failed", false, "a", "a", flyoutRetryLoad + time.Millisecond, showNavigate},
	}
	for _, c := range cases {
		if got := flyoutOnShow(c.loaded, c.loadedTag, c.currentTag, c.sinceNavigate); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestFlyoutOpenLogLine(t *testing.T) {
	t0 := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	cold := flyoutOpen{Click: t0, Shown: t0.Add(900 * time.Millisecond), Ready: t0.Add(1200 * time.Millisecond)}
	line, ok := cold.logLine()
	if !ok {
		t.Error("cold opens are always logged")
	}
	for _, want := range []string{"(cold)", "click to shown 900ms", "shown to ready 300ms", "total 1.2s"} {
		if !strings.Contains(line, want) {
			t.Errorf("cold line %q lacks %q", line, want)
		}
	}

	fast := flyoutOpen{Warm: true, Click: t0, Shown: t0.Add(20 * time.Millisecond), Ready: t0.Add(60 * time.Millisecond)}
	if _, ok := fast.logLine(); ok {
		t.Error("a warm open under the target is not logged")
	}
	slow := flyoutOpen{Warm: true, Click: t0, Shown: t0.Add(20 * time.Millisecond), Ready: t0.Add(400 * time.Millisecond)}
	if line, ok := slow.logLine(); !ok || !strings.Contains(line, "(warm)") {
		t.Errorf("a slow warm open is logged: %q %v", line, ok)
	}

	noClick := flyoutOpen{Shown: t0, Ready: t0.Add(50 * time.Millisecond)}
	if line, _ := noClick.logLine(); !strings.Contains(line, "click to shown 0s") || !strings.Contains(line, "total 50ms") {
		t.Errorf("a show without a click measures from the show: %q", line)
	}
}
