<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: landing polish — motion without layout shift, WCAG 2.2 AA, Core Web Vitals

Version 1.0.0 · 2026-11-18

## Question

How should the landing (`landing/`, prerendered React, terminal/tmux identity) add professional
motion, reach WCAG 2.2 AA and keep Core Web Vitals green on phone and desktop, without new
dependencies?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | web.dev — Cumulative Layout Shift | https://web.dev/articles/cls | 2026-11-17 |
| 2 | web.dev — Optimize CLS | https://web.dev/articles/optimize-cls | 2026-11-17 |
| 3 | web.dev — How to create high-performance CSS animations | https://web.dev/articles/animations-guide | 2026-11-17 |
| 4 | web.dev — prefers-reduced-motion | https://web.dev/articles/prefers-reduced-motion | 2026-11-17 |
| 5 | web.dev — Preload critical assets | https://web.dev/articles/preload-critical-assets | 2026-11-17 |
| 6 | MDN — `animation-timeline: view()` | https://developer.mozilla.org/en-US/docs/Web/CSS/animation-timeline/view | 2026-11-17 |
| 7 | MDN — `Element.getAnimations()` | https://developer.mozilla.org/en-US/docs/Web/API/Element/getAnimations | 2026-11-17 |
| 8 | Chrome for Developers — Same-document view transitions | https://developer.chrome.com/docs/web-platform/view-transitions/same-document | 2026-11-17 |
| 9 | W3C — WCAG 2.2 (SC 1.4.3, 2.2.2, 2.4.1, 2.4.7, 2.5.8) | https://www.w3.org/TR/WCAG22/ | 2026-11-17 |
| 10 | WAI-ARIA APG — Radio Group pattern | https://www.w3.org/WAI/ARIA/apg/patterns/radio/ | 2026-11-17 |
| 11 | WAI-ARIA APG — Disclosure (Show/Hide) pattern, navigation menu example | https://www.w3.org/WAI/ARIA/apg/patterns/disclosure/ | 2026-11-17 |
| 12 | schema.org — SoftwareApplication / codeRepository | https://schema.org/SoftwareApplication · https://schema.org/codeRepository | 2026-11-17 |
| 13 | Google Search Central — Software app structured data | https://developers.google.com/search/docs/appearance/structured-data/software-app | 2026-11-17 |

Sources were consulted from the author's prior reading, not re-fetched in this session:
**no verificado** word for word — re-open a URL before quoting it.

## Findings

- **CLS** counts a shift when a visible element changes position between frames; growing content
  above other content shifts it [1][2]. Measured baseline (Lighthouse 12, mobile): CLS **0.187**,
  caused by the hero terminal mounting its lines one by one and pushing the page down.
- Animating **transform and opacity** stays off layout and paint; layout properties (height,
  max-height, width) re-run layout every frame [3]. The mobile menu animated `max-height`.
- `prefers-reduced-motion: reduce` should remove non-essential motion; a global rule that
  shortens every animation/transition is the common safety net [4].
- Preloading fonts needs `crossorigin` even on the same origin; it removes one hop
  (HTML → CSS → font) for faces used on the first screen [5].
- Scroll-driven animations (`animation-timeline: view()`) run without JS but are not in every
  engine; feature-detect and fall back [6]. `getAnimations()` exposes a running CSS animation's
  `currentTime`, so a replacement node can resume at the same point with a negative delay [7].
- Same-document view transitions accept a custom animation on `::view-transition-new(root)`;
  a `clip-path: circle()` from the click point is the documented theme-toggle pattern [8].
- WCAG 2.2 AA items that applied here [9]: 1.4.3 contrast 4.5:1 (placeholder text included);
  2.2.2 auto-updating content needs pause/stop (the routing board rotated every 2.6s with a
  polite live region announcing each step); 2.4.1 bypass blocks (no skip link); 2.4.7 focus
  visible (the demo input removed its outline); 2.5.8 target size ≥24px (tmux links ~21px,
  copy buttons ~16px).
- A single choice shared by several blocks (the OS picker) is a **radio group**, not tabs:
  one tab stop, arrows move and select [10]. A "jump to section" control is a disclosure
  of links, not a listbox (a listbox must not contain interactive children) [11].
- `codeRepository` is a property of `SoftwareSourceCode`, not of `SoftwareApplication` [12];
  `softwareHelp` expects a `CreativeWork`. Google's software-app rich result also wants
  ratings or reviews [13], which the project does not have, so no rich result is expected —
  the markup is still valid and truthful (free, Apache-2.0, three OSes).

## Measurements (local `vite preview`, Lighthouse 12, headless Chrome)

| Page | Before (perf / a11y / BP / SEO · CLS) | After |
|---|---|---|
| `/` mobile | 87 / 100 / 100 / 100 · CLS 0.187 · LCP 2.4 s | 97 / 100 / 100 / 100 · CLS 0 · LCP 2.3 s (2 runs; a first cold run gave 78 from a TBT spike) |
| `/` desktop | 100 / 100 / 100 / 100 · CLS 0.005 | 100 / 100 / 100 / 100 · CLS 0 |
| `/en/` mobile | 88 / 100 / 100 / 100 · CLS 0.187 · LCP 2.3 s | 97 / 100 / 100 / 100 · CLS 0 · LCP 2.3 s |
| `/en/` desktop | 100 / 100 / 100 / 100 · CLS 0.005 | 100 / 100 / 100 / 100 · CLS 0 |

Lighthouse a11y = 100 before and after: automated checks did not see the issues listed above
(they were found by reading the code and measuring token contrast).
