<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0079: Landing hero call to action, one-screen footer on phones, jump menu hides with the header

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Owner feedback on the landing, 2026-11-20:

- The hero's buttons are small and the left column ends far above the terminal log beside it; the
  hero's call to action should be the strongest on the page.
- On phones the footer is taller than the screen (1077 px on a 390 × 844 viewport), while on
  laptops it fills exactly one screen (ADR-0060).
- On phones the "Ir a" jump menu stays pinned when the header steps aside for the footer.

## Decision Outcome

- **Hero** (`landing/src/components/sections/Hero.tsx`, `index.css`): from 1024 px the buttons are
  60 px tall with larger text and icons; under the note a row of facts a visitor checks before
  downloading — "Gratis · Código abierto · Sin cuenta · Tus datos no salen de tu equipo" (green
  check marks: each is a passed check, the Green Means Up Rule).
- **Footer on phones** (≤ 640 px): the mark (line art, 128 px, the minimum of DESIGN.md) beside the
  claim instead of above it; link groups in rows; the chips hidden (beta, systems and license are
  already in the header, the tagline and the links). Under 720 px the hidden tmux bar no longer
  takes 34 px from the footer, and from 760 px of height the footer is exactly one screen
  (`height: 100svh`); shorter phones scroll naturally. Measured: 844 px on 390 × 844, nothing cut.
- **Jump menu**: `body:has(.nav.is-away) .home-jump` slides away with the header and is hidden
  (not focusable) while the footer shows.

## More Information

- Refines: [ADR-0060](0060-landing-single-theme-bands-and-line-art-marks.md), [ADR-0066](0066-landing-gold-accent-footer-and-fixed-hero-terminal.md), [ADR-0070](0070-landing-instrument-terminal-style-contour-edges-and-short-gold-band.md).
- Docs: `DESIGN.md` (Footer, hero buttons).
