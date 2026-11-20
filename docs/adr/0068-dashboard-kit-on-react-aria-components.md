<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0068: Dashboard kit primitives on React Aria Components; icon-square row actions, packed Inicio, collapsible menu

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

Refines [ADR-0067](0067-dashboard-floating-layers-own-controls-and-action-transitions.md): its
positioning layer (Floating UI + Popover API, `floating.ts`) and its own Select / Combobox /
DateRangePicker / Menu / Tooltip are replaced; its other decisions (actions that wait for the live
state, tables to cards by container width, one primary per row) stay.

## Context and Problem Statement

Owner review of v0.7.0-rc.1 (2026-11-19):

1. The kit's controls were hand-rolled (APG patterns, own keyboard and focus code). The owner
   asks to follow a big, mature component library instead.
2. The Select's open/close animation and the amber bar on the active option looked heavy.
3. Actividad → Registro "Perfil" mixed free text with a list; it should be a searchable
   multi-select of profiles (any of the chosen ones matches).
4. "Tus VPN" mixed text buttons ("Conectar") with icon buttons; groups showed a text button plus
   an unexplained icon.
5. Inicio cards in a fixed two-column grid of unequal heights left a hole under "Avisos".
6. The header's language and theme toggles duplicated Ajustes; the side menu could not be
   collapsed.

## Decision Drivers

- Standards first: a vetted, widely used library for accessibility and i18n, not our own code
  (AGENTS.md).
- The page's CSP stays `style-src 'self'` with no `'unsafe-inline'` and no nonce.
- Layers must work inside the modal "＋ Agregar" sheet (`<dialog>` + `showModal()`).
- Spanish and English only; nothing fetched at run time; bundle cost measured.
- CLI parity for any new filter (AGENTS.md "Product model").

## Considered Options

React Aria Components, Radix Primitives, Ariakit, Headless UI (comparison and sources in the
research note). Only React Aria has a date range picker / range calendar with i18n calendars,
a searchable multi-select pattern (Autocomplete + ListBox), and Tooltip in one library.

## Decision Outcome

1. **Kit on React Aria Components** (`react-aria-components` 1.22.1 and `react-aria` 3.53.1 for
   `UNSAFE_PortalProvider`, `@internationalized/date` 3.12.4; Apache-2.0; exact pins):
   `Select` (Select + ListBox, groups as sections), `MultiSelect` (DialogTrigger + Popover +
   Autocomplete/SearchField + ListBox `selectionMode="multiple"`, `escapeKeyBehavior="none"` so
   Esc never clears the choice, chosen values as a TagGroup of chips + "Quitar todos"),
   `DateRangePicker` (our trigger + presets, RangeCalendar inside a Popover/Dialog: the range is
   painted live up to the pointer or the focused day before the second pick), `Tooltip`
   (TooltipTrigger + Focusable; a tooltip that only repeats the accessible name is not linked
   again, a longer one describes the trigger through a hidden node that is always there).
   `Combobox` (single, free text) and `Menu` ("⋯") had no use left and are deleted;
   `floating.ts` and `@floating-ui/react-dom` are removed (nothing stale).
2. **Layering:** overlays portal to `<body>`; inside the kit `Dialog` they portal into the
   `<dialog>` (`UNSAFE_PortalProvider`), so they share its top layer and are not inert. The
   sheet's backdrop click now needs the press to start on the backdrop too (a press on a select
   ends on React Aria's underlay; the browser then targets the click at the `<dialog>`).
3. **CSP:** React Aria writes positions and sizes through `element.style` (CSSOM, allowed).
   `usePress` would inject one `<style>` (`touch-action` on pressable elements) unless an element
   with id `react-aria-pressable-style` exists; that `<style>` is blocked by the CSP. The rule
   lives in `kit-controls.css` and `lib/ariaStyleGuard.ts` adds a `<meta>` with that id before
   the first render (a test checks the id against the installed `react-aria`). The other
   injected `<style>` (`usePreventScroll`) runs only on iOS WebKit; the panel is a desktop page
   on 127.0.0.1 (not verified on iOS). Checked in Chromium under the daemon's exact header:
   zero violations (research note).
4. **i18n:** `I18nProvider` follows the panel language (`es-ES` / `en-US`: Monday first in
   Spanish); `@react-aria/optimize-locales-plugin` 2.1.0 (Adobe, Apache-2.0, build only) keeps
   only those two locales of React Aria's built-in strings.
5. **Look** (`theme/kit-controls.css`): lists open and close at once (no animation); focused
   option = hover background, chosen = check + subtle amber wash, no side bar.
6. **Audit filter by several profiles:** `GET /api/v1/audit?profile=a&profile=b` (any matches,
   case-insensitive); `security.AuditQuery.Profiles`; `kongtrol logs --profile a,b` (or
   repeated) for CLI parity.
7. **Row actions:** 36px icon squares only (plug = connect, power = disconnect, refresh =
   reconnect, plus = add rule, × = cancel), tooltip + aria-label with the row's name; weight only
   on the state's action (connect filled amber; disconnect ink outline, never amber); spinner
   inside the same square while pending. Groups (`GroupConnect`, Inicio and /profiles/groups):
   none up → connect; all up → disconnect; some up → "Conectar las que faltan (n de m)" +
   "Desconectar todas (n de m conectadas)", and the count says "n de m conectadas". The "⋯"
   fold is gone (the squares fit).
8. **Inicio:** "Tus VPN" full width, then two independent columns (Protección + Avisos, Grupos +
   Tu salida a internet) by the page's container width (≥ 760px); DOM order = reading order.
9. **Shell:** no language/theme toggles in the header (Ajustes › Idioma y apariencia keeps them).
   "Contraer menú" collapses the side menu (≥ 600px) to a 64px icon rail with tooltips;
   `aria-expanded` + `aria-controls`, the toggle keeps focus; remembered in `localStorage`
   (`kongtrol.nav`, read/write never throw) on windows ≥ 1100px, collapsed by default below
   (expanding there lasts the visit). The old 600–839px labelled rail is replaced by it.

### Consequences

- Good: keyboard, focus, screen reader and calendar behaviour come from a library used and
  tested by many products; less own code to keep correct.
- Good: one searchable multi-select for any "pick several" filter; the audit filter has CLI
  parity.
- Bad: bundle. JS total 263.5 → 331.4 kB gzip (+67.8 kB); the first load (`index`) 144.3 →
  150.1 kB (+5.8 kB); the chunk with the form controls (`Field`, loaded only by pages with
  forms) 4.6 → 66.8 kB, mostly React Aria collections and `@internationalized/date` calendars
  (RAC's calendar imports every calendar system; not tree-shakeable from RAC components).
  CSS 23.5 → 23.8 kB.
- Bad: two internals we depend on: the `react-aria-pressable-style` id (guarded by a test) and
  the `UNSAFE_` portal provider (stable for years, but named unsafe).
- Bad: React Aria shows no tooltip on touch (ADR-0067 had a long press); touch users read the
  accessible name instead.

## More Information

- Research note: [2026-11-19-react-aria-components](../research/2026-11-19-react-aria-components.md).
- Related: ADR-0036 (tray flyout uses the same page), ADR-0038, ADR-0043, ADR-0055, ADR-0067.
