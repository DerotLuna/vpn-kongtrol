<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0039: Browser sessions from a one-time nonce; pages never hand out credentials

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

The daemon's pages (`/` legacy dashboard, `/app/` new UI) set the `kongtrol_control` cookie
with the full control token from `~/.kongtrol/api-token` on every request, without any
credential or Host check. Any local process or other local user could `curl` the page, read the
token and call every API route (purge data, change settings, connect tunnels). How does a
browser get API access without the page giving it to anyone who asks?

## Decision Drivers

- Only someone who can already read `~/.kongtrol/api-token` (the user's CLI, tray) may open
  an authenticated browser session.
- OWASP Session Management and CSRF cheat sheets, ASVS V3 (sources in the
  [research note](../research/2026-11-18-local-web-ui-auth.md)).
- Non-technical users: opening the panel from the tray or `kongtrol dashboard` must just work;
  a page opened any other way says what to do.
- Stable API for the other clients (tray flyout, new UI).

## Considered Options

1. Keep the cookie on page load (status quo) — token disclosure to any local process.
2. Long-lived token in the URL (Jupyter `?token=`) — the real credential lands in history.
3. **One-time nonce in the URL fragment, traded for a separate in-memory session cookie.**

## Decision Outcome

Chosen: **option 3**.

- `POST /api/v1/session/nonce` (file token in `X-Kongtrol-Token` / `Authorization: Bearer`
  only; a session cookie is not enough) → `{"nonce": "<base64url, 256 bits>", "expires_in": 60}`.
  Nonces are single use, 60 s, at most 32 pending.
- Clients open `<base><path>#k=<nonce>` built by **one** function,
  `daemonctl.(*Client).PanelURL(path)`: CLI `kongtrol dashboard` and the live view's `o`
  (`daemonPanelURL`), tray "Abrir panel" / first-run Studio (`openDashboardPage`), and the tray
  flyout (`/app/tray`). On error it returns the plain URL.
- The page (`web/dashboard/auth.js`, `web/ui/src/api/session.ts`) strips `#k=` with
  `history.replaceState` and calls `POST /api/v1/session {"nonce": "…"}` — the only `/api`
  route without credentials; Host must be loopback and `Origin` must be present and the
  daemon's own origin; JSON only; failed exchanges are rate-limited (10 per minute, then 429).
  It sets `kongtrol_session=<random 256-bit id>; HttpOnly; SameSite=Strict; Path=/`, distinct
  from the file token. `DELETE /api/v1/session` logs out.
- Sessions live in memory (a daemon restart ends them): 12 h idle (sliding), 7 days absolute,
  at most 64. Only SHA-256 digests of nonces and ids are stored.
- The API accepts the file token in a header or a live session cookie; the file token is never
  accepted from a cookie.
- `/`, `/app/` and every other route validate Host like `/api` (DNS rebinding), and never set
  credentials.
- A 401 shows: "Esta página no tiene permiso para hablar con Kongtrol. Qué hacer: ábrela desde
  Kongtrol, con el ícono de la bandeja («Abrir panel») o con kongtrol dashboard." (legacy
  banner in `auth.js`; new UI `error.unauthorized`).
- `localhost` binds listen on `127.0.0.1` (a hosts file cannot move the listener off loopback);
  non-loopback binds stay refused (`config.ValidateDashboardBind`), so no `Secure` cookie / TLS
  is needed.

### Consequences

- Good: a local process without the file token gets nothing from the pages; the long-lived token
  never reaches a browser, its history or its cookie jar.
- Good: one URL builder; the other clients only call `PanelURL`.
- Bad: a bookmarked `http://127.0.0.1:9741/` works only while its session lives; afterwards the
  page asks to be opened from Kongtrol.
- Bad: if a Windows URL handler dropped the fragment, the page would show that guidance instead
  of signing in (**no verificado** on every Windows version; tracked in an issue).
- Neutral: a daemon restart signs browsers out (open the panel again).

### Confirmation

Tests: `internal/api/session_test.go` (no cookies on pages, Host check, nonce needs the header
token, single use, expiry, idle/absolute timeouts, rate limit, Origin required, file token not
accepted as cookie), `internal/daemonctl/panel_test.go`, `cmd/kongtrol/cli_fixes_test.go`
(`TestDashboard_UsesSharedDaemon`), `web/ui/src/api/session.test.ts`.

## More Information

Code: `internal/api/session.go`, `internal/api/server.go` (`secureHandler`, `authenticated`),
`internal/daemonctl/panel.go`, `web/dashboard/auth.js`, `web/ui/src/api/session.ts`.
Documented in `docs/SECURITY.md` ("Local panel sign-in").
