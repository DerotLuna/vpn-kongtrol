<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0028: Landing review with Apple HIG and Material 3 — text floor, targets, states, readable guide

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

The landing kept its terminal/tmux identity after [ADR-0024](0024-landing-motion-a11y-performance.md),
but a review against Apple HIG and Material 3 found text below the 11px floor, small targets,
links told apart by color only, hover effects on things you cannot click, a copy button that
failed silently, 3-column guide tables cut off on phones, long lines, and prose set in monospace.
Research: [2026-11-18-landing-hig-material3-review](../research/2026-11-18-landing-hig-material3-review.md).
Buttons, hero, header, download section, router diagram and attach demo are out of scope
([ADR-0027](0027-landing-download-ctas.md)).

## Decision Drivers

- Non-technical visitors read the guide and the legal pages on phones.
- Keep identity (tokens, fonts, terminal labels, contour rings) and copy; token names stay stable.
- No new dependencies; prerender + hydration unchanged (`pnpm check:hydration`).
- Lighthouse mobile must not regress.

## Considered Options

1. Leave as is (ADR-0024 already meets WCAG 2.2 AA in automated checks).
2. **Apply HIG/M3 rules through shared tokens and the components this scope owns.**
3. Adopt Material Web components.

## Decision Outcome

Chosen: **option 2**. Option 3 adds a dependency and replaces the identity; option 1 leaves the
issues above.

- Tokens (`landing/src/index.css`): `--text-min: 0.6875rem` (HIG 11 pt minimum = M3
  label-small), `--step-2`, M3 `--ease-standard`, `--dur-short` 150ms, `--dur-medium` 250ms,
  `--state-hover` 8%, `--state-press` 10%. Existing token names unchanged.
- Every label in the owned sections uses at least `--text-min`.
- Targets: footer legal links, OS selector, "ver más" disclosures ≥44px; copy buttons keep their
  28px look with a 44px hit area.
- States: hover/press state layers only on interactive elements; hover removed from security
  rows, "bajo el capó" rows, step cards and table rows. Copy button says when it could not copy.
- Guide/legal readability: prose links underlined; text and callouts capped at `--measure`;
  callouts are tonal containers with a hairline instead of a thick left stripe; section numbers
  use the same ring as the quick-start steps; 3-column tables become one block per row under
  600px (column names via `data-label`); 2-column tables no longer force a sideways scroll;
  legal lists are indented.
- Typographic quotes in the terms ("tal cual" / "as is"); terms version 1.3.1.

### Consequences

- Good: one text floor and one motion/state vocabulary for later work; guide readable on phones.
- Bad: the stacked tables need `data-label` on new 3-column tables (otherwise they still work,
  without the per-value column name).
- Not done here (other owners): text below 11px in `routeboard.css` and `attachdemo.css`.

## Lighthouse 12, mobile (median of 3, `vite preview`)

`before` = `origin/main` at 3220c92, `after` = this change merged on it; runs alternated on the
same machine.

| Page | Perf | A11y | BP | SEO | LCP | CLS | TBT |
|---|---|---|---|---|---|---|---|
| `/` before | 95 | 97 | 100 | 100 | 2.49 s | 0.000 | 59 ms |
| `/` after | 95 | 97 | 100 | 100 | 2.32 s | 0.091 | 6 ms |
| `/guia` before | 96 | 97 | 100 | 100 | 2.44 s | 0 | 11 ms |
| `/guia` after | 96 | 100 | 100 | 100 | 2.45 s | 0 | 23 ms |

`/guia` accessibility 97 → 100 (`link-in-text-block` fixed). On `/` the CLS of 0.091 is the hero
contour (`.hero-contour`) and shows up intermittently on both builds (3 extra runs: 2 of 3 on each),
so it is not caused by this change; the remaining `/` accessibility item is the attach demo's
`.attach-app-btn` contrast (2.4:1). Both belong to [ADR-0027](0027-landing-download-ctas.md)'s scope.

## More Information

Code: `landing/src/index.css`, `landing/src/components/sections/{Guide,HowItWorks}.tsx`,
`landing/src/components/sections/providers.css`, `landing/src/components/ui/{CodeBlock,Footer}.tsx`,
`landing/src/content/guide.{es,en}.tsx`, `landing/src/content/legal/terminos.tsx`.
