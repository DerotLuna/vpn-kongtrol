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
  signal-light: "#1b1813"
  signal-fill-light: "#ffb020"
  landing-gold: "#ffb020"
  landing-gold-hi: "#ffc456"
  on-landing-gold: "#0a0c10"
  signal-hi-light: "#ffc456"
  on-signal-light: "#0a0c10"
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

Versión 1.6.0 · 2026-11-20

## Overview

**Creative North Star: "Sesión en la Torre"**

**Style: "Terminal de instrumentos" (ADR-0070).** Terminal/CLI UI (the page is a session) plus an instrument-panel layer (radar, routing patch panel), warm (Bone paper, one gold), with the topographic contour lines of the monkey mark as its structural signature. It is neither neubrutalism (no thick borders, no hard offset shadows) nor a "hacker" look (no green Matrix clichés outside the CRT easter egg). The landing is where this style is expressive; the dashboard and tray share its tokens and gold but stay plain product UI: an admin panel any non-technical person can run.

The site is not a page about Kongtrol; it is a guest session inside Kongtrol's control tower (window title `kongtrol — terminal`). Everything a visitor reads arrives as terminal output: a "Last login" line, a block-letter ASCII banner, `#` comments, commands with `$`, tunnel rows with `✔`, navigation written as CLI flags (`--inicio`, `--guia`) and a tmux status bar pinned to the bottom with numbered windows (`0:motd`, `1:empezar`…), the kill-switch state (`armed`) and a clock. Plain-language sentences (Inter) sit inside that session so non-technical visitors are never left decoding jargon; the terminal is the frame, not a test.

The landing has **one designed theme** (ADR-0060): bone paper for the page, Tower Night (Ink) bands that alternate with it section by section, and one short gold Signal band for the download call to action (ADR-0070); terminal windows are always Ink. The landing has one accent, gold `#ffb020`, on every tone (ADR-0066). A faint field of concentric contour rings echoes the monkey mark's line art. Density is calm: one terminal window per screen, each key section fits one laptop screen, no decorative imagery. Motion is reserved for things that explain: the boot timeline, the routing diagram, the attach demo, and the cross-fade of a page change. Components themselves stay precise and quiet.

**Key Characteristics:**
- Terminal session as the page frame (window chrome, ASCII banner, tmux bar, CLI-flag navigation).
- One brand accent, gold `#ffb020`, on the landing, the dashboard and the tray flyout; every other color is a state.
- Three type voices: Space Grotesk headings, Inter prose, IBM Plex Mono for everything machine-like.
- Precise components, expressive demos.
- Landing: one theme, sections in bands (Bone, Ink, one Signal band), no theme switch; plus a hidden green-phosphor CRT mode (↑↑↓↓←→←→BA). The dashboard (`web/ui`) keeps two complete themes that follow the OS until the user picks one.

## Colors

One warm accent against warm Bone neutrals and cool Ink neutrals; color carries meaning, never decoration. On the landing the `-light` neutrals are the Bone tone (`:root`) and the plain values the Ink tone (`.tone-ink`), set per section.

**Landing accent (ADR-0066, refines ADR-0060).** One gold, `landing-gold` `#ffb020` (hover `landing-gold-hi` `#ffc456`), on every tone; no deep amber anywhere on the landing. The accent has three roles in `landing/src/index.css`: `--signal-fill` (solid fills: primary buttons, active tmux window and tabs; gold with Ink text `on-landing-gold`, 10.70:1), `--signal` (accent text, lines and focus rings: gold on Ink 10.70:1; on Bone it is Ink `#1b1813`, 15.15:1, because gold text on Bone is 1.57:1) and `--signal-mark` (decorative gold only: link underlines in running text, rings, dots, washes, the footer wordmark; never the only carrier of meaning). The one `.tone-signal` band (the download section) is gold: Ink text (10.70:1; 12.38:1 on the `#ffc456` cards), muted Ink 82 % (7.67 / 8.57), dim Ink 70 % (5.59 / 6.02), card borders Ink 18 %, outlines Ink 55 % (3.65 / 3.82), an inverted primary button (Ink fill, gold text, 10.70:1) and Ink-inset code blocks.

**Dashboard and tray accent (ADR-0069, refines ADR-0066).** `web/ui` (and so the tray flyout `/tray`) uses the same gold and the same roles on both themes in `web/ui/src/theme/tokens.css`: `--signal` (gold on Ink, Ink on Bone), `--signal-fill` (gold, Ink text `--on-signal`), `--signal-mark` (decorative gold) and `--signal-edge`, the 1px edge and the mark of a filled control (switch on, checked check box or radio): gold on Ink, Ink on Bone, because there the fill alone is the state and gold on Bone is 1.57:1. Light-theme focus is a 2px Ink ring; text hovers keep `--signal` and add a gold underline. The dark theme is unchanged.

