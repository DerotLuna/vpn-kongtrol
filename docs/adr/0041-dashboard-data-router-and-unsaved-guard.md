<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0041: Panel nuevo: router en modo datos, aviso de cambios sin guardar y páginas por demanda

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer · Supersedes ADR-0038
decision 1 (router mode only; the rest of ADR-0038 stays in force)

## Context and Problem Statement

[ADR-0038](0038-new-dashboard-architecture-and-patterns.md) chose React Router 8 in declarative
mode. With every page migrated, three forms (Ajustes, the profile editor, the rule and schedule
editors) can lose edits: a click on another section or Back drops them silently. Declarative mode
cannot block a navigation (`useBlocker` is data/framework only), so only tab close/reload was
guarded. The single JS bundle (~680 kB, served uncompressed by the daemon) also slowed the first
paint on phones (Lighthouse mobile performance 61–63).

## Decision Drivers

- No silent data loss (Nielsen: error prevention); WCAG 2.2 AA dialogs.
- Standard mechanism, no own history patching ("Standards first").
- Minimal churn: same routes, same URLs, same tests.

## Considered Options

1. Data router (`createBrowserRouter` + `RouterProvider`) + `useBlocker`.
2. Stay declarative and intercept clicks on links app-wide (misses Back/Forward, reinvents routing).
3. Only the browser `beforeunload` prompt (status quo; misses every in-app navigation).

## Decision Outcome

Chosen: **option 1**, sources in [the research note](../research/2026-11-18-unsaved-changes-guard.md).

- `web/ui/src/App.tsx` builds the same route tree with `createRoutesFromElements` and
  `createBrowserRouter` (basename `/app`).
- Forms only register their state: `useUnsavedChanges(dirty)` (`web/ui/src/app/unsaved.ts`), which
  also arms `beforeunload` while dirty. They never touch the router, so they work in any test.
- The shell's `UnsavedChangesGuard` (`web/ui/src/components/layout/UnsavedChangesGuard.tsx`)
  blocks a change of path while any form is dirty and asks with the kit Dialog (alertdialog):
  "Tienes cambios sin guardar" · **Seguir editando** (filled, first, safe) / **Descartar**
  (danger). Query/hash changes of the same page never ask.
- Route `lazy` loads each page except Inicio on first visit; the current page stays until the next
  is ready (no loading flash).

### Consequences

- Good: no silent loss of edits from links, the bottom bar, the "Más" sheet or Back/Forward.
- Good: smaller first download; pages load on demand.
- Neutral: the first visit to a page waits for its chunk (local server, milliseconds).
- Bad: one more concept (route modules) for contributors.

### Confirmation

Vitest `web/ui/src/pages/SettingsPage.test.tsx` ("cambios sin guardar": stay keeps the edit,
discard leaves, nothing changed never asks) and every page test running on the data router.

## More Information

- React Router `useBlocker`: https://reactrouter.com/api/hooks/useBlocker
- React Router modes: https://reactrouter.com/start/modes
