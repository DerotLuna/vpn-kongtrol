<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: tray flyout sometimes opens slowly (#41)

Version 1.0.0 · 2026-11-20

## Question

The Windows tray flyout (ADR-0036, ADR-0056) sometimes takes a while to open. Where does the
time go on a click, cold (first open after the tray starts) and warm (later opens), and what can
the tray do about it without touching the page (`web/ui` `/tray`)?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Microsoft Learn — Performance best practices for WebView2 apps ("Cold start", "Avoid redundant WebView2 instances", "Use memory management APIs") | https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/performance | 2026-11-20 |
| 2 | github.com/jchv/go-webview2 `pkg/edge` at v0.0.0-20260205173254-56598839c808: `chromium.go` (`Embed` pumps messages until environment + controller exist; `Show`/`Hide` = `put_IsVisible`), `ICoreWebView2NavigationCompletedEventArgs.go` (vtable has `GetIsSuccess`, no Go wrapper) | https://github.com/jchv/go-webview2/tree/56598839c808/pkg/edge | 2026-11-20 (module cache) |
| 3 | Kongtrol code before the change: `cmd/kongtrol-tray/flyout_windows.go`, `flyout.go`, `tray_actions.go` (`ensureDaemon`), `internal/daemonctl/client.go` (`Probe`), `internal/daemonctl/panel.go` (`PanelURL`), `internal/api/session.go` (sessions in memory) | this repository, commit 1cdd13b | 2026-11-20 |
| 4 | Local measurement: `KONGTROL_FLYOUT_SMOKE=1 go test -run TestFlyoutWindowSmoke ./cmd/kongtrol-tray/` (real WebView2, stub page on 127.0.0.1), Windows 11 Pro 10.0.26300 | — | 2026-11-20 |

## Findings

- **Microsoft [1]:** "A common performance bottleneck is the time to create a WebView2 control
  and navigate to a webpage for the first time. On a cold launch, WebView2 must spin up its
  processes and disk caches, which can introduce a noticeable delay". It recommends reusing one
  WebView2 instead of destroying and re-creating it, and, for an instance not used for a while,
  `MemoryUsageTargetLevel = Low` or `TrySuspendAsync`.
- **What the tray did on each left click [3]** (line numbers at commit 1cdd13b):
  - `onTrayClick` (`flyout_windows.go:238`) started `open()` in a goroutine. `open()`
    (`:254`) always ran `ensureDaemon()` first (`tray_actions.go:77`): a mutex shared with
    every other menu action, then `client.Probe()`, an HTTP `GET /api/v1/tunnels` with an
    800 ms timeout through the listener-owner check (`internal/daemonctl/client.go:116`).
    That round trip ran on **every** click, warm ones too, before the window showed.
  - The window, the WebView2 environment and the controller were created **lazily, on the
    first click** (`startOnce.Do(go f.run())`, `:261`; `run` `:272`; `Embed` `:302`), then
    kept for the life of the process and only shown/hidden. So only the first open was cold.
  - On that cold open the page was navigated (`navigate` `:390`, which asks the daemon for a
    one-time sign-in nonce, `PanelURL`, `POST /api/v1/session/nonce`, 3 s timeout) and the
    window was shown right away (`:311`), before the page had loaded: the user saw the
    background colour until the page painted.
  - Warm opens did not reload the page or request a nonce: `showNow` (`:438`) only sent
    `kongtrol:shown` when the page had said `ready` before. It re-navigated only when the page
    had never been ready and the last navigation was older than 3 s.
  - The page's session lives in the daemon's memory (`internal/api/session.go`), so after a
    daemon restart (upgrade, `kongtrol restart`) a page loaded earlier is signed out; the tray
    did not notice.
- **Measured [4]** with the changed code (stub page, not the real `/tray` bundle):
  WebView2 environment + controller 639 ms, page load 106 ms, show to `ready` 83 ms. This is
  one run on one machine with the runtime already in the disk cache; a first run after boot,
  slower machines and the real page are **no verificado**. It is consistent with [1]: the
  controller start dominates the cold open.
- `get_IsSuccess` of the navigation-completed args is in go-webview2's vtable but not wrapped
  [2]; calling it through the vtable (IUnknown's 3 slots, then `IsSuccess(BOOL*)`) worked in
  the smoke test (success `true` for each load).
- go-webview2 at this version has no `MemoryUsageTargetLevel` (ICoreWebView2_19) at all and
  lists `TrySuspend` in the ICoreWebView2_3 vtable without a Go wrapper (it needs a
  completion handler) (checked by searching the module source) — neither is used yet.

## Conclusions

- Pre-warm: create the hidden window and controller and load `/tray` once the daemon first
  answers the tray's status poll, so the first click only shows the window ([1] reuse; ADR-0074).
- Skip the per-click probe when the page is loaded and a status poll answered in the last 5 s.
- Reload the page with a fresh nonce when the daemon restarted (api-token changed) or when the
  page reports its session is gone (`reauth`), rate-limited.
- Real timings on users' machines are **no verificado**; `tray.log` now records them.
