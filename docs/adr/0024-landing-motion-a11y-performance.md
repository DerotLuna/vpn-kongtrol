<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0024: Landing polish — CSS-only motion without layout shift, WCAG 2.2 AA, Core Web Vitals

Version 1.0.0 · 2026-11-18

Status: Accepted; boot-timeline resume (option 3) superseded by [ADR-0026](0026-landing-hydrate-prerendered-html.md) · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

The landing must read well for non-technical visitors, feel professional and rank well, while
keeping its terminal/tmux identity (evolve, not replace). Measured problems: mobile CLS 0.187
(the hero terminal grew line by line and pushed the page), a hero whose plain-language sentence
was small grey monospace, accessibility gaps automated audits did not flag (no skip link,
undersized targets, invisible focus on the demo input, a menu whose hidden links stayed in the
tab order, an auto-rotating board announced every 2.6s, listbox/tab semantics that did not
match the controls), and a few invalid JSON-LD properties
([research](../research/2026-11-17-landing-ux-a11y-perf.md)).

## Decision Drivers

- No new dependencies; animations are CSS + native APIs; every animation respects
  `prefers-reduced-motion`; only transform/opacity animate (clip-path for the typed command,
  clip-path for the theme wipe); no layout shift.
- Prerendered HTML carries the real content (crawlers, no-JS visitors).
- WCAG 2.2 AA, measured, not assumed.
- Keep copy unchanged; identity (tokens, fonts, terminal, tmux bar, contour rings) unchanged.

## Considered Options

1. JS-timed boot (status quo): lines mount one by one; content missing from prerendered HTML.
2. Switch to `hydrateRoot` so prerendered nodes are reused and CSS animations keep running.
3. CSS boot timeline: all lines always in the markup; each fades in at its own `--at`; after
   `createRoot` replaces the prerendered nodes, the new nodes resume at the elapsed time read
   from the old ones with `getAnimations()`.

## Decision Outcome

Chosen: **option 3**. Option 2 is the React-documented path but needs every client-only value
(theme, OS, language, `useId`) to render identically on the server first; that is a wider change
across files other work is editing, so it stays a follow-up. Option 3 removes the shift
(CLS 0.187 → 0) and puts the whole hero in the HTML with a local change.

Also decided:

- **Motion system**: one authored moment (the boot) plus quiet reveals. Reveals use CSS
  scroll-driven animations where supported, an IntersectionObserver fallback elsewhere
  (`useReveal`), never hide what is already on screen, and are off for reduced motion and
  without JS. Theme toggle: view transition wiping a circle from the button; instant without
  the API or with reduced motion. A global reduced-motion rule shortens anything left.
- **Hierarchy**: the hero's plain-language sentence uses the body face at full contrast and a
  larger fluid size; spacing and type scale tokens (`--space-*`, `--step-*`,
  `--section-y`, `--measure` 65ch); one button family with ≥44px height, hover lift and
  `:active` press.
- **Accessibility**: skip link on every page (`<main id="main">`); `--text-dim` darkened
  (dark #838d9d, light #635c4b) so it clears 4.5:1 on every surface incl. `--signal-soft`;
  placeholder at full token color; focus ring on the demo prompt line; targets ≥24px (tmux,
  copy, footer, close) and ≥44px for actions; mobile menu hidden with `visibility` (out of the
  tab order), Escape returns focus, outside tap closes; OS picker = APG radio group; jump menu
  = APG disclosure of links; routing board pauses on hover/focus, stops after a pick, and is
  only announced after the visitor interacts; `aria-current` on nav, tmux and jump links;
  guide diagram titles no longer skip heading levels.
- **SEO/perf**: preload the two first-screen fonts; `theme-color` per scheme; CSS
  `color-scheme` follows the chosen theme; JSON-LD drops `codeRepository` (not a
  SoftwareApplication property) and gives `softwareHelp` a CreativeWork; dead CSS removed
  (~140 lines). Canonical, hreflang (es/en/x-default), OG/Twitter, sitemap and robots were
  already correct and stay as they are.

### Consequences

- Good: CLS 0 on both pages and devices; mobile performance 87–88 → 97; Lighthouse a11y,
  best practices and SEO stay at 100; the hero reads in plain language at a glance.
- Bad: the boot timeline still restarts if `getAnimations()` is missing (old browsers) — it
  then replays from the start, without layout shift.
- Follow-up: `hydrateRoot` (option 2); home meta descriptions are ~400 characters and get
  truncated in results — a copy decision, left to whoever owns the text.

## More Information

Research and before/after scores: `docs/research/2026-11-17-landing-ux-a11y-perf.md`.
Code: `landing/src/index.css` (v2.3), `landing/src/components/sections/Hero.tsx`,
`landing/src/hooks/useReveal.ts`, `landing/src/components/ui/{SkipLink,OsTabs,SectionJumpMenu,Nav}.tsx`,
`landing/src/seo.ts`.
