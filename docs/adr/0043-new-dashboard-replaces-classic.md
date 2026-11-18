<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0043: The new dashboard replaces the classic one and moves to `/`

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

Supersedes [ADR-0011](0011-dashboard-vite-react-ts.md).

## Context and Problem Statement

[ADR-0011](0011-dashboard-vite-react-ts.md) built the new dashboard (`web/ui`) at `/app/` next to
the classic vanilla-JS one (`web/dashboard`, at `/`) and retired the classic one only when every row
of the parity contract (`docs/dashboard-parity.md`) was ticked. Phase 2 shipped every page. Two
dashboards meant two codebases to keep in sync, a CSP that allowed inline scripts and styles only
for the classic pages, and users landing on the old one by default. The maintainer approved the
switch ("mígralo… todo lo que se puede hacer en el panel clásico").

## Decision Drivers

- No capability may be lost: the parity gate must pass first.
- Nothing stale stays (AGENTS.md): delete the classic code, its tests, CI job and assets route.
- Old addresses keep working: bookmarks, older tray builds (first run opened `/studio.html`, the
  flyout `/app/tray`) and the one-time sign-in fragment `#k=` ([ADR-0039](0039-browser-sessions-one-time-nonce.md)).
- Security unchanged or better: same Host/Origin checks, same session bootstrap, stricter CSP.

## Considered Options

1. Keep `/app/` as the dashboard's address and redirect `/` there.
2. Serve the dashboard at `/` and redirect `/app/*` (and the classic page names) to the same page.
3. Keep both dashboards.

## Decision Outcome

Chosen: **option 2**. Short, stable addresses (`http://127.0.0.1:9741/rules`) and one place for
every client; option 1 keeps a migration artefact in every URL forever; option 3 breaks
"nothing stale stays".

1. **Parity gate.** `docs/dashboard-parity.md` is closed at 244/244 rows. Five rows drop only a
   cosmetic part, recorded as `dropped: ADR-0043`: the BETA badge (the version tag shows the
   build), the remembered manual rail collapse (the rail follows the window width, ADR-0038), the
   theme circular reveal, the language toggle flash, and the custom select widget (replaced by the
   native `<select>`, which keyboard and screen readers support without extra code).
2. **Serving** (`internal/api/ui_assets.go`). The UI handler is the mux catch-all `/`: a file of
   the build if it exists; a missing file (under `assets/` or any name with an extension) is a
   404, never HTML; any other path is a client route and gets `index.html`; `/api` and `/api/*`
   never get the page. Gzip and cache rules are unchanged.
3. **Old addresses.** `/app` and `/app/*` answer **308** to the same path at `/` (query kept);
   `/studio.html`, `/security.html`, `/settings.html`, `/audit.html` answer 308 to `/profiles`,
   `/security`, `/settings`, `/activity`. The `#k=` fragment never reaches the server; the browser
   re-attaches it because `Location` carries none (RFC 9110 §10.2.2). A target that would start
   with `/\` (read as another host by browsers) goes to `/` instead.
4. **Clients.** `daemonctl.Client.PanelURL` builds `/#k=…` for the CLI (`kongtrol dashboard`, live
   view `o`) and tray ("Abrir panel"); the tray's first run opens `/profiles` (setup mode guide);
   the flyout loads `/tray` (loopback and path lock unchanged).
5. **Deleted.** `web/dashboard/**`, `web/assets.go`, the `/brand/` route and `assets.Brand()`
   (only the classic pages linked them; `web/ui` bundles brand files with Vite),
   `scripts/i18n/dashboard.test.mjs` and its CI job.
6. **CSP.** Pages get `script-src 'self'; style-src 'self'` (no `'unsafe-inline'`, which only the
   classic pages needed). React sets `style` props through the element's `style` object, which
   `style-src` does not block (MDN). The "UI not built" page allows its one `<style>` block by
   SHA-256 hash (CSP Level 3 hash-source).

### Consequences

- Good: one dashboard, one codebase; shorter URLs; stricter CSP; about 7,000 fewer lines.
- Good: old bookmarks and older tray builds still land on the right page.
- Bad: a binary built without the UI (no Node, development only) has no dashboard at all, only the
  "not built" page that says how to build it; release builds always include it.
- Bad: a newer tray talking to an older, still-running daemon gets a 404 for `/tray` until the
  daemon restarts (installers restart both; **no verificado** for every manual upgrade path).

### Confirmation

- Go: `internal/api/ui_assets_test.go` (root, deep links, `/tray`, assets, 404s, `/api` never the
  page, `/app` and classic-page redirects, open-redirect guard, CSP, Host check),
  `internal/daemonctl/panel_test.go`, `cmd/kongtrol-tray/flyout_test.go`.
- `web/ui`: `src/app/shell.test.tsx` (sections at the root, no classic link), `src/api/auth.test.ts`.
- One-off Playwright smoke (2026-11-18) against a locally built daemon (temp home, loopback port
  19741, no admin): `/app/rules#k=<nonce>` → `/rules` signed in (session cookie set), every
  section by deep link, `/studio.html` → `/profiles`, `/tray`, zero console errors (no CSP
  violations).

## More Information

- Research: [Panel at the site root](../research/2026-11-18-panel-at-root.md)
- Code: `internal/api/ui_assets.go`, `internal/api/server.go`, `internal/daemonctl/panel.go`,
  `cmd/kongtrol-tray/flyout.go`, `cmd/kongtrol-tray/tray_actions.go`, `web/ui/vite.config.ts`,
  `web/ui/src/App.tsx`
- Parity record: [`docs/dashboard-parity.md`](../dashboard-parity.md)
