<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0062: Origin-bound browser sessions, a new API token on every daemon start, and a listener-owner check before sending it

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

Refines [ADR-0039](0039-browser-sessions-one-time-nonce.md) (the nonce bootstrap stays; the cookie
alone no longer authenticates) and [ADR-0040](0040-local-hardening-review-2026-11.md). Research:
[2026-11-19-loopback-cookies-and-listener-owner](../research/2026-11-19-loopback-cookies-and-listener-owner.md).

## Context and Problem Statement

An independent review (2026-11-19) confirmed two medium findings and three low ones on a machine
shared by several OS users:

1. `kongtrol_session` (HttpOnly, SameSite=Strict) is sent by browsers to **every port** of
   127.0.0.1: cookies are not isolated by port (RFC 6265 §8.5). `authenticated()` accepted the
   cookie alone, so another local user's server on another loopback port that the browser visits
   receives it and can replay it against the daemon.
2. The CLI and the tray send the long-lived control token (`X-Kongtrol-Token`, created once) to
   whatever listens on 127.0.0.1:9741. While the daemon is stopped (the tray keeps polling after
   `down`), another local user can bind that port and collect the token.
3. Low: the page CSP allowed `connect-src ws: wss:` (any host); profile names after
   `daemon --restart` were not separated from flags; the elevated `.conf` import held the config
   lock for up to 3 min while `pkexec` waited, past the server's 30 s `WriteTimeout`.

## Decision Drivers

- Keep ADR-0039's properties: the token never reaches a browser; the tray, `kongtrol dashboard`,
  the live view's "o" and the tray flyout keep opening a signed-in page with one click.
- Fail closed when another user's program is on the port; never break the app when the OS cannot
  say who owns it.
- No admin rights, no cgo in the CLI, standards first (RFC 6265, Fetch, CSP3, OWASP).

## Considered Options

Browser sessions:

1. Keep the cookie alone and add the port to the cookie: impossible, cookies have no port attribute.
2. Drop the cookie, keep only a header token: works, but an XSS-free page gains nothing from losing
   the HttpOnly half, and SameSite=Strict still helps against cross-site navigation.
3. **Cookie + per-page session token in a header**; the token lives in `sessionStorage`, which is
   per origin (scheme, host **and port**) per the HTML storage model, so a server on another port
   can never read it. WebSocket: a one-time ticket, because browsers cannot add headers to it.

API token:

1. Owner check only: no protection where the OS gives no answer (macOS without root).
2. Rotation only: the token still leaks once to a squatter that is there when a client asks.
3. **Both**: rotate on every daemon start; before sending the token, check who owns the other end.

## Decision Outcome

Chosen: option 3 for both.

**Sessions** (`internal/api/session.go`, `web/ui/src/api/auth.ts`, `client.ts`, `data/live.ts`).
`POST /api/v1/session {"nonce"}` sets the same cookie and answers `{"session_token"}` (256 bits,
`Cache-Control: no-store`). Every API request needs the cookie **and** a matching
`X-Kongtrol-Session` (SHA-256 digests compared in constant time). A browser that already holds a
live cookie keeps it on the next exchange and gets one more token (at most 16 per cookie), so a
second tab or the tray flyout does not sign out the first. The page keeps its token in
`sessionStorage`: it survives reloads of that tab; a new tab opened without `#k=` has none, gets
401 and shows the existing "open it from Kongtrol" guidance. Only `api/auth.ts` touches it.
WebSocket: `POST /api/v1/ws-ticket` (cookie + header) returns a ticket valid 30 s, single use,
bound to that session; `GET /api/v1/ws/metrics?ticket=…` needs the daemon's `Origin`, the cookie
and the ticket. Native clients keep sending the file token, also on the WebSocket. The cookie
stays without `Secure`: the page is plain http on loopback and Safari still drops Secure cookies
there (WebKit bug 281149; Chromium and Firefox accept them on localhost), so the CodeQL
`go/cookie-secure-not-set` alerts are dismissed with "loopback http, Secure would break Safari;
protected by the Host check and the per-session header".