### Primary
- **Signal Gold in the dashboard** (`signal`, `signal-fill`; light `signal-light` Ink, `signal-fill-light` gold): the single brand accent of the dashboard and tray flyout (ADR-0069). Gold fills with Ink text for primary actions; accent text, links and focus are gold on Ink and Ink on Bone. Hover brightens to `signal-hi`.
- **Landing Gold** (`landing-gold`): the single brand accent of the landing (ADR-0066). Primary buttons (gold fill, Ink text) on every surface, the active tmux window and tabs, the ASCII banner, the `$` prompt on Ink, the "se adjunta" provider group and "auto" OS chips (gold border and wash, Ink text on Bone). On Bone, accent text is Ink with a gold mark.

### Secondary
- **Instrument Steel** (`steel`; light `steel-light`): secondary information and the "conecta directo" provider group; also the "connecting" state. Never used for primary actions.

### Tertiary (states)
- **Live Green** (`live`; light `live-light`): a tunnel that is up and healthy, or a check that passed. Nothing else.
- **Caution Orange** (`caution`; light `caution-light`): a tunnel that is up but not answering ("Sin respuesta", WireGuard liveness, ADR-0030) or any degraded state.
- **Fault Red** (`danger`; light `danger-light`): error, dropped, blocked.

### Neutral
- **Tower Night** (`bg`, `surface`, `surface-2`, `surface-3`): page, terminal body, title bars and raised rows, stepped by tonal lightness.
- **Bone Paper** (`bg-light`, `surface-light`, `surface-2-light`): the landing's base tone and the dashboard's light theme, matched to the ink-on-paper line art of the logo.
- **Hairlines** (`border`, `border-2` and light pairs): 1px dividers and outlines; `border-2` for interactive outlines.
- **Ink** (`text`, `text-muted`, `text-dim` and light pairs): primary, secondary and tertiary text. `text-dim` keeps ≥ 4.5:1 on every surface.

### Named Rules
**The One Accent Rule.** The accent (gold, on the landing, the dashboard and the tray) is the only brand color. If something is the accent color, it is the brand or the action; it is never a state. Gold is a dose, not a background: on the landing only one short band is gold (ADR-0070).

**The Contour Edge Rule.** A tone band never ends on a ruler line by default: it ends on a contour line from the mark (ADR-0070), never the same shape twice in a row, with an occasional straight edge as a rest. Edges are decoration: no meaning, no motion.

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

Single column, max width 1160px, fluid gutter `clamp(16px, 3vw, 24px)`, section rhythm `clamp(48px, 2.5vw + 32px, 88px)`; sections alternate Bone and Ink bands with no hairline between them (the tone change divides; ADR-0060). Bands end on contour-line edges (ADR-0070, `.edge` in `landing/src/index.css`): a CSS mask cuts the band along one line of a topographic map (swell, dune, drift, ring; mirrored where it helps), a different shape at each boundary, depth `clamp(28px, 4.5vw, 72px)`, and some boundaries stay straight on purpose as rests. Static, no motion. On laptops (1366×768, 1440×900, 1920×1080) each key section fits one screen under the header and above the tmux bar: the hero terminal (two columns from 1024px; a fixed-height scrollback with the prompt pinned below it, so commands never grow the window, ADR-0066), the attach demo, the routing map (heading beside the board from 1200px) and the download section (heading beside the recommended file from 1080px); phones keep the natural flow. Spacing follows a 4px base scale (4 → 64). A fixed 66px header and a fixed 32px tmux bar frame the scroll; anchored sections offset for both. Main breakpoints: 1080, 960, 720, 640, 520 and 420px. Grids collapse 4 → 3 → 2 → 1; the routing patch panel stacks under 768px (destinations, a text-free panel strip, exits) so text never scales below the floor. Interactive targets are 44px (24px minimum for dense inline controls).

## Elevation & Depth

Depth is tonal, not shadowed: `bg` → `surface` → `surface-2` → `surface-3` and 1px hairlines separate layers. Shadows appear in exactly two places — the terminal window, which floats above the page, and the primary button's accent glow on hover. The ambient contour field and two soft radial washes (accent top-right, steel bottom-left) give the page atmosphere without lifting any element.

