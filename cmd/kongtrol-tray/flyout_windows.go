//go:build windows

package main

// Tray flyout on Windows (v1.2.0 · 2026-11-20, ADR-0036, ADR-0056, ADR-0074):
// a frameless, topmost tool window next to the tray icon hosting Microsoft
// Edge WebView2 through github.com/jchv/go-webview2/pkg/edge (pure Go, no
// cgo: the tray keeps cross-compiling with CGO_ENABLED=0). The window lives
// on its own locked OS thread with its own message loop. It is pre-warmed
// (ADR-0074): created hidden, with the page loaded, as soon as the daemon
// first answers the tray's status poll (or on the first left click if that
// comes earlier), and then only shown/hidden. A click on a loaded page does
// no network round trip before showing it.
//
// The WebView2 controller's visibility (IsVisible) follows the window on
// every show and hide. The window is created hidden, and without
// PutIsVisible(true) after showing it the page stays document.visibilityState
// "hidden" and never paints: the flyout opened as an empty frame (#40,
// measured on WebView2 157 / Windows 11).

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"fyne.io/systray"
	"github.com/jchv/go-webview2/pkg/edge"
	"github.com/jchv/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	wmActivate   = 0x0006
	wmSize       = 0x0005
	wmClose      = 0x0010
	wmDPIChanged = 0x02E0
	wmApp        = 0x8000
	wmFlyoutShow = wmApp + 0x4B1
	wmFlyoutHide = wmApp + 0x4B2
	// wmFlyoutCheck: flyoutReadyTimeout after a show (wParam = show number).
	wmFlyoutCheck = wmApp + 0x4B3
	// wmFlyoutDaemon: a tray status poll answered (checkDaemon).
	wmFlyoutDaemon = wmApp + 0x4B4
	waInactive     = 0
	wmRButtonUp    = 0x0205

	wsPopup        = 0x80000000
	wsClipChildren = 0x02000000
	wsExToolWindow = 0x00000080
	wsExTopmost    = 0x00000008

	swHide        = 0
	swpShowWindow = 0x0040
	hwndTopmost   = ^uintptr(0) // HWND_TOPMOST (-1)
	idcArrow      = 32512

	monitorDefaultToNearest = 2
	mdtEffectiveDPI         = 0
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4): crisp text on every
	// monitor; Windows 10 1703+, ignored where missing.
	dpiAwarePerMonitorV2 = ^uintptr(3)

	dwmwaWindowCornerPreference = 33 // Windows 11; ignored on Windows 10
	dwmwcpRound                 = 2

	vkEscape = 0x1B

	// fyne.io/systray's hidden window class and notification icon id
	// (systray_windows.go: className "SystrayClass", nid.ID 100). Used only
	// to ask the shell where the icon is; a mismatch after an upgrade falls
	// back to the cursor position.
	systrayWindowClass = "SystrayClass"
	systrayIconID      = 100
	// fyne.io/systray's notification callback message (WM_USER + 1,
	// systray_windows.go); posting it with lParam WM_RBUTTONUP runs its
	// right-click path, which opens the native menu (ADR-0056 fallback).
	systrayCallbackMsg = 0x0400 + 1

	// A click on the icon while the flyout is open first deactivates it (it
	// hides on blur) and then arrives as a tray click: that click must not
	// reopen it.
	flyoutReopenGuard = 400 * time.Millisecond
)

