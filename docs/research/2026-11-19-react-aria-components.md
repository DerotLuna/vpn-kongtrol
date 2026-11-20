<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Headless component libraries for the dashboard kit (React Aria vs. Radix vs. Ariakit vs. Headless UI), CSP and bundle

Version 1.0.0 · 2026-11-19

Investigation behind [ADR-0068](../adr/0068-dashboard-kit-on-react-aria-components.md).
Sources accessed 2026-11-19. "Verified in code" = read in the installed package
(`web/ui/node_modules`, versions pinned in `web/ui/package.json`).

## What the panel needs

Select (single, groups), a searchable multi-select with chips, a date range picker with presets
and a live range while choosing, tooltips, all working inside a modal `<dialog>`, under
`style-src 'self'` with no nonce, in Spanish and English.

## Comparison

| Library | License | Select | Searchable multi-select | Date range / calendar | Tooltip | Source |
|---|---|---|---|---|---|---|
| **React Aria Components** 1.22.1 (Adobe) | Apache-2.0 (verified in code: `LICENSE`, `package.json`) | Select + ListBox | Autocomplete + SearchField + ListBox `selectionMode="multiple"`; TagGroup for chips | DateRangePicker, RangeCalendar, `@internationalized/date` (many calendar systems) | TooltipTrigger + Tooltip | https://react-aria.adobe.com/ , https://react-aria.adobe.com/quality |
| Radix Primitives (WorkOS) | MIT (not stated on the pages read; from the repository, no verificado here) | Select (single value; typeahead, no search) | none documented | none | Tooltip | https://www.radix-ui.com/primitives/docs/components/select , https://www.radix-ui.com/primitives/docs/overview/introduction |
| Ariakit 0.4.43 | MIT (no verificado here) | Select | Combobox (autocomplete; multiple selection not documented on the component index) | none | Tooltip | https://ariakit.com/components |
| Headless UI (Tailwind Labs) | MIT (no verificado here) | Listbox, Select | Combobox with `multiple` | none | none | https://headlessui.com/react/combobox |

Only React Aria covers every control in one library (no second date library, no own tooltip).
React Aria documents localized strings for 30+ languages and `I18nProvider` to set the locale
(https://react-aria.adobe.com/quality).

## Facts verified in code (react-aria 3.53.1, react-aria-components 1.22.1)

- **Positioning:** overlays get `top/left/maxHeight` through React's `style` prop and
  `--trigger-width` on the popover (CSSOM writes). CSP3 does not apply `style-src` to CSSOM
  writes, only to `<style>` elements and `style` attributes (same finding and test as the
  2026-11-19 floating-layers note).
- **Injected `<style>` elements:** `interactions/usePress.mjs` creates
  `<style id="react-aria-pressable-style">` (`touch-action: pan-x pan-y pinch-zoom` on
  `[data-react-aria-pressable]`) unless an element with that id exists, with a nonce from
  `<meta property="csp-nonce">` if present (`utils/getNonce.mjs`); skipped when
  `NODE_ENV === 'test'`. `overlays/usePreventScroll.mjs` injects one only on iOS WebKit.
  Under our CSP (no nonce) the first would be blocked → the rule is in our stylesheet and a
  `<meta id="react-aria-pressable-style">` is added before render (`lib/ariaStyleGuard.ts`).
- **Portals:** overlays render into `document.body` unless `UNSAFE_PortalProvider`
  (`react-aria/PortalProvider`) gives another container; inside a modal `<dialog>` the body is
  inert, so the kit Dialog provides itself as the container.
- **Calendar:** RAC's Calendar uses `props.createCalendar || createCalendar` from
  `@internationalized/date` (`private/Calendar.mjs`), so every calendar system is bundled.
- **ListBox Esc:** selectable collections clear the selection on Esc by default
  (`escapeKeyBehavior`); the multi-select sets `"none"` so Esc only closes.
- **Tooltip hover:** opens on hover only when the interaction modality is `pointer`
  (`tooltip/useTooltipTrigger.mjs`), set by any pointer move; keyboard focus opens it at once.
- **Locales:** `@react-aria/optimize-locales-plugin` 2.1.0 (Apache-2.0, `LICENSE` in the package)
  replaces React Aria's locale string modules outside the given locales at build time
  (`LocalesPlugin.js`). Official docs page for it not re-read today (the old URL redirects to
  https://react-aria.adobe.com/quality, which does not mention it): no verificado online.

## Measurements (this repository, `pnpm build`, gzip -9)

| | Before (ADR-0067) | After (ADR-0068) |
|---|---|---|
| All JS | 263.5 kB | 331.4 kB |
| First load `index-*.js` | 144.3 kB | 150.1 kB |
| `Field-*.js` (pages with forms) | 4.6 kB | 66.8 kB |
| All CSS | 23.5 kB | 23.8 kB |

Without the locales plugin the JS total was 349.2 kB.

## CSP check

The built panel served with the daemon's exact header
(`default-src 'self'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' data:;
connect-src 'self' ws://127.0.0.1:<port>; frame-ancestors 'none'`, `internal/api/server.go`),
mocked API, Chromium (Playwright), every page at 1440/1280/1024/768/390 plus open Select,
multi-select with chips, date range mid-selection, a Select inside the "＋ Agregar" sheet and the
collapsed menu with a tooltip; a `securitypolicyviolation` listener and the console were
watched: zero violations.
