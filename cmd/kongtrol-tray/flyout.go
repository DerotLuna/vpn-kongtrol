package main

// Tray flyout — portable rules (v1.2.0 · 2026-11-20, ADR-0036, ADR-0056, ADR-0074).
//
// The flyout is a small branded window that opens next to the tray icon on a
// left click (like the Windows 11 network flyout, Tailscale or Docker
// Desktop); the native menu stays on the right click and is the left-click
// fallback wherever the flyout cannot run. It shows one page of the UI the
// daemon already serves (web/ui, route /tray), so it is a client of the
// local API like the dashboard: same HttpOnly cookie, no token in the URL.
//
// This file holds the OS-independent rules (which URL may load, where the
// window goes, which page messages exist) so they are unit-tested on every
// OS. flyout_windows.go draws the window with WebView2; other OSes keep the
// native menu (flyout_other.go).

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/systray"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
)

const (
	// flyoutPath is the only page the flyout window may show.
	flyoutPath = "/tray"
	// Size in device-independent pixels (scaled by the monitor's DPI).
	flyoutWidth  = 360
	flyoutHeight = 520
	// flyoutGap is the space between the flyout and the taskbar / screen
	// edge, close to the Windows 11 quick settings flyout (no verificado:
	// measured by eye, not documented).
	flyoutGap = 12
)

// quitTray closes only the tray icon (the daemon and its VPNs keep running).
var quitTray = systray.Quit

var errFlyoutURL = errors.New("flyout: daemon address is not a loopback http URL")

// flyoutURL builds the flyout page URL from the daemon's API base
// (daemonctl.APIBase). Only plain http on a loopback host with an explicit
// port is accepted: the page carries the API capability (cookie), so the
// window must never load anything that is not this machine's daemon.
func flyoutURL(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "http" || u.Port() == "" || u.User != nil {
		return "", errFlyoutURL
	}
	if !isLoopbackHost(u.Hostname()) {
		return "", errFlyoutURL
	}
	return "http://" + u.Host + flyoutPath, nil
}

// flyoutLoadURL is the ONE place that turns the flyout page URL into the
// address the window navigates to, called before every navigation. It asks
// the daemon for a one-time sign-in nonce (daemonctl.Client.PanelURL,
// authenticated with the file token, which never enters a URL; ADR-0039)
// and returns pageURL + "#k=<nonce>". A fragment keeps the path /tray,
// so the request filter and flyoutInitScript stay unchanged, and the
// fragment is never sent over HTTP nor logged. Without a nonce (older
// daemon, daemon not answering) the page loads plain and shows how to
// reopen it.
func flyoutLoadURL(pageURL string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil || u.Path != flyoutPath {
		return "", errFlyoutURL
	}
	got, _ := flyoutPanelURL("http://"+u.Host, flyoutPath)
	if !flyoutRequestAllowed(pageURL, got) {
		return pageURL, nil
	}
	return got, nil
}

