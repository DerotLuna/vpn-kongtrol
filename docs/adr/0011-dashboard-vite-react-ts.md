<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0011: Dashboard migration to Vite + React + TS by phases, served at `/app/`

Version 1.0.0 · 2026-11-17

Status: Superseded by [ADR-0043](0043-new-dashboard-replaces-classic.md) · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

The embedded dashboard (`web/dashboard`) is vanilla JS/CSS with no bundler. As it became the
primary GUI ([ADR-0003](0003-product-model-gui-first-cli-complete.md)), it needs types, tests and
components. A big-bang rewrite would block features; contributors must still `go build` without
Node.

## Decision Drivers

- Single binary: the UI stays embedded.
- `go build` works without Node.
- Migrate page by page; no regression window.

## Considered Options

1. Keep vanilla JS — no types/tests; harder to grow.
2. Big-bang rewrite — long freeze.
3. New Vite + React + TS app alongside the legacy one, migrated by phases.

## Decision Outcome

Chosen: **option 3**. `web/ui/` (Vite + React + TS, pnpm, oxlint, Vitest) builds to
`web/ui/dist`, embedded by `web/ui_embed.go`, served at `/app/` by `internal/api/ui_assets.go`;
legacy `web/dashboard` keeps `/` until migrated. Only `dist/.gitkeep` is tracked, so `go build`
works without Node (shows a "not built" page); release builds always build the UI. `make ui-dev`
proxies `/api` to the daemon. Strings in `src/i18n/messages.ts` (ES+EN parity tested); external
URLs only in `src/links.ts`.

Precedents (verified 2026-11-17): Tailscale's `client/web` is Vite + React + TypeScript and
proxies to the Vite dev server, with built assets from a separate prebuilt module; PocketBase
embeds its UI with `//go:embed all:dist`; Gitea embeds assets only with the `bindata` build tag.

### Consequences

- Good: typed, tested UI; same binary; no Node needed for Go work.
- Bad: two dashboards during migration; both must stay in sync ("Docs stay in sync" rule).

### Confirmation

Vitest suite in `web/ui`; CI `ui` job runs only when `web/ui` changes ([ADR-0015](0015-ci-scoped-builds-generated-docs.md)).

## More Information

- Code: `web/ui/`, `web/ui_embed.go`, `internal/api/ui_assets.go`, `Makefile` (`ui-build`, `ui-dev`)
- Research: [Go-embedded SPA](../research/2026-11-17-go-embedded-spa.md)
