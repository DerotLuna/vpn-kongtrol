---
name: Kongtrol
description: "Sesión en la Torre" — the site is a guest terminal session inside Kongtrol's control tower.
colors:
  bg: "#0a0c10"
  surface: "#12151b"
  surface-2: "#171b23"
  surface-3: "#1d222c"
  border: "#232833"
  border-2: "#313847"
  signal: "#ffb020"
  signal-hi: "#ffc456"
  on-signal: "#0a0c10"
  steel: "#7fb4d4"
  live: "#3ecf72"
  caution: "#ff8a3d"
  danger: "#ff5a52"
  text: "#eef1f5"
  text-muted: "#99a3b3"
  text-dim: "#8e98a8"
  bg-light: "#f1ede4"
  surface-light: "#fbf9f4"
  surface-2-light: "#f4f0e6"
  border-light: "#ddd5c4"
  border-2-light: "#c9bfa8"
  signal-light: "#904f00"
  signal-hi-light: "#7a4300"
  on-signal-light: "#fbf9f4"
  steel-light: "#2f6a8f"
  live-light: "#126f39"
  caution-light: "#a54006"
  danger-light: "#b3261e"
  text-light: "#1b1813"
  text-muted-light: "#4c4638"
  text-dim-light: "#635c4b"
typography:
  display:
    fontFamily: "Space Grotesk, system-ui, sans-serif"
    fontSize: "clamp(2rem, 1.5rem + 2.4vw, 3rem)"
    fontWeight: 600
    lineHeight: 1.08
    letterSpacing: "-0.02em"
  headline:
    fontFamily: "Space Grotesk, system-ui, sans-serif"
    fontSize: "clamp(1.5rem, 1.3rem + 1vw, 2.1rem)"
    fontWeight: 600
    lineHeight: 1.08
    letterSpacing: "-0.01em"
  title:
    fontFamily: "Space Grotesk, system-ui, sans-serif"
    fontSize: "clamp(1.15rem, 1.08rem + 0.35vw, 1.35rem)"
    fontWeight: 600
    lineHeight: 1.08
    letterSpacing: "-0.01em"
  body:
    fontFamily: "Inter, system-ui, sans-serif"
    fontSize: "clamp(1rem, 0.96rem + 0.2vw, 1.08rem)"
    fontWeight: 400
    lineHeight: 1.65
  label:
    fontFamily: "IBM Plex Mono, ui-monospace, monospace"
    fontSize: "0.76rem"
    fontWeight: 500
    letterSpacing: "0.04em"
  button:
    fontFamily: "IBM Plex Mono, ui-monospace, monospace"
    fontSize: "0.82rem"
    fontWeight: 600
    letterSpacing: "0.04em"
  label-min:
    fontFamily: "IBM Plex Mono, ui-monospace, monospace"
    fontSize: "0.6875rem"
    fontWeight: 400
rounded:
  tmux: "3px"
  sm: "6px"
  tile: "11px"
  lg: "12px"
  pill: "999px"
spacing:
  "1": "4px"
  "2": "8px"
  "3": "12px"
  "4": "16px"
  "5": "24px"
  "6": "32px"
  "7": "48px"
  "8": "64px"
  tap: "44px"
components:
  button-primary:
    backgroundColor: "{colors.signal}"
    textColor: "{colors.on-signal}"
    typography: "{typography.button}"
    rounded: "{rounded.sm}"
    padding: "12px 22px"
    height: "44px"
  button-primary-hover:
    backgroundColor: "{colors.signal-hi}"
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.text}"
    rounded: "{rounded.sm}"
    padding: "12px 20px"
    height: "44px"
  terminal-window:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
  terminal-titlebar:
    backgroundColor: "{colors.surface-2}"
    padding: "12px 16px"
  tmux-bar:
    backgroundColor: "{colors.surface-2}"
    typography: "{typography.label-min}"
    height: "32px"
  tmux-window-active:
    backgroundColor: "{colors.signal}"
    textColor: "{colors.on-signal}"
    rounded: "{rounded.tmux}"
  provider-card:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.sm}"
    padding: "14px"
  provider-monogram:
    textColor: "{colors.text}"
    rounded: "{rounded.tile}"
    size: "44px"
  os-chip:
    textColor: "{colors.text-muted}"
    typography: "{typography.label-min}"
    rounded: "{rounded.pill}"
