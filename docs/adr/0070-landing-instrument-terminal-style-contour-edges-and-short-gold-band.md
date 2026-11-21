<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0070: Landing style "Terminal de instrumentos": contour-line section edges and a short gold band

Version 1.0.0 · 2026-11-20

Status: Accepted (download contour animation superseded by ADR-0089) · Date: 2026-11-20 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Owner feedback, 2026-11-20, on the landing after ADR-0066:

- The gold download band is long (about 1.5 laptop screens) and so much gold overwhelms.
- Every section boundary is a straight line; the owner wants edges that vary, are creative and
  catch the eye, some straight, the rest not always the same shape.
- The style needs a name to design by ("¿neobrutalismo? ¿hacker GUI? ¿cuál es el nuestro?").
- The landing is where we are creative; the dashboard stays an admin panel (Cloudflare-like) that a
  non-technical user can run, and the tray stays for non-technical users.

## Decision Drivers

- Lean on known styles and techniques (research note), innovate where we can say why.
- One accent in small doses; calm reading for reference content.
- WCAG 2.2 AA, reduced motion, the landing's CSP unchanged, no layout shift.
- Build only what changed: CSS + small SVGs, no new dependency.

## Considered Options

Download band: (1) split, gold only for the call to action, then Bone; (2) an Ink section with a
faint gold field; (3) a gold → Bone gradient. Edges: (a) contour lines from the mark; (b) stepped
pixel/dither edges; (c) a mix of curves, diagonals and straight lines. Name: "Terminal de
instrumentos", "Hacker GUI", "Neobrutalismo cálido".

## Decision Outcome

Chosen 1, a, "Terminal de instrumentos" (owner, 2026-11-20).

- **Style.** "Terminal de instrumentos" = Terminal/CLI UI (the hero is a session, `$` commands,
  tmux bar) + an instrument-panel layer (radar, routing patch panel), warm (Bone paper, one gold),
  with topographic contour lines, the rings of the monkey mark, as its structural signature. Not
  neubrutalism (no thick borders or hard shadows) and not "hacker" clichés (no green Matrix look
  outside the CRT easter egg), so non-technical visitors stay at ease. It applies to the landing;
  the dashboard and tray keep the same tokens and gold (ADR-0069) but stay plain, usable product UI.
- **Short gold band.** `Download.tsx`: the `.tone-signal` band (`.dl-band`) holds only the heading
  and the visitor's file (its one filled button). It ends on the `ring` contour edge, echoed by two
  thin gold contour lines on the Bone page (`.dl-echo`, `aria-hidden`), drawn in as they scroll into
  view where CSS scroll-driven animations exist, static otherwise and with reduced motion. "Otras
  formas de instalar", the script, notes, the beta notice (caution orange again) and the source
  blocks sit on Bone; their code blocks stay Ink.
- **Contour edges.** `index.css` `.edge` + `edge-t-*` / `edge-b-*`: a tone band (Ink or gold) is
  shaped by a CSS mask; Bone is the page (transparent sections), so the part past the curve shows
  the page and nothing overlaps. One family of curves (`landing/src/assets/edges/`: swell, dune,
  drift, ring; one mirrored), different per boundary, depth `--edge-h` `clamp(28px, 4.5vw, 72px)`,
  and the section padding grows by 60 % of it so content keeps its rhythm. Home: Cómo empezar
  (swell in, dune out), Mapa (straight in, drift out), Detalles técnicos (mirrored dune in, straight
  into the gold band), gold band (ring out). Straight edges are deliberate rests; a new section picks
  a shape not used by its neighbours.

### Consequences

- Good: gold is the call to action again, not the background; boundaries read as one designed
  family; no new requests or dependencies (SVGs are inlined as `data:`, allowed by the CSP).
- Neutral: a masked section clips anything overflowing its edge (none today); a future sticky or
  floating child inside an edged section must live outside the mask.
- Supersedes nothing; refines ADR-0060 (rhythm of bands) and ADR-0066 (the gold band's extent).

## More Information

- Code: `landing/src/index.css` (`.edge`), `landing/src/assets/edges/*.svg`,
  `landing/src/components/sections/{Download.tsx,download.css,HowItWorks.tsx,Features.tsx,UnderTheHood.tsx}`.
- Research: [2026-11-20-landing-style-and-contour-edges](../research/2026-11-20-landing-style-and-contour-edges.md).
- Docs: `DESIGN.md` (Overview, Named Rules, Components).
- Refines: [ADR-0060](0060-landing-single-theme-bands-and-line-art-marks.md), [ADR-0066](0066-landing-gold-accent-footer-and-fixed-hero-terminal.md).
