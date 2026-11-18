<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: landing review against Apple HIG and Material Design 3

Version 1.0.0 · 2026-11-18

## Question

Which ideas from Apple's Human Interface Guidelines (HIG) and Material Design 3 (M3) apply to the
landing (home, `/guia`, legal pages, ES+EN, desktop + phone, dark + light), so it reads as
professional and is easy for non-technical visitors? Scope: everything except the hero, header,
"Para todos", download section, router diagram, attach demo and button base styles
(covered by [ADR-0027](../adr/0027-landing-download-ctas.md)).

## Sources (accessed 2026-11-18)

Apple pages were read from the JSON that renders developer.apple.com
(`/tutorials/data/design/human-interface-guidelines/<page>.json`); the HTML is client-rendered.
m3.material.io is client-rendered too, so M3 values come from Google's own token source
(material-web) and Android docs.

| # | Source | What it says (short) |
|---|---|---|
| 1 | [HIG — Buttons](https://developer.apple.com/design/human-interface-guidelines/buttons) | "a button needs a hit region of at least 44x44 pt"; one primary role per view |
| 2 | [HIG — Typography](https://developer.apple.com/design/human-interface-guidelines/typography) | iOS default 17 pt, minimum 11 pt; hierarchy by weight, size and color; avoid mixing too many typefaces |
| 3 | [HIG — Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility) | WCAG AA contrast (4.5:1 up to 17 pt); "Convey information with more than color alone" |
| 4 | [HIG — Feedback](https://developer.apple.com/design/human-interface-guidelines/feedback) | Feedback tells people the result of an action; put status near the item it describes |
| 5 | [HIG — Motion](https://developer.apple.com/design/human-interface-guidelines/motion) | "Aim for brevity and precision in feedback animations"; avoid unnecessary motion |
| 6 | [HIG — Layout](https://developer.apple.com/design/human-interface-guidelines/layout) | Order content by importance; alignment and indentation convey hierarchy |
| 7 | [material-web tokens: md.sys.motion](https://github.com/material-components/material-web/blob/main/tokens/versions/latest/sass/_md-sys-motion.scss) | `easing-standard` `cubic-bezier(0.2, 0, 0, 1)`; `duration-short3` 150ms, `medium1` 250ms |
| 8 | [material-web tokens: md.sys.state](https://github.com/material-components/material-web/blob/main/tokens/versions/latest/sass/_md-sys-state.scss) | hover state layer 0.08, pressed 0.10, focus 0.10 |
| 9 | [material-web tokens: md.sys.typescale](https://github.com/material-components/material-web/blob/main/tokens/versions/latest/sass/_md-sys-typescale.scss) | body-large 1rem/1.5rem; label-small 0.6875rem (11px); headline-small 1.5rem |
| 10 | [Android — Compose accessibility API defaults](https://developer.android.com/develop/ui/compose/accessibility/api-defaults) | Material components keep a 48dp minimum touch target |
| 11 | [WCAG 2.2 — 1.4.1 Use of Color](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color) / [2.5.8 Target Size (Minimum)](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum) | links in text need a non-color cue; targets ≥24px |

The M3 guideline pages themselves (m3.material.io: states, touch targets, containers) could not be
fetched as text: **no verificado** beyond the token files and the Android doc above.

## Findings (before)

Measured on a local production build (`vite preview`), 390px and 1440px, dark and light.

- **Text below the 11px floor** [2, 9]: code-block language label 10.2px, table headers 10.4px,
  footer facts 9.9px, badges and the guide sidebar title 10.6px, provider OS states 10.2px.
- **Small targets** [1, 10, 11]: footer legal links ~20px tall; copy buttons 28px; OS selector 40px;
  "ver más" disclosures ~40px.
- **Links told apart by color only** in guide/legal prose (Lighthouse `link-in-text-block` failed
  on `/guia`) [3, 11].
- **Hover on things you cannot click** (security rows, "bajo el capó" rows, step cards lifting,
  table rows): M3 applies state layers to interactive components only [8]; on a page for
  non-technical visitors it suggests a click that does nothing.
- **Silent failure**: the copy button did nothing when the clipboard was refused [4].
- **Wide tables on phones**: 3-column guide tables scrolled sideways inside a 390px screen;
  the third column (what to check / what to do) was cut off [6].
- **Callouts** used a thick colored left stripe; M3 uses tonal containers.
- **Line length**: the guide intro and the "¿Primera vez?" card ran up to ~150 characters per line
  on desktop (readability target 45–75ch).
- **Guide section numbers** were an 11px digit floating above each heading, not tied to the
  quick-start's numbered rings.
- **Prose set in monospace**: the step-card notes ("después, más VPN desde el panel…") looked like
  terminal output.
- **Typographic quotes**: Space Grotesk draws `"` as a closing quote, so `"tal cual"` read as
  `”tal cual”` in the terms heading.
- The footer logo had no width/height (Lighthouse `unsized-images`).

## Lighthouse 12, mobile (median of 3, same machine, `vite preview`)

See [ADR-0028](../adr/0028-landing-hig-material3-review.md) for the table; `before` = `origin/main`
at 3220c92, `after` = this change merged on it.
