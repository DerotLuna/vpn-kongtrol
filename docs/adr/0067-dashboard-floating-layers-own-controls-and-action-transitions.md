<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0067: Dashboard: one positioning layer (Floating UI + Popover API), own Select / Combobox / DateRangePicker / Menu, and actions that wait for the live state

Version 1.0.0 · 2026-11-19

Status: Accepted; refined by [ADR-0068](0068-dashboard-kit-on-react-aria-components.md) (positioning and kit controls) · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

Number: 0062 and 0063 are taken by work in other branches (0063 seen in the repository's refs);
0067 (renumbered at merge: 0064–0066 were taken by other branches).

## Context and Problem Statement

Owner review of the dashboard (`web/ui`, screenshots of 2026-11-19) found:

1. The Inicio "Tus VPN" actions had no consistent hierarchy, and the row's "＋" dropped to a
   second line at medium widths.
2. The routes panel under a tunnel read right-aligned.
3. The tooltip was absolutely positioned below its trigger with a fixed offset: at the right
   edge of the window it was cut off, inside a table it could be clipped.
4. Connect showed a spinner only while the HTTP request was pending. The daemon answers a
   connect with 202 at once and works in the background (`internal/api/handlers_tunnels.go`
   `handleConnect`), so the row went back to "Conectar" while the tunnel was still coming up.
5. Native `<select>`, `<datalist>` and two `<input type="date">` (Actividad → Registro) look
   different on every system and ignore the panel's theme.
6. The "＋ Agregar" sheet was large and its buttons did not follow primary / secondary.
7. Several pages overflowed or wrapped badly at 768–1280 px.

ADR-0043 had kept the native `<select>` on purpose (keyboard and screen readers for free; the
classic panel's custom select leaked listeners). The owner now asks for own, branded controls,
which needs a reliable way to float a list, a menu, a calendar and a tooltip next to a trigger.

## Decision Drivers

- Standards first: WAI-ARIA APG patterns for every control; a vetted library for positioning
  instead of our own collision math (AGENTS.md "Standards first, then improve").
- CSP of the page stays `style-src 'self'` (no `'unsafe-inline'`, AGENTS.md "Dashboard").
- Small, MIT/BSD, maintained dependencies only; nothing that fetches at run time.
- Layers must work above the modal "＋ Agregar" sheet (`<dialog>` + `showModal()`, ADR-0055).
- Non-technical users: an action in progress must say so until it really ended, and say what
  to do if it takes too long (AGENTS.md "User guidance").

## Considered Options

Positioning:
1. Hand-rolled positioning (the old tooltip): no flip/shift, no scroll/resize tracking.
2. CSS Anchor Positioning (`anchor-name`, `position-try`): not yet in every browser the
   dashboard opens in (no verified support in Firefox at the time of writing; no verificado).
3. **Floating UI** (`@floating-ui/react-dom` 2.1.9, MIT): flip, shift, size, autoUpdate;
   ~10 kB gzip with its `dom`/`core`/`utils` packages.
4. react-aria / Radix: complete widgets, but far larger, and Radix portals into `<body>`, under
   a modal `<dialog>`.

Layering: portals to `<body>` (under a modal dialog's top layer) vs. **the Popover API**
(`popover="manual"` + `showPopover()`), which puts the layer in the browser's top layer above
an already open modal dialog, with no z-index or portal.

Controls: keep native vs. **own kit controls following the APG patterns**.

Connect feedback: spinner while the request runs (old) vs. **pending until the live state
reaches an end state**, with a timeout.

## Decision Outcome

1. **Positioning:** `@floating-ui/react-dom` 2.1.9 (exact pin), one hook `useAnchored`
   (`web/ui/src/components/ui/floating.ts`) with offset + flip + shift + size inside an 8 px
   viewport margin and `autoUpdate` while open. Lists cap their height to the room left and
   scroll inside; the calendar (it grows while open) is flipped and shifted on both axes.
2. **Layer:** every floating element is mounted only while open and shown with
   `popover="manual"` + `showPopover()` (top layer). Dismissal (Esc, outside press) is ours, so
   it behaves the same in every browser and in jsdom; Esc on a layer inside the sheet does not
   close the sheet.
3. **CSP:** Floating UI's coordinates reach the element through React's `style` prop and our
   CSS variables through `element.style.setProperty`; both are CSSOM writes, which
   `style-src 'self'` does not block. A `style` attribute in markup or set with
   `setAttribute('style', …)` is blocked. Verified in Chromium under the daemon's exact header
   (research note).
4. **Own controls in the kit** (`web/ui/src/components/ui`), shown at `/_kit`:
   `Tooltip` (hover delay 350 ms, instant on keyboard focus, hoverable, Esc, long press on touch
   that does not also press, `aria-describedby` to a node that is always present, visual bubble
   `aria-hidden`); `Menu` (APG menu button: "⋯" overflow actions); `Select` (APG select-only
   combobox: keyboard, type-ahead, groups like `<optgroup>`); `Combobox` (APG editable combobox
   with list autocomplete; optional free text); `DateRangePicker` (one "Desde–Hasta" control,
   presets Hoy / Últimos 7 días / Últimos 30 días / Personalizado, APG date-grid keyboard, week
   starts Monday in Spanish and Sunday in English, Intl names, no future days); check boxes and
   radios get the kit's box in CSS. Every `<select>`, `<datalist>` and date input of the panel
   uses them. Native `<input type="time">` stays (typing a time is faster than a picker); only
   its picker glyph is themed.
5. **Actions wait for the live state** (`web/ui/src/data/transition.ts`): after the daemon
   accepts a connect / cancel / disconnect / reconnect, the tunnel stays pending until the
   tunnels cache (WebSocket frames or a re-read) shows its end state — connected; an error or
   `last_error`; a provider app waiting for the user (`attach.phase` `waiting_user`, whose line
   tells the user what to do); disconnected (after it was seen moving); a reconnect with a new
   `ConnectedAt` — or until a timeout (connect 90 s, disconnect 45 s), which ends with a toast
   and the "what to do" line. A connect in progress offers Cancelar. Group actions mark each
   member they touch; the tray flyout uses the same end-state rule.
6. **Layout rules:** one primary (amber) action per row or step; Desconectar / Cancelar are
   outlined, never amber; secondary actions are 36 px icon buttons with the kit Tooltip; when a
   list is narrow they fold into "⋯". Tables switch to cards by the width of their own container
   (CSS container queries), not the window: the kit DataList at 680 px, Inicio at 980 px (with
   "⋯" between 980 and 1079 px), Actividad at 900 px.

### Consequences

- Good: no tooltip, list or calendar leaves the screen or hides under the sheet; one
  positioning code path; branded controls with the same keyboard contract as the platform's.
- Good: "Conectando…" now means the tunnel is still coming up; a slow connect says what to do.
- Bad: one more runtime dependency (~10 kB gzip). Mitigation: exact pin, MIT, used only through
  `floating.ts`.
- Bad: own controls must keep APG behaviour by tests (`kit.test.tsx`, page tests) instead of
  getting it from the browser.
- ADR-0043's choice of the native `<select>` (dashboard-parity.md "Themed select") is replaced
  by this ADR for the dashboard; ADR-0043 is otherwise unchanged and keeps its status.

## More Information

- Research note: [2026-11-19-floating-layers-and-own-controls](../research/2026-11-19-floating-layers-and-own-controls.md).
- Related: ADR-0036 (tray flyout), ADR-0038 (dashboard architecture), ADR-0043, ADR-0055.