// flyoutPanelURL is daemonctl's PanelURL for base (a test seam).
var flyoutPanelURL = func(base, path string) (string, error) {
	tok := ""
	if client != nil {
		tok = client.CurrentToken()
	}
	return daemonctl.New(base, tok).PanelURL(path)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(strings.TrimSuffix(host, "."), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// flyoutRequestAllowed is the web-request filter of the flyout window.
// pageURL is what flyoutURL returned. Every request (page, scripts, styles,
// fonts, /api calls, the WebSocket feed) must stay on the daemon's origin:
// same host and port, plain http or ws, no credentials in the URL. The page
// path itself is pinned by flyoutInitScript.
func flyoutRequestAllowed(pageURL, raw string) bool {
	page, err := url.Parse(pageURL)
	if err != nil {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Opaque != "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "ws" {
		return false
	}
	return strings.EqualFold(u.Host, page.Host)
}

// flyoutOffPage is sent (without the nonce) by flyoutInitScript when the
// window is not on the flyout page; its only effect is loading it again.
const flyoutOffPage = "kongtrol:off-page"

// flyoutInitScript is injected before any page script of every document the
// window loads. On the flyout page (exact origin and path) it exposes
// window.kongtrolTray.post, which prefixes the per-launch nonce; the tray
// only acts on messages that carry it. Anywhere else (another daemon page
// reached by a link, or a page the request filter missed) it exposes
// nothing and asks the tray, without the nonce, to load the flyout page
// again. It also disables window.open: the flyout never opens windows.
func flyoutInitScript(pageURL, nonce string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil || u.Host == "" {
		return "", errFlyoutURL
	}
	// Every value reaches the script as one JSON object (json.Marshal
	// escapes quotes, <, >, & and U+2028/U+2029, so it is a valid JS
	// literal); nothing else is concatenated into the source.
	cfg, err := json.Marshal(struct {
		Origin  string `json:"origin"`
		Path    string `json:"path"`
		Nonce   string `json:"nonce"`
		OffPage string `json:"offPage"`
	}{u.Scheme + "://" + u.Host, u.EscapedPath(), nonce, flyoutOffPage})
	if err != nil {
		return "", err
	}
	return flyoutScriptHead + string(cfg) + flyoutScriptBody, nil
}

// flyoutScriptHead + JSON config + flyoutScriptBody is flyoutInitScript.
const (
	flyoutScriptHead = "(function () {\n  var cfg = "
	flyoutScriptBody = `;
  var w = window.chrome && window.chrome.webview;
  if (!w) return;
  window.open = function () { return null; };
  if (location.origin !== cfg.origin || location.pathname !== cfg.path) {
    w.postMessage(cfg.offPage);
    return;
  }
  var nonce = cfg.nonce;
  Object.defineProperty(window, "kongtrolTray", {
    value: Object.freeze({ post: function (m) { w.postMessage(nonce + " " + String(m)); } }),
  });
})();`
)

// rect is a screen rectangle in pixels (right/bottom exclusive, as Win32 RECT).
type rect struct{ Left, Top, Right, Bottom int }

func (r rect) width() int  { return r.Right - r.Left }
func (r rect) height() int { return r.Bottom - r.Top }
func (r rect) empty() bool { return r.width() <= 0 || r.height() <= 0 }

// taskbarEdge is the screen edge the taskbar sits on.
type taskbarEdge int

const (
	edgeBottom taskbarEdge = iota
	edgeTop
	edgeLeft
	edgeRight
)

// detectTaskbarEdge finds the taskbar edge from the monitor and its work
// area (the monitor minus the taskbar): the side where the work area is
// smaller. With an auto-hidden taskbar both are equal, so the edge closest to
// the icon wins. Bottom is the default (Windows' default taskbar).
func detectTaskbarEdge(icon, work, monitor rect) taskbarEdge {
	switch {
	case work.Bottom < monitor.Bottom:
		return edgeBottom
	case work.Top > monitor.Top:
		return edgeTop
	case work.Left > monitor.Left:
		return edgeLeft
	case work.Right < monitor.Right:
		return edgeRight
	}
	if icon.empty() {
		return edgeBottom
	}
	cx, cy := (icon.Left+icon.Right)/2, (icon.Top+icon.Bottom)/2
	best, dist := edgeBottom, monitor.Bottom-cy
	for _, c := range []struct {
		edge taskbarEdge
		d    int
	}{{edgeTop, cy - monitor.Top}, {edgeLeft, cx - monitor.Left}, {edgeRight, monitor.Right - cx}} {
		if c.d < dist {
			best, dist = c.edge, c.d
		}
	}
	return best
}

// placeFlyout returns the top-left corner for a w×h flyout anchored to the
// tray icon: next to the taskbar edge, centered on the icon along that edge,
// and always inside the work area with gap pixels of margin (a flyout larger
// than the work area is pinned to its left/top edge).
func placeFlyout(icon, work, monitor rect, w, h, gap int) (x, y int) {
	if work.empty() {
		work = monitor
	}
	cx, cy := (icon.Left+icon.Right)/2, (icon.Top+icon.Bottom)/2
	switch detectTaskbarEdge(icon, work, monitor) {
	case edgeTop:
		x, y = cx-w/2, work.Top+gap
	case edgeLeft:
		x, y = work.Left+gap, cy-h/2
	case edgeRight:
		x, y = work.Right-gap-w, cy-h/2
	default:
		x, y = cx-w/2, work.Bottom-gap-h
	}
	x = clamp(x, work.Left+gap, work.Right-gap-w)
	y = clamp(y, work.Top+gap, work.Bottom-gap-h)
	return x, y
}

// clamp keeps v in [lo, hi]; when the range is empty (hi < lo) lo wins.
func clamp(v, lo, hi int) int {
	if v > hi {
		v = hi
	}
	if v < lo {
		v = lo
	}
	return v
}

// scaleDIP converts device-independent pixels to pixels at dpi (96 = 100%).
func scaleDIP(v int, dpi uint32) int {
	if dpi == 0 {
		dpi = 96
	}
	return (v*int(dpi) + 48) / 96
}

// flyoutAction is one message the flyout page may send to the tray. The set
// is closed: anything else is ignored, so the page can never make the tray
// run something it was not built for.
type flyoutAction int

const (
	flyoutNone flyoutAction = iota
	// flyoutReady: the page is on screen (sent after it loads and after
	// every show; flyoutWatch).
	flyoutReady
	// flyoutClose: Escape, or an action that ends the flyout's job.
	flyoutClose
	// flyoutOpenDashboard: "Abrir panel", same as the menu item.
	flyoutOpenDashboard
	// flyoutQuitTray: close only the tray icon; VPNs keep running.
	flyoutQuitTray
	// flyoutQuitAll: "Salir y desconectar todo" (daemon shutdown + quit).
	flyoutQuitAll
	// flyoutOpenAdd: "Agregar VPN" when there is none yet: the panel's
	// "＋ Agregar" sheet on the VPN step (ADR-0055 deep link).
	flyoutOpenAdd
	flyoutOpenDiagnostics
	// flyoutReauth: the page's session is gone (401; sessions live in the
	// daemon's memory, so a daemon restart ends them). The tray loads the
	// page again with a fresh one-time nonce (ADR-0074), at most once per
	// flyoutReauthEvery, without hiding the window.
	flyoutReauth
	// flyoutOpenAdd<Provider>: a notice's "Agregar <App>" (ADR-0102): the
	// "＋ Agregar" sheet with that provider chosen. One message per provider
	// keeps the list closed (no free-form argument from the page).
	flyoutOpenAddProtonVPN
	flyoutOpenAddNordVPN
	flyoutOpenAddSurfshark
	flyoutOpenAddFortiClient
	flyoutOpenAddMullvad
	flyoutOpenAddPia
	flyoutOpenAddExpressvpn
	flyoutOpenAddWindscribe
	flyoutOpenAddCyberghost
)

// flyoutAddProviders maps each "open-add-<type>" action to its adapter type.
var flyoutAddProviders = map[flyoutAction]string{
	flyoutOpenAddProtonVPN:   "protonvpn",
	flyoutOpenAddNordVPN:     "nordvpn",
	flyoutOpenAddSurfshark:   "surfshark",
	flyoutOpenAddFortiClient: "forticlient",
	flyoutOpenAddMullvad:     "mullvad",
	flyoutOpenAddPia:         "pia",
	flyoutOpenAddExpressvpn:  "expressvpn",
	flyoutOpenAddWindscribe:  "windscribe",
	flyoutOpenAddCyberghost:  "cyberghost",
}

// addProviderPath is the deep link of "Agregar <App>" (web/ui
// src/app/addLink.ts `type`), or "" for a type outside the closed list.
func addProviderPath(adapterType string) string {
	for _, t := range flyoutAddProviders {
		if t == adapterType {
			return addVPNPath + "&type=" + t
		}
	}
	return ""
}

// flyoutReauthEvery rate-limits "reauth" so a daemon that keeps refusing
// the sign-in cannot make the flyout reload in a loop.
const flyoutReauthEvery = 5 * time.Second

// flyoutReauthAllowed reports whether a "reauth" at now may reload the page
// after the previous one at last (zero = never).
func flyoutReauthAllowed(last, now time.Time) bool {
	return last.IsZero() || now.Sub(last) >= flyoutReauthEvery || now.Before(last)
}

// addVPNPath is the panel deep link that opens the "＋ Agregar" sheet on
// the VPN step (web/ui src/app/addLink.ts).
const addVPNPath = "/?add=vpn"

// flyoutMessages maps the names web/ui/src/tray/bridge.ts sends.
var flyoutMessages = map[string]flyoutAction{
	"ready":            flyoutReady,
	"close":            flyoutClose,
	"open-dashboard":   flyoutOpenDashboard,
	"quit-tray":        flyoutQuitTray,
	"quit-all":         flyoutQuitAll,
	"open-add":         flyoutOpenAdd,
	"open-diagnostics": flyoutOpenDiagnostics,
	"reauth":           flyoutReauth,
	// ADR-0102: a notice's "Agregar <App>".
	"open-add-protonvpn":   flyoutOpenAddProtonVPN,
	"open-add-nordvpn":     flyoutOpenAddNordVPN,
	"open-add-surfshark":   flyoutOpenAddSurfshark,
	"open-add-forticlient": flyoutOpenAddFortiClient,
	"open-add-mullvad":     flyoutOpenAddMullvad,
	"open-add-pia":         flyoutOpenAddPia,
	"open-add-expressvpn":  flyoutOpenAddExpressvpn,
	"open-add-windscribe":  flyoutOpenAddWindscribe,
	"open-add-cyberghost":  flyoutOpenAddCyberghost,
}

// parseFlyoutMessage returns the action of a page message, which must be
// "<nonce> <name>" with this launch's nonce. Anything else is flyoutNone.
func parseFlyoutMessage(msg, nonce string) flyoutAction {
	if nonce == "" {
		return flyoutNone
	}
	got, name, ok := strings.Cut(msg, " ")
	if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(nonce)) != 1 {
		return flyoutNone
	}
	return flyoutMessages[name]
}

// trayLeftClick says what a left click on the tray icon does on goos: the
// flyout where it is implemented and its web runtime is present, otherwise
// the native menu (the reason is logged once).
func trayLeftClick(goos string, webviewInstalled bool) (useFlyout bool, reason string) {
	switch {
	case goos != "windows":
		return false, "flyout not implemented on " + goos + "; left click opens the native menu"
	case !webviewInstalled:
		return false, "Microsoft Edge WebView2 Runtime not installed; left click opens the native menu"
	default:
		return true, ""
	}
}

// runFlyoutAction does what the page asked for. hide closes the flyout
// window (OS-specific, called on the window's thread).
func runFlyoutAction(a flyoutAction, hide func()) {
	switch a {
	case flyoutClose:
		hide()
	case flyoutOpenDashboard:
		hide()
		go openDashboard()
	case flyoutQuitTray:
		hide()
		quitTray()
	case flyoutQuitAll:
		hide()
		go stopKongtrol()
	case flyoutOpenAdd:
		hide()
		go openDashboardPage(addVPNPath)
	case flyoutOpenDiagnostics:
		hide()
		go openDiagnostics()
	case flyoutOpenAddProtonVPN, flyoutOpenAddNordVPN, flyoutOpenAddSurfshark, flyoutOpenAddFortiClient:
		hide()
		go openDashboardPage(addProviderPath(flyoutAddProviders[a]))
	}
}

// flyoutReadyTimeout is how long a shown flyout may wait for the page's
// "ready" before the tray gives up on it (ADR-0056): enough for a cold
// WebView2 start (the page answered in under 1 s on one Windows 11 machine;
// slower machines no verificado), short enough that nobody stares at an
// empty window.
const flyoutReadyTimeout = 3 * time.Second

// flyoutWatch pairs every show of the flyout with the page's "ready"
// (ADR-0056). The page answers only while it is visible and painting, so a
// show without an answer means the user is looking at an empty window.
type flyoutWatch struct{ shown, acked atomic.Uint64 }

// show starts waiting for a new answer and returns the show's number.
func (w *flyoutWatch) show() uint64 { return w.shown.Add(1) }

// ack records the page's "ready" for the latest show.
func (w *flyoutWatch) ack() { w.acked.Store(w.shown.Load()) }

// unanswered reports whether show seq is still the latest one and the page
// has not answered it.
func (w *flyoutWatch) unanswered(seq uint64) bool {
	return w.shown.Load() == seq && w.acked.Load() < seq
}

// ── Pre-warming and open timing (ADR-0074, issue #41) ─────────────────────

const (
	// flyoutSlowOpen is the target for a warm open, click to "ready"
	// (issue #41: almost instant, under ~150 ms). Warm opens slower than
	// this are logged; cold ones always are.
	flyoutSlowOpen = 150 * time.Millisecond
	// flyoutDaemonFresh: a tray status poll (every 3 s, tray_menu.go) that
	// answered this recently proves the daemon is up, so a click on a
	// loaded flyout shows it at once instead of probing the daemon first.
	flyoutDaemonFresh = 5 * time.Second
	// flyoutRetryLoad: a show that finds the page not loaded this long
	// after its last navigation started (the load failed) loads it again.
	flyoutRetryLoad = 3 * time.Second
)

// daemonAnsweredAt is the UnixNano of the last tray status poll the daemon
// answered (0 = never).
var daemonAnsweredAt atomic.Int64

// noteDaemonAnswered is called after every status poll the daemon
// answered. It records the time and pre-warms the flyout (OS-specific; a
// no-op where there is no flyout or it is already started).
func noteDaemonAnswered() {
	daemonAnsweredAt.Store(time.Now().UnixNano())
	prewarmFlyout()
}

// daemonAnsweredWithin reports whether the daemon answered a status poll
// in the last d before now.
func daemonAnsweredWithin(now time.Time, d time.Duration) bool {
	at := daemonAnsweredAt.Load()
	if at == 0 {
		return false
	}
	ago := now.Sub(time.Unix(0, at))
	return ago >= 0 && ago <= d
}

// flyoutNeedsDaemonCheck says whether a left click must first make sure the
// daemon answers (ensureDaemon: a probe, and a spawn after `kongtrol down`).
// A loaded page and a fresh status poll skip it: that probe was the only
// network round trip left on a warm click.
func flyoutNeedsDaemonCheck(pageLoaded bool, now time.Time) bool {
	return !pageLoaded || !daemonAnsweredWithin(now, flyoutDaemonFresh)
}

// tokenTag identifies the daemon the page signed in to without keeping
// another copy of its token: the token is rewritten on every daemon start
// (AGENTS.md "Daemon lifecycle"), so another tag means another daemon, and
// the page's session died with the old one. Empty token, empty tag.
func tokenTag(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:8])
}

