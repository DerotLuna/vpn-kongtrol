<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0031: Landing load performance — inline CSS, hashed fonts, immutable caching, quieter main thread

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer · Builds on
[ADR-0022](0022-landing-on-workers-static-assets.md), [ADR-0026](0026-landing-hydrate-prerendered-html.md)

## Context and Problem Statement

Live audits of https://kongtrol.app (PageSpeed mobile 98, Best Practices 92; Cloudflare speed
test mobile 90, TBT 280 ms) reported: `Permissions-Policy: interest-cohort` unrecognized; hashed
assets served with `max-age=0, must-revalidate`; three render-blocking stylesheets (`fonts.css`
and two Vite CSS files, ~450 ms estimated on mobile); a forced reflow; six non-composited
animations; one long main-thread task on load. Research with sources:
[2026-11-18-landing-load-performance](../research/2026-11-18-landing-load-performance.md).
The Cloudflare-injected scripts (Web Analytics beacon, challenge platform) are dashboard settings
the maintainer turns off; the CSP is not loosened for them.

## Decision Drivers

- Standards and vendor docs first (Cloudflare `_headers`, RFC 8246, Lighthouse, React 18).
- CSP unchanged (`style-src` already allows `'unsafe-inline'`; no new script hash).
- No hydration mismatch (`pnpm check:hydration`), same visual design, reduced motion respected.
- No new dependency.

## Considered Options

Render-blocking CSS:
1. Inline only `fonts.css`; keep two stylesheets (still blocking).
2. `build.cssCodeSplit: false` (one file, still one blocking request).
3. Critical-CSS tool (Beasties): its non-blocking strategies need an `onload` attribute or a
   script (CSP), and on a fully prerendered page it inlines nearly everything anyway.
4. **Inline the page's whole bundled CSS (~15 KB gzip) at prerender time and bundle the fonts.**

Long task: code-splitting the demos with `React.lazy` needs streaming SSR (`renderToString`
cannot resolve lazy components); **`<Suspense>` boundaries without lazy** keep `renderToString`
and let React 18 hydrate each boundary in its own task.

## Decision Outcome

1. `interest-cohort=()` removed from `Permissions-Policy` (FLoC is gone; Chrome warns).
2. `_headers`: `/assets/*` → `public, max-age=31536000, immutable` (Vite names are content
   hashes); `/favicon.svg`, `/logo*`, `/og-1200x630.jpg` → `public, max-age=86400` (stable URLs
   linked from outside; ETag revalidation after a day). HTML keeps Cloudflare's default. Per the
   Cloudflare docs a `_headers` value overrides the default; `! Cache-Control` is not needed
   because no earlier rule sets it.
3. Fonts are imported by `src/index.css` from `assets/brand/fonts` (alias `@brand`), so Vite
   hashes them into `/assets/`; a build plugin adds the two font preloads with the hashed URLs;
   `public/fonts/` is gone and `404.html` receives the `@font-face` rules at prerender.
   `scripts/prerender.mjs` replaces the stylesheet links with one `<style>` at the end of
   `<head>` and deletes the unused `.css` files (fails if anything still references them).
4. Forced reflow: `useReveal` reads all positions before writing; `Nav` reads `scrollY` once per
   frame (rAF) and gets its height from a `ResizeObserver` instead of `offsetHeight`.
