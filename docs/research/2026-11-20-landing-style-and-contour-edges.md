<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Landing style name and contour section edges

Version 1.0.0 · 2026-11-20 · Backs [ADR-0070](../adr/0070-landing-instrument-terminal-style-contour-edges-and-short-gold-band.md)

## Question

Which established web style is Kongtrol's landing closest to, so we can name it and lean on
others' experience? How do others draw non-straight section edges, and what must they respect
(accessibility, performance, CSP)?

## Sources (accessed 2026-11-20)

- daisyUI design trends, "Terminal / CLI UI": https://trends.daisyui.com/trend/terminal-cli-ui/
- daisyUI, "Cassette Futurism" (amber screens, industrial panels, analog controls):
  https://trends.daisyui.com/trend/cassette-futurism/
- daisyUI, "FUI / HUD" (telemetry, grids, diagrams): https://trends.daisyui.com/trend/fui-hud/
- daisyUI, "Retro-futurism": https://trends.daisyui.com/trend/retro-futurism/
- LogRocket, "Retro-futuristic UX designs" (keep it modern and functional under the skin):
  https://blog.logrocket.com/ux-design/retro-futuristic-ux-designs-bringing-back-the-future
- UX Planet, "UI design styles explained" (neubrutalism: raw, confrontational; thick black
  borders, hard shadows): https://uxplanet.org/ui-design-styles-explained-670cfabdf430
- CSS Tip, "A CSS-only wavy divider": https://css-tip.com/wavy-divider/
- freeCodeCamp, "Section divider using CSS" (clip-path / SVG masks):
  https://www.freecodecamp.org/news/section-divider-using-css/
- CodyHouse, section divider component (decorative SVG, `aria-hidden`):
  https://codyhouse.co/ds/components/info/section-divider
- MDN, `mask` and `mask-image`: https://developer.mozilla.org/en-US/docs/Web/CSS/mask
- web.dev, `prefers-reduced-motion`: https://web.dev/articles/prefers-reduced-motion
- MDN, CSS scroll-driven animations (`animation-timeline: view()`):
  https://developer.mozilla.org/en-US/docs/Web/CSS/animation-timeline

## Findings

1. **Style.** Trend catalogs name "Terminal / CLI UI" (monospace, box drawing, prompts, amber or
   green text) and "Cassette Futurism" (amber screens, instrument panels) as live families; FUI/HUD
   covers telemetry and diagrams. Kongtrol's landing already is Terminal/CLI UI (the hero is an SSH
   session, tmux bar, `$` commands) with an instrument-panel layer (radar sweep, contour rings,
   routing patch panel). The sources warn against pure nostalgia: the look is a skin over modern,
   usable patterns. None of them uses topographic contour lines as the structural motif; that part
   is ours (no published name found, **no verificado** as absent everywhere).
2. **Not neubrutalism.** Neubrutalism is raw and confrontational (thick black borders, hard offset
   shadows, flat saturated blocks). The full-width gold band with a black button drifted towards it,
   which is part of why it felt heavy next to the calm rest of the page.
3. **Large saturated areas.** A saturated accent works best in small doses; the common 60-30-10
   heuristic (60 % neutral, 30 % secondary, 10 % accent) is a rule of thumb, not a standard
   (**no verificado** as research-backed). A 1.5-screen gold band is far beyond it.
4. **Edges.** Common techniques: CSS gradients (simple waves), `clip-path` (angles) and SVG used as
   a mask (any shape, scales). SVG masks fit custom brand shapes; the SVG is decorative, so it stays
   out of the accessibility tree (a CSS mask never enters it). `preserveAspectRatio="none"` lets one
   path stretch to any width; keep the depth small on phones so curves do not flatten into noise.
   Animated dividers must honour `prefers-reduced-motion`.
5. **CSP.** The landing's CSP allows `img-src 'self' data:`; Vite inlines the small edge SVGs as
   `data:` URLs in the CSS, so masks load under the current policy with no change.

## What we take

- Name the style **"Terminal de instrumentos"**: Terminal/CLI UI + instrument panel, warm (Bone
  paper, one gold), with contour lines as the structural signature.
- Shorten the gold band to the call to action; the rest goes to the Bone page.
- Contour-line edges from the mark's rings, varied per boundary, some straight on purpose; static
  masks, plus one decorative scroll-drawn echo that is off with reduced motion.
