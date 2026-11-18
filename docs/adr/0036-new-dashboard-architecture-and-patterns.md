<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0036: Panel nuevo: arquitectura y patrones (router, datos en vivo, kit, navegación)

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

[ADR-0011](0011-dashboard-vite-react-ts.md) started `web/ui` with one page and hand-written
`useState` hooks per resource. Issue #19 now migrates **everything** the classic dashboard does
(`docs/dashboard-parity.md`: 244 rows, 57 API routes) for two audiences with equal weight
(non-technical people and developers), with live data from the daemon (WebSocket + REST). Doing
that page by page in parallel needs a shared foundation first: routes for every section, one data
layer, one component kit and one shell. Per-page ad-hoc fetching would repeat requests, flicker on
each tick and handle errors differently on every page.

## Decision Drivers

- GUI-first, CLI-complete, no feature dropped (parity contract).
- Live data without flicker or layout shift; only changed rows re-render.
- One way to read, cache and mutate each daemon resource; optimistic changes with rollback.
- WCAG 2.2 AA, ES+EN, DESIGN.md Operate mode (Four States, Green Means Up, One Accent, 11px floor,
  Flat-By-Default, motion only for state).
- Prefer vetted libraries over our own code ("Standards first"); keep the embedded bundle sane.
- Auth stays isolated so the planned session bootstrap changes one module.

## Considered Options

Router: (A) React Router 8 declarative mode · (B) TanStack Router · (C) wouter · (D) none (state
switch). Data: (E) TanStack Query fed by the WebSocket · (F) in-house store + hooks (status quo).
Overlays: (G) native `<dialog>` + in-house toasts · (H) Radix / a toast library.

## Decision Outcome

Chosen: **A + E + G**, sources in [the research note](../research/2026-11-18-dashboard-ui-patterns.md).

1. **Router: `react-router` 8.5.1, declarative mode, `basename` = Vite's base (`/app`).** Most
   mature option with `basename`, `NavLink` (`aria-current`) and React 19 support; the daemon
   already serves `index.html` for any `/app/...` path (`internal/api/ui_assets.go`). TanStack
   Router's type-safe routes do not pay off for ~10 flat routes; wouter is far smaller but has
   fewer guarantees. Cost: the JS bundle grows to ~110 kB gzip (fine for a local panel served from
   127.0.0.1; revisit if it matters). Versions are pinned and were published ≥ 2 weeks before
   adoption.
2. **Data: `@tanstack/react-query` 5.105.3** gives cache keyed by resource (`src/data/keys.ts`),
   stale-while-revalidate, request de-dup, retries and optimistic mutations with rollback
   (`src/data/optimistic.ts`), instead of re-implementing them. Settings: `networkMode: 'always'`
   (the API is local; `navigator.onLine` is irrelevant), `staleTime` 5 s = the daemon's heartbeat,
   retries only for "daemon not answering" and 5xx (max 2, full-jitter backoff), never for 4xx or
   mutations.
3. **One live feed per page** (`src/data/live.ts`): the WebSocket writes each frame into the same
   cache entry as REST (`keys.tunnels`); structural sharing keeps unchanged tunnels identical and
   memoized rows (`DataList`) skip re-rendering. While the socket is down the tunnels query polls
   REST every 5 s; reconnect uses exponential backoff with full jitter (cap 30 s); after an outage
   every resource is refetched.