---

<img src="assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Design System: Kongtrol

Versión 1.1.0 · 2026-11-19

## Overview

**Creative North Star: "Sesión en la Torre"**

The site is not a page about Kongtrol; it is a guest session inside Kongtrol's control tower (`ssh guest@kongtrol.tower`). Everything a visitor reads arrives as terminal output: a "Last login" line, a block-letter ASCII banner, `#` comments, commands with `$`, tunnel rows with `✔`, navigation written as CLI flags (`--inicio`, `--guia`) and a tmux status bar pinned to the bottom with numbered windows (`0:motd`, `1:empezar`…), the kill-switch state (`armed`) and a clock. Plain-language sentences (Inter) sit inside that session so non-technical visitors are never left decoding jargon; the terminal is the frame, not a test.

The material is amber phosphor on near-black in the dark theme and ink on bone paper in the light theme, with a faint field of concentric contour rings echoing the monkey mark's line art. Density is calm: one terminal window per screen, generous section rhythm, no decorative imagery. Motion is reserved for things that explain: the boot timeline, the routing diagram, the attach demo, the theme wipe. Components themselves stay precise and quiet.

**Key Characteristics:**
- Terminal session as the page frame (window chrome, ASCII banner, tmux bar, CLI-flag navigation).
- One brand accent (amber signal); every other color is a state.
- Three type voices: Space Grotesk headings, Inter prose, IBM Plex Mono for everything machine-like.
- Precise components, expressive demos.
- Two complete themes (follows the OS preference until the visitor picks one; light on bone paper) plus a hidden green-phosphor CRT mode (↑↑↓↓←→←→BA).

## Colors

One warm accent against cool neutrals in the dark theme and warm neutrals in the light theme; color carries meaning, never decoration.

### Primary
- **Phosphor Amber** (`signal`; light `signal-light`): the single brand accent. Primary actions, links, focus rings, the active tmux window, the ASCII banner, the `$` prompt, the "se adjunta" provider group and "auto" OS chips. Hover deepens to `signal-hi`.

### Secondary
- **Instrument Steel** (`steel`; light `steel-light`): secondary information and the "conecta directo" provider group; also the "connecting" state. Never used for primary actions.

### Tertiary (states)
- **Live Green** (`live`; light `live-light`): a tunnel that is up and healthy, or a check that passed. Nothing else.
- **Caution Orange** (`caution`; light `caution-light`): a tunnel that is up but not answering ("Sin respuesta", WireGuard liveness, ADR-0030) or any degraded state.
- **Fault Red** (`danger`; light `danger-light`): error, dropped, blocked.

### Neutral
- **Tower Night** (`bg`, `surface`, `surface-2`, `surface-3`): page, terminal body, title bars and raised rows, stepped by tonal lightness.
- **Bone Paper** (`bg-light`, `surface-light`, `surface-2-light`): the light theme, matched to the ink-on-paper line art of the logo.
- **Hairlines** (`border`, `border-2` and light pairs): 1px dividers and outlines; `border-2` for interactive outlines.
- **Ink** (`text`, `text-muted`, `text-dim` and light pairs): primary, secondary and tertiary text. `text-dim` keeps ≥ 4.5:1 on every surface.

### Named Rules
**The One Accent Rule.** Amber is the only brand color. If something is amber, it is the brand or the action; it is never a state.

**The Green Means Up Rule.** Green means up or OK: a connected or healthy tunnel, a passed check (`✔` in terminal output), the attach demo's "Conectado". Groups, categories, "auto" support and decoration never use green.

**The Four States Rule.** Connected = green, connecting = steel, no response = caution orange, error = red. Each state also carries a word or icon; color is never the only signal.

## Typography

**Display Font:** Space Grotesk (with system-ui)
**Body Font:** Inter (with system-ui)
**Label/Mono Font:** IBM Plex Mono (with ui-monospace)

**Character:** Space Grotesk gives headings an engineered, slightly quirky geometry; Inter carries the plain-language sentences; IBM Plex Mono is the voice of the machine — commands, flags, labels, the tmux bar, filenames.