var (
	user32                           = windows.NewLazySystemDLL("user32.dll")
	shell32                          = windows.NewLazySystemDLL("shell32.dll")
	shcore                           = windows.NewLazySystemDLL("shcore.dll")
	dwmapi                           = windows.NewLazySystemDLL("dwmapi.dll")
	procRegisterClassExW             = user32.NewProc("RegisterClassExW")
	procCreateWindowExW              = user32.NewProc("CreateWindowExW")
	procDefWindowProcW               = user32.NewProc("DefWindowProcW")
	procDestroyWindow                = user32.NewProc("DestroyWindow")
	procGetMessageW                  = user32.NewProc("GetMessageW")
	procTranslateMessage             = user32.NewProc("TranslateMessage")
	procDispatchMessageW             = user32.NewProc("DispatchMessageW")
	procPostMessageW                 = user32.NewProc("PostMessageW")
	procShowWindow                   = user32.NewProc("ShowWindow")
	procSetWindowPos                 = user32.NewProc("SetWindowPos")
	procSetForegroundWindow          = user32.NewProc("SetForegroundWindow")
	procGetCursorPos                 = user32.NewProc("GetCursorPos")
	procMonitorFromRect              = user32.NewProc("MonitorFromRect")
	procGetMonitorInfoW              = user32.NewProc("GetMonitorInfoW")
	procEnumThreadWindows            = user32.NewProc("EnumThreadWindows")
	procGetClassNameW                = user32.NewProc("GetClassNameW")
	procLoadCursorW                  = user32.NewProc("LoadCursorW")
	procSetThreadDpiAwarenessContext = user32.NewProc("SetThreadDpiAwarenessContext")
	procShellNotifyIconGetRect       = shell32.NewProc("Shell_NotifyIconGetRect")
	procGetDpiForMonitor             = shcore.NewProc("GetDpiForMonitor")
	procDwmSetWindowAttribute        = dwmapi.NewProc("DwmSetWindowAttribute")
)

type winRect struct{ Left, Top, Right, Bottom int32 }

func (r winRect) rect() rect {
	return rect{int(r.Left), int(r.Top), int(r.Right), int(r.Bottom)}
}

type winPoint struct{ X, Y int32 }

type winMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      winPoint
	Private uint32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type monitorInfo struct {
	Size    uint32
	Monitor winRect
	Work    winRect
	Flags   uint32
}

// NOTIFYICONIDENTIFIER (shellapi.h).
type notifyIconIdentifier struct {
	Size     uint32
	Wnd      windows.Handle
	ID       uint32
	GUIDItem windows.GUID
}

// winFlyout is the one flyout window. Fields marked "window thread" are
// only touched on the flyout's own OS thread.
type winFlyout struct {
	url   string
	nonce string

	startOnce   sync.Once
	enabled     atomic.Bool  // setupFlyout chose the flyout; fail() clears it
	loaded      atomic.Bool  // the last navigation succeeded (page in memory)
	clickAt     atomic.Int64 // UnixNano of the click waiting for its show
	hwnd        atomic.Uintptr
	trayHwnd    atomic.Uintptr // read by the window thread, found on systray's
	visible     atomic.Bool
	ready       atomic.Bool
	pendingShow atomic.Bool
	hiddenAt    atomic.Int64 // UnixNano of the last hide
	watch       flyoutWatch  // each show waits for the page's "ready"
	gaveUp      atomic.Bool  // fallback ran: left click = native menu
	navigated   atomic.Bool  // a navigation completed (for the fallback's log)

	chromium    *edge.Chromium // window thread
	embedded    bool           // window thread
	navigatedAt time.Time      // window thread
	loadedTag   string         // window thread: tokenTag at the last navigation
	open1       flyoutOpen     // window thread: the show waiting for "ready"
	openPending bool           // window thread: open1 is waiting
	lastReauth  time.Time      // window thread: last "reauth" reload
}

var (
	fly             winFlyout
	flySetupOnce    sync.Once
	flyWndProc      = windows.NewCallback(flyoutWndProc)
	trayWindowFound uintptr
	trayEnumProc    = windows.NewCallback(func(hwnd, _ uintptr) uintptr {
		buf := make([]uint16, 64)
		n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if windows.UTF16ToString(buf[:n]) == systrayWindowClass {
			trayWindowFound = hwnd
			return 0 // stop
		}
		return 1
	})
)

