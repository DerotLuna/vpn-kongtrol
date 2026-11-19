<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: tray flyout opens as an empty frame (#40)

Version 1.0.0 · 2026-11-19

## Question

On Windows 11 with v0.6.0-rc.5/6 one click on the tray icon opens the flyout (ADR-0036) as a
translucent rounded frame with no content. Why, which click opens what, and how should the tray
react when the page does not show up (ADR-0056)?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | fyne.io/systray v1.12.2 — `systray_windows.go` (`WndProc`, `systrayLeftClick`, `systrayRightClick`, `wmSystrayMessage = WM_USER + 1`, `showMenu`) | https://github.com/fyne-io/systray/blob/v1.12.2/systray_windows.go | 2026-11-19 |
| 2 | github.com/jchv/go-webview2 `pkg/edge/chromium.go` (`Embed`, `Show`/`Hide` = `ICoreWebView2Controller.put_IsVisible`, `NavigationCompletedCallback`) at v0.0.0-20260205173254-56598839c808 | https://github.com/jchv/go-webview2/blob/56598839c808/pkg/edge/chromium.go | 2026-11-19 |
| 3 | Microsoft Learn — ICoreWebView2Controller `IsVisible` | https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2controller | not reachable from this environment |
| 4 | Local measurement: WebView2 Runtime 157.0.4341.60, Windows 11 Pro 10.0.26300, the real flyout code (`winFlyout.run`) against a daemon at 127.0.0.1 with a diagnostic script reporting `document.visibilityState`, `#root` children and errors | — | 2026-11-19 |

## Findings

- **Root cause, measured [4]:** the page loads and renders completely (the DOM had the six
  VPNs, the groups and the bridge; no script error, no CSP violation, no blocked request, no
  401), but `document.visibilityState` stayed `"hidden"` after the window was shown, so
  WebView2 never painted it: the user sees only the DWM rounded frame. The flyout window is
  created hidden (no `WS_VISIBLE`) and shown later with `SetWindowPos(SWP_SHOWWINDOW)`; the
  tray never set the controller's `IsVisible` [2]. Calling `put_IsVisible(TRUE)` after showing
  the window made the same page `"visible"` at once and it painted. Hiding the window now also
  sets `IsVisible(FALSE)`.
- The URL, the 403 filter, the ADR-0039 nonce flow, the CSP and the `/tray` route after
  ADR-0043 were all fine [4]: none of them was involved.
- Microsoft's wording on `IsVisible` (default, and whether it tracks the parent window): **no
  verificado**, the page could not be fetched [3]; the behaviour above is measured, not read.
- **Which click:** fyne.io/systray maps the icon's `WM_LBUTTONUP` to the `SetOnTapped` handler
  and `WM_RBUTTONUP` to `SetOnSecondaryTapped`, falling back to the native menu when a handler is
  nil [1]. Kongtrol sets only `SetOnTapped` (the flyout), so left = flyout and right = native
  menu, as ADR-0036 says. With "swap primary and secondary buttons" on, Windows reports the
  logical buttons, so the primary click still opens the flyout (reasoned, **no verificado**).
  The owner's "right click" report could not be reproduced; the code has no path from a right
  click to the flyout.
- **Opening the native menu programmatically:** fyne.io/systray exports no "show menu" call;
  its right-click path runs `showMenu` (TrackPopupMenu at the cursor). Posting its own callback
  message (`WM_USER + 1`, lParam `WM_RBUTTONUP`) to its window runs that path [1]. This depends
  on a private constant of v1.12.2, like the existing `SystrayClass` / icon id lookups, so it
  must be re-checked when systray is upgraded.
- go-webview2 exposes `NavigationCompletedCallback` but not `IsSuccess` / `WebErrorStatus` on
  its args type [2], so the fallback's log can say whether a navigation finished, not whether
  it failed.

## Confidence

High for the root cause (reproduced and fixed on the real code path, negative control: with
`IsVisible(FALSE)` the page never answers and the fallback runs). Medium for the click mapping
(read in the library source, the owner's report not reproduced).

## Open questions

- Microsoft's documented default for `IsVisible` with a hidden parent: re-check [3] when
  reachable.

## Used by

- ADR-0056