5. Non-composited animations: the attach demo's color/background/stroke animations became
   stacked layers cross-faded with `opacity`; the switchboard list's dashed wire moves with
   `transform` instead of `background-position`; both demos (and the board's auto-rotation)
   run only while on screen (`useInView`, `.is-live`); the OS tabs no longer animate the
   detected-OS switch (transition only on `:hover`). The desktop SVG wire (`stroke-dashoffset`)
   stays, but only runs while visible.
6. Long task: the hero's 1-second uptime clock moved into `BootRows`, so a tick re-renders three
   rows instead of the hero; home's below-the-fold sections and footer are memoized `<Suspense>`
   boundaries (memoized so a re-render of `App` before they hydrate does not discard their HTML);
   on `/guia`, the command reference and the footer.

### Consequences

- Good: no render-blocking request; fonts and JS cached for a year; demo loops cost nothing off
  screen; hydration split into shorter tasks.
- Bad: each HTML page carries its CSS (~15 KB gzip), not shared across pages in the cache; a
  changed logo/og image can be stale for up to 24 h; the attach demo's check mark fades in
  instead of being drawn.
- Bad (lab): on `/guia` mobile, TBT went from ~23 ms to ~144 ms (median; 1 of 5 runs 18 ms).
  Total main-thread work did not grow and the longest task shrank; with the CSS inline the first
  paint comes earlier in Lighthouse's simulation, so the hydration script now falls inside the
  FCP→TTI window that TBT counts. Still under the 200 ms "good" threshold. On `/guia` only the
  command reference and the footer are separate boundaries: the rest of `Guide` takes `os`,
  which changes right after hydration and would force a client render of any boundary.

### Confirmation

`pnpm install --frozen-lockfile`, `typecheck`, `check:i18n`, `build`, `check:hydration`
(40 runs, no mismatch). Built `dist/` served with the `_headers` CSP; every ES/EN page loaded in
headless Chrome (desktop, mobile, reduced motion) and scrolled to the end: no console errors and
no CSP violations (only the expected 404 status on the 404 page). `.is-live` verified on/off.

Lighthouse 12.8.2, `vite preview`, before/after interleaved on the same machine (other Chrome
processes running, so single runs vary widely), medians of 5:

| Page · device | Perf | FCP ms | LCP ms | TBT ms | SI ms | Render-blocking (est. FCP savings) | Non-composited animations | Best Practices |
|---|---|---|---|---|---|---|---|---|
| `/` mobile before | 97 | 1723 | 2308 | 51 | 1813 | 450 ms | 12 | 100 |
| `/` mobile after | 97 | 1716 | 2313 | 33 | 1738 | 0 | 0 | 100 |
| `/guia` mobile before | 96 | 1962 | 2442 | 23 | 1962 | 300 ms | 0 | 100 |
| `/guia` mobile after | 95 | 1960 | 2352 | 144 | 1960 | 0 | 0 | 100 |
| `/` desktop before | 100 | 413 | 508 | 0 | 499 | 100 ms | 12 | 100 |
| `/` desktop after | 100 | 421 | 506 | 0 | 515 | 0 | 0 | 100 |
| `/guia` desktop before | 100 | 452 | 544 | 0 | 452 | 100 ms | 0 | 100 |
| `/guia` desktop after | 100 | 449 | 517 | 0 | 451 | 0 | 0 | 100 |

Per-run mobile `/` performance: before 92, 97, 97, 97, 97 (TBT 246, 105, 51, 20, 34); after
97 ×5 (TBT 33, 44, 50, 14, 3). Best Practices is 100 locally on both sides: the live 92 comes
from the Cloudflare-injected scripts and the `interest-cohort` warning, neither present in
`vite preview` (not verified on production until deployed). `vite preview` serves no
`_headers`, so the caching gain is not in these numbers; it shows on repeat visits only.

## More Information

Code: `landing/public/_headers`, `landing/vite.config.ts`, `landing/scripts/{prerender,copy-brand}.mjs`,
`landing/src/index.css`, `landing/src/seo.ts`, `landing/src/hooks/{useInView,useReveal}.ts`,
`landing/src/components/ui/Nav.tsx`, `landing/src/components/sections/{Hero,Features,AttachDemo}.tsx`,
`landing/src/components/sections/{attachdemo,routeboard}.css`, `landing/src/components/sections/Guide.tsx`,
`landing/src/pages/{home,guia}/App.tsx`.
Docs: `docs/deploy-cloudflare.md`, `landing/README.md`.
