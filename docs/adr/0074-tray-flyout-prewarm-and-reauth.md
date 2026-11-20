<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0074: Tray flyout is pre-warmed, opens without a round trip and signs in again on its own

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer (owner)

Refines [ADR-0036](0036-tray-flyout-window.md) and [ADR-0056](0056-tray-flyout-ready-fallback.md)
(both stay accepted).

## Context and Problem Statement

The Windows flyout sometimes opens slowly (#41). Reading the code
([research note](../research/2026-11-20-tray-flyout-open-latency.md)): the window and the
WebView2 controller were created on the first left click, so the first open paid the WebView2
cold start (Microsoft names it the usual bottleneck) plus the page load, and showed the window
before the page was there. Every click, warm ones too, first probed the daemon over HTTP
(`ensureDaemon`, up to 800 ms, behind a mutex shared with the menu actions). And because
browser sessions live in the daemon's memory, a page loaded before a daemon restart stayed
signed out (every request 401, stale tunnel states).

## Decision Drivers

- Warm open almost instant (target under ~150 ms), cold open under ~1 s (#41).
- Keep ADR-0056: `IsVisible` follows show/hide, `ready` after every show, 3 s fallback.
- Measure before optimising further: timings in `tray.log` without flooding it.
- Microsoft: reuse one WebView2 rather than re-creating it; the cold start is the cost.

## Considered Options

1. Keep the lazy start; only remove the per-click probe.
2. **Pre-warm at tray start, skip the probe when the daemon is known up, reload with a fresh
   nonce when the daemon restarted or the page says its session is gone.**
3. Pre-warm and also suspend the hidden page (`TrySuspend`, memory target "low").

## Decision Outcome

Option 2. Option 3 is not wrapped by the WebView2 binding we use (go-webview2) and needs
measurements first; revisit if memory becomes an issue.

- **Pre-warm.** After the first tray status poll the daemon answers (`noteDaemonAnswered`,
  every 3 s), the tray creates the flyout window hidden, starts WebView2, sets the controller
  hidden (it follows the window) and loads `/tray` with a one-time nonce. A click before that
  still starts it (as before). Clicks then only show/hide.
- **No round trip on a warm click.** When the page is loaded (last navigation succeeded,
  read from `NavigationCompleted` `IsSuccess`) and a status poll answered in the last 5 s, the
  click shows the window at once; otherwise it runs `ensureDaemon` as before (so after
  `kongtrol down` a left click still starts the daemon).
- **Show.** Loaded page → `kongtrol:shown` (refresh, focus, `ready` again; a pre-warmed page
  that was never shown handles it too). Load still running → wait for `ready` (the page
  announces it when it becomes visible). Load failed more than 3 s ago → load again.
- **Daemon restarted.** The api-token is rewritten on every daemon start; the tray remembers
  a short hash of it (`tokenTag`, never the token) at each navigation. A different token at a
  show, or after a status poll while hidden, reloads the page with a fresh nonce.
- **`reauth`.** New message in the closed list (`flyoutMessages`, mirrored in
  `web/ui/src/tray/bridge.ts`): the page posts it when its session is gone. The tray reloads
  `/tray` with a fresh nonce through the same `flyoutLoadURL` path as the first load, without
  hiding the window, at most once every 5 s (`flyoutReauthAllowed`). If the window is visible
  the reloaded page must say `ready` within 3 s like any show (ADR-0056).
- **Timing in `tray.log`.** One line per open: cold always, warm only above 150 ms
  (`flyoutOpen.logLine`): click to shown, shown to ready, total. Also logged once per load:
  WebView2 start time and each page load time with its result. The tray has no debug level,
  so this threshold replaces one instead of adding a setting.

### Consequences

- Good: the first click no longer pays the WebView2 start; warm clicks do no network call
  before showing; a daemon restart no longer leaves a signed-out flyout.
- Good: real timings from users' machines end up in `tray.log`.
- Bad: WebView2's processes (browser, renderer, GPU) now run from tray start even for users
  who never left-click (memory cost no verificado on our machines).
- Bad: `IsSuccess` is read through go-webview2's vtable (not wrapped); re-check on upgrades,
  like the systray private message of ADR-0056.
- Bad: a hidden page's timers are throttled by the browser; `kongtrol:shown` refreshes its
  data on every show.

## More Information

- Code: `cmd/kongtrol-tray/flyout.go` (`noteDaemonAnswered`, `flyoutNeedsDaemonCheck`,
  `tokenTag`, `flyoutOnShow`, `flyoutOpen`, `flyoutReauth`, `flyoutReauthAllowed`),
  `cmd/kongtrol-tray/flyout_windows.go` (`prewarmFlyout`, `open`, `showNow`, `reauth`,
  `checkDaemon`, `navigationSucceeded`), `cmd/kongtrol-tray/tray_menu.go`
  (`updateTrayStatus` calls `noteDaemonAnswered`).
- Page side (`reauth`, bridge list): `web/ui/src/tray/bridge.ts`, `web/ui/src/pages/TrayPage.tsx`.
- Research: [tray flyout open latency](../research/2026-11-20-tray-flyout-open-latency.md).
