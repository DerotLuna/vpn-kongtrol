<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Floating layers, own form controls and CSP in the dashboard

Version 1.0.0 · 2026-11-19

Investigation behind [ADR-0064](../adr/0064-dashboard-floating-layers-own-controls-and-action-transitions.md).
Sources accessed 2026-11-19.

## Positioning library

| Option | License | Size (gzip) | Maintenance | Notes |
|---|---|---|---|---|
| `@floating-ui/react-dom` 2.1.9 (+ `dom` 1.8.0, `core` 1.8.0, `utils` 0.2.12) | MIT | ≈ 10 kB upper bound (min+gzip of the four entry files measured locally: core 4.7 kB, dom 4.0 kB, utils/dom 1.8 kB, react-dom 3.0 kB); the dashboard's main chunk grew 10.7 kB gzip with this change, including our new components | releases 2026-01, 2026-03, 2026-07 on npm; repo github.com/floating-ui/floating-ui | middleware `offset`, `flip`, `shift`, `size`; `autoUpdate` follows scroll, resize and layout shifts |
| react-aria(-components) | Apache-2.0 | far larger (whole widget set + i18n date libraries) | Adobe | complete APG widgets; more than the dashboard needs |
| Radix primitives | MIT | per widget | WorkOS | portals into `<body>`: under a modal `<dialog>` in the top layer |
| CSS Anchor Positioning | — | 0 | spec in progress | not verified in every target browser at this date (no verificado) |

Sources: https://floating-ui.com/docs/react , https://floating-ui.com/docs/middleware ,
https://floating-ui.com/docs/autoUpdate , `pnpm view @floating-ui/react-dom` (version, license,
publish dates), https://react-spectrum.adobe.com/react-aria/ , https://www.radix-ui.com/primitives ,
https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_anchor_positioning .

## Top layer: the Popover API

`popover` + `showPopover()` puts an element in the top layer, after anything already there, so
a list opened inside a modal `<dialog>` paints above it without portals or z-index (seen in
the "＋ Agregar" sheet screenshots). Baseline since 2024 (Chrome 114, Safari 17, Firefox 125:
versions from the sources below, not re-tested here); WebView2 (tray flyout) is Chromium.
`popover="manual"` keeps light-dismiss and Esc in our hands, so behaviour is identical in every
browser and testable in jsdom (which has no `showPopover()`; tests show `[popover]` with a
stylesheet in `src/test/setup.ts`).

Sources: https://developer.mozilla.org/en-US/docs/Web/API/Popover_API ,
https://html.spec.whatwg.org/multipage/popover.html .

## CSP: style attributes vs. CSSOM

CSP Level 3 applies `style-src` to inline `<style>` and to `style` attributes (the "inline
style" checks); scripts changing `element.style` (CSSOM) are not inline styles and are not
checked. Sources: https://www.w3.org/TR/CSP3/#directive-style-src ,
https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/style-src
(summary: inline style attributes and `setAttribute('style', …)` need `'unsafe-inline'`; CSSOM
property assignment does not). The table below is what was actually measured.

Verified locally (Playwright, Chromium 159.0.8154.6) with the daemon's exact header
`default-src 'self'; script-src 'self'; style-src 'self'; …` (`internal/api/server.go`):

| How the style was written | Result |
|---|---|
| `style="left:13px"` in markup | blocked (CSP violation logged) |
| `el.setAttribute('style', 'left:34px')` | blocked (CSP violation logged) |
| `el.style.left = '21px'`, `el.style.setProperty('--x', '99px')` | applied, no violation |

The whole dashboard was then driven at 390–1600 px with the same header on every response
(menus, lists, calendar and tooltips open): zero CSP violations. React writes the `style` prop
through CSSOM in the browser, which is why Floating UI's `floatingStyles` works.

## WAI-ARIA patterns followed

- Tooltip: https://www.w3.org/WAI/ARIA/apg/patterns/tooltip/ and WCAG 2.2 SC 1.4.13 (hoverable,
  dismissible, persistent). Touch: Material 3 shows plain tooltips on long press
  (https://m3.material.io/components/tooltips/guidelines).
- Menu button: https://www.w3.org/WAI/ARIA/apg/patterns/menu-button/
- Select-only combobox: https://www.w3.org/WAI/ARIA/apg/patterns/combobox/examples/combobox-select-only/
- Editable combobox with list autocomplete: https://www.w3.org/WAI/ARIA/apg/patterns/combobox/examples/combobox-autocomplete-list/
- Date picker dialog (grid keys: arrows, Home/End, PageUp/PageDown, Shift+Page for years):
  https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/examples/datepicker-dialog/
- Week start: Spanish locales start on Monday (CLDR `firstDay` for ES and most of Latin America
  is Monday; US English Sunday). Source: https://unicode.org/reports/tr35/tr35-dates.html#Week_Data
  — a per-country week start from `Intl.Locale#getWeekInfo` is reasoned but not adopted (the
  dashboard knows the language, not the country).

## Connect is asynchronous in the daemon

`POST /api/v1/tunnels/{name}/connect` answers 202 `connecting` before the tunnel is up and
records the outcome later (`internal/api/handlers_tunnels.go` `handleConnect`,
`collector.RecordConnectError`); provider apps report `attach.phase` `waiting_user` while they
wait for the user. The end-state rule in `web/ui/src/data/transition.ts` follows those fields.