### Shadow Vocabulary
- **Terminal float** (`box-shadow: 0 0 0 1px rgba(255,176,32,0.05), 0 30px 70px rgba(0,0,0,0.45)`; light `0 18px 44px rgba(60,50,25,0.12)`): the hero terminal window only.
- **Signal glow** (`box-shadow: 0 10px 28px -6px var(--signal-glow)`): primary button hover only.

### Named Rules
**The Flat-By-Default Rule.** Surfaces are flat at rest. Only the terminal window floats; everything else gains depth through tone or responds to interaction.

## Shapes

Mostly squared and technical: 6px corners on buttons, cards and code blocks; 12px on the terminal window and large panels; 11px on the 44px provider monogram tiles; 3px on tmux windows; full pills only for OS chips and status chips. Borders are 1px hairlines; focus is a 2px outline in `--signal` (gold on Ink, Ink on Bone and on the gold band) offset 3px. Circles belong to the brand: contour rings, status dots, the macOS-style window dots, and the tiny contour-ring glyph (11px) that opens every section label.

## Components

### Buttons
- **Shape:** gently squared (6px).
- **Primary:** solid accent (gold on the landing on every tone, inverted to Ink with gold text on the gold band), ink text, IBM Plex Mono 600 uppercase, 12px × 22px, 44px tall, download icon on the left.
- **Hover / Focus:** the fill brightens (`landing-gold-hi`; `signal-hi` in the dashboard), lifts 1px, soft accent glow; press scales to 0.98; 2px focus ring in `--signal` (gold on Ink, Ink on Bone and on the gold band).
- **Secondary:** transparent with a `border-2` hairline and text color; on hover the border, text and a 12% accent wash turn to the accent (`--signal`). One primary per section.