// setupFlyout makes a left click on the tray icon open the flyout when the
// WebView2 Runtime is installed (preinstalled on Windows 11, delivered to
// Windows 10 by Microsoft; detection as documented by Microsoft). Otherwise
// the left click keeps opening the native menu. The right click always
// opens the native menu.
func setupFlyout() {
	flySetupOnce.Do(func() {
		version, err := webviewloader.GetInstalledVersion()
		if err != nil {
			log.Printf("kongtrol-tray: WebView2 check failed: %v", err)
		}
		use, reason := trayLeftClick("windows", err == nil && version != "")
		if !use {
			log.Printf("kongtrol-tray: %s", reason)
			return
		}
		pageURL, err := flyoutURL(client.Base)
		if err != nil {
			log.Printf("kongtrol-tray: %v; left click opens the native menu", err)
			return
		}
		nonce := make([]byte, 32)
		if _, err := rand.Read(nonce); err != nil {
			log.Printf("kongtrol-tray: flyout disabled: %v", err)
			return
		}
		fly.url, fly.nonce = pageURL, hex.EncodeToString(nonce)
		fly.enabled.Store(true)
		systray.SetOnTapped(fly.onTrayClick)
		log.Printf("kongtrol-tray: left click opens the flyout (WebView2 %s)", version)
	})
}

// prewarmFlyout starts the flyout window hidden and loads its page, once,
// so the first click only shows it (ADR-0074). Called after every status
// poll the daemon answered (noteDaemonAnswered); cheap after the first.
func prewarmFlyout() {
	if !fly.enabled.Load() || fly.gaveUp.Load() {
		return
	}
	fly.start("pre-warm")
	// Once running: reload a hidden page whose daemon restarted.
	fly.post(wmFlyoutDaemon)
}

// start runs the window thread once; why ("pre-warm" or "click") is logged.
func (f *winFlyout) start(why string) {
	f.startOnce.Do(func() {
		log.Printf("kongtrol-tray: flyout starting WebView2 (%s)", why)
		go f.run()
	})
}

// onTrayClick runs on systray's thread (inside its window procedure): it
// must not block.
func (f *winFlyout) onTrayClick() {
	now := time.Now()
	if f.trayHwnd.Load() == 0 {
		trayWindowFound = 0
		_, _, _ = procEnumThreadWindows.Call(uintptr(windows.GetCurrentThreadId()), trayEnumProc, 0)
		f.trayHwnd.Store(trayWindowFound)
	}
	if f.visible.Load() {
		f.post(wmFlyoutHide)
		return
	}
	if time.Since(time.Unix(0, f.hiddenAt.Load())) < flyoutReopenGuard {
		return
	}
	go f.open(now)
}

func (f *winFlyout) open(clicked time.Time) {
	// Same as the menu items: the page needs the daemon. A loaded page with
	// a fresh status poll skips the probe (ADR-0074); after `kongtrol down`
	// the poll fails, so the click still starts the daemon as before.
	if flyoutNeedsDaemonCheck(f.loaded.Load(), time.Now()) {
		if err := ensureDaemon(); err != nil {
			notifyError(trayT("tray.err_start"), err)
			return
		}
	}
	f.clickAt.Store(clicked.UnixNano())
	f.pendingShow.Store(true)
	f.start("click")
	f.post(wmFlyoutShow)
}

func (f *winFlyout) post(msg uint32) {
	if h := f.hwnd.Load(); h != 0 {
		_, _, _ = procPostMessageW.Call(h, uintptr(msg), 0, 0)
	}
}

