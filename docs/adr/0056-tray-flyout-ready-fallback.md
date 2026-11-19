<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0056: Tray flyout waits for "ready" on every show, else the native menu

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

Refines [ADR-0036](0036-tray-flyout-window.md) (which stays accepted).

## Context and Problem Statement

In v0.6.0-rc.5/6 the Windows flyout opened as an empty rounded frame (#40). The page had
loaded and rendered, but WebView2 never painted it: the controller's `IsVisible` was not set
when the hidden window was shown ([research note](../research/2026-11-19-tray-flyout-empty-window.md)).
The tray had no way to notice: the page sent `ready` once when it loaded, whether anyone could
see it or not, and the tray only used it to choose between refresh and reload.

## Decision Drivers

- Never leave a non-technical user in front of an empty window (AGENTS.md "User guidance").
- The native menu (ADR-0037) already does everything the flyout does.
- Keep the bridge's closed message list (ADR-0036); no new message.
- Log the cause so a report can be diagnosed from `tray.log`.

## Considered Options

1. Fix `IsVisible` only.
2. **Fix `IsVisible`, and make every show wait for a `ready` sent only when the page is on
   screen; without it, fall back to the native menu.**
3. A native "loading" placeholder inside the flyout window.

## Decision Outcome

Option 2.

- The tray sets the WebView2 controller visible on every show and hidden on every hide.
- `ready` now means "on screen": `web/ui` sends it (`announceReady`, `src/tray/bridge.ts`)
  when the document is `visible` and a frame was painted (next animation frame), after the
  first load and again after every `kongtrol:shown`.
- Every show gets a number (`flyoutWatch`, `cmd/kongtrol-tray/flyout.go`); if the latest show
  has no `ready` after `flyoutReadyTimeout` (3 s) while the window is still open, the tray
  closes the flyout, opens the native menu at the cursor and logs one line with the cause
  (never finished navigating / navigated but never on screen / stopped answering). The left
  click then opens the native menu until the tray restarts, so the user never gets the empty
  window twice.
- Click mapping unchanged and verified in fyne.io/systray's source: left = flyout
  (`SetOnTapped`), right = native menu.

### Consequences

- Good: the empty-window class of failures (not painted, script error, daemon not answering)
  now ends in a working menu within 3 s and a log line.
- Bad: opening the native menu relies on fyne.io/systray's private callback message
  (`WM_USER + 1`), like the existing window-class lookup; re-check on upgrades.
- Bad: a tray paired with an older daemon whose page sends `ready` only once falls back on the
  second open. Tray and daemon ship together, so only a stale daemon hits it, and the result is
  the native menu, not a broken window.

## More Information

- Code: `cmd/kongtrol-tray/flyout.go` (`flyoutWatch`, `flyoutReadyTimeout`),
  `cmd/kongtrol-tray/flyout_windows.go` (`showNow`, `hideNow`, `giveUp`),
  `web/ui/src/tray/bridge.ts` (`announceReady`), `web/ui/src/pages/TrayPage.tsx`.
- Research: [tray flyout empty window](../research/2026-11-19-tray-flyout-empty-window.md).
