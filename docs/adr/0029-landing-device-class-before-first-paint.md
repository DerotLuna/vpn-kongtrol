<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0029: Landing marks phones before first paint so the hero never shifts

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer · Refines
[ADR-0027](0027-landing-download-ctas.md)

## Context and Problem Statement

[ADR-0027](0027-landing-download-ctas.md) keeps everything visitor-dependent out of the first
render. On phones the hero therefore painted the desktop buttons from the prerender and swapped
them for the taller "Enviar a mi computador" block after hydration. The hero grew, and the
980px contour behind it (`top: 46%` of the section) moved with it: Lighthouse 12 mobile measured
CLS 0.0905 in 3 of 5 runs on `/` (0 when hydration landed before first paint). Research:
[2026-11-18-landing-hig-material3-review](../research/2026-11-18-landing-hig-material3-review.md#follow-up-hero-cls-and-small-text-on-phones).

## Decision Drivers

- CLS 0 on every run; no hydration mismatch (`pnpm check:hydration`).
- No new request, no new dependency; CSP keeps one hash per inline script.

## Considered Options

1. Reserve the taller block's height in the hero for everyone (empty space on desktop).
2. Decide by CSS media features (`pointer: coarse`): differs from the UA heuristic in edge cases.
3. **The existing inline boot script (`THEME_BOOT`) also sets `<html data-device="mobile">`
   with the same test as `detectMobile()`; the hero renders both variants and CSS shows one.**

## Decision Outcome

Chosen: **option 3**. `<html>` is outside the React root, so hydration is unaffected; both
variants are in the prerender and in the first client render. The hidden one is `display:none`
(not focusable, not announced). Also: the contour is anchored a fixed distance from the section
top, and the "send" button keeps both labels in one grid cell so the Web Share label switch
after hydration cannot move it.

### Consequences

- Good: CLS 0.0000 in 5/5 Lighthouse runs; same heuristic as before.
- Bad: the heuristic lives twice (`seo.ts` string + `useVisitor.ts`), cross-referenced in both;
  editing the boot script changes its CSP hash (the build fails until `public/_headers` is updated).

### Confirmation

Lighthouse 12 mobile, `vite preview`, same machine:

| Page | Before (CLS per run) | After (CLS per run) | Accessibility before → after |
|---|---|---|---|
| `/` | 0.0905, 0, 0.0905, 0.0905, 0 | 0, 0, 0, 0, 0 | 97 → 100 |
| `/guia` | 0, 0 | 0, 0 | 100 → 100 |

## More Information

Code: `landing/src/seo.ts` (`THEME_BOOT`), `landing/src/components/sections/Hero.tsx`,
`landing/src/components/ui/{SendToComputer.tsx,download-cta.css}`, `landing/src/index.css`
(`.th-contour`), `landing/scripts/check-hydration.mjs`, `landing/public/_headers`.
