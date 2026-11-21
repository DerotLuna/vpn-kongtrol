<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: Panel session recovery after restart

Version 1.0.0 · 2026-11-21

| Source | Accessed |
|---|---|
| [WHATWG HTML Standard, Web Storage](https://html.spec.whatwg.org/multipage/webstorage.html#webstorage) | 2026-11-21 |
| [Profile sync](../../internal/api/profile_sync.go), [restart handler](../../internal/api/handlers_setup.go) and [session implementation](../../internal/api/session.go) | 2026-11-21 |

- Setup mode returns `restart_required` after the first profile is saved. The restart handler
  spawns a fresh process. Its session map starts empty and credentials issued by the old
  instance receive a 401 from authenticated endpoints.
- The earlier Add sheet fixture simulated an unavailable socket during restart and accepted
  the old browser credentials afterwards. The updated test exchanges a new nonce before
  continuing to rule creation.
- Web Storage limits `localStorage` to the current origin and shares it across tabs, while
  `sessionStorage` belongs to each tab. Storage can be unavailable due to browser policy.
  Local storage can outlast the tab, so the receiver enforces an application deadline and
  removes the entry while reading it.
- The paths explain the reported symptoms. The exact sequence behind the original capture
  remains unverified.

Used by [ADR-0082](../adr/0082-resume-first-rule-after-panel-sign-in.md).
