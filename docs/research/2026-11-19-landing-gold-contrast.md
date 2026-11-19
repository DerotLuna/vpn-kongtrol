<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Landing gold accent: contrast and a fixed-height terminal log

Version 1.0.0 · 2026-11-19 · Backs [ADR-0066](../adr/0066-landing-gold-accent-footer-and-fixed-hero-terminal.md)

## Question

Can the landing use one bright gold (`#ffb020`) on Bone, Ink and a gold download band while every
text and UI part stays WCAG 2.2 AA? How should the hero's demo terminal keep a fixed size?

## Sources (accessed 2026-11-19)

- WCAG 2.2, SC 1.4.3 Contrast (Minimum): https://www.w3.org/TR/WCAG22/#contrast-minimum
- WCAG 2.2, SC 1.4.11 Non-text Contrast: https://www.w3.org/TR/WCAG22/#non-text-contrast
- WCAG 2.2, SC 1.4.1 Use of Color: https://www.w3.org/TR/WCAG22/#use-of-color
- WCAG 2.2 relative luminance / contrast ratio definitions: https://www.w3.org/TR/WCAG22/#dfn-relative-luminance
- WAI-ARIA 1.2, `log` role (implicit `aria-live="polite"`): https://www.w3.org/TR/wai-aria-1.2/#log
- Chrome 130 keyboard-focusable scrollers: https://developer.chrome.com/blog/keyboard-focusable-scrollers
- MDN `Element.scrollTo()` (`behavior`): https://developer.mozilla.org/en-US/docs/Web/API/Element/scrollTo

## Findings

Ratios computed with the WCAG formula (alpha colours flattened over their surface):

| Pair | Ratio | Use |
|---|---|---|
| Ink `#0a0c10` on gold `#ffb020` | 10.70 | primary button text, band text |
| Ink on gold hover `#ffc456` | 12.38 | button hover, band cards |
| gold on Ink `#0a0c10` / `#1d222c` | 10.70 / 8.72 | accent text on Ink, inverted band button |
| gold on Bone `#f1ede4` | 1.57 | fails as text and as UI part: decorative only |
| Ink `#1b1813` on Bone | 15.15 | accent text on Bone |
| Ink on gold 20 % over Bone | 13.78 | inline code, auto chips |
| dim `#635c4b` on gold 20 % over Bone / surface | 5.17 / 5.62 | dim text on washes |
| Ink 82 % on `#ffb020` / `#ffc456` | 7.67 / 8.57 | band muted text |
| Ink 70 % on `#ffb020` / `#ffc456` / Ink 8 % wash | 5.59 / 6.02 / 5.10 | band dim text |
| Ink 55 % on `#ffb020` / `#ffc456` | 3.65 / 3.82 | band outlines (≥ 3:1) |
| `#0b5a2c` on `#ffb020` / `#ffc456` | 4.57 / 5.29 | live state in the band |
| `#b3261e` on `#ffb020` | 3.57 | fails: danger/caution use Ink in the band |

- Gold text on Bone cannot pass, so on Bone the accent text is Ink and gold appears as fills,
  washes and underlines; links in running text keep a non-colour cue (underline), SC 1.4.1.
- A scroll container needs keyboard access: Chrome 130+ focuses scrollers itself, other browsers
  do not, so the log gets `tabIndex=0` and an accessible name. `role="log"` announces additions
  only, not the static boot lines.
- `scrollTo({ behavior })` with `'auto'` under `prefers-reduced-motion: reduce` jumps instantly.
