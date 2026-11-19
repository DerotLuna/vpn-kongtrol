<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: authenticating a browser to a local daemon's web UI

Version 1.0.0 · 2026-11-18

## Question

`GET /` and `GET /app/` set the `kongtrol_control` cookie with the full control token from
`~/.kongtrol/api-token`, with no credential and no Host check. Any local process (or another
user on the same machine) could fetch the page, read `Set-Cookie` and then call every API
route (`POST /api/v1/data/purge`, …). How do mature local web UIs hand a browser its
credential, and what do OWASP session and CSRF guidance require?

## Root cause (code, verified)

`internal/api/server.go` (`mux.Handle("/", …)` and the `/app/` handler) called
`setControlCookie` (`internal/api/ui_assets.go`) on every page request. `secureHandler` only
checked Host and auth under `/api/`. `curl -i http://127.0.0.1:9741/` was enough.

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | OWASP Session Management Cheat Sheet | https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html | 2026-11-18 |
| 2 | OWASP CSRF Prevention Cheat Sheet | https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html | 2026-11-18 |
| 3 | OWASP ASVS 4.0.3, V3 Session Management | https://github.com/OWASP/ASVS/blob/v4.0.3/4.0/en/0x12-V3-Session-management.md | 2026-11-18 |
| 4 | Jupyter Server, security docs | https://jupyter-server.readthedocs.io/en/latest/operators/security.html | 2026-11-18 |
| 5 | Syncthing FAQ (GUI listen address, Host check) | https://docs.syncthing.net/users/faq.html | 2026-11-18 |

## Findings

- **Session ids** (1): at least 64 bits of entropy from a CSPRNG; cookies with `HttpOnly` and
  `SameSite=Strict` (preferred); idle **and** absolute timeouts enforced on the server; a new id
  after any privilege change (login); never accept session ids from URLs ("might disclose the
  session ID" through history, logs, Referer).
- **CSRF** (2): `SameSite` is defense in depth, not the only control; verify `Origin` when present
  and do not trust a missing Origin or `Origin: null`; reject simple content types (text/plain,
  form) on JSON APIs so cross-site requests need a preflight; login CSRF is a real case.
- **ASVS V3** (3): V3.1 fundamentals (no tokens in URLs), V3.2 session binding (3.2.2 "at least 64 bits of entropy"), V3.3 termination (3.3.1 "logout and expiration invalidate the session token"),
  V3.4 cookie attributes, V3.7 defenses. Not every level-2/3 item
  applies to a loopback-only UI without TLS (`Secure` cookie needs HTTPS).
- **Jupyter** (4): the token goes in the URL once (`/tree/?token=…`) or in the login form; after
  that "a cookie will be set in your browser and you won't need to use the token again" — the
  long-lived token itself travels in the URL.
- **Syncthing** (5): listens on 127.0.0.1 by default "for security reasons"; since 0.14.6 checks
  the Host header when bound to localhost, which "protects against most forms of DNS
  rebinding". Remote access needs a password and HTTPS.

## Design reasoned from this (Kongtrol)

- Improve on Jupyter: the URL carries a **one-time nonce** (60 s, single use), not the long-lived
  token, and in the **fragment** (`#k=`), which browsers do not send to the server or in
  Referer (RFC 3986 §3.5; `Referrer-Policy: no-referrer` is set anyway). The page strips it with
  `history.replaceState`.
- The nonce is minted only by a native client holding the file token (`POST
  /api/v1/session/nonce`, header auth); the page trades it for a **separate** random session id
  (`POST /api/v1/session`, Host + Origin required) in an `HttpOnly; SameSite=Strict; Path=/`
  cookie. The file token is never accepted from a cookie.
- Sessions: in memory (a daemon restart ends them), 256-bit ids, 12 h idle (sliding), 7 days
  absolute, at most 64; only SHA-256 digests are stored (lookups compare digests, so timing says
  nothing about the secret). Failed exchanges are rate-limited (10/min).
- Host is validated for every route, pages included (as Syncthing), so DNS rebinding cannot read
  them either.
- `Secure` is not set: the API is loopback-only HTTP by design (non-loopback binds are refused by
  `config.ValidateDashboardBind`). **Reasoned**, not an OWASP default.

## Confidence

High for the OWASP items (primary docs). Medium for how every OS URL handler passes a fragment:
`cmd /c start` and `rundll32 url.dll,FileProtocolHandler` with `http://…#k=…` are **no
verificado** on every Windows version; without the fragment the page shows how to open it from
Kongtrol (no silent failure).

## Open questions

- Windows URL handlers and fragments ([#39](https://github.com/DerotLuna/vpn-kongtrol/issues/39)).

## Used by

- ADR-0039
