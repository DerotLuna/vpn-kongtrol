<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0044: Tray glyph relleno — the solid mark derived from the logo, with a smaller status dot

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

Refines point 3 of [ADR-0037](0037-native-tray-menu-states-and-installer-upgrade.md) (which glyph
the tray icon is rendered from, and the badge size). The rest of ADR-0037 stands, so it is not
superseded.

## Context and Problem Statement

ADR-0037 renders the tray / menu bar icon from the vector logo `source/logo-kong.svg`. That mark
is drawn in concentric lines: at 16–24 px (the Windows notification area at 100–150 %, the macOS
menu bar) the lines blur into a grey disc next to the system's solid glyphs (owner feedback on
Windows 11). A first round of three redrawn glyphs was rejected because they did not look like
the logo. Research: [2026-11-18-native-tray-menu](../research/2026-11-18-native-tray-menu.md)
(section "A simplified tray glyph").

## Decision Drivers

- The tray glyph must read as the Kongtrol logo, not as a new drawing.
- Small sizes are rendered per size, monochrome, on a transparent background, tinted by the
  system or by the taskbar theme (Microsoft notification-area and icon-construction guidance,
  Apple HIG template images; cited in the research note, **no verificado** in this session).
- The state badge must never cover the face, at any size.
- One source of truth: brand files live in `assets/brand/` and are generated, never hand-edited.

## Considered Options

1. Keep the line-art mark at every size (ADR-0037 as is).
2. Redraw a simplified glyph by hand (silhouette, ring or face; rejected by the owner).
3. **Derive** a solid version of the same head from the line-art master and use it for the tray.

## Decision Outcome

Chosen: **option 3**.

1. **Glyph.** `assets/brand/source/logo-kong-solid.svg` is derived from `source/logo-kong.svg`
   by `assets/brand/generate.go` (`go run -tags brandgen assets/brand/generate.go`): closing the
   gaps between the lines, filling from outside, keeping the face plate (eyes, nose, mouth) as
   negative space, then tracing back to Bézier outlines. It is regenerated, never edited.
2. **Masks.** `assets/brand/tray-<px>.png` (16, 20, 24, 32, 48, 64) are rasterized from the solid
   SVG, cropped to the mark like before, black + alpha, with an edge-contrast hint at ≤32 px.
   The full line-art logo stays for large surfaces (installer, app icons, about, dashboard,
   README); packaging icons (`assets/logo.ico`, `logo-kong-512.png` for the macOS `.icns`, the
   Linux 512 px PNG) are app icons, not the tray glyph, and do not change.
3. **Badge.** The state dot is smaller: radius 17 % of the icon (at least 2.5 px, was 20 %), in
   the bottom-right corner, with a transparent gap ring of size/16 (at least 1 px).
   `assets.BadgeGeometry` is the single definition; `TrayMark`/`TrayPNG`/`TrayICO` use it, so the
   Windows `.ico`, the macOS template image and the Linux PNG all get it.

### Consequences

- Good: the tray icon reads as the logo at 16 px on dark and light taskbars.
- Good: the dot never paints over the face at any size (`TestBadgeOffFace`).
- Bad: the dot's gap ring thins the face plate's 1 px rim at the chin's lower right, so on the
  smallest sizes the plate touches the gap there. A 15 % radius keeps the rim closed at every
  size (checked with the same test); 17 % was kept because it is the size the owner approved.

### Confirmation

- `assets/state_icons_test.go`: `TestTrayMarkTransparentAndTinted`, `TestTrayBadges`,
  `TestBadgeGeometryInsideIcon`, `TestBadgeOffFace` (fails with the old 20 % radius),
  `TestTrayICO`.
- Visual check (2026-11-18): 16/20/24/32 px masks rendered with each badge on a dark and a
  light background.

## More Information

- Research: [Native tray menu](../research/2026-11-18-native-tray-menu.md)
- Code: `assets/brand/generate.go` (`deriveSolid`, `renderTrayMasks`), `assets/icon.go`,
  `assets/state_icons.go`
