<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0027: Landing download buttons — one primary per view, recommended OS, phones send the page to a computer

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

Every download entry point on the landing (hero, header, "Para todos", download section) went to
the GitHub releases page, the hero had two equally loud buttons, the download section showed
three equal cards with filenames that broke mid-word, and phone visitors were offered files they
can't run. Research: [2026-11-18-landing-download-ctas](../research/2026-11-18-landing-download-ctas.md).

## Decision Drivers

- Non-technical visitors: the right file for their computer, in short words, ES+EN.
- Apple HIG / Material 3 button hierarchy; ≥44 px targets.
- Prerendered + hydrated landing ([ADR-0026](0026-landing-hydrate-prerendered-html.md)): nothing
  that depends on the visitor may be in the first render.
- The landing only advertises what works: the current release ships CLI binaries, no installer yet.

## Considered Options

1. Hero button downloads the detected file directly.
2. **Hero, header and "Para todos" scroll to the download section (`#install`), which recommends
   the detected OS in one featured card.**
3. Keep linking GitHub releases.

## Decision Outcome

Chosen: **option 2**, until a release ships the Windows installer (issue #32).

- Hero: filled "Descargar para Windows/Linux/Mac" ("Descargar gratis" in the prerender) +
  outlined "Ver en GitHub". Header: GitHub icon link always; the download button appears once
  the hero's has scrolled out of view (`IntersectionObserver`, `visibility` so it is never
  focusable while hidden, no layout shift, no animation with reduced motion); on other pages it
  shows from the start and links to `/#install`.
- Download section: featured "Recomendado para tu equipo" card with the page's only filled
  button; "Otros sistemas" below with outlined buttons; macOS offers Apple Silicon and Intel and
  explains the About This Mac check; filenames on one line with ellipsis.
- Phones/tablets (`useIsMobile`, heuristic documented in `landing/src/hooks/useVisitor.ts`):
  download buttons become "Estás en un celular: Kongtrol se instala en tu computador." +
  "Enviar a mi computador" (Web Share API) or "Copiar enlace" with a `role="status"` confirmation.
- Asset filenames live only in `RELEASE_ASSETS` (`landing/src/links.ts`).
- `pnpm check:hydration` adds an iPhone visitor and asserts the hero's button per visitor.

### Consequences

- Good: one clear action per view; no mismatch; renaming release assets is a one-line change.
- Bad: one extra scroll before the download; the phone heuristic misses "desktop site" mode.
- When the installer ships, revisit option 1 with a new ADR.

## More Information

Code: `landing/src/components/sections/{Hero,Start,Download}.tsx`,
`landing/src/components/ui/{Nav,SendToComputer,downloadCta}.tsx`,
`landing/src/hooks/useVisitor.ts`, `landing/scripts/check-hydration.mjs`.