// flyoutShowStep is what a show does with the page besides showing it.
type flyoutShowStep int

const (
	// showRefresh: the page is loaded; send kongtrol:shown (refresh data,
	// focus, announce "ready" again).
	showRefresh flyoutShowStep = iota
	// showNavigate: load the page again (fresh one-time sign-in nonce).
	showNavigate
	// showWait: a load is still running; the page says "ready" once it is
	// visible (announceReady listens to visibilitychange).
	showWait
)

// flyoutOnShow picks the show's step. loadedTag and currentTag are the
// tokenTag of the token when the page loaded and now: a page signed in to a
// daemon that has since restarted is loaded again instead of shown signed
// out.
func flyoutOnShow(loaded bool, loadedTag, currentTag string, sinceNavigate time.Duration) flyoutShowStep {
	switch {
	case loaded && loadedTag != "" && currentTag != "" && loadedTag != currentTag:
		return showNavigate
	case loaded:
		return showRefresh
	case sinceNavigate > flyoutRetryLoad:
		return showNavigate
	default:
		return showWait
	}
}

// flyoutOpen is one open of the flyout, for tray.log.
type flyoutOpen struct {
	Warm  bool      // the page was already loaded when the window showed
	Click time.Time // tray icon clicked (zero: shown without a click)
	Shown time.Time // window on screen
	Ready time.Time // page said "ready"
}

// logLine returns the tray.log line for a finished open and whether to
// write it: cold opens always, warm ones only above flyoutSlowOpen, so the
// log does not grow with every click.
func (o flyoutOpen) logLine() (string, bool) {
	start := o.Click
	if start.IsZero() || start.After(o.Shown) {
		start = o.Shown
	}
	total := o.Ready.Sub(start)
	kind := "cold"
	if o.Warm {
		kind = "warm"
	}
	line := fmt.Sprintf("kongtrol-tray: flyout open (%s): click to shown %s, shown to ready %s, total %s",
		kind, roundMS(o.Shown.Sub(start)), roundMS(o.Ready.Sub(o.Shown)), roundMS(total))
	return line, !o.Warm || total > flyoutSlowOpen
}

func roundMS(d time.Duration) time.Duration {
	if d < 0 {
		d = 0
	}
	return d.Round(time.Millisecond)
}
