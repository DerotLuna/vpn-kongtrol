<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0078: Panel motion, hover that says what the click does, and fixed action slots

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Owner feedback on the panel after v0.7.0-rc.3, 2026-11-20:

- Changing tabs (Actividad, Perfiles · Grupos) jumps the selected mark; it should slide.
- Hovering "Desconectar" should turn red; hovering "Conectar" barely shows in the light theme.
- While connecting, "Cancelar" appears to the right of the main square and pushes things around;
  the main square must never move (Cancelar and Reconectar get their own place). The change from
  connecting to connected should read as one fluid motion.
- Inicio's lead runs to two lines; Tráfico shows "Sin mediciones de latencia todavía." where a
  loader reads better.

## Decision Outcome

- **Sliding tab mark** (`components/ui/Tabs.tsx` `useInk`, `theme/kit.css`): one `.k-tabs__ink` per
  list, placed under the selected tab (`aria-selected` / `aria-current`) by CSSOM writes (allowed by
  the CSP) after layout and on resize; transform + width transition (`--dur-med`,
  `--ease-standard`), none on first paint (`data-ready` after the first placement) or with reduced
  motion. Lists without ink keep the per-tab `::after`.
- **Hover says what the click does** (`theme/kit-controls.css`, `kit.css`): the state's "ends the
  connection" square (`k-icon-btn--strong`, Desconectar) turns danger red on hover (border, icon,
  soft wash). Gold fills brighten and add the accent glow; filled icon squares also take the
  `--signal-edge` border (Ink on Bone), so the hover shows on light surfaces too.
- **Fixed action slots** (`components/home/TunnelRow.tsx`, `home.css`): every row is
  `[secondary] [main] [＋]`; secondary is Cancelar while connecting, Reconectar while connected, an
  empty 36px slot otherwise (44px in phone cards). The main square never moves.
- **Fluid state change** (`theme/global.css` `.pill`): pill colours blend (`--dur-med`), the
  connecting ring breathes, the connected dot settles in; off with reduced motion.
- **Copy**: Inicio lead in one line ("Tus VPN en vivo: qué está conectado y qué hacer si algo
  falla."); Tráfico's empty latency shows a spinner with "Midiendo la latencia…" for screen readers.

### Consequences

- Good: motion explains changes, hover predicts the result, the row layout is stable under the pointer.
- Neutral: the connected dot also settles in when a page first renders a connected tunnel (once, 250 ms).

## More Information

- Docs: `DESIGN.md` (Panel controls: row actions, hover).
- Refines: [ADR-0067](0067-dashboard-floating-layers-own-controls-and-action-transitions.md), [ADR-0068](0068-dashboard-kit-on-react-aria-components.md), [ADR-0071](0071-panel-navigation-polish-traffic-order-and-release-tag-pinning.md).
