<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: UI patterns for the dashboard (`web/ui`, served at `/app/`)

Version 1.0.0 · 2026-11-18

## Question

Which rules from platform design systems (Apple HIG, Material 3, Fluent 2), WCAG 2.2 AA and
usability research apply to the new dashboard, a live-data panel used by non-technical people and
devs (ES+EN, desktop first, phone width supported)? Which router, data-fetch, toast and dialog
building blocks fit a small Vite SPA embedded in the Go binary under `/app/`?

## Sources (accessed 2026-11-18)

All sources below were accessed on 2026-11-18. Apple pages were read from the JSON that renders
developer.apple.com (`/tutorials/data/design/human-interface-guidelines/<page>.json`).
m3.material.io is client-rendered and could not be read as text, so M3 values come from Google's
own token files (material-web), the Material Components for Android docs and the androidx source.
npm data comes from `https://registry.npmjs.org/<pkg>`; sizes from the bundlephobia API
(`/api/size?package=<pkg>@<ver>`, whole entry point minified + gzip, **before** tree-shaking).

| # | Source | What it says (short) |
|---|---|---|
| 1 | [HIG — Sidebars](https://developer.apple.com/design/human-interface-guidelines/sidebars) | "show no more than two levels of hierarchy in a sidebar"; let people hide it but "Avoid hiding the sidebar by default"; a tab bar is the compact alternative |
| 2 | [HIG — Tab bars](https://developer.apple.com/design/human-interface-guidelines/tab-bars) | "Use a tab bar to support navigation, not to provide actions"; keep it visible; avoid overflow ("More") tabs; don't disable or hide tab items |
| 3 | [HIG — Layout](https://developer.apple.com/design/human-interface-guidelines/layout) | order content by importance (top, leading side first); group related items; progressive disclosure |
| 4 | [HIG — Feedback](https://developer.apple.com/design/human-interface-guidelines/feedback) | put status "near the items it describes"; alerts only for critical, actionable info; confirm only significant actions; say why a command can't run |
| 5 | [HIG — Alerts](https://developer.apple.com/design/human-interface-guidelines/alerts) | "Use alerts sparingly"; no alert for common undoable actions; no alert at app start — show cached data plus a nonintrusive label |
| 6 | [HIG — Sheets](https://developer.apple.com/design/human-interface-guidelines/sheets) | for a simple, scoped task; "Display only one sheet at a time"; Cancel/Done buttons; complex flows deserve another surface |
| 7 | [HIG — Toggles](https://developer.apple.com/design/human-interface-guidelines/toggles) | toggles manage state between two opposing values; identify what they affect; "Avoid relying solely on different colors" |
| 8 | [HIG — Progress indicators](https://developer.apple.com/design/human-interface-guidelines/progress-indicators) | prefer determinate; "Keep progress indicators moving"; a stalled process needs feedback on what to do; don't switch circular ↔ bar |
| 9 | [Android — Window size classes](https://developer.android.com/develop/ui/compose/layouts/adaptive/window-size-classes) | compact `< 600dp`, medium `600–839dp`, expanded `840–1199dp`, large `1200–1599dp`, extra-large `≥ 1600dp` |
| 10 | [MDC Android — Navigation bar](https://github.com/material-components/material-components-android/blob/master/docs/components/BottomNavigation.md) / [Navigation rail](https://github.com/material-components/material-components-android/blob/master/docs/components/NavigationRail.md) | bar: "three to five destinations"; rail: "three to no more than seven" destinations, badges for counts |
| 11 | [material-web — md.sys.state](https://github.com/material-components/material-web/blob/main/tokens/versions/latest/sass/_md-sys-state.scss) | state-layer opacity: hover 0.08, focus 0.10, pressed 0.10, dragged 0.16; disabled 0.38 |
| 12 | [material-web — md.sys.motion](https://github.com/material-components/material-web/blob/main/tokens/versions/latest/sass/_md-sys-motion.scss) | `easing-standard` `cubic-bezier(0.2, 0, 0, 1)`, decelerate `(0, 0, 0, 1)`, accelerate `(0.3, 0, 1, 1)`; durations short1–4 = 50/100/150/200ms, medium1–4 = 250/300/350/400ms, long1–2 = 450/500ms |
| 13 | [MDC Android — Snackbar](https://github.com/material-components/material-components-android/blob/master/docs/components/Snackbar.md) + [androidx `SnackbarHost.kt`](https://github.com/androidx/androidx/blob/androidx-main/compose/material3/material3/src/commonMain/kotlin/androidx/compose/material3/SnackbarHost.kt) | snackbars "don't require user input to disappear"; Compose M3: Short 4000ms, Long 10000ms, Indefinite; one shown at a time (host queues) |
| 14 | [MDC Android — Dialog](https://github.com/material-components/material-components-android/blob/master/docs/components/Dialog.md) / [Progress indicator](https://github.com/material-components/material-components-android/blob/master/docs/components/ProgressIndicator.md) | "Dialogs are purposefully interruptive, so they should be used sparingly"; progress has determinate and indeterminate variants |
| 15 | [WinUI — NavigationView](https://learn.microsoft.com/en-us/windows/apps/design/controls/navigationview) | top nav when "5 or fewer top-level navigation categories that are equally important"; left pane otherwise; expanded left pane at "1008px or greater"; LeftCompact / LeftMinimal below |
| 16 | [WinUI — InfoBar](https://learn.microsoft.com/en-us/windows/apps/design/controls/infobar) | inline, "app-wide status messages … highly visible yet non-intrusive", severity levels, optional action, dismissible or persistent |
| 17 | [Fluent 2 — Message bar](https://fluent2.microsoft.design/components/web/react/core/messagebar/usage) | state of the product or surface; warnings for temporary states "like connectivity issues"; "Warnings must include a button, link, or both" |
| 18 | [Fluent 2 — Toast](https://fluent2.microsoft.design/components/web/react/core/toast/usage) | temporary, "useful and relevant, but not critical"; "For critical messages, try a modal"; confirmation toasts follow the user's action |
| 19 | [Fluent 2 — Switch](https://fluent2.microsoft.design/components/web/react/core/switch/usage) / [WinUI — Toggle switches](https://learn.microsoft.com/en-us/windows/apps/design/controls/toggles) | switch = "immediate change"; "Use a toggle switch for binary settings when changes become effective immediately"; label switches |
| 20 | [Fluent 2 — Dialog](https://fluent2.microsoft.design/components/web/react/core/dialog/usage) / [WinUI — Dialogs](https://learn.microsoft.com/en-us/windows/apps/design/controls/dialogs-and-flyouts/dialogs) | dialogs interrupt, use for important actions; "Don't use dialogs for errors that are contextual"; always one safe button; one or two "do it" buttons |
| 21 | [WCAG 2.2](https://www.w3.org/TR/WCAG22/) + Understanding pages ([2.4.11](https://www.w3.org/WAI/WCAG22/Understanding/focus-not-obscured-minimum), [2.5.8](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum), [4.1.3](https://www.w3.org/WAI/WCAG22/Understanding/status-messages), [2.2.2](https://www.w3.org/WAI/WCAG22/Understanding/pause-stop-hide), [1.4.13](https://www.w3.org/WAI/WCAG22/Understanding/content-on-hover-or-focus)) | criteria texts quoted in §4 |
| 22 | [NN/g — 10 Usability Heuristics](https://www.nngroup.com/articles/ten-usability-heuristics/) | the ten heuristics listed in §5 |
| 23 | [NN/g — Response Times: 3 Limits](https://www.nngroup.com/articles/response-times-3-important-limits/) | 0.1s feels instant; 1.0s keeps flow; 10s keeps attention; percent-done for long waits |
| 24 | [NN/g — Skeleton Screens 101](https://www.nngroup.com/articles/skeleton-screens/) | skeletons "used exclusively for full-page loads"; spinners best for 2–10s; under 10s both work |
| 25 | [RFC 5861](https://www.rfc-editor.org/rfc/rfc5861.html) | `stale-while-revalidate=<delta-seconds>`: serve stale while revalidating in the background |
| 26 | [TanStack Query — Important defaults](https://tanstack.com/query/latest/docs/framework/react/guides/important-defaults) / [Optimistic updates](https://tanstack.com/query/latest/docs/framework/react/guides/optimistic-updates) | data stale immediately; background refetch on mount, window focus, reconnect; inactive cache GC after 5 minutes; failed queries "silently retried 3 times, with exponential backoff"; `onMutate` snapshot + `onError` rollback |
| 27 | [SWR](https://swr.vercel.app) / [SWR API](https://swr.vercel.app/docs/api) | named after RFC 5861; one request per key shared by components; `dedupingInterval = 2000` |
| 28 | [TkDodo — WebSockets with React Query](https://tkdodo.eu/blog/using-web-sockets-with-react-query) | WebSocket events update or invalidate the query cache; `staleTime: Infinity` when the socket keeps data fresh |
| 29 | [React — useSyncExternalStore](https://react.dev/reference/react/useSyncExternalStore) | the hook "lets you subscribe to an external store" (subscribe + getSnapshot), tear-free in concurrent rendering |
| 30 | [AWS Architecture — Exponential Backoff And Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/) | capped exponential backoff alone still synchronizes clients; "Full Jitter" (`sleep = random(0, min(cap, base·2^n))`) and "Decorrelated Jitter" do less work |
| 31 | [MDN — `<dialog>`](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/dialog) / [ARIA status role](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Roles/status_role) | dialog Baseline "since March 2022"; `showModal()` renders in the top layer, makes the rest `inert`, Esc closes, `::backdrop`, `closedby`; `role="status"` is polite, never move focus to it |
| 32 | [React Router — Modes](https://reactrouter.com/start/modes) / [createBrowserRouter](https://reactrouter.com/api/data-routers/createBrowserRouter) | Declarative, Data and Framework modes; `basename` option |
| 33 | [TanStack Router — RouterOptions](https://tanstack.com/router/latest/docs/framework/react/api/router/RouterOptionsType) | `basepath` "for mounting a router instance at a subpath" |
| 34 | [wouter README](https://github.com/molefrog/wouter) | `<Router base={basepath}>`; nested routes |

**No verificado:** the m3.material.io pages themselves (skeleton/loading guidance, card and
dialog specs), Fluent's default toast timeout, WinUI's `CompactModeThresholdWidth` default value
(641px from memory) and Fluent 2 web Nav guidance (page not read).

## 1. Apple HIG

- **Navigation.** Sidebar for an app with several areas; ≤ 2 levels; visible by default,
  hideable; tab bar is the compact form [1][2]. Tabs navigate, never act; never hide or disable
  a destination [2].
  *Applies to Kongtrol panel as:* a left sidebar with flat destinations (Tunnels, Policies,
  Profiles, Security, Settings), always visible on desktop, collapsing to a bottom bar on phones.
- **Layout.** Most important first (top, leading side); group related; progressive disclosure [3].
  *Applies as:* overall state ("Protegido / 2 VPN conectadas") on top of every page; advanced
  fields behind "Más opciones".
- **Feedback.** Status next to the item; alerts only for critical, actionable info; no alert on
  start — show cached data with a small label [4][5].
  *Applies as:* per-tunnel status inside each tunnel card; "daemon unreachable" is an inline
  banner over the last known data, never a modal.
- **Sheets.** One at a time, simple task, Cancel/Done [6].
  *Applies as:* add/edit profile in one dialog; multi-step setup is a page (the wizard), not stacked dialogs.
- **Toggles.** Two opposing states, say what they affect, not color alone [7].
  *Applies as:* kill switch / DNS guard switches with a text state ("Activado") next to the color.
- **Progress.** Prefer determinate, keep it moving, explain a stall [8].
  *Applies as:* connect shows phases ("Abriendo la app… Esperando que conectes…") from the
  daemon phase keys, and a stalled phase shows the "what to do" line.

## 2. Material 3

- **Navigation rail vs bar.** Breakpoints: compact `< 600dp`, medium `600–839dp`, expanded
  `≥ 840dp` (then large `≥ 1200`, extra-large `≥ 1600`) [9]. Bar: 3–5 destinations; rail: 3–7 [10].
  *Applies as:* bottom bar below 600px (max 5 items), rail 600–839px, full sidebar ≥ 840px.
- **Cards.** Card specs not read (no verificado).
  *Applies as:* one card per tunnel/profile, a container with a single primary action.
- **State layers.** Hover 8%, focus 10%, pressed 10%, dragged 16%, disabled content 38% [11].
  *Applies as:* one `--state-hover/focus/pressed` token set used by every interactive element.
- **Motion.** Standard easing `cubic-bezier(0.2, 0, 0, 1)`; 150–250ms for small changes,
  300–500ms for larger transitions [12].
  *Applies as:* motion tokens `--dur-short: 150ms`, `--dur-medium: 250ms`, standard easing,
  all disabled under `prefers-reduced-motion`.
- **Snackbars.** Auto-dismiss, one at a time, queue the rest, at most one action; Compose: 4s
  short, 10s long [13].
  *Applies as:* toasts for results of the user's own action; 4s, or 10s with an action ("Deshacer").
- **Dialogs.** Purposefully interruptive, use sparingly [14].
  *Applies as:* dialogs only for confirmation of destructive or irreversible actions ("Salir y desconectar todo").
- **Progress / loading.** Determinate and indeterminate, linear and circular [14]. Whether M3
  gives skeleton guidance: no verificado.
  *Applies as:* linear indeterminate bar on the page header for background refreshes; no full spinners.

## 3. Microsoft Fluent 2 / WinUI (most users are on Windows)

- **NavigationView.** Top nav only for ≤ 5 equally important categories; left pane otherwise;
  expanded pane ≥ 1008px, compact (icons) or minimal (hamburger) below [15].
  *Applies as:* consistent with §1/§2: left nav on desktop; labelled icons when compact.
- **MessageBar / InfoBar.** Inline status of the product or surface, severity, optional action;
  a warning must carry a button or link [16][17].
  *Applies as:* the app-wide banner (daemon offline, helper not installed, update available)
  is a MessageBar with one "what to do" button.
- **Toast.** Temporary, not critical; critical goes to a modal [18].
  *Applies as:* toasts never carry the only copy of an error; the error also stays on the item.
- **Switch.** Immediate effect, always labelled [19].
  *Applies as:* switches apply on change (no Save button); settings forms that need Save use checkboxes.
- **Dialog.** Not for contextual errors; always a safe button; ≤ 2 "do it" buttons [20].
  *Applies as:* validation errors inline under the field; confirm dialog = "Cancelar" + one verb button.

## 4. WCAG 2.2 AA (live-data dashboard)

| SC | Requirement | Applies to Kongtrol panel as |
|---|---|---|
| 2.4.1 Bypass Blocks (A) | a mechanism to skip repeated blocks | "Saltar al contenido" link + `<main>` landmark |
| 2.4.11 Focus Not Obscured (Minimum) (AA) | a focused component is "not entirely hidden due to author-created content" | toasts and sticky header/bottom bar never cover focus; `scroll-padding` equal to their height |
| 2.5.8 Target Size (Minimum) (AA) | targets ≥ 24×24 CSS px, or spaced so a 24px circle doesn't overlap | 24px floor everywhere; 44px for primary actions (HIG) |
| 4.1.3 Status Messages (AA) | status messages are programmatically determinable without focus | one polite live region (`role="status"`) for toasts and connect phases; `role="alert"` only for errors [31] |
| 2.2.2 Pause, Stop, Hide (A) | auto-updating info that starts automatically and "lasts more than five seconds" with other content needs pause/stop/hide | live charts get a pause control; live numbers are not announced on every tick |
| 1.4.13 Content on Hover or Focus (AA) | Dismissible (Esc), Hoverable, Persistent | tooltips close on Esc, stay while hovered, never hold essential info |
| 3.3.1 / 3.3.3 (A / AA) | identify the error in text; give suggestions when known | API `hint` (`.fix` key) shown under the field, linked by `aria-describedby` |
| 3.2.6 Consistent Help (A) | help in the same relative order on every page | Guide link always in the same place of the sidebar |
| 3.3.7 Redundant Entry (A) | don't ask again for info already given in the same process | edit forms are pre-filled; the wizard carries values between steps |
| 2.3.3 Animation from Interactions (AAA, not required) | motion from interaction can be disabled | honour `prefers-reduced-motion` anyway (cheap) |

## 5. Nielsen's 10 heuristics → panel rules [22]

1. **Visibility of system status** → global protection state always visible; data age ("hace 3 s") when stale.
2. **Match with the real world** → "Conectada", "Bloqueado", no "daemon/API/token" without a short explanation (AGENTS.md).
3. **User control and freedom** → Cancel during connect; undo for removals where possible.
4. **Consistency and standards** → same component for the same thing (switch = immediate, one status chip style).
5. **Error prevention** → validate before save (`config.Validate` trial), disable nothing silently — say why.
6. **Recognition over recall** → pickers for profiles/groups, not free text.
7. **Flexibility and efficiency** → keyboard shortcuts and favorites for devs; defaults for everyone else.
8. **Aesthetic and minimalist design** → advanced fields hidden; one primary action per card.
9. **Recognize, diagnose, recover from errors** → every error = what happened + one "what to do" line + button.
10. **Help and documentation** → contextual "?" links to `/guia` sections.

## 6. Live-data patterns

- **Stale-while-revalidate.** RFC 5861 serves stale content while revalidating [25]; SWR and
  TanStack Query apply it on the client: show cached data, refetch on mount/focus/reconnect,
  dedupe by key (SWR 2s), retry 3× with exponential backoff, GC unused data after 5 min [26][27].
  Optimistic updates snapshot the old value in `onMutate` and roll back in `onError` [26].
  *Applies as:* never blank a page on refresh; toggles flip optimistically and roll back with an error toast.
- **WebSocket + cache.** Socket events write into or invalidate the query cache; with a live
  socket, `staleTime: Infinity` avoids duplicate polling [28].
  *Applies as:* the daemon WebSocket snapshot feeds the cache; REST refetch only on reconnect.
- **No flicker.** External stores via `useSyncExternalStore` (tear-free) [29]; stable keys and
  memoized rows; `font-variant-numeric: tabular-nums` for counters; reserved space for async
  content to avoid layout shift (CLS). *Applies as:* fixed-width numeric cells and fixed-height
  card slots; no layout change when data arrives.
- **Reconnect.** Capped exponential backoff plus jitter (Full Jitter) [30].
  *Applies as:* WebSocket reconnect `random(0, min(30s, 0.5s·2^n))`, reset on success; show
  "Reconectando…" in the banner after the first failure.
- **Response times.** 0.1s instant, 1s flow, 10s attention [23]. Skeletons for full-page
  loads, spinners for 2–10s, progress indicator beyond 10s [24].
  *Applies as:* no indicator under 1s (delay it ~300–500ms, reasoned, not verified as a
  standard); skeleton only on first load; connect (can take minutes) shows phases, not a spinner.
- **Toast a11y.** Polite live region, never steal focus [31]; 4–10s per M3 [13]; persistent
  for anything critical (Fluent) [18]. WCAG 2.2.1 Timing Adjustable may apply when a toast holds
  an action: give it 10s and keep the action elsewhere too (reasoned; no verificado as required).

## 7. Router and data-fetch library

npm registry, accessed 2026-11-18. The repo rule requires releases ≥ 2 weeks old (published
before 2026-11-04). React in `web/ui` is 19.3.1.

| Package | `latest` (date) | Newest ≥ 2 weeks | Peer `react` | Size min / gzip (bundlephobia) | First release |
|---|---|---|---|---|---|
| `react-router` | 8.5.2 (2026-11-18, too new) | **8.5.1 (2026-10-27)** | `>=19.2.7` | 206.8 kB / 64.6 kB (whole entry, not tree-shaken) | 2014; v7 2024-11-22, v8 2026-06-17 |
| `@tanstack/react-router` | 1.170.46 (2026-10-22) | **1.170.46 (2026-10-22)** | `>=18 \|\| >=19` | 90.9 kB / 31.0 kB | 2022 |
| `wouter` | 3.13.0 (2026-09-30) | **3.13.0 (2026-09-30)** | `>=16.8` | 5.7 kB / 2.7 kB | 2019 (Unlicense) |
| `@tanstack/react-query` | 5.108.0 (2026-11-17, too new) | **5.105.3 (2026-11-02)** | `^18 \|\| ^19` | 46.2 kB / 13.3 kB | 2022 (v5 2023-10-17) |

Note: the current React Router major is **v8**, not v7 (v8.0.0 published 2026-06-17).

- **React Router (Declarative or Data mode).** Most used and oldest; `basename` for `/app/` [32];
  Data mode adds loaders/actions we would not use if TanStack Query owns data. Real cost after
  tree-shaking in Declarative mode is lower than the table (no verificado: not measured).
- **TanStack Router.** Fully type-safe params and search params; `basepath` [33]; file-based
  routing optional (needs a Vite plugin). Pays off with many routes and typed search state.
- **wouter.** ~2.7 kB, `base` prop [34], hooks API; fewer features (no loaders, no typed routes),
  one main maintainer.
- *Applies to Kongtrol panel as:* ~6–8 flat routes and no SSR → a small router suffices. React
  Router 8.5.1 Declarative mode is the safe default (maturity, docs, contributors familiar with it);
  wouter if bundle size matters most. TanStack Router is justified only if typed search params grow.
- **TanStack Query vs in-house store.** In-house needs: per-key cache, dedupe, refetch on focus
  and reconnect, retry with backoff, optimistic update + rollback, mutation invalidation [26].
  That is most of TanStack Query (~13 kB gzip, tested, devtools). *Applies as:* use
  `@tanstack/react-query` 5.105.3; keep the WebSocket in a tiny `useSyncExternalStore` store that
  writes into the query cache, instead of writing our own cache.

## 8. Toasts and dialogs

| Package | Version (date) | Size min / gzip |
|---|---|---|
| `sonner` | 2.0.8 (2026-08-09) | 34.1 kB / 9.4 kB |
| `react-hot-toast` | 2.6.1 (2026-09-16) | 11.9 kB / 4.8 kB |
| `@radix-ui/react-dialog` | 1.3.1 (2026-10-12) | 39.5 kB / 13.2 kB |

- **Toasts.** The needed behaviour is small: a queue showing one at a time [13], 4/10s timers
  paused on hover/focus, one polite live region [31], Esc to dismiss, i18n keys rendered by the
  client. *Applies as:* in-house (~100 lines + tests) keeps full control of a11y and i18n;
  `sonner` is the fallback if requirements grow (stacking, swipe).
- **Dialog.** Native `<dialog>` with `showModal()` is Baseline since March 2022: top layer,
  background `inert`, Esc closes, `::backdrop`, `autofocus`, `closedby` [31]. Radix Dialog adds a
  focus trap and scroll lock in JS (13 kB gzip). *Applies as:* native `<dialog>` wrapped in one
  `<Dialog>` component (labelled by its heading, focus returned to the opener on close, body
  scroll lock in CSS); no Radix.

## Summary of recommendations

1. Left sidebar ≥ 840px, rail 600–839px, bottom bar < 600px; ≤ 5 flat destinations, always visible.
2. Global protection state on top of every page; per-tunnel status inside its card.
3. Daemon offline / helper missing / update = inline MessageBar with one "what to do" button, over the last known data; never a modal.
4. Dialogs only for destructive or irreversible confirmations; Cancel + one verb button; native `<dialog>`.
5. Switches apply immediately with text state + color; forms needing Save use checkboxes.
6. Toasts: in-house, one at a time, 4s (10s with action), polite live region, never the only copy of an error.
7. Validation errors inline under the field with the API `hint` (`aria-describedby`).
8. Tokens: state layers 8/10/10%, motion 150/250ms with `cubic-bezier(0.2,0,0,1)`, all off under `prefers-reduced-motion`.
9. WCAG 2.2 AA: skip link, 24px minimum targets (44px primary), focus never under toasts/sticky bars, pause for live charts, Esc-dismissible tooltips, Guide link always in the same place.
10. Stale-while-revalidate: never blank on refresh; skeleton only on first load; no indicator under ~0.5s.
11. Connect shows daemon phases, not a spinner; a stalled phase shows the "what to do" line.
12. No flicker: tabular-nums, fixed-size cells and cards, stable keys, `useSyncExternalStore` for the WebSocket store.
13. WebSocket reconnect: Full Jitter, cap 30s, reset on success, "Reconectando…" visible.
14. Router: `react-router` 8.5.1 (Declarative mode, `basename: "/app"`); wouter 3.13.0 if size wins.
15. Data: `@tanstack/react-query` 5.105.3 (cache, dedupe, retry, optimistic + rollback), fed by the WebSocket.
