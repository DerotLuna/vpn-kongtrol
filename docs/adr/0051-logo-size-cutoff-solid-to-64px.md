<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0051: Logo size cutoff — solid mark up to ~64 px, line art from ~128 px

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

ADR-0047 point 4 put the cutoff between the two marks at 32/36 px: the solid silhouette up to
32 px, the line-art original from 36 px up, so the landing header (40 px, 36 px under 960 px)
used the line art. At 36–40 px its concentric lines are about one device pixel apart on a
standard screen and still read as a grey disc next to the header buttons; the line art only
shows its detail when it is large. The redesigned landing footer (2026-11-19) carries a large
mark, which gives the line art its place.

## Decision Drivers

- Small marks must read as the head, not as grey noise (the same reason as ADR-0044 and
  ADR-0047).
- The line art is the signature drawing: show it where its lines can be seen.
- Keep both marks derived from the masters in `assets/brand/source/` (ADR-0013, ADR-0044).

## Considered Options

1. Keep the 32/36 px cutoff (ADR-0047).
2. **Solid up to ~64 px, line art from ~128 px; nothing in between.**

## Decision Outcome

Option 2 (owner decision, 2026-11-19). This refines only the size cutoff of ADR-0047 point 4
and the header size in point 5; the name, tagline and every other point of ADR-0047 stand, so
ADR-0047 stays Accepted.

- **Solid silhouette** (`assets/brand/source/logo-kong-solid.svg`) up to ~64 px: favicon, tray,
  panel and flyout marks (as before), plus the landing header mark at 40 px (36 px under 960 px)
  and the badge of the landing's routing patch panel. The landing gets it as
  `logo-kong-solid.svg` through `landing/scripts/copy-brand.mjs` (no committed copy, no PNG),
  black on transparent, inverted on the dark theme like every logo `<img>`.
- **Line art** (`logo-kong.svg`) from ~128 px: the landing footer mark (≥ 160 px), README, app
  icons ≥ 180 px.
- Sizes between ~64 and ~128 px are avoided; the footer's old 96 px lockup is gone.

### Consequences

- Good: the header mark reads at a glance; the line art appears only where its detail shows.
- Neutral: the landing serves one more brand file (`logo-kong-solid.svg`, ~15 KB, cached with
  the other `/logo*` files by `landing/public/_headers`).

### Confirmation

- Landing lint, build (CSP check), `check:hydration`, `check:i18n`; screenshots at 1280 and
  375 px, light and dark — see the commit.

## More Information

- Code: `landing/src/components/ui/Nav.tsx`, `landing/src/components/ui/Footer.tsx`,
  `landing/src/components/sections/Features.tsx`, `landing/scripts/copy-brand.mjs`.
- Docs: `DESIGN.md` ("Logo (name and sizes)").
- Refines: [ADR-0047](0047-product-name-kongtrol-and-logo-sizes.md) point 4 (cutoff) and point 5
  (header mark).