### Panel controls (dashboard `web/ui`, ADR-0067, ADR-0068)
- **Behaviour from React Aria Components** (Adobe, Apache-2.0): Select, MultiSelect (search + ListBox + chips), DateRangePicker (RangeCalendar), Tooltip; the look below is ours.
- **Button ladder:** filled gold = the one primary per row, card or step (Conectar, Guardar, Aplicar); tonal = secondary emphasis (Conectar todas); outlined = neutral (Desconectar, Cancelar, Probar); text = lowest (Volver, Agregar otro, Borrar fechas; "Volver" in muted text at the footer's start). Desconectar and Cancelar are never gold. Small buttons are 36px; icon buttons beside them are 36px too (44px in phone cards).
- **Row actions (VPN and group rows):** icon-only 36px squares, all the same size, each with a tooltip and an accessible name that carries the row ("Conectar trabajo"); the same icons as "Conectar todas" (plug) / "Desconectar todas" (power), Reconectar = refresh, add rule = plus. Only the state's own action carries weight: Conectar filled gold when it is down, Desconectar with an ink outline on `surface-2` when it is up (never gold; danger red on hover, since it ends the connection); the rest are outlined. Fixed slots, so the main square never moves: [Cancelar while connecting | Reconectar while connected | empty] [main] [＋] (ADR-0078). Tabs slide one selection mark to the chosen tab; status pills blend colours, the connecting ring breathes and the connected dot settles in. A group partly up shows both, "Conectar las que faltan" (filled) and "Desconectar todas" (outlined), and its count says how many are up. An action in progress shows a spinner inside the same square until the live state ends it; the other squares stay in place, inert. A row never wraps its actions (44px squares in phone cards).
- **Tooltip:** Ink-on-light inverse bubble (`text` on `bg`), 12px, ≤ 280px, 6px from the trigger; flips and shifts to stay 8px inside the viewport; shows after 350ms on hover, at once on keyboard focus.
- **Floating layers** (listbox, multi-select, calendar): `surface` with a `border-2` hairline, 6px corners and the one floating shadow the panel allows besides dialog and toast (`0 12px 32px rgba(0,0,0,.28)`); they open and close at once (no animation); 40px rows; the focused row gets the hover background, the chosen one a check plus a subtle gold wash (`state-select`); never a side bar.
- **Select / MultiSelect / DateRangePicker:** the input's box (`bg`, `border-2`, 44px) with a chevron or calendar glyph; open = accent border (`--signal`). The multi-select lists what is chosen as chips under the field (× on each, "Quitar todos"); nothing chosen reads "Todos los …". The date range is one control with presets (Hoy, Últimos 7 días, Últimos 30 días, Personalizado) and a month grid (Monday first in Spanish); range ends solid gold (Ink text), days between `signal-soft`, painted live up to the pointer before the second click.
- **Check boxes, radios and switches:** the kit's 18px box; when on, gold fill with an edge and mark in `--signal-edge` (Ink on Bone, ADR-0069); 2px `--signal` focus ring.
- **Tables:** become one card per row by the width of their own container (DataList 680px; Inicio 980px; Actividad 900px). Chips in a detail panel read left to right and are sized to their text.
- **Inicio layout:** "Tus VPN" full width, then two independent columns (Protección + Avisos, Grupos + Tu salida a internet) so a short card never leaves a hole; one column when the page is under 760px. DOM order = reading order.
- **Header and side menu:** the header holds logo, status, "＋ Agregar", Guía and version only (language and theme live in Ajustes). The side menu collapses to a 64px icon rail ("Contraer menú" at its foot, tooltips on the icons), remembered on this device and collapsed by default under 1100px.

### Chips
- **OS chips:** pill, mono 11px, hairline border; "auto" support with a gold border and wash (landing), "parcial" with a short mono qualifier. No green.
- **Status indicators** (dashboards, CLI live view, tray; the landing has none): a leading dot or glyph in the state color plus the state word (Four States Rule).

### Cards / Containers
- **Corner Style:** 6px.
- **Background:** `surface`, hover `surface-2` only when the card is interactive.
- **Shadow Strategy:** none (Flat-By-Default).
- **Border:** 1px `border`; provider groups tint it at 45% with their group color on hover.
- **Internal Padding:** 14–24px.

### Navigation
- **Header:** logo + letter-spaced "KONGTROL" + BETA badge; links written as CLI flags in mono (`--inicio`, `--guia`) whose dashes stay dim until hover; icon buttons for GitHub and language (the landing has no theme switch). Always Ink, over Bone and Ink sections alike. The language switch keeps the section being read (`#fragment`) and cross-fades the page change (cross-document View Transitions, header and tmux bar held in place; reduced motion: none). The download CTA hides while the hero CTA or the download section is on screen; the whole header slides away while the footer is on screen and returns on leaving it.
- **tmux bar:** fixed bottom bar, always Ink, `[kongtrol]` session in gold, numbered windows mapped to sections with the active one in solid gold (Ink text), right side shows kill-switch state and clock. Between 721 and 960px the right side keeps only the clock; under 720px the bar is hidden and a "Ir a" jump menu replaces it. The bar lists only the sections of the page it is on; other pages are reached from the header (`--guia`).
- **Footer:** exactly one screen above the tmux bar on laptops, and one screen on phones from 760 px of height (mark beside the claim, links in rows, no chips; ADR-0079) (sizes clamped by `svh`, nothing cut): the large line-art mark (≥ 128 px) with the tagline, one primary "Descargar para <sistema>" (the same component as the hero and the download section) with its note and "Otras formas de instalar" link, a "Guía de instalación" link, link columns, the `kongtrol version` lines, the facts, and then the facts row and the `no place like 127.0.0.1` wink, and the giant faint gold wordmark alone at the bottom (absolute, no extra height; the content keeps above it, ADR-0066). GitHub appears once, as the `$ git clone` line (no column link).
- **Back to top:** a round 48px Ink icon button (up arrow), fixed bottom-right above the tmux bar on every page, shown after about one screen of scrolling; accessible name and tooltip "Volver al inicio" / "Back to top", 2px focus ring; it scrolls to the top (instantly under reduced motion) and focuses `<main>`.
- **Download section:** a short Signal band, gold with Ink text (ADR-0066), holding only the call to action (ADR-0070): heading, then not a boxed card but a terminal comment `# Tu equipo: <sistema>`, the filled button (Ink, gold text) and the file line, set off by an Ink hairline. The band ends on the `ring` contour edge, echoed by two thin gold contour lines that draw in on scroll (static with reduced motion). The featured file is the app on every system (ADR-0077): the Windows installer, the universal macOS app, and on Linux the one-line install with a link to the .deb/.rpm packages. Then, on Bone, a "¿Qué descargo?" block (three short definitions: the app, command only, the icon needs the command) and one "Otras formas de instalar" row (bare binaries labelled "Solo el comando"): every other file exactly once, grouped by system, and the install script with a tab and a copy button per shell.

### Logo (name and sizes)
- **Name:** the product is just **Kongtrol** (ADR-0047). "VPN" lives only in the tagline ("Todas tus VPN bajo Kongtrol" / "All your VPNs under Kongtrol", a play on "control", ADR-0048), never in the name or the lockup.
- **Solid silhouette up to ~64 px** (`assets/brand/source/logo-kong-solid.svg`, derived by `assets/brand/generate.go`; ADR-0051 refines the cutoff of ADR-0047): favicon (`favicon.svg`, transparent, black or white by `prefers-color-scheme`; `logo-kong-32.png` fallback), tray / menu bar, the panel's 32 px header mark, the 28 px tray flyout mark, and the landing attach demo's tray mock (28 px, with the real badge geometry: r = 15 %, gap size/16; hollow grey → live green). At these sizes the concentric lines blur into grey.
- **Line-art original from ~128 px up** (`logo-kong.svg`, lockup `logo.svg`): landing footer mark ≥ 128 px, README, the installer's side panel. ADR-0060 adds, by owner decision, the landing header (54 px, 44 px under 960 px), the attach demo hub (108 px) and the routing patch panel's badge (40 SVG units; 28 in the phone strip). Other sizes between ~64 and ~128 px are avoided. On Ink bands every mark inverts to Bone (`.logo-mark`); on Bone and on the gold band it stays black. Always give `<img>` its width and height (no layout shift).
- **App icon: the only logo with a background** (ADR-0052). A squircle plate in Ink (`text-light` `#1b1813`) with the solid mark in Bone Paper (`bg-light` `#f1ede4`), generated by `assets/brand/generate.go` (`source/app-icon.svg`, `assets/logo.ico`, `app-icon-*.png`): Windows binaries and installer, macOS `.icns`, Linux menus, the landing's touch icon. On macOS the plate keeps Apple's grid margin (824 of 1024 px); on Windows and Linux it fills 44 of 48 px. Up to 24 px (title bar, context menu, a 100 % taskbar) the `.ico` shows the plain solid mark in mid-tone `#857c6b`, which keeps 3:1 on light and dark taskbars. Every other logo stays background-free. The installer's side panel (`assets/brand/installer/`) is the lockup in Bone on Ink with no words; the tagline is installer text, so it follows the installer's language.

### Terminal Window (signature)
macOS-style title bar with three dots and a centered mono title (`kongtrol — terminal`), always Ink (also on Bone sections), body in `surface`, 14px mono text at full opacity (every line ≥ 4.5:1), a blinking idle caret at the prompt, 12px corners, the only floating element. Content boots on a CSS timeline already present in the prerendered HTML (no layout shift); visitors can type `help`.

### Provider Monogram (signature)
44×44 tile, 11px corners, two letters in Space Grotesk, subtle gradient and a 5px corner dot in the group color. Our own marks only — never a third-party logo or brand color (ADR-0023).

## Do's and Don'ts

### Do:
- **Do** keep the accent (gold on the landing) as the only brand color and use it for the one primary action per section.
- **Do** use green only for "connected" or a passed check, caution orange for "no response", red for errors, steel for "connecting", always with a word or icon.
- **Do** write anything machine-like in IBM Plex Mono and anything for people in Inter.
- **Do** keep motion for demos that explain (routing, attach, boot) and animate only transform/opacity, on screen, respecting reduced motion. The only UI motion is state feedback the visitor triggers: disclosures open/close by height (`::details-content`), tab panels and copy confirmations fade in, never on load.
- **Do** use the shared motion tokens in the panel (`web/ui/src/theme/tokens.css`): arrivals (page change, dialog and sheet open, toast, a newly added row) rise `--rise` over `--dur-med` with `--ease-enter`; exits (dialog and sheet close, toast) fade over the shorter `--dur-exit` with `--ease-exit`; switches answer a press with a slightly larger thumb. Nothing animates on first load or on live data ticks.
- **Do** keep the terminal frame (window chrome, ASCII banner, CLI-flag nav, tmux bar) when adding sections.

### Don't:
- **Don't** use green for groups, categories, support levels or decoration.
- **Don't** use the accent to signal a state; it collides with the brand and the primary button.
- **Don't** set gold text on Bone (1.57:1): use Ink text with a gold mark (ADR-0066).
- **Don't** add glow, float or breathing effects to buttons and cards; expression belongs to the demos.
- **Don't** render text below 11px or targets below 24px.
- **Don't** use third-party VPN logos or brand colors.
- **Don't** add shadows to anything except the terminal window and the primary button hover (in the panel also the dialog, the toast and floating layers: menu, listbox, calendar).
- **Don't** use native `<select>`, `<datalist>` or date inputs in the panel; use the kit controls (ADR-0067).
