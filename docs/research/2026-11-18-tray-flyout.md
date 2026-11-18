<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: a branded flyout window next to the tray icon

Version 1.0.0 · 2026-11-18

## Question

Mature desktop apps open a small window next to the tray icon on a left click (Windows 11
network/volume flyouts, Tailscale on macOS, Docker Desktop, 1Password mini) instead of a plain
native menu. How can Kongtrol's tray (Go, `fyne.io/systray`, pure Go on Windows and Linux,
cgo only on macOS) do the same on Windows, macOS and Linux, without breaking the CGO-free
cross-compiled release builds, and keeping "local only" and accessibility?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Wails v3 docs — system tray, attached window | https://v3.wails.io/features/menus/systray | 2026-11-18 |
| 2 | Wails v3 beta announcement (status: beta, not final) | https://docs-preview.wails.io/blog/wails-v3-beta/ | 2026-11-18 |
| 3 | `fyne.io/systray` v1.12.2 source (`SetOnTapped`, `SetOnSecondaryTapped`, `systray_windows.go` window class `SystrayClass`, icon id 100) | https://github.com/fyne-io/systray/tree/v1.12.2 | 2026-11-18 |
| 4 | `github.com/jchv/go-webview2` source and README (pure Go WebView2 binding, embedded `WebView2Loader.dll` via go-winloader, MIT) | https://github.com/jchv/go-webview2 | 2026-11-18 |
| 5 | Microsoft — Distribute your app and the WebView2 Runtime (Evergreen preinstalled on Windows 11; installed on eligible Windows 10, "a small number" without it; detect with the `pv` registry value or `GetAvailableCoreWebView2BrowserVersionString`) | https://learn.microsoft.com/microsoft-edge/webview2/concepts/distribution | 2026-11-18 (search summary; page fetch blocked here) |
| 6 | Microsoft — `Shell_NotifyIconGetRect` (Windows 7+, `NOTIFYICONIDENTIFIER` = hWnd + uID or GUID) | https://learn.microsoft.com/windows/win32/api/shellapi/nf-shellapi-shell_notifyicongetrect | 2026-11-18 (**no verificado**: fetch blocked; signature from the Windows SDK headers as known) |
| 7 | `github.com/webview/webview_go` (C/C++ library, needs cgo on every OS) | https://github.com/webview/webview_go | 2026-11-18 |
| 8 | Tailscale — windowed macOS UI (moved from a menu to a window anchored to the menu bar) | https://tailscale.com/blog/windowed-macos-ui-beta | 2026-11-18 |
| 9 | Tailscale Windows install doc (right-click menu on the tray icon) | https://tailscale.com/docs/install/windows.md | 2026-11-18 |
| 10 | StatusNotifierItem spec (`Activate(x, y)`, `ItemIsMenu`; position is a hint the host may not send) | https://www.freedesktop.org/wiki/Specifications/StatusNotifierItem/ | 2026-11-18 (**no verificado** in this session: from the spec as known and `fyne.io/systray` `systray_unix.go`) |

## Options

| Option | Windows | macOS | Linux | Release build impact | Verdict |
|---|---|---|---|---|---|
| **Wails v3** tray + attached window [1][2] | yes (WebView2) | yes (WKWebView) | yes (WebKitGTK) | Replaces fyne.io/systray and the app loop; cgo on macOS and Linux (GTK/WebKitGTK); beta | Too big a migration for one feature, beta, and breaks the CGO-free Linux tray |
| **webview/webview_go** [7] | yes | yes | yes | cgo everywhere (C++), so the Windows/Linux tray no longer cross-compiles in GoReleaser | Rejected: release pipeline and binary size |
| **jchv/go-webview2 `pkg/edge`** + own Win32 popup [4] | yes | — | — | Pure Go (`CGO_ENABLED=0`), windows/amd64 + arm64; embedded loader DLL | **Chosen for Windows** |
| `lxn/walk` native widgets | yes | — | — | Pure Go | Rejected: no reuse of web/ui (two UIs to keep in sync); project looks unmaintained (**no verificado**) |
| NSPopover (macOS) | — | yes | — | cgo + Objective-C; fyne.io/systray does not expose its `NSStatusItem` button to anchor to | Later, needs either a fork of the tray library or a different tray library |
| Linux | — | — | SNI hosts do not reliably send the click position [10]; a web view needs WebKitGTK (cgo) | — | Native menu stays |

## Findings

- `fyne.io/systray` v1.12 adds `SetOnTapped`/`SetOnSecondaryTapped` [3]. On Windows the left
  click calls the handler and the right click still opens the menu; on Linux setting a handler
  flips `ItemIsMenu`, so we do not set it there.
- Icon position on Windows: `Shell_NotifyIconGetRect` with the tray's hidden window
  (`SystrayClass`, found with `EnumThreadWindows` on systray's thread) and icon id 100 [3][6].
  The taskbar edge comes from the monitor rectangle vs its work area; an auto-hidden taskbar
  falls back to the edge closest to the icon. The cursor position is the fallback (icon in the
  overflow area).
- WebView2 availability [5]: preinstalled on Windows 11, present on most Windows 10. Detected
  with `webviewloader.GetInstalledVersion` (wraps `GetAvailableCoreWebView2BrowserVersionString`);
  when missing the left click keeps the native menu (no download, no prompt).
- go-webview2 caveats found in its code: `log.Fatal` in a few failure callbacks (environment /
  controller creation) and in `AddWebResourceRequestedFilter`; `WebView2Loader` is looked up
  by name with `NewLazyDLL` before the embedded copy (DLL search order); the edge package's
  `init` locks the main thread and initializes COM (STA) there. None blocks use; they are
  listed in the ADR as risks.
- The binding gives no `NavigationStarting` event nor the message source origin, so the page
  path and the message bridge are pinned by a document-created script plus a per-launch nonce
  (see ADR-0036), on top of a request filter limited to the daemon's origin.
- Authentication: the daemon already gives pages it serves an HttpOnly `SameSite=Strict`
  cookie (`internal/api/ui_assets.go`), so the window loads `http://127.0.0.1:<port>/app/tray`
  with no token in the URL, exactly like the dashboard.
- Tailscale on macOS moved to a window anchored to the menu bar [8]; on Windows it keeps a
  right-click menu [9]. Our split (flyout on left click, menu on right click) follows the
  Windows 11 system flyouts.
