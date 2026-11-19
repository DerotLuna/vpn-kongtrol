<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0066: Landing accent is one gold on every tone; footer facts above the wordmark; fixed-height hero terminal

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Owner feedback on the single-theme landing (ADR-0060), 2026-11-19:

- The deep amber (`#904f00`, the download band and the amber primary buttons on Bone) reads
  brownish. The bright gold used on Ink (header "DESCARGAS" button, the active tmux window, the
  hero ASCII banner) is the colour the owner likes.
- The footer shows GitHub twice (a column link and the `$ git clone` line), and the facts row and
  `no place like 127.0.0.1` sit on top of the giant "KONGTROL" wordmark.
- Typing in the hero terminal (e.g. `help`) makes the whole hero taller; the idle block caret
  shows in front of the placeholder text.

## Decision Drivers

- One accent everywhere, the owner's gold; no brownish amber left on the landing.
- WCAG 2.2 AA: text ≥ 4.5:1, UI parts and focus ≥ 3:1, links not by colour alone.
- Footer and hero keep fitting one laptop screen (1366×768, 1440×900, 1920×1080).
- The dashboard (`web/ui`) is out of scope and keeps its own tokens.

## Considered Options

1. Keep amber on Bone and gold only on Ink (status quo, ADR-0060).
2. A darker "gold" on Bone that passes as text (it ends up brown again).
3. **One gold `#ffb020` on every tone, split by role: fills, accent text, decorative marks.**

## Decision Outcome

Option 3 (owner decision, 2026-11-19). This refines the colour part of ADR-0060 (its Signal band
and the Bone `--signal`); the rest of ADR-0060 stands, so it stays Accepted.

- **Roles** (`landing/src/index.css`): `--signal-fill` for solid fills (primary buttons, the active
  tmux window, active tabs, selection) is gold `#ffb020` with Ink `#0a0c10` text on it (10.70:1),
  hover `#ffc456` (12.38:1); `--signal` for accent text, lines and focus rings is gold on Ink
  (10.70:1) and Ink `#1b1813` on Bone (15.15:1 on `#f1ede4`; gold text on Bone would be 1.57:1);
  `--signal-mark` is decorative gold only (link underlines in running text, section-label rings,
  dots, washes, the hero radar, the footer wordmark) and never the only carrier of meaning.
  Links inside running text get a 2px gold underline (WCAG 1.4.1). `--signal-soft` on Bone is gold
  at 20 %: Ink text on it 13.78:1, dim text `#635c4b` 5.17:1.
- **Download band** (`.tone-signal`): gold `#ffb020` surface, cards `#ffc456`, Ink text (10.70 /
  12.38), muted Ink 82 % (7.67 / 8.57), dim Ink 70 % (5.59 / 6.02; 5.10 on the Ink 8 % wash),
  card borders Ink 18 % (decorative), outlines Ink 55 % (3.65 / 3.82, ≥ 3:1), live `#0b5a2c`
  (4.57 / 5.29), the primary button inverted (Ink fill, gold text 10.70:1; hover `#1d222c` 8.72:1),
  code blocks inset in Ink. Logos stay black there (only `.tone-ink` inverts them).
- **Dashboard**: unchanged; DESIGN.md keeps `signal` / `signal-light` for it and adds
  `landing-gold`, `landing-gold-hi`, `on-landing-gold` for the landing.
- **Footer**: GitHub only once, as the `$ git clone github.com/DerotLuna/vpn-kongtrol` line (it is
  in the terminal voice of the footer and shows the real address; the column link is removed). The
  facts row and the wink sit between the link columns and the wordmark: `.footer-inner` reserves
  the wordmark's height (`--giant-h` = 0.62 × `min(16vw, 30svh, 300px)`) as bottom padding, so the
  wordmark is alone at the bottom. Vertical sizes were tightened (top padding 4.5svh, mark 20svh,
  grid gap 3svh) so it still fits exactly one screen.
- **Hero terminal**: the log column (`.th-scroll`, `role="log"`, focusable, labelled) has a fixed
  height (16.5rem from 1024 px, `min(60svh, 30rem)` below) and scrolls inside with a thin themed
  scrollbar; the prompt line stays pinned under it; each command scrolls to the newest line
  (instantly under reduced motion); `clear` empties it; the input keeps focus. `role="log"`
  announces only new output (implicit `aria-live="polite"`). The idle block caret is removed: the
  input's own gold caret is the only caret. The first render is unchanged between prerender and
  hydration (`check:hydration`).

### Consequences

- Good: one recognisable gold across the page; AA everywhere (ratios above); the footer no longer
  overlaps; commands never resize the hero.
- Neutral: on Bone, small accent text (section commands, `$` prompts outside terminals) is Ink
  with gold marks around it instead of coloured text.
- Bad: none known; the README GIFs (`assets/readme/{hero,routing,attach}.gif`) were regenerated
  in the same change (`node scripts/readme-media.mjs`, each under 2 MB).

### Confirmation

Landing `lint`, `build` (CSP hashes), `check:hydration`, `check:i18n`; Playwright screenshots at
1366×657, 1440×789, 1920×969 (laptop screens minus browser chrome) and 390×844; footer
`scrollHeight == height` at each, facts row above the wordmark; hero terminal height identical
before and after two `help` commands (543 px at 1366, 559 px at 1440, 1060 px at 390).

## More Information

- Code: `landing/src/index.css`, `landing/src/components/sections/{Hero.tsx,download.css,providers.css,attachdemo.css}`,
  `landing/src/components/ui/{Footer.tsx,download-cta.css}`.
- Research: [2026-11-19-landing-gold-contrast](../research/2026-11-19-landing-gold-contrast.md).
- Docs: `DESIGN.md` (Colors, Elevation, Shapes, Components, Do's and Don'ts).
- Refines: [ADR-0060](0060-landing-single-theme-bands-and-line-art-marks.md) (colour part).
