<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0060: Landing in one designed theme with section bands; line-art marks in the header and illustrations

Version 1.0.0 · 2026-11-19

Status: Accepted; refined by [ADR-0070](0070-landing-instrument-terminal-style-contour-edges-and-short-gold-band.md) (contour-line band edges, short gold band) · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Owner feedback on the live landing (2026-11-19): sections look flat and alike, some are too far
apart, the key ones (routing map, attach demo, download, footer) do not fit a laptop screen, the
download section repeats the same files, the hero terminal looks washed out, the header mark is
too small to read, and switching ES ↔ EN reloads the page and loses the place. The landing had a
light/dark switch (ADR-0024 theme wipe, ADR-0026 theme store); two complete themes made every
section the same tone in either theme.

## Decision Drivers

- A landing that persuades: visual rhythm between sections, each key section in one screen.
- Same tokens as DESIGN.md "Sesión en la Torre" (no new palette), WCAG AA everywhere.
- The line-art mark is the signature; show it where it can be read (owner decision).
- No stale code: remove what a single theme no longer needs.

## Considered Options

1. Keep light/dark and only add section backgrounds per theme.
2. **One designed theme: Bone page, Ink bands, one Signal-amber band.**

## Decision Outcome

Option 2 (owner decision, 2026-11-19).

- **Theme (landing only).** No theme switch and no `prefers-color-scheme` on the landing: the
  `kgtheme` key, `useTheme`, the theme button, the theme-wipe transition and the theme part of
  the boot script (now `DEVICE_BOOT`) are gone; `404.html` is Ink only. `:root` carries the Bone
  tokens; `.tone-ink` redefines every colour token for its subtree (the old dark values), and
  `.tone-signal` does it once, for the download section. Sections alternate: hero and attach demo
  Bone (the hero terminal window is Ink), steps Ink, providers Bone, map Ink, security Bone (its
  terminal Ink), details Ink, download Signal, footer Bone. The header, the tmux bar and the
  back-to-top button are always Ink. Logos invert through `.tone-ink/.tone-signal .logo-mark`.
  The dashboard (`web/ui`) keeps its light/dark themes; the favicon still follows the browser.
  This refines the landing-theme parts of ADR-0024 (theme toggle wipe) and ADR-0026 (theme store
  and its hydration check); their other points stand, so they stay Accepted.
- **Signal band.** The download section uses the deep amber of the hero ASCII banner (`#904f00`)
  with Bone text, an inverted primary button (Bone fill, `#6b3b00` text) and code blocks inset in
  Ink; contrast in the research note. Amber stays an accent everywhere else.
- **Rhythm and fit.** `--section-y` is `clamp(48px, 2.5vw + 32px, 88px)`; no hairlines between
  sections (the change of tone divides). From 1200 px the map heading sits beside the board
  (the SVG never scales under 1:1, so its 11 px labels keep the floor); from 1080 px the download
  heading sits beside the recommended file; from 1024 px the hero terminal has two columns under a
  one-line h1 and no inner scroll. The footer is exactly `100svh − tmux bar` on laptops; its giant
  wordmark is absolute (no extra height).
- **Download.** One recommended card (installer for Windows; both Mac files; Linux 64-bit), then
  one "Otras formas de instalar" row where every other file appears once and the install script
  has a tab and a copy button per shell (`DevInstall`).
- **Back to top.** The footer's text link is replaced by a round 48 px Ink icon button, fixed
  bottom-right above the tmux bar, shown after about one screen, accessible name "Volver al
  inicio" / "Back to top", tooltip, focus ring; reduced motion jumps instantly.
- **Language switch.** Still separate prerendered URLs with `hreflang`. The toggle adds the
  section being read as `#fragment`; `@view-transition { navigation: auto; }` cross-fades the
  pages with the header and tmux bar as named groups (Chrome 126+; other browsers just navigate;
  none under reduced motion); `rel=expect blocking=render` on `#site-footer`; smooth scrolling is
  enabled only after hydration (`html[data-ready]`), so the new page lands on the section at once.
- **Latest release in the footer.** The `kongtrol version` lines show the latest stable release
  (`kongtrol v0.6.0 · último release`, linked to GitHub Releases), resolved at build time by
  `landing/vite.config.ts` (env `KONGTROL_RELEASE` → GitHub Releases API `releases/latest`, 5 s →
  newest stable git tag → line left out; cached one hour so build, prerender and the hydration
  check agree). Visitors make no third-party request (CSP `connect-src 'self'`); a new release
  appears on the next landing build.
- **Logos (refines ADR-0051).** Line art also for the landing header (54 px, 44 px under 960 px),
  the attach demo hub (108 px) and the routing panel badge (40 units in the SVG, 28 in the phone
  strip). The solid mark stays for the favicon, the tray, the panel and flyout marks, and the
  attach demo's tray mock, drawn with the real badge geometry (`assets.BadgeGeometry`: r = 15 %,
  gap size/16; hollow grey → live green when the tunnel is detected; reduced motion shows the
  final state).

### Consequences

- Good: clear section rhythm; each key section fits 1366×768, 1440×900 and 1920×1080 laptop
  screens; no duplicated downloads; the mark reads in the header; the language switch keeps the
  place without a visible reload where supported.
- Neutral: one fewer stored preference (privacy page updated); `.impeccable/design.json` still
  carries the old two-theme wording until its tool regenerates it from DESIGN.md.
- Bad: the README GIFs (`scripts/readme-media.mjs`: hero, routing, attach) show the old look until
  they are regenerated after merge.

### Confirmation

- Landing `lint`, `build` (CSP hashes), `check:hydration` (now also fails if `data-theme` appears),
  `check:i18n`; Playwright screenshots and measurements at 1366×657, 1440×789, 1920×969 and
  390×844 (see the commit and the research note).

## More Information

- Code: `landing/src/index.css`, `landing/src/components/ui/{Nav,Footer,BackToTop,TmuxBar}.tsx`,
  `landing/src/components/sections/{Hero,Download,AttachDemo,Features}.tsx` and their CSS,
  `landing/src/seo.ts`, `landing/public/{_headers,404.html}`, `landing/scripts/check-hydration.mjs`.
- Research: [2026-11-19-landing-single-theme-and-page-transitions](../research/2026-11-19-landing-single-theme-and-page-transitions.md).
- Docs: `DESIGN.md` (Overview, Colors, Layout, Navigation, Footer, Logo).
- Refines: [ADR-0051](0051-logo-size-cutoff-solid-to-64px.md) (header and illustration marks),
  [ADR-0024](0024-landing-motion-a11y-performance.md) and
  [ADR-0026](0026-landing-hydrate-prerendered-html.md) (landing theme parts only).