### Hierarchy
- **Display** (600, fluid 2–3rem, 1.08, -0.02em): section titles; balanced, ≤ 30ch.
- **Headline** (600, fluid 1.5–2.1rem, 1.08): guide section titles and sub-sections.
- **Title** (600, fluid 1.15–1.35rem, 1.08): card and group headings.
- **Body** (400, fluid 1–1.08rem, 1.65): prose, max 65ch (`--measure`).
- **Label** (500, 0.76rem, 0.04em): nav flags, chips, table heads.
- **Button** (600, 0.82rem, 0.04em, uppercase on primary): button text.
- **Label-min** (400, 0.6875rem): the 11px floor; tmux bar and fine print. Nothing renders smaller.

### Named Rules
**The Machine Speaks Mono Rule.** Anything a computer would print or accept — commands, flags, paths, IPs, filenames, states — is set in IBM Plex Mono. Sentences meant for people are Inter.

**The 11px Floor Rule.** No rendered text below 11px, on any screen size (ADR-0028).

## Layout

Single column, max width 1160px, fluid gutter `clamp(16px, 3vw, 24px)`, section rhythm `clamp(72px, 6vw + 40px, 120px)`. Spacing follows a 4px base scale (4 → 64). A fixed 66px header and a fixed 32px tmux bar frame the scroll; anchored sections offset for both. Main breakpoints: 1080, 960, 720, 640, 520 and 420px. Grids collapse 4 → 3 → 2 → 1; the routing diagram swaps to an HTML list under 768px so text never scales below the floor. Interactive targets are 44px (24px minimum for dense inline controls).

## Elevation & Depth

Depth is tonal, not shadowed: `bg` → `surface` → `surface-2` → `surface-3` and 1px hairlines separate layers. Shadows appear in exactly two places — the terminal window, which floats above the page, and the primary button's amber glow on hover. The ambient contour field and two soft radial washes (amber top-right, steel bottom-left) give the page atmosphere without lifting any element.

### Shadow Vocabulary
- **Terminal float** (`box-shadow: 0 0 0 1px rgba(255,176,32,0.05), 0 30px 70px rgba(0,0,0,0.45)`; light `0 18px 44px rgba(60,50,25,0.12)`): the hero terminal window only.
- **Signal glow** (`box-shadow: 0 10px 28px -6px var(--signal-glow)`): primary button hover only.

### Named Rules
**The Flat-By-Default Rule.** Surfaces are flat at rest. Only the terminal window floats; everything else gains depth through tone or responds to interaction.

## Shapes

Mostly squared and technical: 6px corners on buttons, cards and code blocks; 12px on the terminal window and large panels; 11px on the 44px provider monogram tiles; 3px on tmux windows; full pills only for OS chips and status chips. Borders are 1px hairlines; focus is a 2px amber outline offset 3px. Circles belong to the brand: contour rings, status dots, the macOS-style window dots, and the tiny contour-ring glyph (11px) that opens every section label.

## Components

### Buttons
- **Shape:** gently squared (6px).
- **Primary:** solid amber, ink text, IBM Plex Mono 600 uppercase, 12px × 22px, 44px tall, download icon on the left.
- **Hover / Focus:** amber deepens, lifts 1px, soft amber glow; press scales to 0.98; 2px amber focus ring.
- **Secondary:** transparent with a `border-2` hairline and text color; on hover the border, text and a 12% amber wash turn amber. One primary per section.

### Chips
- **OS chips:** pill, mono 11px, hairline border; "auto" support tinted amber, "parcial" with a short mono qualifier. No green.
- **Status indicators** (dashboards, CLI live view, tray; the landing has none): a leading dot or glyph in the state color plus the state word (Four States Rule).

### Cards / Containers
- **Corner Style:** 6px.
- **Background:** `surface`, hover `surface-2` only when the card is interactive.
- **Shadow Strategy:** none (Flat-By-Default).
- **Border:** 1px `border`; provider groups tint it at 45% with their group color on hover.
- **Internal Padding:** 14–24px.