4. **Reachability and auth are app state, not errors per page.** `src/data/connection.ts`
   (`useSyncExternalStore`) tracks "daemon answering"; one banner says what to do and the page
   reconnects by itself (there is no API to start a stopped daemon from a page it no longer serves,
   so the banner guides instead of offering "Iniciar"). `src/api/auth.ts` is the **only** module
   that deals with authentication: it strips a `#k=` bootstrap nonce from the URL at boot and trades
   it at `POST /api/v1/session` (server side pending; a 404 keeps today's cookie path), and any 401
   shows "abre el panel desde Kongtrol".
5. **Helper state is shared and immediate.** Every install button uses one mutation key; success
   writes the new status into the cache (all buttons disappear at once), a status change from any
   surface (tray, CLI, classic panel; polled every 10 s while missing) raises one toast "Ayudante
   listo · ya no hace falta volver a instalarlo", progress names the OS prompt, and a failure shows
   its "what to do" line inline and as a toast.
6. **Information architecture** (target of every parity row): Inicio `/`, Perfiles `/profiles`
   (+ `/profiles/groups`), Reglas `/rules` (+ `/rules/schedule`), Tráfico `/traffic`, Seguridad
   `/security`, Actividad `/activity`, Ajustes `/settings`. Sub-sections are link tabs with their own
   URL. Navigation by width (Material 3 window classes): bottom bar < 600 px (Inicio, Perfiles,
   Reglas, Seguridad + "Más" sheet), rail 600–839 px, sidebar with labels ≥ 840 px. Seven
   destinations exceed the "≤ 5" guidance for bars, so the bar shows four plus "Más"; the rail and
   sidebar (3–7 allowed) show all.
7. **Kit** (`src/components/ui`): Button (filled/tonal/outlined/text/danger, loading), IconButton +
   accessible Tooltip (WCAG 1.4.13), Switch (`role="switch"`, word + color, optimistic), StatusPill
   (Four States + idle/unknown/on, distinct dot shapes), Card (flat), DataList (table → cards under
   640 px, memoized rows, tabular numbers), Dialog/Sheet on native `<dialog>` (focus trap, Esc,
   inert background from the platform), Toaster (one at a time, 4 s / 10 s with action, errors stay,
   pauses on hover/focus, polite + assertive regions), Tabs (APG pattern) and TabLinks, Field
   (label, hint, inline error = problem + what to do), Skeleton (first load only), EmptyState,
   ErrorState, Banner, Kbd, Logo. Tokens: Material 3 standard easing `cubic-bezier(0.2,0,0,1)`,
   150/250 ms, state layers 8/10 %, all motion off with reduced motion. `/app/_kit` reviews them in
   development builds only.
8. **Brand images are vector and theme-adaptive.** The logo is the line-art master
   `assets/brand/logo-kong.svg` used as a CSS mask in `currentColor` (no disc, crisp at any size);
   the favicon is `assets/brand/favicon.svg`, whose embedded `prefers-color-scheme` style adapts to
   the browser theme. `web/ui` ships no bitmap: the panel runs in desktop browsers that support SVG
   favicons and there is no PWA manifest. Safari before version 26 ignores SVG favicons
   (**no verificado**); it then shows no icon, which is acceptable for a local panel.

### Consequences

- Good: pages in phase 2 only compose hooks + kit; no page fetches on its own; live ticks touch
  only changed rows; errors and helper/daemon states read the same everywhere.
- Good: phase 2 can run in parallel by page (one owner per `src/pages/<Page>` + its keys/hooks).
- Bad: two runtime dependencies more (react-router, TanStack Query) and a larger bundle.
- Neutral: on-demand actions (doctor, export, purge) stay local state; they are not resources.

### Confirmation

Vitest: `src/data/data.test.tsx` (de-dup, backoff, retry policy, WebSocket identity, only changed
rows re-render, optimistic rollback, helper toast from another surface), `src/api/auth.test.ts`,
`src/components/ui/kit.test.tsx` (roles, keyboard), `src/app/shell.test.tsx` (routes, focus,
titles, notices), `src/i18n/messages.test.ts` (ES/EN parity).

## More Information

- Code: `web/ui/src/{App.tsx,app/,data/,api/auth.ts,components/ui/,components/layout/,theme/}`
- Parity contract: [docs/dashboard-parity.md](../dashboard-parity.md)
- Research: [Dashboard UI patterns](../research/2026-11-18-dashboard-ui-patterns.md)
- Builds on [ADR-0011](0011-dashboard-vite-react-ts.md), [ADR-0013](0013-brand-single-source.md),
  [ADR-0014](0014-i18n-language-resolution.md)
