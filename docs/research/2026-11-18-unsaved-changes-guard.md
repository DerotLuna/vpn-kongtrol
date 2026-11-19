<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Unsaved-changes guard and router mode for `web/ui`

Version 1.0.0 · 2026-11-18

Question: how do mature web apps stop a user from losing form edits when they leave the page, and
what does React Router 8 need for it? Accessed 2026-11-18.

## Findings

- **Leaving the tab (close, reload, typed URL).** The only standard mechanism is the
  `beforeunload` event: call `preventDefault()` (and set `returnValue` for older Chromium) and the
  browser shows its own prompt; custom text is ignored by every current browser.
  Sources: HTML Standard, "unloading documents" —
  https://html.spec.whatwg.org/multipage/browsing-the-web.html#unloading-documents ;
  MDN `beforeunload` — https://developer.mozilla.org/en-US/docs/Web/API/Window/beforeunload_event
- **Leaving inside a single-page app.** React Router's `useBlocker` blocks in-app navigations and
  returns a blocker with `state` (`unblocked` / `blocked` / `proceeding`), `proceed()` and `reset()`;
  it does not handle hard reloads or cross-origin navigations (that part stays `beforeunload`). It
  is "Available with Framework" and "Available with Data", **not with Declarative** mode.
  Source: https://reactrouter.com/api/hooks/useBlocker (v8.5.1 docs) and the shipped typings
  `node_modules/react-router/dist/development/index.d.ts`.
- **Back/Forward.** The blocker function receives `historyAction`, so POP navigations go through it;
  the docs do not state it in words. Checked in Chromium with the built panel (Back on a dirty
  form shows the prompt and the URL stays) — see the screenshots run of 2026-11-18.
- **Router modes.** Declarative (`<BrowserRouter>`) vs data (`createBrowserRouter` +
  `<RouterProvider>`): the route tree can be kept as JSX with `createRoutesFromElements`, so the
  change is local to `App.tsx`. Data mode also offers route `lazy` (code splitting per route; the
  current page stays until the next one is loaded). Source: https://reactrouter.com/start/modes
- **Prompt wording and actions.** Platform guidance: the safe action ("keep editing") is the
  default and first; the destructive one ("discard") is clearly marked. Apple HIG Alerts —
  https://developer.apple.com/design/human-interface-guidelines/alerts ; Material 3 Dialogs —
  https://m3.material.io/components/dialogs/guidelines

## Used by

ADR-0041.