// run owns the window thread for the rest of the process.
func (f *winFlyout) run() {
	runtime.LockOSThread() // never unlocked: the window belongs to this thread
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err != nil {
		log.Printf("kongtrol-tray: flyout COM init: %v", err)
	}
	if procSetThreadDpiAwarenessContext.Find() == nil {
		_, _, _ = procSetThreadDpiAwarenessContext.Call(dpiAwarePerMonitorV2)
	}
	hwnd, err := createFlyoutWindow()
	if err != nil {
		f.fail(err)
		return
	}
	f.hwnd.Store(hwnd)

	ch := edge.NewChromium()
	ch.DataPath = flyoutDataDir()
	ch.MessageCallback = f.onMessage
	ch.WebResourceRequestedCallback = f.onRequest
	ch.NavigationCompletedCallback = func(_ *edge.ICoreWebView2, args *edge.ICoreWebView2NavigationCompletedEventArgs) {
		ok := navigationSucceeded(args)
		f.loaded.Store(ok)
		f.navigated.Store(true)
		log.Printf("kongtrol-tray: flyout page load finished in %s (success %t)", roundMS(time.Since(f.navigatedAt)), ok)
	}
	ch.AcceleratorKeyCallback = func(vk uint) bool {
		if vk == vkEscape {
			f.hideNow()
			return true
		}
		return false
	}
	f.chromium = ch
	embedStart := time.Now()
	if !ch.Embed(hwnd) {
		_, _, _ = procDestroyWindow.Call(hwnd)
		f.hwnd.Store(0)
		f.fail(errors.New("WebView2 did not start"))
		return
	}
	log.Printf("kongtrol-tray: flyout WebView2 ready in %s (environment + controller)", roundMS(time.Since(embedStart)))
	f.configure(ch)
	// The window is hidden until a show: the controller follows it
	// (ADR-0056), so a pre-warmed page loads without painting and says
	// "ready" only once it is shown (announceReady waits for "visible").
	_ = ch.Hide()
	f.embedded = true
	f.navigate()
	if f.pendingShow.Swap(false) {
		f.showNow()
	}

	var msg winMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// fail turns the flyout off for this session: the next left click opens the
// native menu again.
func (f *winFlyout) fail(err error) {
	log.Printf("kongtrol-tray: flyout unavailable, using the native menu: %v", err)
	f.enabled.Store(false)
	// fyne.io/systray reads this handler on its own thread without a lock;
	// a single pointer write, done once.
	systray.SetOnTapped(nil)
	notifyError(trayT("tray.flyout_failed"), err)
}

// giveUp is the fallback of ADR-0056 (window thread): the page did not say
// it was on screen within flyoutReadyTimeout of a show. It closes the empty
// flyout, opens the native menu in its place, logs why, and leaves the left
// click on the native menu until the tray restarts, so the user never gets
// the empty window twice.
func (f *winFlyout) giveUp() {
	var cause string
	switch {
	case !f.navigated.Load():
		cause = "the page never finished loading; is the daemon answering?"
	case !f.ready.Load():
		cause = "navigation finished (maybe as an error page) but the page never reported being on screen: not painted, daemon error or script error"
	default:
		cause = "the page stopped answering after it was shown again"
	}
	log.Printf("kongtrol-tray: flyout gave no sign of life within %s (%s); closed it and opened the native menu; left click opens the native menu until the tray restarts", flyoutReadyTimeout, cause)
	f.hideNow()
	if f.gaveUp.Swap(true) {
		return
	}
	// Same single pointer write as fail(); later clicks go to the menu.
	systray.SetOnTapped(nil)
	if h := f.trayHwnd.Load(); h != 0 {
		_, _, _ = procPostMessageW.Call(h, systrayCallbackMsg, 0, wmRButtonUp)
	}
}

// configure locks the browser down: no dev tools, context menu, zoom,
// status bar, host objects or browser shortcuts; only the daemon's origin
// may load (onRequest); the page path and the message bridge are pinned by
// flyoutInitScript.
func (f *winFlyout) configure(ch *edge.Chromium) {
	if s, err := ch.GetSettings(); err == nil {
		_ = s.PutAreDevToolsEnabled(false)
		_ = s.PutAreDefaultContextMenusEnabled(false)
		_ = s.PutIsStatusBarEnabled(false)
		_ = s.PutIsZoomControlEnabled(false)
		_ = s.PutIsPinchZoomEnabled(false)
		_ = s.PutIsSwipeNavigationEnabled(false)
		_ = s.PutAreHostObjectsAllowed(false)
		_ = s.PutAreDefaultScriptDialogsEnabled(false)
		_ = s.PutAreBrowserAcceleratorKeysEnabled(false)
	}
	ch.AddWebResourceRequestedFilter("*", edge.COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL)
	if script, err := flyoutInitScript(f.url, f.nonce); err == nil {
		ch.Init(script)
	}
	if c2 := ch.GetController().GetICoreWebView2Controller2(); c2 != nil {
		_ = c2.PutDefaultBackgroundColor(flyoutBackground())
	}
}

// navigate loads the flyout page; the address always comes from
// flyoutLoadURL (one-time auth step lives there, never the file token).
func (f *winFlyout) navigate() {
	f.navigatedAt = time.Now()
	f.loaded.Store(false)
	f.loadedTag = currentTokenTag()
	target, err := flyoutLoadURL(f.url)
	if err != nil {
		log.Printf("kongtrol-tray: flyout could not prepare its page: %v", err)
		return
	}
	f.chromium.Navigate(target)
}

// currentTokenTag is tokenTag of the daemon's current file token (read
// per call; rewritten on every daemon start).
func currentTokenTag() string {
	if client == nil {
		return ""
	}
	return tokenTag(client.CurrentToken())
}

// navigationSucceeded reads ICoreWebView2NavigationCompletedEventArgs::
// get_IsSuccess, which github.com/jchv/go-webview2 declares in its vtable
// but does not wrap. Layout: IUnknown (3 slots), then IsSuccess (BOOL*).
// Any error counts as a failed load (the next show loads the page again).
func navigationSucceeded(args *edge.ICoreWebView2NavigationCompletedEventArgs) bool {
	if args == nil {
		return false
	}
	type vtbl struct{ queryInterface, addRef, release, getIsSuccess uintptr }
	obj := (*struct{ v *vtbl })(unsafe.Pointer(args))
	if obj.v == nil || obj.v.getIsSuccess == 0 {
		return false
	}
	var ok int32
	hr, _, _ := syscall.SyscallN(obj.v.getIsSuccess, uintptr(unsafe.Pointer(args)), uintptr(unsafe.Pointer(&ok)))
	return hr == 0 && ok != 0
}

// onRequest answers 403 to anything outside the daemon's origin. Only the
// scheme and host are logged, never the full URL.
func (f *winFlyout) onRequest(req *edge.ICoreWebView2WebResourceRequest, args *edge.ICoreWebView2WebResourceRequestedEventArgs) {
	raw, err := req.GetUri()
	if err == nil && flyoutRequestAllowed(f.url, raw) {
		return
	}
	where := "?"
	if u, perr := url.Parse(raw); perr == nil {
		where = u.Scheme + "://" + u.Host
	}
	log.Printf("kongtrol-tray: flyout blocked a request to %s", where)
	if env := f.chromium.Environment(); env != nil {
		if resp, err := env.CreateWebResourceResponse(nil, 403, "Forbidden", ""); err == nil {
			_ = args.PutResponse(resp)
		}
	}
}

// onMessage runs on the window thread.
func (f *winFlyout) onMessage(msg string) {
	if msg == flyoutOffPage {
		log.Println("kongtrol-tray: flyout left its page; loading it again")
		f.navigate()
		return
	}
	switch a := parseFlyoutMessage(msg, f.nonce); a {
	case flyoutReady:
		f.ready.Store(true)
		f.watch.ack()
		if f.openPending {
			f.openPending = false
			f.open1.Ready = time.Now()
			if line, ok := f.open1.logLine(); ok {
				log.Print(line)
			}
		}
	case flyoutReauth:
		f.reauth()
	case flyoutNone:
	default:
		runFlyoutAction(a, f.hideNow)
	}
}

// reauth reloads the page with a fresh one-time nonce after the page said
// its session is gone (window thread, ADR-0074). The window stays as it is;
// a visible one must hear "ready" from the new page within
// flyoutReadyTimeout like any show (ADR-0056).
func (f *winFlyout) reauth() {
	now := time.Now()
	if !flyoutReauthAllowed(f.lastReauth, now) {
		log.Println("kongtrol-tray: flyout asked to sign in again too soon; ignored")
		return
	}
	f.lastReauth = now
	log.Println("kongtrol-tray: flyout session expired (daemon restarted?); loading the page with a new sign-in")
	f.navigate()
	if f.visible.Load() {
		f.armWatch(f.hwnd.Load())
	}
}

// checkDaemon runs on the window thread after a status poll answered
// (wmFlyoutDaemon): a hidden page signed in to a daemon that has since
// restarted is loaded again now, so the next click is still warm.
func (f *winFlyout) checkDaemon() {
	if !f.embedded || f.visible.Load() || !f.loaded.Load() {
		return
	}
	if flyoutOnShow(true, f.loadedTag, currentTokenTag(), 0) == showNavigate {
		log.Println("kongtrol-tray: daemon restarted; reloading the hidden flyout page with a new sign-in")
		f.navigate()
	}
}

// armWatch starts waiting for the page's "ready" for a new show number.
func (f *winFlyout) armWatch(hwnd uintptr) {
	seq := f.watch.show()
	time.AfterFunc(flyoutReadyTimeout, func() {
		_, _, _ = procPostMessageW.Call(hwnd, wmFlyoutCheck, uintptr(seq), 0)
	})
}

// showNow places the window next to the tray icon and shows it (window
// thread). Coordinates are physical pixels (per-monitor DPI aware thread).
func (f *winFlyout) showNow() {
	hwnd := f.hwnd.Load()
	icon := trayIconRect(f.trayHwnd.Load())
	mon, _, _ := procMonitorFromRect.Call(uintptr(unsafe.Pointer(&icon)), monitorDefaultToNearest)
	mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	_, _, _ = procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	dpi := monitorDPI(mon)
	w, h := scaleDIP(flyoutWidth, dpi), scaleDIP(flyoutHeight, dpi)
	x, y := placeFlyout(icon.rect(), mi.Work.rect(), mi.Monitor.rect(), w, h, scaleDIP(flyoutGap, dpi))

	_, _, _ = procSetWindowPos.Call(hwnd, hwndTopmost, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpShowWindow)
	f.chromium.Resize()
	// The page paints only while the controller is visible (see the file
	// comment); the window is hidden in between.
	if err := f.chromium.Show(); err != nil {
		log.Printf("kongtrol-tray: flyout could not make its page visible: %v", err)
	}
	_, _, _ = procSetForegroundWindow.Call(hwnd)
	f.chromium.Focus()
	f.visible.Store(true)

	// Timing for tray.log (ADR-0074): from the click (if any) to "ready".
	var clicked time.Time
	if at := f.clickAt.Swap(0); at != 0 {
		clicked = time.Unix(0, at)
	}
	loaded := f.loaded.Load()
	f.open1 = flyoutOpen{Warm: loaded, Click: clicked, Shown: time.Now()}
	f.openPending = true

	// The page answers "ready" once it is on screen; without that answer the
	// window would stay an empty frame (ADR-0056).
	f.armWatch(hwnd)

	switch flyoutOnShow(loaded, f.loadedTag, currentTokenTag(), time.Since(f.navigatedAt)) {
	case showRefresh:
		// Refresh data and move focus into the page (web/ui/src/tray). A
		// pre-warmed page that was never shown handles it too: it re-arms
		// its "ready" announcement and refreshes its data.
		f.chromium.Eval("window.dispatchEvent(new Event('kongtrol:shown'))")
	case showNavigate:
		// The load failed (daemon was starting) or the daemon restarted
		// since (its sign-in is gone): load the page again.
		f.open1.Warm = false
		f.navigate()
	case showWait:
		// Still loading; the page says "ready" when it is visible.
	}
}

func (f *winFlyout) hideNow() {
	if h := f.hwnd.Load(); h != 0 {
		_, _, _ = procShowWindow.Call(h, swHide)
	}
	if f.embedded {
		// Hidden page: WebView2 stops rendering it (document "hidden").
		_ = f.chromium.Hide()
	}
	if f.visible.Swap(false) {
		f.hiddenAt.Store(time.Now().UnixNano())
	}
}

func flyoutWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case wmFlyoutShow:
		if fly.embedded {
			fly.pendingShow.Store(false)
			fly.showNow()
		}
		return 0
	case wmFlyoutHide:
		fly.hideNow()
		return 0
	case wmFlyoutDaemon:
		fly.checkDaemon()
		return 0
	case wmFlyoutCheck:
		if fly.visible.Load() && fly.watch.unanswered(uint64(wParam)) {
			fly.giveUp()
		}
		return 0
	case wmActivate:
		if wParam&0xFFFF == waInactive && fly.visible.Load() {
			fly.hideNow() // click outside: close, like the system flyouts
		}
	case wmSize:
		if fly.embedded {
			fly.chromium.Resize()
		}
	case wmClose:
		fly.hideNow() // Alt+F4 hides; the window is reused
		return 0
	case wmDPIChanged:
		return 0 // showNow already sized the window for its monitor
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
}

