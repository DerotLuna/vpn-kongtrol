<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0072: Settings changes reach every open page at once; the tray flyout signs in again after a daemon restart

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Owner feedback testing v0.7.0-rc.2 and v0.6.0, 2026-11-20:

- A change made in the panel (theme, language, profiles, groups) does not reach the tray flyout
  until it is closed and opened again. The flyout is a hidden page that is shown and hidden, never
  reloaded: it read the preferences once and refetched only groups and security on each show; the
  WebSocket carried only tunnel metrics.
- After an update the flyout showed "Desconectada · No se pudo" for a VPN that was connected. Browser
  sessions live in the daemon's memory (ADR-0039/0062), so a new daemon process (upgrade, restart)
  does not know the hidden page's session: every request answers 401, the page kept its last cached
  states, and its "No se pudo" note never cleared.
- On open, the first switch showed a focus ring (#41).

## Decision Drivers

- No new connection, no polling: reuse the one WebSocket each page already has (ADR-0038).
- Keep `kongtrol up`'s live view (bare snapshot map over the same WebSocket) unchanged.
- Never show stale states as if live; tell the user what is happening.

## Considered Options

1. Poll preferences and every resource on a timer in the flyout.
2. A second WebSocket or Server-Sent Events stream for change notices.
3. Opt-in envelopes on the existing WebSocket (`?events=1`) with a data-free "changed" frame.

## Decision Outcome

Chosen option 3.

- **Daemon** (`internal/api/changes.go`, `ws.go`, `server.go`): every successful (2xx) mutating
  request through `secureHandler` bumps a revision in a `changeHub` (`serveAnnounced`, after the
  audit wrapper). A WebSocket opened with `?events=1` gets envelopes — `{"type":"metrics",
  "tunnels":{…}}` and `{"type":"changed","rev":N}` — and wakes on both the collector and the hub;
  without the parameter frames stay the bare map. Wake-ups coalesce (buffer 1). Changes made by the
  daemon itself (scheduler, kill switch state) are not announced here; tunnel states keep arriving
  as metrics.
- **Page** (`web/ui/src/data/live.ts`, `app/PreferencesProvider.tsx`): the panel and the flyout ask
  for events; a "changed" frame refetches every query except tunnels and fires `kongtrol:changed`;
  the preferences are read again on that event and on every `kongtrol:shown`, so theme and language
  follow at once.
- **Flyout after a restart** (`pages/TrayPage.tsx`, `tray/bridge.ts`; Go side in ADR-0074): when the
  page's session is gone (auth state `unauthorized`), it hides the rows, says "Kongtrol se reinició:
  volviendo a conectar esta vista…" and posts the new closed-list message `reauth` once per show; the
  tray reloads the page with a fresh one-time nonce. In a browser tab the page says to reopen it from
  the Kongtrol icon. A "No se pudo" note also clears as soon as the feed shows the profile up.
- **Focus on open** (#41): the window itself takes focus on show (`tabIndex=-1`, no ring); a control
  that asks for it (`data-autofocus`: quit choices, Reintentar, Agregar VPN) still gets it.

### Consequences

- Good: one source of truth, instant updates across the panel's tabs and the tray, no extra
  requests while nothing changes, no stale "Desconectada" after an update.
- Neutral: a change refetches the visible resources of every open page (a handful of small local
  requests); data-free notices keep the WebSocket's payload unchanged.

## More Information

- Code: `internal/api/{changes.go,ws.go,server.go}`, `web/ui/src/{data/live.ts,app/PreferencesProvider.tsx,pages/TrayPage.tsx,tray/bridge.ts,theme/tray.css}`.
- Tests: `internal/api/ws_test.go` (`TestHandleWebSocket_EventsEnvelope`, `TestServeAnnounced`),
  `web/ui/src/data/data.test.tsx`, `web/ui/src/pages/TrayPage.test.tsx`.
- Docs: `docs/API.md` (WebSocket frames), `AGENTS.md` (Dashboard, Tray flyout).
- Refines: [ADR-0036](0036-tray-flyout-window.md), [ADR-0038](0038-new-dashboard-architecture-and-patterns.md), [ADR-0062](0062-origin-bound-sessions-token-rotation-listener-owner.md).
