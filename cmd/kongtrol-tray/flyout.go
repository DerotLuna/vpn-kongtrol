package main

// Tray flyout — portable rules (v1.0.0 · 2026-11-18, ADR-0036).
//
// The flyout is a small branded window that opens next to the tray icon on a
// left click (like the Windows 11 network flyout, Tailscale or Docker
// Desktop); the native menu stays on the right click and is the left-click
// fallback wherever the flyout cannot run. It shows one page of the UI the
// daemon already serves (web/ui, route /app/tray), so it is a client of the
// local API like the dashboard: same HttpOnly cookie, no token in the URL.
//
// This file holds the OS-independent rules (which URL may load, where the
// window goes, which page messages exist) so they are unit-tested on every
// OS. flyout_windows.go draws the window with WebView2; other OSes keep the
// native menu (flyout_other.go).

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"

	"fyne.io/systray"

	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
)

const (
	// flyoutPath is the only page the flyout window may show.
	flyoutPath = "/app/tray"
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
// and returns pageURL + "#k=<nonce>". A fragment keeps the path /app/tray,
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
		tok = client.Token
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
	origin, _ := json.Marshal(u.Scheme + "://" + u.Host)
	path, _ := json.Marshal(u.EscapedPath())
	n, _ := json.Marshal(nonce)
	return `(function () {
  var w = window.chrome && window.chrome.webview;
  if (!w) return;
  window.open = function () { return null; };
  if (location.origin !== ` + string(origin) + ` || location.pathname !== ` + string(path) + `) {
    w.postMessage("` + flyoutOffPage + `");
    return;
  }
  var nonce = ` + string(n) + `;
  Object.defineProperty(window, "kongtrolTray", {
    value: Object.freeze({ post: function (m) { w.postMessage(nonce + " " + String(m)); } }),
  });
})();`, nil
}

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
	// flyoutReady: the page loaded.
	flyoutReady
	// flyoutClose: Escape, or an action that ends the flyout's job.
	flyoutClose
	// flyoutOpenDashboard: "Abrir panel", same as the menu item.
	flyoutOpenDashboard
	// flyoutQuitTray: close only the tray icon; VPNs keep running.
	flyoutQuitTray
	// flyoutQuitAll: "Salir y desconectar todo" (daemon shutdown + quit).
	flyoutQuitAll
)

// flyoutMessages maps the names web/ui/src/tray/bridge.ts sends.
var flyoutMessages = map[string]flyoutAction{
	"ready":          flyoutReady,
	"close":          flyoutClose,
	"open-dashboard": flyoutOpenDashboard,
	"quit-tray":      flyoutQuitTray,
	"quit-all":       flyoutQuitAll,
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
	}
}