func createFlyoutWindow() (uintptr, error) {
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		return 0, err
	}
	className, _ := windows.UTF16PtrFromString("KongtrolFlyout")
	title, _ := windows.UTF16PtrFromString("Kongtrol")
	cursor, _, _ := procLoadCursorW.Call(0, idcArrow)
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   flyWndProc,
		Instance:  instance,
		Cursor:    windows.Handle(cursor),
		ClassName: className,
	}
	if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return 0, err
	}
	hwnd, _, err := procCreateWindowExW.Call(
		wsExToolWindow|wsExTopmost, // no taskbar button, above other windows
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsPopup|wsClipChildren, // frameless
		0, 0, flyoutWidth, flyoutHeight,
		0, 0, uintptr(instance), 0,
	)
	if hwnd == 0 {
		return 0, err
	}
	pref := int32(dwmwcpRound)
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, dwmwaWindowCornerPreference, uintptr(unsafe.Pointer(&pref)), unsafe.Sizeof(pref))
	return hwnd, nil
}

// trayIconRect asks the shell where the tray icon is; when it can't (icon in
// the overflow area, unknown window) the cursor position stands in, since
// the user just clicked the icon.
func trayIconRect(trayHwnd uintptr) winRect {
	if trayHwnd != 0 && procShellNotifyIconGetRect.Find() == nil {
		id := notifyIconIdentifier{Wnd: windows.Handle(trayHwnd), ID: systrayIconID}
		id.Size = uint32(unsafe.Sizeof(id))
		var r winRect
		hr, _, _ := procShellNotifyIconGetRect.Call(uintptr(unsafe.Pointer(&id)), uintptr(unsafe.Pointer(&r)))
		if hr == 0 && !r.rect().empty() {
			return r
		}
	}
	var p winPoint
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return winRect{p.X, p.Y, p.X + 1, p.Y + 1}
}

func monitorDPI(mon uintptr) uint32 {
	if procGetDpiForMonitor.Find() != nil {
		return 96
	}
	var x, y uint32
	if hr, _, _ := procGetDpiForMonitor.Call(mon, mdtEffectiveDPI, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y))); hr != 0 {
		return 96
	}
	return x
}

// flyoutDataDir keeps WebView2's profile (cache, localStorage) under
// ~/.kongtrol so "Borrar todos mis datos" and uninstall cover it.
func flyoutDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kongtrol", "webview2")
}

// flyoutBackground is shown before the page paints: the page's own
// background for the system's app theme (DESIGN.md bg / bg-light), so
// opening never flashes white in dark mode.
func flyoutBackground() edge.COREWEBVIEW2_COLOR {
	dark := edge.COREWEBVIEW2_COLOR{A: 255, R: 0x0a, G: 0x0c, B: 0x10}
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return dark
	}
	defer k.Close()
	if v, _, err := k.GetIntegerValue("AppsUseLightTheme"); err == nil && v == 1 {
		return edge.COREWEBVIEW2_COLOR{A: 255, R: 0xf1, G: 0xed, B: 0xe4}
	}
	return dark
}
