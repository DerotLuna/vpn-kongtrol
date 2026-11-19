<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Loopback cookies, per-origin storage, CSP for WebSockets and who owns a local port

Version 1.0.0 · 2026-11-19 · Backs [ADR-0062](../adr/0062-origin-bound-sessions-token-rotation-listener-owner.md)

Access date for every source: 2026-11-19.

## Questions

1. Does a browser send a cookie set by 127.0.0.1:9741 to another port of 127.0.0.1?
2. Where can a page keep a secret that another port of the same host cannot read?
3. Do browsers accept `Secure` cookies over http on 127.0.0.1 / localhost?
4. Does CSP `connect-src 'self'` cover `ws://` to the same host and port?
5. How can a process find out, without admin, which user owns a loopback listener?

## Findings

| Source | What it says | Status |
|---|---|---|
| RFC 6265 §8.5 "Weak Confidentiality" — https://www.rfc-editor.org/rfc/rfc6265#section-8.5 | "Cookies do not provide isolation by port. If a cookie is readable by a service running on one port, the cookie is also readable by a service running on another port of the same server." Servers SHOULD NOT run mutually distrusting services on different ports and use cookies for security-sensitive data. | verified |
| RFC 6265bis (draft) same-site definition — https://httpwg.org/http-extensions/draft-ietf-httpbis-rfc6265bis.html | "Same-site" compares scheme and registrable domain (host for IPs), not ports, so SameSite=Strict does not stop a page on 127.0.0.1:8080 from receiving the 127.0.0.1 cookie. | no verificado (read from memory of the draft; consistent with RFC 6265 §8.5) |
| HTML Standard, Web storage — https://html.spec.whatwg.org/multipage/webstorage.html | `sessionStorage` is keyed by the top-level browsing context and the document's origin (scheme, host, port); it survives reloads of the tab and is copied only when a tab is duplicated. | no verificado (page not re-read in this session; long-standing behaviour) |
| WebKit bug 281149 — https://bugs.webkit.org/show_bug.cgi?id=281149 ; WebKit bug 231035 — https://bugs.webkit.org/show_bug.cgi?id=231035 | Safari does not set `Secure` cookies over http://localhost; a March 2026 patch for loopback hosts was discussed, release status unknown. | verified (bug pages); shipping status no verificado |
| httpwg issue 2605 — https://github.com/httpwg/http-extensions/issues/2605 | Chrome allows Secure cookies on localhost; Firefox moved to the Secure Contexts "potentially trustworthy" rule (Mozilla bug 1618113 — https://bugzilla.mozilla.org/show_bug.cgi?id=1618113); Safari had not decided. | verified |
| CSP Level 3 §1.3 — https://www.w3.org/TR/CSP3/ | "'self' now matches `https:` and `wss:` variants of the page's origin, even on pages whose scheme is `http`." The exact `ws:` matching step was not read. | partly verified |
| MDN `connect-src` — https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/connect-src ; WebKit bug 201591 — https://bugs.webkit.org/show_bug.cgi?id=201591 | `'self'` did not resolve to WebSocket schemes in every browser (Safari 13 refused same-host `ws:` under `connect-src 'self'`). Naming the WebSocket origin explicitly avoids the difference. | verified (bug page); current Safari no verificado |
| Microsoft Learn, `GetExtendedTcpTable` — https://learn.microsoft.com/windows/win32/api/iphlpapi/nf-iphlpapi-getextendedtcptable | `TCP_TABLE_OWNER_PID_ALL` lists every TCP endpoint with its owning PID; no elevation is required. Already used by `internal/monitor` (ADR-0054) and run here as a standard user. | verified (doc + test `TestCheckRealLoopback`) |
| Microsoft Learn, `OpenProcessToken`, `PROCESS_QUERY_LIMITED_INFORMATION` — https://learn.microsoft.com/windows/win32/api/processthreadsapi/nf-processthreadsapi-openprocesstoken , https://learn.microsoft.com/windows/win32/procthread/process-security-and-access-rights | `OpenProcessToken` needs `PROCESS_QUERY_LIMITED_INFORMATION`; the token's DACL decides `TOKEN_QUERY`. Another user's process normally denies it. | verified for own processes (test); other-user and same-user-elevated behaviour no verificado |
| Linux `proc(5)`, `/proc/net/tcp` — https://man7.org/linux/man-pages/man5/proc_net.5.html | Each row has `uid`, the effective uid of the socket's creator; readable by every user. An accepted socket carries the listener's uid (checked: `TestCheckRealLoopback` passes under WSL Ubuntu with the connection row). | verified |
| macOS `lsof(8)` — https://ss64.com/mac/lsof.html | Without root, `lsof` only reports file descriptors of processes the caller may inspect (its own), so another user's listener is invisible rather than identified. libproc (`proc_pidfdinfo`) needs cgo. | no verificado (not tested on macOS) |
| OWASP Session Management Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html | Bind sessions to the client where possible; rotate secrets; short-lived single-use tokens for upgrades. | verified |

## Conclusions

- The cookie alone cannot identify the daemon's own origin on loopback; a per-page token in
  `sessionStorage` sent as a header can, and a one-time ticket covers the WebSocket.
- `Secure` would lock Safari users out; the cookie stays without it (dismissal reason for CodeQL).
- Name `ws://<host>:<port>` in `connect-src` instead of relying on `'self'`.
- Windows and Linux can identify the owner of a loopback listener without admin; macOS cannot
  without cgo or root, so the token also rotates on every daemon start.