### Navigation
- **Header:** logo + letter-spaced "KONGTROL" + BETA badge; links written as CLI flags in mono (`--inicio`, `--guia`) whose dashes stay dim until hover; icon buttons for GitHub, theme and language. The download CTA hides while the hero CTA or the download section is on screen.
- **tmux bar:** fixed bottom bar, `[kongtrol]` session in amber, numbered windows mapped to sections with the active one in solid amber, right side shows kill-switch state and clock. Between 721 and 960px the right side keeps only the clock; under 720px the bar is hidden and a "Ir a" jump menu replaces it.

### Logo (name and sizes)
- **Name:** the product is just **Kongtrol** (ADR-0047). "VPN" lives only in the tagline ("Todas tus VPN bajo Kongtrol" / "All your VPNs under Kongtrol", a play on "control", ADR-0048), never in the name or the lockup.
- **Solid silhouette up to 32 px** (`assets/brand/source/logo-kong-solid.svg`, derived by `assets/brand/generate.go`): favicon (`favicon.svg`, transparent, black or white by `prefers-color-scheme`; `logo-kong-32.png` fallback), tray / menu bar, the panel's 32 px header mark and the 28 px tray flyout mark. At that size the concentric lines blur into grey.
- **Line-art original from 36 px up** (`logo-kong.svg`, lockup `logo.svg`): landing header 40 px (36 px under 960 px, next to the 37 px header buttons), footer lockup 96 px, README, the installer's side panel. Always give `<img>` its width and height (no layout shift).
- **App icon: the only logo with a background** (ADR-0052). A squircle plate in Ink (`text-light` `#1b1813`) with the solid mark in Bone Paper (`bg-light` `#f1ede4`), generated by `assets/brand/generate.go` (`source/app-icon.svg`, `assets/logo.ico`, `app-icon-*.png`): Windows binaries and installer, macOS `.icns`, Linux menus, the landing's touch icon. On macOS the plate keeps Apple's grid margin (824 of 1024 px); on Windows and Linux it fills 44 of 48 px. Up to 24 px (title bar, context menu, a 100 % taskbar) the `.ico` shows the plain solid mark in mid-tone `#857c6b`, which keeps 3:1 on light and dark taskbars. Every other logo stays background-free. The installer's side panel (`assets/brand/installer/`) is the lockup in Bone on Ink with no words; the tagline is installer text, so it follows the installer's language.

### Terminal Window (signature)
macOS-style title bar with three dots and a centered mono title (`ssh guest@kongtrol.tower`), body in `surface`, 12px corners, the only floating element. Content boots on a CSS timeline already present in the prerendered HTML (no layout shift); visitors can type `help`.

### Provider Monogram (signature)
44×44 tile, 11px corners, two letters in Space Grotesk, subtle gradient and a 5px corner dot in the group color. Our own marks only — never a third-party logo or brand color (ADR-0023).

## Do's and Don'ts

### Do:
- **Do** keep amber as the only accent and use it for the one primary action per section.
- **Do** use green only for "connected" or a passed check, caution orange for "no response", red for errors, steel for "connecting", always with a word or icon.
- **Do** write anything machine-like in IBM Plex Mono and anything for people in Inter.
- **Do** keep motion for demos that explain (routing, attach, boot) and animate only transform/opacity, on screen, respecting reduced motion. The only UI motion is state feedback the visitor triggers: disclosures open/close by height (`::details-content`), tab panels and copy confirmations fade in, never on load.
- **Do** use the shared motion tokens in the panel (`web/ui/src/theme/tokens.css`): arrivals (page change, dialog and sheet open, toast, a newly added row) rise `--rise` over `--dur-med` with `--ease-enter`; exits (dialog and sheet close, toast) fade over the shorter `--dur-exit` with `--ease-exit`; switches answer a press with a slightly larger thumb. Nothing animates on first load or on live data ticks.
- **Do** keep the terminal frame (window chrome, ASCII banner, CLI-flag nav, tmux bar) when adding sections.

### Don't:
- **Don't** use green for groups, categories, support levels or decoration.
- **Don't** use amber to signal a state; it collides with the brand and the primary button.
- **Don't** add glow, float or breathing effects to buttons and cards; expression belongs to the demos.
- **Don't** render text below 11px or targets below 24px.
- **Don't** use third-party VPN logos or brand colors.
- **Don't** add shadows to anything except the terminal window and the primary button hover.
