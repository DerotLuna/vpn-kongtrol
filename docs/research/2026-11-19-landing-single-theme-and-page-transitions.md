<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: landing in one theme, smooth language switch, back to top

Version 1.0.0 · 2026-11-19

## Question

For ADR-0060: how to switch ES ↔ EN between two prerendered pages without a visible reload
(keeping the section being read), how a back-to-top control is usually built, and whether the
one Signal-amber band keeps WCAG AA contrast.

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | MDN — `@view-transition`: `navigation: auto` opts both documents into a view transition for same-origin `push` / `replace` / `traverse` navigations started by the user; not Baseline | https://developer.mozilla.org/en-US/docs/Web/CSS/@view-transition | 2026-11-19, verified (fetched; the compatibility table was not in the fetched text) |
| 2 | Chrome for Developers — Cross-document view transitions: "supported in Chrome from Chrome 126"; `<link rel="expect" href="#id" blocking="render">` keeps the first render until the element is in the DOM | https://developer.chrome.com/docs/web-platform/view-transitions/cross-document | 2026-11-19, verified (fetched) |
| 3 | W3C — CSS View Transitions Module Level 2 (cross-document, `@view-transition`) | https://www.w3.org/TR/css-view-transitions-2/ | 2026-11-19, **no verificado** (not fetched; cited as the spec) |
| 4 | Safari 18.2 support for cross-document view transitions; Firefox status | WebKit / Firefox release notes | **no verificado** — browsers without support simply navigate (graceful fallback, nothing breaks) |
| 5 | Nielsen Norman Group — Back-to-Top Button Design Guidelines (show after several screens of scrolling, bottom-right, small and persistent, clear arrow + label/tooltip) | https://www.nngroup.com/articles/back-to-top/ | 2026-11-19, **no verificado** (fetch returned no usable text) |
| 6 | WCAG 2.2 — 1.4.3 Contrast (Minimum) 4.5:1 text; 1.4.11 Non-text Contrast 3:1 | https://www.w3.org/TR/WCAG22/ | 2026-11-19, standard (ratios computed locally with the WCAG relative-luminance formula) |
| 7 | Local checks: Playwright 1.65 (Chromium) on the built landing at 1366×657, 1440×789, 1920×969 (laptop screens minus browser chrome) and 390×844 | — | 2026-11-19, verified |

## Findings

- A language switch is a real navigation (separate URLs and `hreflang` stay, which SEO needs). The
  standard way to make it feel continuous is a cross-document view transition (1, 2): both pages
  opt in with `@view-transition { navigation: auto; }`; named elements (header, tmux bar) stay in
  place while the rest cross-fades. Unsupported browsers navigate normally.
- Keeping the place: the link gets the current section as `#fragment` when clicked; the new page
  jumps there on load. `rel=expect … blocking=render` on `#site-footer` (the last element) holds
  the snapshot until the whole prerendered page is parsed (2). Measured: with
  `html { scroll-behavior: smooth }` the load-time jump to the fragment was animated (map 541 →
  83 → 24 px over ~1 s); with smooth scrolling enabled only after hydration (`html[data-ready]`)
  the new page lands on the section at once (24 px, the same offset as the old page).
- Back to top (5): a persistent round icon button bottom-right, shown after scrolling, with an
  accessible name and tooltip, is the common pattern; a text link only in the footer is easy to
  miss. Moving focus to `<main>` keeps keyboard users in step.
- Signal band contrast (6), computed: on `#904f00` — `#fbf9f4` 6.04:1, `#f1ede4` 5.44:1,
  `#e6dccb` 4.68:1; on the `#7a4300` cards 7.59 / 6.83 / 5.88:1; button text `#6b3b00` on
  `#fbf9f4` 8.87:1; Ink `#0a0c10` on the amber only 3.08:1, so code blocks are inset in Ink
  instead of sitting on the amber. Orange caution text on amber fails, so the beta notice turns
  Bone in that band.
- Measured fit (no inner scroll, no horizontal scroll): hero window bottom 619 px of 625 at
  1366×657; map board bottom 592; download "other ways" bottom 613; footer exactly
  `100svh − tmux bar` at all three laptop sizes.
