<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0036: Branded tray flyout on left click (Windows first), native menu kept

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

The tray is the first thing non-technical users touch (GUI-first, ADR-0003), but it only offers
a plain native menu. Mature apps open a small branded window next to the tray icon (Windows 11
quick settings, Tailscale on macOS, Docker Desktop). We want the same as the primary tray UI
without breaking the CGO-free cross-compiled tray builds (`.goreleaser.yaml`, `release.yml`),
"local only", accessibility, or the native menu that another track keeps improving.

## Decision Drivers

- Reuse `web/ui` (already embedded, served by the daemon at `/app/`, i18n, themes, tests).
- Windows and Linux trays stay pure Go (`CGO_ENABLED=0`); macOS already needs cgo.
- The window must only ever show this machine's daemon; no token in URLs.
- Native menu stays (right click, and left click wherever the flyout cannot run).

## Considered Options

1. Wails v3 tray with attached window — beta, replaces the tray library and app loop, cgo on
   Linux/macOS.
2. `webview/webview_go` — cgo on every OS; Windows/Linux tray stops cross-compiling.
3. **`github.com/jchv/go-webview2/pkg/edge` (pure Go WebView2) in our own Win32 popup on
   Windows; native menu elsewhere.**
4. Native widgets (`lxn/walk`) — a second UI to maintain, no reuse of web/ui.

## Decision Outcome

Chosen: **option 3** (sources and trade-offs in the
[research note](../research/2026-11-18-tray-flyout.md)).

- `fyne.io/systray` v1.12.2: `SetOnTapped` takes the left click on Windows only; right click
  keeps the menu. Linux and macOS do not set it (`cmd/kongtrol-tray/flyout_other.go`).
- Windows (`cmd/kongtrol-tray/flyout_windows.go`): frameless `WS_POPUP`, tool window (no
  taskbar button), topmost, rounded corners on Windows 11, per-monitor DPI aware thread, own
  message loop; created on the first click and then shown/hidden. Placed next to the icon
  (`Shell_NotifyIconGetRect`, taskbar edge from monitor vs work area, cursor fallback; math in
  `placeFlyout`, tested). Closes on blur, Escape, Alt+F4 (hide). Size 360×520 DIP.
- Page: `web/ui` route `/app/tray` (`web/ui/src/pages/TrayPage.tsx`): aggregate state, VPN rows
  with the Four States and a `role="switch"`, groups, kill switch / DNS guard / public IP state
  (read only, no lookup), "Abrir panel", "Salir" (close icon only, or quit and disconnect all,
  same as the menu), version.
- Missing WebView2 Runtime → left click keeps the native menu (no download or prompt). If the
  window fails at runtime, the tray logs it, notifies once and returns the left click to the
  menu.

### Security model

- URL: only `http://<loopback>:<port>/app/tray` built from the API base (`flyoutURL`; a LAN
  bind is refused). Auth is the existing HttpOnly `SameSite=Strict` cookie; no token in URLs
  or logs. Every navigation takes its address from `flyoutLoadURL`, the single place where
  the planned one-time bootstrap nonce (`/app/tray#k=<nonce>`, exchanged by the page for a
  session cookie) will be added; the fragment keeps the pinned path.
- Every web request outside the daemon's origin gets 403 (`WebResourceRequested`, filter
  `*`; only scheme + host are logged).
- A script injected before any page script pins the page: on any other origin or path it
  exposes nothing and asks the tray to reload the flyout page; `window.open` is disabled.
- Page → tray messages are a closed list (`ready`, `close`, `open-dashboard`, `quit-tray`,
  `quit-all`), each prefixed with a random per-launch nonce known only to that script on
  that page. Unknown or unsigned messages are ignored.
- Dev tools, context menu, zoom, status bar, host objects, script dialogs and browser
  shortcuts are disabled. The page has the daemon's CSP (`default-src 'self'`).
- Profile data (cache, localStorage) in `~/.kongtrol/webview2`, removed with the user's data.
- No new outbound call: the window only talks to 127.0.0.1.

### Consequences

- Good: one UI codebase; Windows gets the flyout with no cgo and no pipeline change
  (`CGO_ENABLED=0` builds for windows/amd64 and arm64 verified); native menu unchanged.
- Bad: a WebView2 process tree (~100 MB, **no verificado** on this machine) lives while the
  flyout exists after the first open; acceptable for instant reopen, revisit if users complain.
- Risks (from go-webview2's code): `log.Fatal` on some WebView2 init failures would end the
  tray process (daemon and tunnels keep running); runtime presence is checked first to make
  this unlikely. `WebView2Loader.dll` is looked up by name before the embedded copy (DLL
  search order: the tray's folder). fyne.io/systray reads the tap handler without a lock; we
  write it once at setup and at most once more on failure.
- macOS keeps the native menu until an NSPopover anchored to the status item is possible
  (fyne.io/systray does not expose it). Linux keeps the native menu (no reliable click
  position on StatusNotifierItem hosts; WebKitGTK needs cgo).

## Confirmation

- Go tests (`cmd/kongtrol-tray/flyout_test.go`): loopback-only URL, request filter, injected
  script, message parsing with nonce, per-OS selection, placement for the four taskbar edges,
  auto-hide, negative coordinates, small work areas, DPI scaling.
- `web/ui` tests (`src/pages/TrayPage.test.tsx`, `src/tray/model.test.ts`): Four States,
  switch flow, failures, offline, bridge messages, Escape, ES/EN.
- Manual on Windows: open/close by click, blur, Escape; position with the taskbar on each
  edge and on a second monitor; 100/150/200 % scaling; Narrator reads switches and states.
