<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0069: The dashboard and the tray flyout use the landing gold on both themes; tray flyout usability

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

ADR-0066 made gold `#ffb020` the one landing accent on every tone and left the dashboard
(`web/ui`) on its own tokens. The dashboard's dark theme already used `#ffb020`, but its light
theme kept the deep amber `#904f00` with Bone text. The tray flyout (`/tray`, ADR-0036) loads the
same tokens, so its light "ABRIR PANEL" button showed the same brown. Owner decision, 2026-11-20:
the panel and the tray use the landing gold, the same brand everywhere (primary buttons gold with
dark text), before v0.7.0-rc.2.

## Decision Drivers

- One brand accent on landing, dashboard and tray; no brownish amber left anywhere.
- WCAG 2.2 AA: text ≥ 4.5:1 (1.4.3), UI parts and states ≥ 3:1 (1.4.11), focus visible.
- Reuse the landing's roles (ADR-0066) instead of a second scheme.

## Considered Options

1. Keep the dashboard light theme on `#904f00` (status quo).
2. Swap `#904f00` for `#ffb020` everywhere `--signal` is used.
3. The landing's three roles in `web/ui/src/theme/tokens.css`, plus an Ink edge for filled
   controls on Bone.

## Decision Outcome

Chosen option 3. Option 2 would put gold text, links and focus rings on Bone (1.57:1, research
note of ADR-0066), failing 1.4.3 and 1.4.11.

- `--signal`: accent text, lines and focus. Gold on Ink; Ink `#1b1813` on Bone (15.15:1).
- `--signal-fill`: solid fills (filled buttons and icon buttons, the tray primary button, switch
  on, checked check boxes and radios, date range ends, skip link): gold on both themes, with
  `--on-signal` Ink `#0a0c10` on it (10.70:1). Hover `--signal-hi` `#ffc456`.
- `--signal-edge`: the 1px edge of a filled control and the mark inside it come from Ink on Bone.
  A switch, check box or radio shows its state only by its fill, and gold on Bone is 1.57:1
  (below 1.4.11's 3:1), so on Bone the Ink edge and Ink mark (check, radio dot, switch thumb)
  carry the state. On Ink the edge is gold (no visible change).
- `--signal-mark`: decorative gold only (selected tab underline, link hover underline, the found
  dot on Perfiles); never the only carrier of meaning.
- Light theme: `--signal-soft` gold 20 % (Ink text 13.78:1), `--signal-line` gold, `--focus` a
  2px Ink ring. Text hovers no longer turn `--signal-hi` (gold text on Bone): they stay
  `--signal` and add the gold underline, like the landing.
- The dark theme looks the same as before (all roles are `#ffb020`); radios on Ink now fill gold
  with an Ink dot instead of a gold dot on Ink, to match check boxes.
- The native tray menu and the tray icon are drawn by the OS and the generated masks
  (ADR-0044/0051); they carry no accent and do not change.

### Tray flyout usability (same change, owner request 2026-11-20)

- Rows list connected profiles first, then by name, like the native menu (ADR-0037); the order
  is taken when the flyout is shown (and when profiles come or go), never on each update, so a row
  never jumps away under the pointer mid-toggle (`trayOrder` / `applyOrder`, `web/ui/src/tray/model.ts`).
- A kill switch that is blocking traffic shows a caution notice at the top with what to do,
  visible without scrolling past the list.
- With no VPN the one primary action is "Agregar VPN": a new closed-list message `open-add`
  (`cmd/kongtrol-tray/flyout.go` `flyoutOpenAdd` → `openDashboardPage("/?add=vpn")`, the
  ADR-0055 deep link; `web/ui/src/tray/bridge.ts`); "Abrir panel" steps back to outlined.
- A failed connect shows the panel's "what to do" line for the daemon's `last_error_key`
  (`ERROR_KEYS`), or a generic one; never the raw English error.
- The title bar's contour rings are faint gold (`--signal-mark`), the landing's signature.

### Consequences

- Good: one gold across landing, dashboard and tray flyout; light-theme contrast now meets AA for
  text, states and focus.
- Neutral: gold fills on Bone keep a low edge contrast (1.57:1) for buttons, as on the landing;
  the Ink label identifies the button (1.4.11 asks 3:1 for what identifies the component).

## More Information

- Code: `web/ui/src/theme/{tokens.css,global.css,kit.css,kit-controls.css,tray.css}`,
  `web/ui/src/components/{home/home.css,profiles/profiles.css,settings/settings.css}`,
  `web/ui/src/pages/TrayPage.tsx`, `web/ui/src/tray/{model.ts,bridge.ts}`, `cmd/kongtrol-tray/flyout.go`.
- Contrast figures: [2026-11-19-landing-gold-contrast](../research/2026-11-19-landing-gold-contrast.md)
  (same colours). WCAG 2.2 1.4.11: https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html
- Docs: `DESIGN.md` (tokens, Colors, One Accent Rule, Components).
- Refines: [ADR-0066](0066-landing-gold-accent-footer-and-fixed-hero-terminal.md) (its "the dashboard keeps its own tokens").