**Token rotation** (`config.RotateAPIToken`, `cmd/kongtrol/daemon_cmd.go`). The daemon writes a
fresh `~/.kongtrol/api-token` (0600, temp file + rename, `config.WriteFileAtomic`) after it found
no daemon answering and before it listens. Clients read the file per request
(`daemonctl.Client.TokenSource = config.CurrentAPIToken`; the CLI, the tray, the flyout's
`PanelURL` and the live view's WebSocket), so `down`/`up`, `kongtrol restart` and the dashboard's
restart (`daemon --restart`, which rotates when it starts) keep working without restarting the tray.

**Listener-owner check** (`internal/sockowner`, `internal/daemonctl/owner.go`). Every connection
that carries the token goes through `daemonctl.DialContext` (`HTTPClient`, the live view's
WebSocket dialer). After connecting and before the first byte it looks up the server end of that
very connection (falling back to the LISTEN row of the port):

| OS | How (no admin) | Verdict |
|---|---|---|
| Windows | `GetExtendedTcpTable(TCP_TABLE_OWNER_PID_ALL)` → owning PID → `OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` + `OpenProcessToken(TOKEN_QUERY)` → user SID | same SID → send; other SID → refuse; access denied → send only if the PID is the one in `~/.kongtrol/run/kongtrol.pid` (this user's elevated daemon), else refuse |
| Linux | `/proc/net/tcp{,6}` uid column | own uid or root → send; other uid → refuse |
| macOS | none verified (`lsof` lists only this user's processes without root; libproc needs cgo) | unknown → send; rotation covers it |

A refused connection returns `*daemonctl.ForeignListenerError` with a translated "what to do"
line (`daemon.err.foreign_listener[.fix]`): ask that user to close the program or change
Kongtrol's port (`kongtrol config dashboard set-port`). `Spawner.Base` makes `up`, `restart` and
the tray report it instead of starting a daemon that cannot bind. When the lookup fails (table
unreadable, row not found) the client proceeds and logs at debug level: the app keeps working and
the token still rotates on the next start.

**Low findings.** CSP `connect-src 'self' ws://<Host>` (Host already checked to be loopback with
the daemon's port) instead of `ws: wss:`: CSP3 says `'self'` covers the `wss:` variant, but
browsers differed on `ws:` (MDN, WebKit bug 201591), so the one origin is named. Replacement
arguments are `daemon … --restart -- <names>` and names failing `config.ValidateName` are left
out (the replacement validates kongtrol.yaml again and drops unknown profiles). The import route
no longer runs under the config lock: the file is read (and an elevated read waits) without it,
the lock is taken only to store the file, and the elevated path extends its own write deadline to
4 min with `http.ResponseController` (only that request; the UI flow is unchanged).

### Consequences

- Good: a captured cookie is worthless without the page token; a squatter on the port gets no
  token on Windows/Linux and only a token that dies on the next start on macOS.
- Good: no UX change for the normal paths (tray, flyout, `kongtrol dashboard`, reloads).
- Bad: a dashboard tab opened by hand (bookmark, new tab, copied URL without `#k=`) must be opened
  from Kongtrol again; `make ui-dev` needs the `#k=` address with the dev port.
- Bad: a tray or CLI older than this version holds the token it read at launch and gets 401 after
  a new daemon starts; upgrades replace both binaries, and restarting the tray fixes it.
- Neutral: one file read and one TCP-table lookup per new connection (keep-alive reuses checked
  connections).

### Confirmation

Go tests: `TestSessionCookieNeedsPageToken`, `TestSecondPageKeepsFirstSignedIn`,
`TestWebSocketNeedsOneTimeTicket`, `TestWSTicketRequiresSessionHeader`,
`TestCSPNamesOnlyThisWebSocketOrigin` (`internal/api`); `TestClientRefusesForeignListener`,
`TestClientFollowsRotatedToken`, `TestSpawnerRefusesForeignPort` (`internal/daemonctl`);
`internal/sockowner` (fakes per OS and a real loopback socket, also run on Linux/WSL);
`TestRotateAPIToken`; `TestReplacementArgs_NamesAreNeverFlags`;
`TestImportElevatedReadDoesNotHoldConfigLock`. web/ui: `api/auth.test.ts`, `data/data.test.tsx`.
A headless Edge smoke test against a test daemon (temp HOME, port 19741) confirmed sign-in, API
and WebSocket (101) with the header and ticket, 401 for the replayed cookie alone, survival of a
reload, guidance in a new tab without `#k=`, and the token changing on every start and on
`kongtrol restart`.

## More Information

- Not verified: macOS has no owner check; another user's listener on Windows that this user may
  inspect is compared by SID, but a same-user elevated daemon is only recognized through the PID
  file, which the daemon writes right after it starts listening (a non-elevated client asking in
  that instant is refused for that request). Whether a standard user can open a same-user
  elevated process with `PROCESS_QUERY_LIMITED_INFORMATION` + `TOKEN_QUERY` was not tested here.
- Code: `internal/api/session.go`, `internal/api/server.go`, `internal/api/handlers_vpn_import.go`,
  `internal/config/control.go`, `internal/sockowner/`, `internal/daemonctl/owner.go`,
  `cmd/kongtrol/daemon_cmd.go`, `cmd/kongtrol/daemon_restart.go`, `web/ui/src/api/auth.ts`.
