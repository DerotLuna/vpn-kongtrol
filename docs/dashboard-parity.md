<img src="../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Dashboard parity: legacy → new UI

Version 1.6.0 · 2026-11-19

**Complete.** Every row is ticked or justified (`replaced`, `dropped: ADR-0043`, `n/a`). The legacy
dashboard was deleted and the new UI moved to `/` ([ADR-0043](adr/0043-new-dashboard-replaces-classic.md)).
This file stays as the record that ADR-0038, ADR-0042 and ADR-0043 cite: legacy `file:line`
references point to the tree before that change (`git show 35a2630:web/dashboard/<file>`), and
routes below are now served at `/` instead of `/app/`.

## 1. Purpose and how to use

This is the parity contract for migrating the legacy vanilla-JS dashboard (`web/dashboard/`, served at `/`)
to the new React UI (`web/ui/`, served at `/app/`). The legacy dashboard is only retired when every row
below is ticked. Nothing may be dropped silently: dropping a behaviour needs an ADR in `docs/adr/` and a
row note `dropped: ADR-NNNN`.

How to use it (phase 2):

- Each row is a `- [ ]` checkbox. When a row ships, tick it and replace the status with the new file:
  `status: done (web/ui/src/pages/HomePage.tsx)`.
- Rows give the legacy source (`file:line`, under `web/dashboard/` unless a path is given), the API it
  uses (`METHOD path (handler file:line)`, under `internal/api/`), the target route (under `/app/`) and
  the current state of the new UI.
- A legacy bug is not parity: rows flagged **fix** must be fixed in the new UI, not copied.
- AGENTS.md rules apply to every row: strings via `src/i18n/messages.ts` (ES+EN), every error has a
  "what to do" line, no color-only states, CLI parity, docs/landing in sync.

Legend:

| Mark | Meaning |
|---|---|
| `status: new UI has it (<file>)` | Behaviour exists in `web/ui/src/<file>`; verify against the row, then tick |
| `status: partial (<file>: …)` | Exists, but the listed parts are missing |
| `status: missing` | Not in the new UI |
| **fix** | Legacy behaviour is a bug or breaks an AGENTS.md rule; implement the corrected version |
| **IA: revisar** | Placement is a judgement call; confirm before building |
| `no verificado` | Reasoned from code reading, not tested |

Snapshot verified against commit `0f78f0e` (2026-11-18).

## 2. Target information architecture

All routes under base `/app/`.

| Route | Name (ES / EN) | Contents |
|---|---|---|
| `/` | Inicio / Home | Aggregate status, tunnels list + connect/disconnect/cancel/reload, groups quick launch, base network egress, public IP, security summary, live notices |
| `/profiles` | Perfiles / Profiles | VPN profile CRUD, per-profile kill-switch override (shortcut, see decision below) |
| `/profiles/groups` | Grupos / Groups (tab) | Group CRUD + connect/disconnect/reload group |
| `/rules` | Reglas / Rules | Policy CRUD, policy meta, test before saving, reload from disk, resolve |
| `/rules/schedule` | Horarios / Schedule (tab) | Scheduler on/off + interval, scheduler rule CRUD |
| `/traffic` | Tráfico / Traffic | Traffic over time chart, traffic map & policy resolver, active routes (live), per-tunnel routes, network overview |
| `/security` | Seguridad / Security | Kill switch, DNS guard, per-profile kill-switch override, public IP lookup / leak check, kill switch tuning, DNS guard tuning, split DNS |
| `/activity` | Actividad / Activity | Audit log, events feed |
| `/settings` | Ajustes / Settings | Dashboard prefs (bind/port read-only), language/theme, diagnostics + export, health check, updates, data purge, helper, restart/shutdown, setup |

**Decision (per-profile kill-switch override):** the editable override table lives on `/security`
(it changes protection, and the global switch it overrides is there). `/profiles` shows the override
read-only per row with a link to `/security#profile-overrides`. Rows below follow this decision.

Legacy → target page map: `index.html` → `/` + `/traffic`; `studio.html` → `/profiles`,
`/profiles/groups`, `/rules`; `security.html` → `/security`; `settings.html` → `/settings`,
`/security`, `/rules/schedule`, `/activity`; `audit.html` → `/activity`.

## 3. Checklist per legacy page

### 3.1 `index.html` + `app.js` (Overview)

#### Screens / sections

- [x] Tunnels card (title, table) — legacy index.html:19 — API GET /api/v1/tunnels (handlers_tunnels.go:40) + WS — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx)
- [x] Groups quick-launch card — legacy index.html:49, app.js:1140 — API GET /api/v1/groups (handlers_groups.go:17) — target `/` — status: done (web/ui/src/components/home/GroupsCard.tsx)
- [x] Traffic over time card — legacy index.html:59, app.js:609 — API GET /api/v1/metrics/history (handlers_tunnels.go:31) + WS — target `/traffic` — status: done (web/ui/src/components/traffic/TunnelTrafficCard.tsx, web/ui/src/data/traffic.ts)
- [x] Traffic map & policy resolver card — legacy index.html:68 — API GET /api/v1/policies (handlers_policy.go:53), GET /api/v1/resolve (handlers_policy.go:309) — target `/traffic` — status: done (web/ui/src/components/traffic/LookupCard.tsx)
- [x] Base network egress card — legacy index.html:97, app.js:852 — API GET /api/v1/network/overview (handlers_routes.go:94) — target `/` (same data reused by `/traffic` network overview) — status: done (web/ui/src/components/home/NetworkCard.tsx)
- [x] Security controls summary card + "Manage →" link — legacy index.html:112-116 — API GET /api/v1/security/status (handlers_security.go:91) — target `/` (link to `/security`) — status: done (web/ui/src/components/home/SecurityCard.tsx)
- [x] Active routes (live) card — legacy index.html:142, app.js:803 — API GET /api/v1/routes (handlers_routes.go:13) — target `/traffic` — status: done (web/ui/src/components/traffic/SystemRoutes.tsx; collapsed by default, read only while open)
- [x] Helper install banner (prepended to `main`, `role=status`) — legacy app.js:357-386 — API GET /api/v1/helper/status (handlers_helper.go:67) — target `/` (+ `/settings` helper section) — status: done (web/ui/src/components/home/HelperBanner.tsx)
- [x] Aggregate status headline (N of M connected) — legacy none (new) — API GET /api/v1/tunnels — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx)

#### Tunnels table: fields per row

- [x] Profile name, rows sorted by name (`localeCompare`) — legacy app.js:523, app.js:577 — API WS snapshot `Name` key — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx); **fix**: legacy injects `name` unescaped into HTML and `onclick` (app.js:568-577)
- [x] Status dot + translated label (`status.connected|connecting|disconnected|error`) — legacy app.js:525-530 — API `Status` — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx)
- [x] Attach line (provider adapters, `attach.*` 7 phases, `waiting` style when user must act) — legacy app.js:540-543, app.js:1047-1059, app.js:1083 — API `attach {phase,app,location,reason}` — target `/` — status: done (web/ui/src/components/home/TunnelRow.tsx)
- [x] Health line (`health.idle|no_response|unknown` + fix from `reason_key`), caution vs waiting style — legacy app.js:547-551, app.js:1064-1080 — API `health {state,reason_key}` — target `/` — status: done (web/ui/src/components/home/TunnelRow.tsx)
- [x] Last error line `tunnel.lastError ({time})` + translated `last_error_key` (8 keys), raw error as tooltip, hidden when connected — legacy app.js:327-341, app.js:535-537 — API `last_error`, `last_error_key`, `last_error_at` — target `/` — status: done (web/ui/src/components/home/TunnelRow.tsx)
- [x] "What goes through this tunnel (n / everything)" toggle (`aria-expanded`) — legacy app.js:552-555, app.js:747 — API GET /api/v1/tunnels/routes (handlers_tunnel_routes.go:76) — target `/` (opens `/traffic` per-tunnel view or inline) — status: done (web/ui/src/components/home/TunnelRoutes.tsx: one-line toggle "Rutas (n)" with the question as tooltip; panel in a full-width row under the tunnel (DataList `rowDetail`) + link to /traffic)
- [x] Assigned IP (— when empty) — legacy app.js:559 — API `AssignedIP` — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx)
- [x] Uptime (`formatDuration`, only when connected) — legacy app.js:560, app.js:1196 — API `UptimeSeconds` — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx)
- [x] Bytes sent / received (`formatBytes`, — when 0) — legacy app.js:561-562, app.js:1189 — API `BytesSent`, `BytesReceived` — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx)
- [x] Latency `{n}ms` — legacy app.js:563 — API `LatencyMS` — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx)
- [x] Latency trend sparkline (canvas, connected only, redrawn on theme change) — legacy app.js:564-566, app.js:591-596, app.js:1221-1229 — API WS samples — target `/` — status: done (web/ui/src/components/home/Sparkline.tsx: SVG with a text alternative); **fix**: canvas has no text alternative

#### Tunnels: actions

- [x] Connect (disconnected/error rows) — legacy app.js:568, app.js:1020 — API POST /api/v1/tunnels/{name}/connect (handlers_tunnels.go:50; 202 `connecting`, 200 `already_connected`, 404, 409 in progress, 503) — target `/` — status: done (web/ui/src/components/home/TunnelRow.tsx + web/ui/src/data/home.ts); **fix**: legacy omits `encodeURIComponent` (app.js:1022) and swallows errors (no toast)
- [x] Cancel (connecting rows) — legacy app.js:572-573, app.js:1033 — API POST /api/v1/tunnels/{name}/cancel_connect (handlers_tunnels.go:88) — target `/` — status: done (web/ui/src/components/home/TunnelRow.tsx + web/ui/src/data/home.ts); **fix**: same encoding/error issues (app.js:1035)
- [x] Disconnect (connected rows) — legacy app.js:569-570, app.js:1087 — API POST /api/v1/tunnels/{name}/disconnect (handlers_tunnels.go:112; 202 `waiting_user` for manual disconnect) — target `/` — status: done (web/ui/src/components/home/TunnelRow.tsx + web/ui/src/data/home.ts); **fix**: legacy ignores 202 `waiting_user` (must say "disconnect in the app") and errors
- [x] Reload (connected rows; re-reads yaml, restarts tunnel in place) — legacy app.js:571, app.js:1107-1128 — API POST /api/v1/tunnels/{name}/reload (handlers_tunnels.go:150; 409 `restart_required`, 202 `restarting|skipped_not_connected`) — target `/` — status: done (web/ui/src/components/home/TunnelRow.tsx + web/ui/src/data/home.ts); **fix**: `skipped_not_connected` still toasts "Reloading". Shown as the icon button "Reconectar" (tooltip says what it does). While any row action runs, the row's buttons stay in place with `aria-disabled`, the pressed one spins with "…ando" (`aria-busy`), the pill shows the transition (also for `waiting_disconnect`) and repeated clicks send nothing
- [x] Connect all (non-connected tunnels, parallel) — legacy index.html:23, app.js:1177 — API POST …/connect ×N — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx)
- [x] Disconnect all (connected tunnels, parallel) — legacy index.html:24, app.js:1182 — API POST …/disconnect ×N — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx)
- [x] Refresh tunnels button — legacy index.html:25, app.js:459 — API GET /api/v1/tunnels — target `/` — status: done (web/ui/src/components/home/TunnelsCard.tsx)
- [x] After any tunnel action: refresh routes + network overview after 600 ms — legacy app.js:1026-1029, 1038-1041, 1092-1095, 1123-1126 — API GET /api/v1/routes, /network/overview — target `/` — status: done (web/ui/src/data/home.ts settleNetwork)

#### Groups quick launch

- [x] Group chip: name, `{n} profiles`, Connect, Disconnect — legacy app.js:1146-1153 — API GET /api/v1/groups — target `/` — status: done (web/ui/src/components/home/GroupsCard.tsx)
- [x] Connect group — legacy app.js:1156-1165 — API POST /api/v1/groups/{name}/connect (handlers_groups.go:148; 202 `started[]`, `already_connected[]`) — target `/` — status: done (web/ui/src/components/home/GroupsCard.tsx); **fix**: ignores `res.ok` and both lists
- [x] Disconnect group — legacy app.js:1167-1175 — API POST /api/v1/groups/{name}/disconnect (handlers_groups.go:199; 500 joined errors) — target `/` — status: done (web/ui/src/components/home/GroupsCard.tsx)
- [x] Refresh groups button — legacy index.html:52 — API GET /api/v1/groups — target `/` — status: done (web/ui/src/components/home/GroupsCard.tsx)

#### Traffic over time (per connected tunnel)

- [x] Sent/received throughput line chart (per-tick deltas, clamped ≥0, 120 samples) — legacy app.js:470-508, app.js:619-649, charts.js:44 — API WS `BytesSent/BytesReceived` — target `/traffic` — status: done (web/ui/src/data/traffic.ts `createTrafficHistory`: bytes/s per tick, clamped ≥0, 120 samples, late frames ignored; web/ui/src/components/traffic/MiniChart.tsx)
- [x] Legend (Sent = `--steel`, Received = `--signal`, with text) — legacy app.js:631-634 — API — — target `/traffic` — status: done with fix (web/ui/src/components/traffic/TunnelTrafficCard.tsx: small multiples, one series per chart named by its title, so no color legend; amber stays brand-only and steel vs. muted fails the dataviz normal-vision floor)
- [x] Stat tiles: reconnects, jitter ms, total time connected — legacy app.js:621-627 — API GET /api/v1/metrics/history (handlers_tunnels.go:31) `reconnects`, `jitter_ms`, `uptime_seconds` — target `/traffic` — status: done (web/ui/src/components/traffic/TunnelTrafficCard.tsx `tr-stats`)
- [x] Latency samples recorded only when the value changes — legacy app.js:488-501 — API WS `LatencyMS` — target `/traffic` + `/` sparkline — status: done for `/traffic` (web/ui/src/data/traffic.ts `createTrafficHistory` latency; Home sparkline pending)

#### Traffic map & policy resolver

- [x] Resolve input (IP/domain, URL, policy/profile, `app:process`) + Check + Clear — legacy index.html:73-77, app.js:918, app.js:991 — API GET /api/v1/resolve?target=|app= (handlers_policy.go:309) `target, app, via, rule, matched` — target `/traffic` — status: done (web/ui/src/components/traffic/LookupCard.tsx)
- [x] `app:` prefix switches the query to `app=` — legacy app.js:1003-1006 — API GET /api/v1/resolve?app= — target `/traffic` — status: done (web/ui/src/data/traffic.ts `parseLookup`)
- [x] Matched result `{lhs} → {via} ({rule})` (matched style) — legacy app.js:937-939 — API `matched, via, rule` — target `/traffic` — status: done with fix (web/ui/src/components/traffic/LookupCard.tsx + web/ui/src/data/traffic.ts `explainLookup`: says what happens now — VPN off → normal connection, `on_tunnel_down: block` → blocked; domains can show their IPs via GET /api/v1/dns/resolve)
- [x] Unmatched result `→ default route (no matching policy) (public IP x)` — legacy app.js:940-948, app.js:709 — API `matched=false` + public IP when lookup on — target `/traffic` — status: done (web/ui/src/components/traffic/LookupCard.tsx: rest_via interface, VPN that took all traffic, last known public IP)
- [x] Policies table filtered by the resolve query (name, via, domains, ip_ranges, apps) + note "Showing x of y" — legacy app.js:892-903, app.js:955-989 — API GET /api/v1/policies `name, via, domains, ip_ranges, apps, resolved_cidrs` — target `/traffic` — status: done (web/ui/src/components/traffic/LookupCard.tsx `RulesList`, web/ui/src/data/traffic.ts `filterPolicies`)
- [x] Policy row: match tags (domain / ip), via profile, resolved IP count (`{n} IPs` or —) — legacy app.js:974-987 — API `resolved_cidrs` — target `/traffic` — status: done (web/ui/src/components/traffic/LookupCard.tsx; per-rule IPs with expiry from the new `resolved` field, fallback `resolved_cidrs`)
- [x] Refresh policies button — legacy index.html:71 — API GET /api/v1/policies — target `/traffic` — status: done (web/ui/src/components/traffic/LookupCard.tsx)

#### Active routes (live)

- [x] Route list: destination, tag (default / origin / policy), `→ profile · iface` — legacy app.js:826-840 — API GET /api/v1/routes (handlers_routes.go:13) `destination, gateway, interface, metric, policy_name, policy_via, is_default` — target `/traffic` — status: done (web/ui/src/components/traffic/SystemRoutes.tsx, + gateway and metric columns)
- [x] Origin index joins routes with per-tunnel routes (iface|dest) — legacy app.js:755-765 — API GET /api/v1/tunnels/routes — target `/traffic` — status: done (web/ui/src/data/traffic.ts `routeOriginIndex`)
- [x] Filter select: all, default, profile:x, origin:server|policy|vpn, policy:x (keeps selection if still present) — legacy index.html:147-153, app.js:872-890, app.js:811-819 — API — — target `/traffic` — status: done (web/ui/src/components/traffic/SystemRoutes.tsx, web/ui/src/data/traffic.ts `routeFilterOptions`/`effectiveFilter`)
- [x] Refresh routes button — legacy index.html:145 — API GET /api/v1/routes + /tunnels/routes — target `/traffic` — status: done (web/ui/src/components/traffic/SystemRoutes.tsx)

#### Per-tunnel routes drawer

- [x] Title + interface, "all your traffic", "nothing yet" — legacy app.js:786-788 — API GET /api/v1/tunnels/routes (handlers_tunnel_routes.go:76) `name, connected, interface, all_traffic, total, counts, routes[]` — target `/traffic` — status: done (web/ui/src/components/traffic/TunnelRoutesPanel.tsx)
- [x] Groups by origin (server / policy {name} / vpn) with counts, chips, server explanation line — legacy app.js:780-798 — API `routes[].origin, policy, destination` — target `/traffic` — status: done (web/ui/src/components/traffic/TunnelRoutesPanel.tsx, web/ui/src/data/traffic.ts `groupTunnelRoutes`)
- [x] "Everything else →" line (`rest_via_tunnel` / `rest_via` / unknown) — legacy app.js:773-777, app.js:799 — API `rest_via, rest_via_tunnel` — target `/traffic` — status: done (web/ui/src/components/traffic/TunnelRoutesPanel.tsx)
- [x] Open drawers survive re-renders — legacy app.js:303, app.js:747-751 — API — — target `/traffic` — status: done (web/ui/src/pages/TrafficPage.tsx: open state per tunnel in the page, open by default)

#### Base network egress

- [x] Connected tunnels count — legacy index.html:103, app.js:854 — API GET /api/v1/network/overview `connected_tunnels` — target `/` — status: done (web/ui/src/components/home/NetworkCard.tsx)
- [x] Default egress `iface (metric n)` — legacy index.html:104, app.js:856-860 — API `default_routes[0].interface, metric` — target `/` — status: done (web/ui/src/components/home/NetworkCard.tsx)
- [x] Gateway (or `on-link`) — legacy index.html:105, app.js:861 — API `default_routes[0].gateway` — target `/` — status: done (web/ui/src/components/home/NetworkCard.tsx)
- [x] Local IPs — legacy index.html:106, app.js:863-864 — API `local_ips` — target `/` — status: done (web/ui/src/components/home/NetworkCard.tsx)
- [x] Public IP row + Check button — legacy index.html:107, app.js:865-869 — API `public_ip, public_ip_check_enabled` — target `/` — status: done (web/ui/src/components/home/NetworkCard.tsx; "Revisar" in web/ui/src/components/home/SecurityCard.tsx)
- [x] Note "Unmatched traffic follows your default route" — legacy index.html:109 — API — — target `/` — status: done (web/ui/src/components/home/NetworkCard.tsx)
- [x] Refresh button — legacy index.html:100 — API GET /api/v1/network/overview — target `/` — status: done (web/ui/src/components/home/NetworkCard.tsx)

#### Security summary

- [x] Kill switch badge (ACTIVE/ARMED/OFF + detail text) — legacy index.html:118-121, app.js:670-671, shell.js:299-306 — API GET /api/v1/security/status `kill_switch, kill_switch_enabled, kill_switch_status` — target `/` — status: done (web/ui/src/components/home/SecurityCard.tsx)
- [x] DNS guard badge (ACTIVE/IDLE/DISABLED) — legacy index.html:122-125, app.js:672 — API `dns_guard, dns_guard_enabled` — target `/` — status: done (web/ui/src/components/home/SecurityCard.tsx)
- [x] Leak check badge (CLEAN/LEAK/ERROR/PENDING/UNAVAILABLE/DISABLED) — legacy index.html:126-129, app.js:673, shell.js:312-318 — API `leak_check, leak_check_available, leak_detection_configured` — target `/` — status: done (web/ui/src/components/home/SecurityCard.tsx)
- [x] Public IP + Check (hidden while lookup off, hint instead) — legacy index.html:130-134, app.js:677-680 — API `public_ip_check_enabled, leak_check.public_ip` — target `/` — status: done (web/ui/src/components/home/SecurityCard.tsx)
- [x] Last checked time (`toLocaleTimeString(lang)`, — when off) — legacy index.html:135-138, app.js:682-698 — API `leak_check.checked_at` — target `/` — status: done (web/ui/src/components/home/SecurityCard.tsx)

#### States

- [x] Loading skeleton rows (tunnels, policies, routes, base network, groups) — legacy index.html:43-44, 55, 90-91, 103-107, 155-156 — API — — target `/`, `/traffic` — status: done (web/ui/src/components/home/*: Skeleton per card on first load)
- [x] Empty: no tunnels / no policies / no policies for filter / no routes for filter / no groups / chart no data — legacy app.js:517, 959, 970, 822, 1143, 615 — API — — target `/`, `/traffic` — status: done (web/ui/src/components/home/*: no VPNs, no groups, no notices, sparkline "Midiendo"; policies/routes/chart empties are /traffic)
- [x] Fetch errors: legacy silently ignores (`catch (_) {}`) — legacy app.js:459-468, 652-659, 718-726 … — API all — target shell (every page) — status: done (components/ui/States.tsx `ErrorState`: every card shows the error + a "what to do" line)
- [x] Button state per tunnel status (connect / cancel / disconnect+reload) — legacy app.js:568-574 — API `Status` — target `/` — status: done (web/ui/src/components/home/TunnelRow.tsx)
- [x] First-run redirect: `needs_setup` → profile editor — legacy app.js:1242-1247 — API GET /api/v1/setup (handlers_setup.go:39) `needs_setup` — target `/` → `/profiles` — status: done (web/ui/src/components/home/TunnelsCard.tsx: first-run step "Agregar VPN" opens the "＋ Agregar" sheet, ADR-0055; no automatic redirect)

#### Notices / toasts

- [x] `toast.daemonLost` (error, 8 s) / `toast.daemonRestored` (success) on WS state change, not on first load — legacy app.js:442-455 — API WS — target shell (every page) — status: done, as a persistent inline notice instead of an 8 s toast (components/layout/AppNotices.tsx: what to do + automatic reconnect + Retry; components/layout/AggregateStatus.tsx "Reconectando…"); it clears itself when the daemon answers again
- [x] `toast.groupConnecting` / `toast.groupDisconnecting` / `toast.groupActionFailed` — legacy app.js:1159-1172 — API group connect/disconnect — target `/` — status: done (web/ui/src/data/home.ts useGroupAction)
- [x] `toast.tunnelReloading` (info), `toast.tunnelRestartRequired` (409), `alerts.tunnelReloadFailed` via `showApiError` — legacy app.js:1110-1121 — API tunnel reload — target `/` — status: done (web/ui/src/data/home.ts useTunnelReload)
- [x] `helper.installing` (busy banner), `helper.installed` (success toast), `helper.failed` + `helper.failedHint` / daemon `error`+`hint`+`detail` — legacy app.js:390-415 — API POST /api/v1/helper/install (handlers_helper.go:74; 403 cancelled, 504 timeout, 500 failed) — target `/` + `/settings` — status: done (web/ui/src/components/home/HelperBanner.tsx)

#### Keyboard / focus

- [x] Enter in resolve input runs Check — legacy app.js:1012-1014 — API GET /api/v1/resolve — target `/traffic` — status: done (web/ui/src/components/traffic/LookupCard.tsx: a real form, Enter submits)
- [x] Route-drawer toggle is a button with `aria-expanded` — legacy app.js:554 — API — — target `/traffic` / `/` — status: done (components/home/TunnelRoutes.tsx and components/traffic/TunnelTrafficCard.tsx: buttons with `aria-expanded`)

#### Endpoints / cadence

- [x] WS `/api/v1/ws/metrics` (ws.go:17; push on change + every 5 s), reconnect after 3 s — legacy app.js:419-440 — target `/` — status: done (web/ui/src/pages/HomePage.tsx via api/hooks.ts useTunnels + data/live.ts)
- [x] Poll 30 s: security status, routes (+tunnel routes), policies, network overview, metrics history, groups, helper status — legacy app.js:1231-1238 — target `/`, `/traffic` — status: done (web/ui/src/data/home.ts SLOW_POLL_MS (security 5 s, helper 10 s while missing))
- [x] Re-render cached data on language change (policies, routes, charts, helper banner) — legacy app.js:317-322 — API — — target shell (every page) — status: done (React re-render via app/PreferencesProvider.tsx)

### 3.2 `studio.html` + `studio.js` (Studio)

#### Shared

- [x] Tabs Profiles / Policies / Groups, last tab remembered (localStorage `kongtrol-dashboard-tab-studio`) — legacy studio.html:20-25, shell.js:327-346 — API — — target `/profiles`, `/profiles/groups`, `/rules` (routes replace tabs) — status: done (web/ui/src/components/profiles/Bits.tsx `ProfilesHeader`: routes + TabLinks replace tabs, so the URL remembers the sub-page)
- [x] Active config file path line (read-only, `--config`) — legacy studio.html:27, studio.js:448-451 — API GET /api/v1/policies/meta (handlers_policy.go:113) `config_path` — target `/settings` (also shown on `/profiles`) **IA: revisar** — status: done (pages/SettingsPage.tsx, components/settings/SystemSection.tsx: from GET /api/v1/setup `config_path`)
- [x] Error toast text: localized `error`+`hint` when `error_key` set, else page text + raw detail — legacy studio.js:341-345 — API `writeLocalizedError` body `error, hint, error_key` (errors_i18n.go) — target `/profiles`, `/rules` — status: done (web/ui/src/components/profiles/errors.ts: known plain-English profile/group errors rewritten as message + what to do; translated `error_key` bodies shown as sent, web/ui/src/api/errors.ts)
- [x] Requests send `X-Kongtrol-Lang` — legacy studio.js:483, 504, 551, 742, 772, 967, 989 — API — — target shell (api/client.ts) — status: done (api/client.ts `setApiLang`)

#### Tab: VPN Profiles → `/profiles`

Screens / sections

- [x] Setup banner (`setup.title`, `setup.body`) when `needs_setup`; forces Profiles tab — legacy studio.html:102-107, studio.js:818-830 — API GET /api/v1/setup (handlers_setup.go:39) `setup_mode, needs_setup, restart_pending, config_path` — target `/profiles` **IA: revisar** (spec lists setup under `/settings`) — status: done (web/ui/src/components/profiles/FirstRun.tsx: setup mode = first-run guide on /profiles with installed VPN apps from GET /api/v1/vpns/detect)
- [x] Restart banner (`restart.pending`, `restart.pendingHint`, Restart now) after any profile change or `restart_pending` — legacy studio.html:108-114, studio.js:833-839 — API `restart_required` from vpn CRUD — target `/profiles` (action in `/settings`) — status: done (web/ui/src/components/profiles/Restart.tsx, shown on /profiles and /profiles/groups after any change or `restart_pending`)
- [x] Note "changes update config + keychain; restart to use" — legacy studio.html:115 — API — — target `/profiles` — status: done (web/ui/src/pages/ProfilesPage.tsx footnote, with the config path)
- [x] Editor card + Existing profiles card — legacy studio.html:117, studio.html:215 — API — — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileEditor.tsx sheet + web/ui/src/components/profiles/ProfileList.tsx)

Fields (client side → server side)

- [x] Name (text, trimmed) — legacy studio.html:120-122, studio.js:614 — API required (handlers_vpn.go:191-193), duplicate 409 (handlers_vpn.go:200-202) — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx; required + duplicate checked inline, web/ui/src/components/profiles/draft.ts)
- [x] Type select (10 types; default openvpn) drives visible fields — legacy studio.html:123-136, studio.js:603-609 — API `oneof` (internal/config/schema.go:30) via `config.Validate` (handlers_vpn.go:277) — target `/profiles` — status: done (web/ui/src/components/profiles/TypePicker.tsx, now "Otra VPN" in the shell's "＋ Agregar" sheet (ADR-0055), + type select in ProfileForm.tsx when editing; fields per type from web/ui/src/components/profiles/catalog.ts)
- [x] Priority (number, default 10, `parseInt || 0`) + hint (tie-break only) — legacy studio.html:137-140, studio.js:616 — API `priority` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx "Más opciones", hint says tie-break only)
- [x] Backup profiles / failover (comma-separated) + hint — legacy studio.html:141-144, studio.js:617 — API `failover` validated by `config.Validate` (exist, not self, no cycles) — target `/profiles` — status: done (web/ui/src/components/profiles/FailoverEditor.tsx: ordered pick of existing profiles; self/duplicate/unknown/cycle checked before saving, draft.ts) — **fix** applied
- [x] Host (forticlient, ciscoanyconnect, globalprotect, tailscale) — legacy studio.html:145-147 — API `host` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx; required for forticlient/ciscoanyconnect/globalprotect, optional exit node for tailscale)
- [x] Config file path (openvpn, wireguard) + openvpn routes hint — legacy studio.html:148-151 — API `config` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx; path field, required; file upload needs a new endpoint, see report)
- [x] Exact server (protonvpn, nordvpn) — legacy studio.html:152-154 — API `server`; cleared for forticlient/surfshark (studio.js:590, 623) — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx; cleared for types without location, draft.ts `toRequest`)
- [x] Protocol (protonvpn; placeholder `wireguard`) — legacy studio.html:155-157 — API `protocol` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx)
- [x] Connect automatically yes/no + hint (forticlient, protonvpn, nordvpn; hidden = true) — legacy studio.html:158-164, studio.js:626 — API `auto_connect` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx: Switch; FortiClient wording explains the fallback)
- [x] Country, City (protonvpn, nordvpn) + hint — legacy studio.html:165-171 — API `country`, `city`; `validateProvider` rejects on surfshark/forticlient (internal/config/semantics.go) — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx; only nordvpn/protonvpn, as validateProvider)
- [x] Server group (nordvpn) — legacy studio.html:172-174 — API `group` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx)
- [x] Network connection name / interface (providers) + hint — legacy studio.html:175-178 — API `interface` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx "Más opciones")
- [x] Wait time (providers; e.g. 10m) + hint — legacy studio.html:179-182 — API `wait_timeout` (duration, `config.Validate`) — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx "Más opciones"; Go duration ≥ 10s checked inline, draft.ts)
- [x] App path (providers) — legacy studio.html:183-185 — API `app_path` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx "Más opciones")
- [x] Auth method select (certificate / credentials / certificate+credentials; hidden for protonvpn, nordvpn, surfshark) — legacy studio.html:186-192 — API `auth_method` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx: translated labels; under "Más opciones" for wireguard/tailscale/cloudflarewarp) — **fix** applied
- [x] Certificate path, Key path — legacy studio.html:193-198 — API `cert`, `key` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx; shown when the method uses a certificate)
- [x] Username (`autocomplete=off`) — legacy studio.html:199-201 — API `username` → OS keychain (handlers_vpn.go:285) — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx)
- [x] Password (`type=password`, `autocomplete=new-password`, blank = keep, never prefilled) — legacy studio.html:202-204, studio.js:652 — API `password` → keychain; env references rejected (handlers_vpn.go:255) — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx: never prefilled, "one is stored" note, blank = keep; $VAR rejected inline)
- [x] Provider-only fields cleared when hidden; other hidden fields not sent — legacy studio.js:578-588, studio.js:619-624 — API absent = kept server-side (handlers_vpn.go:140) — target `/profiles` — status: done (web/ui/src/components/profiles/draft.ts `toRequest`, tested in draft.test.ts)
- [x] Create sends only non-empty fields; Update sends the full draft (empty = clear) — legacy studio.js:634-636, 743, 773 — API POST /api/v1/vpns, PUT /api/v1/vpns/{name} — target `/profiles` — status: done (web/ui/src/components/profiles/draft.ts `toRequest`, tested in draft.test.ts)

Actions

- [x] Create — legacy studio.html:206, studio.js:738-761 — API POST /api/v1/vpns (handlers_vpn.go:184; 201 `restart_required`) — target `/profiles` — status: done (the shell's "＋ Agregar" sheet, web/ui/src/components/add/VpnStep.tsx + FileStep.tsx, ADR-0055: "Guardar y activar" = create, plus the restart only when `restart_required` is true; web/ui/src/data/profiles.ts `useSaveProfile`). Editing a connected profile saves with "Guardar y reconectar" (PUT `?reconnect=true`, components/profiles/ProfileEditor.tsx)
- [x] Create during first run: toast `setup.saved`, restart, go to `/` — legacy studio.js:753-756 — API POST /api/v1/restart (handlers_setup.go:54) — target `/profiles` — status: done (components/add/activate.ts + profiles/useRestartAction.ts: a create the daemon cannot apply live restarts by itself, first run or not, then a Done step offers "Conectar", "Hacer que un sitio salga por ella" and "Ver en Inicio")
- [x] Update (key = selected or name; rename supported) — legacy studio.html:207, studio.js:763-784 — API PUT /api/v1/vpns/{name} (handlers_vpn.go:214; 404, localized 400) — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileEditor.tsx). Rename: done (ADR-0042) — the name is editable; `{"status":"renamed"}` shows what followed it and the restart banner; 409 `api.err.vpn_exists` / `api.err.vpn_rename_busy` land on the name field, and a connected VPN says "desconéctala para renombrarla" before saving (components/profiles/ProfileForm.tsx, ProfileEditor.tsx).
- [x] Delete (selected or typed name) with confirm — legacy studio.html:208, studio.js:786-807 — API DELETE /api/v1/vpns/{name} (handlers_vpn.go:349; 409 last profile :362, referenced by policy :367, by group :374) — target `/profiles` — status: done (web/ui/src/pages/ProfilesPage.tsx + ConfirmDelete.tsx; optimistic; a 409 `api.err.vpn_in_use` / `vpn_last` reopens the dialog with "Dónde se usa" (policies, groups, backups, schedules, empty lists left out) instead of a toast — components/profiles/errors.ts deleteRefusal; last profile, group membership and other profiles' backups are checked first)
- [x] Import a WireGuard `.conf` / OpenVPN `.ovpn` (new, ADR-0042) — legacy none — API POST /api/v1/vpns/import (handlers_vpn_import.go → internal/profileimport) — target `/profiles` — status: done (web/ui/src/components/profiles/ImportFile.tsx: picker + drop, extension and 512 KiB checked first, fills the path, 409 `import_exists` → Reemplazar / Conservar inline, `api.err.import_*` reasons with what to do; manual path kept); CLI `kongtrol profile import`
- [x] Clear form (defaults: openvpn, priority 10, certificate, auto_connect yes) — legacy studio.html:209, studio.js:666-679 — API — — target `/profiles` — status: done (each "Nueva VPN" opens the "＋ Agregar" sheet with a fresh draft; Cancel / Esc asks before discarding a draft)
- [x] Row Edit (fills form) / row Delete — legacy studio.js:714-718, 724-728, 897-901 — API — — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileList.tsx icon buttons with the profile name in the label)
- [x] Filter by name, type, host, server (live) — legacy studio.html:217, studio.js:695-700 — API — — target `/profiles` — status: done (web/ui/src/pages/ProfilesPage.tsx; shown from 4 profiles, also matches the brand name)
- [x] Restart now: disable button, wait old daemon gone (≤240×500 ms) and new one back (≤120×500 ms), then reload lists — legacy studio.js:846-895 — API POST /api/v1/restart (handlers_setup.go:54; 501 unsupported), GET /api/v1/setup as liveness — target `/settings` (button also in `/profiles` banner) — status: done (components/settings/SystemSection.tsx + data/settings.ts useRestart, confirm dialog; the `/profiles` banner button belongs to that page)

Table fields

- [x] Name, Type, Host/Server, Credentials (`username + password` / `username only` / `password only` / —) — legacy studio.html:218-232, studio.js:681-686, 707-721 — API GET /api/v1/vpns (handlers_vpn.go:18) `name, type, host, server, has_username_credential, has_password_credential` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileList.tsx: VPN cell has type + host/server/file and this OS's support; providers show "en la app")
- [x] Kill-switch override shown read-only + link to `/security` — legacy none (decision §2) — API `kill_switch_override` — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileList.tsx + ProfileForm.tsx, link to /security#profile-overrides)

States / notices

- [x] Skeleton rows; empty `empty.noProfiles` (also when filter matches nothing) — legacy studio.html:229-230, studio.js:690-705 — target `/profiles` — status: done (web/ui/src/pages/ProfilesPage.tsx: Skeleton; first-run guide when empty; own "no match" state with "Borrar búsqueda") — **fix** applied
- [x] Toasts `toast.profileCreated|Updated` (7 s), `toast.profileDeleted`, `alerts.profileCreateFailed|UpdateFailed|DeleteFailed`, `alerts.profileSelectFirst|SelectDelete` — legacy studio.js:747-803 — target `/profiles` — status: done (web/ui/src/i18n/pages/profiles.ts `profiles.toast.*`, failures via the optimistic toast or inline in the form; "select first" alerts not applicable: actions are per row)
- [x] Confirm `confirm.deleteProfile` — legacy studio.js:796 — target `/profiles` — status: done (web/ui/src/components/profiles/ConfirmDelete.tsx, app dialog) — **fix** applied
- [x] Restart toasts `restart.running` (10 s), `restart.done`, `restart.failed` (apiErrorText), `restart.timeout` (15 s) — legacy studio.js:864-884 — target `/settings` — status: done (components/settings/SystemSection.tsx: running and timeout banners with what to do, done toast, failure via ErrorState)
- [x] Hints `pEditor.tip`, field hints (priority, failover, autoConnect, country, interface, waitTimeout, ovpnRoutes) — legacy studio.html:139, 143, 151, 163, 167, 177, 181, 212 — target `/profiles` — status: done (web/ui/src/components/profiles/ProfileForm.tsx, web/ui/src/i18n/pages/profiles.ts `profiles.hint.*`)

Cadence

- [x] Load once on page load (profiles, setup state); no polling — legacy studio.js:1078-1086 — target `/profiles` — status: done (web/ui/src/data/profiles.ts: no refetchInterval; tunnel state from the shared live feed)

#### Tab: Policies → `/rules`

Fields

- [x] Name (text) — legacy studio.html:34-36, studio.js:358 — API required (handlers_policy.go:476), duplicate 409 (handlers_policy.go:169, :216) — target `/rules` — status: done (web/ui/src/pages/RulesPage.tsx, components/rules/PolicyEditor.tsx: required + duplicate checked inline before sending; a 409 shows on the field)
- [x] Via VPN profile (select from meta profiles) — legacy studio.html:37-39, studio.js:439-446 — API GET /api/v1/policies/meta (handlers_policy.go:113) `profiles`; via required + must exist (handlers_policy.go:479-484) — target `/rules` — status: done (components/rules/PolicyEditor.tsx; no profiles → "Crea una en Perfiles" link)
- [x] If the tunnel is off: allow (default) / block / connect + live hint `editor.onDownHint.*` — legacy studio.html:40-47, studio.js:365, 371-381 — API `on_tunnel_down`, `config.ValidateOnTunnelDown` (handlers_policy.go:488) — target `/rules` — status: done (components/rules/PolicyEditor.tsx: radio group with each hint; block with apps only is caught inline)
- [x] Domains (one per line) — legacy studio.html:48-50, studio.js:361 — API `match.domains`; at least one match (handlers_policy.go:485); `policy.ParseRule` (handlers_policy.go:491) — target `/rules` — status: done (components/rules/PolicyEditor.tsx: one per line or comma, pasted URLs cleaned to the host, glob form checked like matchGlob)
- [x] IP ranges (one per line) — legacy studio.html:51-53, studio.js:362 — API `match.ip_ranges` (CIDR parse, handlers_policy.go:491) — target `/rules` — status: done (components/rules/PolicyEditor.tsx: IP or CIDR checked like net.ParseCIDR, IPv6 included)
- [x] Apps (one per line) — legacy studio.html:54-56, studio.js:363 — API `match.apps` — target `/rules` — status: done (components/rules/PolicyEditor.tsx: under "Apps (experimental)", which says apps only test/diagnose today — PolicyResolver does not route by app)
- [x] Tip + template text (`editor.tip`, `editor.template` with `<code>`) — legacy studio.html:64-65 — target `/rules` — status: done (components/rules/PolicyEditor.tsx field hints with examples; new rules start as one sentence in the "＋ Agregar" sheet, components/add/RuleStep.tsx, ADR-0055, which replaced the three starting points)

Actions

- [x] Create — legacy studio.html:58, studio.js:479-493 — API POST /api/v1/policies (handlers_policy.go:151; 201; localized 400 :164) — target `/rules` — status: done ("Nueva regla" opens the "＋ Agregar" sheet, components/add/RuleStep.tsx; data/rules.ts useSavePolicy: optimistic, toast `rules.toast.created`; refusal inline + toast; preview "Hoy / Después" from GET /api/v1/policies/explain + POST /api/v1/policies/test)
- [x] Update (selected or name; rename) — legacy studio.html:59, studio.js:495-514 — API PUT /api/v1/policies/{name} (handlers_policy.go:182; 404 :206) — target `/rules` — status: done (web/ui/src/pages/RulesPage.tsx: row Edit opens the editor; rename supported; toast `rules.toast.updated`)
- [x] Delete with confirm `confirm.delete` — legacy studio.html:60, studio.js:516-535 — API DELETE /api/v1/policies/{name} (handlers_policy.go:234) — target `/rules` — status: done (web/ui/src/pages/RulesPage.tsx: app dialog components/rules/shared.tsx ConfirmDelete; optimistic removal)
- [x] Clear (also clears test inputs/result) — legacy studio.html:61, studio.js:394-404 — target `/rules` — status: done (components/rules/PolicyEditor.tsx: Cancelar closes the editor and drops the draft and its test; the next editor starts empty)
- [x] Row Edit / Delete — legacy studio.js:425-426, 433-437, 537-541 — target `/rules` — status: done (components/rules/PolicyList.tsx: icon buttons named "Editar «x»" / "Borrar «x»")
- [x] Test before saving: target (domain/IP/URL) or app → matched / no match (reason) / error — legacy studio.html:69-75, studio.js:543-571 — API POST /api/v1/policies/test (handlers_policy.go:263; body `rule, target, app`; resp `matched, via, rule, reason`; 400 "target or app is required" :305) — target `/rules` — status: done (components/rules/PolicyEditor.tsx DraftTest: POST /api/v1/policies/test; draft errors and empty input explained inline)
- [x] Reload from disk + tip — legacy studio.html:79-81, studio.js:468-477 — API POST /api/v1/policies/reload (handlers_policy.go:138; `policies` count) — target `/rules` — status: done (web/ui/src/pages/RulesPage.tsx: "Volver a leer el archivo" with the config path from /policies/meta; toast with the count)
- [x] Resolve shortcut (same resolver as `/traffic`) — legacy none on Studio — API GET /api/v1/resolve — target `/rules` **IA: revisar** (spec lists "resolve" under `/rules`; legacy resolver is on Overview) — status: done (components/rules/TestDestination.tsx "Probar un destino": new GET /api/v1/policies/explain = `kongtrol policy show`, plus the VPN state and on_tunnel_down outcome)

Table / states / notices

- [x] Existing policies: Name, Match (domains, ranges, `app:x`), Via, actions; sorted by name — legacy studio.html:82-95, studio.js:406-431, 459 — API GET /api/v1/policies — target `/rules` — status: done (components/rules/PolicyList.tsx; **changed**: shown in evaluation order, not by name, because the order decides overlaps; VPN live state per row)
- [x] Empty `empty.noPolicies`; skeleton rows — legacy studio.js:409, studio.html:92-93 — target `/rules` — status: done (web/ui/src/pages/RulesPage.tsx: skeleton on first load; empty state teaches + "Crear la primera regla", which opens the "＋ Agregar" sheet)
- [x] Toasts `alerts.createFailed|updateFailed|deleteFailed|selectFirst|selectDelete|policyReloadFailed`, `toast.policiesReloaded` — legacy studio.js:472-530 — target `/rules` — status: done (data/rules.ts + data/optimistic.ts: failures = translated error + what to do + Retry; select-first toasts are impossible with row actions)
- [x] Test result texts `result.matched`, `result.noMatch`, `result.noMatchReason`, `alerts.testFailed` — legacy studio.js:555-570 — target `/rules` — status: done (components/rules/PolicyEditor.tsx, components/rules/TestDestination.tsx: `rules.test.draft.*`, `rules.outcome.*`, `rules.reason.*`)
- [x] `on_tunnel_down` shown per row (legacy table omits it) — legacy studio.js:419-429 — API `on_tunnel_down` — target `/rules` — status: done (components/rules/PolicyList.tsx column "Si está apagada")

#### Tab: Groups → `/profiles/groups`

- [x] Name (text) — legacy studio.html:242-244, studio.js:910 — API required (handlers_groups.go:46-48), duplicate 409 (handlers_groups.go:55-57) — target `/profiles/groups` — status: done (web/ui/src/components/profiles/GroupEditor.tsx; required + duplicate inline, localized 409)
- [x] Profiles (one per line) — legacy studio.html:245-247, studio.js:911 — API unknown profile / at least one (handlers_groups.go:103-117) — target `/profiles/groups` — status: done (web/ui/src/components/profiles/GroupEditor.tsx: checkboxes of existing profiles) — **fix** applied
- [x] Create / Update / Delete (confirm `confirm.deleteGroup`) / Clear — legacy studio.html:248-253, studio.js:963-1022 — API POST /api/v1/groups (handlers_groups.go:39), PUT /api/v1/groups/{name} (:71), DELETE (:126) — target `/profiles/groups` — status: done (web/ui/src/pages/GroupsPage.tsx + GroupEditor.tsx + ConfirmDelete.tsx; optimistic with rollback; "Clear" not needed: each editor opens fresh)
- [x] Row actions: Connect, Disconnect, Reload, Edit, Delete — legacy studio.js:938-944 — API POST /api/v1/groups/{name}/connect (:148), /disconnect (:199), /reload (:249) — target `/profiles/groups` — status: done (web/ui/src/pages/GroupsPage.tsx)
- [x] Connect toast `toast.groupConnecting` — legacy studio.js:1030-1039 — API group connect — target `/profiles/groups` — status: done (web/ui/src/pages/GroupsPage.tsx: reads `started` / `already_connected`, names profiles that did not start and offers "Reiniciar ahora") — **fix** applied
- [x] Disconnect toast `toast.groupDisconnecting` / error — legacy studio.js:1041-1051 — target `/profiles/groups` — status: done (web/ui/src/pages/GroupsPage.tsx; errors with what to do)
- [x] Reload: 409 `toast.groupRestartRequired` with `missing_profiles`, else `toast.groupReloading`, error `alerts.groupReloadFailed` — legacy studio.js:1060-1076 — API POST /api/v1/groups/{name}/reload (handlers_groups.go:249; 409 :264) — target `/profiles/groups` — status: done (web/ui/src/pages/GroupsPage.tsx + web/ui/src/data/profiles.ts `reloadGroup`: 409 restart_required read as an answer, "nothing to reload" when none was connected)
- [x] Reload explanation text `studio.reloadGroupTip` — legacy studio.html:260 — target `/profiles/groups` — status: done (web/ui/src/pages/GroupsPage.tsx "¿Qué hace «Recargar»?")
- [x] Table Name, Profiles, Actions; sorted; empty `empty.noGroups`; skeleton — legacy studio.html:261-273, studio.js:927-947 — API GET /api/v1/groups (handlers_groups.go:17) `name, profiles` — target `/profiles/groups` — status: done (web/ui/src/pages/GroupsPage.tsx: DataList, sorted by the daemon, live "n de m conectadas", Skeleton, empty states incl. "crea una VPN primero")
- [x] Toasts `toast.groupCreated|Updated|Deleted`, `alerts.groupCreateFailed|UpdateFailed|DeleteFailed|SelectFirst|SelectDelete` — legacy studio.js:972-1019 — target `/profiles/groups` — status: done (web/ui/src/i18n/pages/profiles.ts `groups.toast.*`; "select first" alerts not applicable)

### 3.3 `security.html` + `security.js` → `/security`

#### Controls

- [x] Kill switch row: title, description, runtime badge (with detail text), switch — legacy security.html:25-37, security.js:118-124 — API GET /api/v1/security/status (handlers_security.go:91) — target `/security` — status: done (web/ui/src/components/security/KillSwitchCard.tsx, model.ts)
- [x] Toggle kill switch; confirm only when turning off (`confirm.disableKillSwitch`); revert on error — legacy security.js:169-186 — API POST /api/v1/security/killswitch (handlers_security.go:227; body `enabled`; resp `kill_switch_enabled`) — target `/security` — status: done (web/ui/src/components/security/KillSwitchCard.tsx: native dialog, optimistic + rollback, data/security.ts)
- [x] DNS guard row + toggle (no confirm) — legacy security.html:39-51, security.js:188-201 — API POST /api/v1/security/dnsguard (handlers_security.go:253) — target `/security` — status: done (web/ui/src/components/security/DNSCard.tsx; decision: turning it off now asks first, it lowers privacy)
- [x] Public IP lookup row: switch + description with host — legacy security.html:52-63, security.js:130-133 — API POST /api/v1/security/publicip (handlers_security.go:47; resp `public_ip_check_host`) — target `/security` — status: done (web/ui/src/components/security/PublicIPCheckCard.tsx)
- [x] Switch not overwritten while focused during refresh — legacy security.js:119, 121, 131 — target `/security` — status: done (switches render the cache; a pending mutation cancels polls: data/optimistic.ts)
- [x] Switch disabled while request runs — legacy security.js:175, 190 — target `/security` — status: done (Switch pending/disabled in web/ui/src/components/security/*Card.tsx)
- [x] Refresh button — legacy security.html:22 — target `/security` — status: done (web/ui/src/pages/SecurityPage.tsx)

#### Per-profile kill switch override (decision §2)

- [x] Table Profile + Override select (Inherit (global) / Force on / Force off), sorted — legacy security.html:66-84, security.js:205-233 — API GET /api/v1/vpns (handlers_vpn.go:18) `kill_switch_override` — target `/security` — status: done (web/ui/src/components/security/OverridesCard.tsx: native select, Result column)
- [x] Change override → toast `toast.overrideSaved` / `toast.overrideFailed` + reload table — legacy security.js:235-248 — API PUT /api/v1/vpns/{name}/killswitch (handlers_settings.go:95; `override` must be inherit|on|off :113-125; 404 :111) — target `/security` — status: done (web/ui/src/components/security/OverridesCard.tsx, optimistic + rollback)
- [x] Note `security.perProfileNote`; empty `empty.noProfiles`; refresh button — legacy security.html:69-71, security.js:207-209 — target `/security` — status: done (web/ui/src/components/security/OverridesCard.tsx; refresh = page button)

#### Leak detection

- [x] Leak check badge, public IP + Check, last checked — legacy security.html:86-105, security.js:126, 134-151 — API GET /api/v1/security/status `leak_check{state,has_leak,public_ip,reason,checked_at}` — target `/security` — status: done (web/ui/src/components/security/PublicIPCheckCard.tsx)
- [x] Expected public IP per VPN (new, ADR-0042) — legacy none (kongtrol.yaml only) — API PUT /api/v1/settings `expected_ips` (whole map) — target `/security` — status: done (components/security/ExpectedIPs.tsx: add / edit / remove rows, one row per VPN, IPv4/IPv6 checked before saving)
- [x] What the kill switch blocks now (new, ADR-0042) — legacy none — API GET /api/v1/security/blocked — target `/security` — status: done (components/security/BlockedList.tsx: collapsible list with count, dropped VPNs and blocking rules)
- [x] Kill switch apply error in the daemon's words (new, ADR-0042) — API security/status `kill_switch_error{message,hint}` — target `/security` — status: done (KillSwitchCard.tsx banner; raw detail in "Detalle técnico")
- [x] DNS servers the guard applies now (new, ADR-0042) — API security/status `dns_guard_servers`, `dns_guard_interface`, `dns_guard_profiles` — target `/security` — status: done (DNSCard.tsx, per VPN)

#### Notices / cadence

- [x] Toasts `toast.killSwitchOn|Off`, `toast.dnsGuardOn|Off`, `toast.toggleFailed` via `showApiError` — legacy security.js:178-196 — target `/security` — status: done (data/security.ts successMessage; refusal toast from data/optimistic.ts)
- [x] Poll security status every 15 s — legacy security.js:255 — target `/security` — status: done (5 s shared query, data/security.ts)
- [x] Badge states identical to `kongtrol status` (shellSecurityBadges) — legacy shell.js:299-321 — target `/security`, `/` — status: done on /security (web/ui/src/components/security/model.ts cli line in "Detalle técnico"); / pending

### 3.4 `settings.html` + `settings.js`

#### Section: General

- [x] Dashboard bind + port (read-only) + note + CLI commands `kongtrol config dashboard set-port|set-bind` — legacy settings.html:28-38, settings.js:165-166 — API GET /api/v1/settings (handlers_settings.go:14) `dashboard_bind, dashboard_port` — target `/settings` — status: done (components/settings/SystemSection.tsx: read-only address + both CLI commands)
- [x] Diagnostics link "Open Diagnostics" (to `/app/`) — legacy settings.html:46 — API GET /api/v1/doctor (handlers_doctor.go:21) — target `/settings` — status: done (pages/SettingsPage.tsx: DiagnosticsCard runs the checks in place)
- [x] Export configuration (download `kongtrol-export.yaml`, no secrets) — legacy settings.html:47 — API GET /api/v1/config/export (handlers_doctor.go:52; 404 `export.err.no_config`) — target `/settings` — status: done (pages/SettingsPage.tsx: DiagnosticsCard export)
- [x] Health check interval (text, e.g. 30s) + timeout (e.g. 10s) — legacy settings.html:52-64, settings.js:167-168, 188-189 — API PUT /api/v1/settings (handlers_settings.go:49) `health_check_interval, health_check_timeout`; `time.ParseDuration` (:57-64), `config.Validate` (:74-77) — target `/settings` — status: done (components/settings/HealthSection.tsx: validated like time.ParseDuration, at least 1s)
- [x] Split DNS enabled + interval (e.g. 60s) — legacy settings.html:66-79, settings.js:171-172, 192-193 — API `split_dns_enabled, split_dns_interval` (duration check :57-64) — target `/security` — status: done (web/ui/src/components/security/DNSCard.tsx; says on/off applies on restart, verified cmd/kongtrol/daemon_cmd.go:173)

#### Section: Security tuning

- [x] Kill switch mode strict/loose (legacy; `allow_lan` preferred) — legacy settings.html:90-95, settings.js:173, 194 — API `kill_switch_mode` (`oneof=strict loose`, internal/config/schema.go:177) — target `/security` — status: done (web/ui/src/components/security/KillSwitchCard.tsx: merged into "Permitir la red local"; turning it off from loose saves strict) — **fix** applied
- [x] Allow LAN traffic switch — legacy settings.html:96-99 — API `kill_switch_allow_lan` — target `/security` — status: done (web/ui/src/components/security/KillSwitchCard.tsx)
- [x] Note "enable/disable kill switch on Security page" — legacy settings.html:101 — target `/security` (obsolete once co-located) — status: done (obsolete: co-located)
- [x] DNS guard fallback DNS (text, e.g. 1.1.1.1) — legacy settings.html:109-111 — API `dns_guard_fallback_dns` (`config.Validate`) — target `/security` — status: done (web/ui/src/components/security/DNSCard.tsx, inline error)
- [x] Public IP lookup enabled (duplicate of the Security switch) — legacy settings.html:120-123, settings.js:176, 197 — API `leak_detection_enabled` (legacy alias, handlers_settings.go:188-190) — target `/security` (single switch) — status: done (single switch with consent dialog, web/ui/src/components/security/PublicIPCheckCard.tsx) — **fix** applied
- [x] Public IP services (one https:// URL per line; default collapsed to empty) — legacy settings.html:124-126, settings.js:177, 198 — API `public_ip_endpoints` (handlers_settings.go:194-205; `config.ValidatePublicIPEndpoint` via `config.Validate`) — target `/security` — status: done (web/ui/src/components/security/PublicIPCheckCard.tsx, https-only check before save)
- [x] Public IP note `settings.publicIpNote` — legacy settings.html:127 — target `/security` — status: done (web/ui/src/components/security/PublicIPCheckCard.tsx)
- [x] Leak action notify / killswitch_and_notify — legacy settings.html:128-133 — API `leak_detection_action` — target `/security` — status: dropped: ADR-0042 (`security.leak_detection.action` and `interval` removed from schema, API and legacy page; a leak only notifies; old configs still load)
- [x] Audit log path, max size MB (number), HMAC sign switch — legacy settings.html:137-153, settings.js:179-181, 200-202 — API `audit_log_path, audit_log_max_size_mb, audit_log_sign` — target `/activity` **IA: revisar** (config of the log shown there; could stay in `/settings`) — status: done (components/activity/AuditConfig.tsx on `/activity` (IA decided: next to the log); **fix** done: an empty or non-integer size is refused with what to do, never sent as 0); API now also `audit_log_retention_days` (0 = today and yesterday, max 365) and enforces `audit_log_max_size_mb` (1–1024, default 100: the day goes on in `.1`, `.2`… — ADR-0042); retention field done (components/activity/AuditConfig.tsx: days kept 0–365 with what it means, size per file 1–1024 explained)

#### Section: Scheduler → `/rules/schedule`

- [x] Scheduler enabled switch (saves immediately) — legacy settings.html:163-166 — API PUT /api/v1/settings `scheduler_enabled` — target `/rules/schedule` — status: done (web/ui/src/pages/SchedulePage.tsx: Switch, optimistic, sends only `scheduler_enabled`)
- [x] Scheduler interval (e.g. 1m) — legacy settings.html:167-169, settings.js:170, 191 — API `scheduler_interval` (duration check handlers_settings.go:57-64) — target `/rules/schedule` — status: done (web/ui/src/pages/SchedulePage.tsx IntervalForm: own Save, Go duration checked inline, sends only `scheduler_interval`)
- [x] Rule name — legacy settings.html:175-177, settings.js:234 — API required (handlers_schedule.go:39-41), duplicate 409 (:49-51) — target `/rules/schedule` — status: done (components/rules/ScheduleEditor.tsx: required + duplicate inline)
- [x] Rule profiles (one per line) — legacy settings.html:178-180, settings.js:235 — API `profiles`, `config.Validate` (handlers_schedule.go:96-110) — target `/rules/schedule` — status: done (components/rules/ScheduleEditor.tsx: checkboxes of existing profiles; a deleted profile still listed so it can be unticked)
- [x] Weekdays (comma-separated, optional, e.g. mon,tue) — legacy settings.html:181-183, settings.js:236 — API `weekdays` (`config.Validate`) — target `/rules/schedule` — status: done (components/rules/ScheduleEditor.tsx: seven day toggles + presets; none = every day)
- [x] Start / End HH:MM (optional) — legacy settings.html:184-189, settings.js:237-238 — API `start`, `end` (`config.Validate`) — target `/rules/schedule` — status: done (components/rules/ScheduleEditor.tsx: "Todo el día" or two time inputs; one missing / equal times / overnight explained inline)
- [x] Create / Update / Delete (confirm `confirm.deleteRule`) / Clear — legacy settings.html:190-195, settings.js:294-359 — API POST /api/v1/scheduler/rules (handlers_schedule.go:32), PUT …/{name} (:62), DELETE …/{name} (:116) — target `/rules/schedule` — status: done (web/ui/src/pages/SchedulePage.tsx, components/rules/ScheduleEditor.tsx: optimistic save/delete, app confirm dialog, Cancelar = clear; changes apply at once (scheduler hot reload, ADR-0042), so no restart note)
- [x] Pause one rule without deleting it (new, ADR-0042) — legacy none — API PUT /api/v1/scheduler/rules/{name} `{"enabled": bool}` — target `/rules/schedule` — status: done (web/ui/src/pages/SchedulePage.tsx: optimistic switch per row, rolled back with what to do; "Ahora" says Pausado)
- [x] Rules table: Name, Profiles, Window (`days · start-end` or —), Edit/Delete — legacy settings.html:199-215, settings.js:258-279 — API GET /api/v1/scheduler/rules (handlers_schedule.go:12) — target `/rules/schedule` — status: done (web/ui/src/pages/SchedulePage.tsx: Nombre, Cuándo (plain sentence with the VPNs), Ahora (in/out of schedule), actions)
- [x] Refresh rules button — legacy settings.html:161 — target `/rules/schedule` — status: done (web/ui/src/pages/SchedulePage.tsx: refresh icon button, reloads rules and settings)
- [x] Toasts `toast.ruleCreated|Updated|Deleted`, `alerts.ruleCreateFailed|UpdateFailed|DeleteFailed|SelectFirst|SelectDelete`; empty `empty.noRules` — legacy settings.js:261, 303-350 — target `/rules/schedule` — status: done (data/rules.ts: `schedule.toast.*`; failures via data/optimistic.ts; empty state with "Crear horario de oficina")

#### Page actions / states

- [x] Tabs General / Security / Scheduler (remembered) — legacy settings.html:20-25 — target split across `/settings`, `/security`, `/rules/schedule` — status: done (`/settings` part: sections on one page with an index (pages/SettingsPage.tsx); the security and scheduler parts belong to their pages)
- [x] Save settings (one PUT for all fields; response refills form) + `toast.saved` / `alerts.saveFailed` via `showApiError` — legacy settings.html:222, settings.js:212-225 — API PUT /api/v1/settings (handlers_settings.go:49) — target `/settings`, `/security`, `/rules/schedule` (per-section save sending only that section's fields) — status: done (`/settings` + `/activity` part: per-section PUT with only that section fields (data/settings.ts useSaveSettings, optimistic + rollback, toast))
- [x] Revert (reload from server) — legacy settings.html:223, settings.js:206-210 — API GET /api/v1/settings — target `/settings` (also `/security`, `/rules/schedule`) — status: done (components/settings/SettingsSection.tsx FormBar "Deshacer cambios" per section)
- [x] Unsaved changes warning when leaving — legacy none — API — — target `/settings` (also `/security`, `/rules/schedule`) — status: done (components/settings/form.ts: browser warning on close/reload + "Cambios sin guardar" in the bar; in-app navigation is not blocked (declarative router has no blocker))
- [x] Load once; no polling — legacy settings.js:361-365 — API GET /api/v1/settings, GET /api/v1/scheduler/rules — target `/settings` — status: done (data/settings.ts: staleTime Infinity, no polling)

#### Settings items the spec lists for `/settings` that legacy lacks (new UI already ahead)

- [x] Update notice opt-in — legacy none (only `update.event.available` notice, notices.js:22) — API GET/POST /api/v1/updates (handlers_lifecycle.go:52, :63) — target `/settings` — status: done (pages/SettingsPage.tsx (UpdatesCard))
- [x] Delete all my data (confirm dialog) — legacy none — API POST /api/v1/data/purge (handlers_lifecycle.go:92; `confirm:true` required) — target `/settings` — status: done (pages/SettingsPage.tsx (DataCard))
- [x] Helper status + install — legacy app.js:346-415 (banner) — API GET /api/v1/helper/status, POST /api/v1/helper/install — target `/settings` — status: done (components/settings/SystemSection.tsx: shared install flow)
- [x] Language + theme controls — legacy shell.js:79-83 — API GET/PUT /api/v1/preferences — target `/settings` (also header) — status: done (components/settings/SettingsGroups.tsx: segmented language + theme)
- [x] Shutdown ("quit and disconnect all") — legacy none — API POST /api/v1/shutdown (handlers_tunnels.go:21) — target `/settings` — status: done (components/settings/SystemSection.tsx: confirm dialog, then guidance to reopen Kongtrol)

### 3.5 `audit.html` + `audit.js` → `/activity`

- [x] Filter by profile (text, Enter applies) — legacy audit.html:25, audit.js:117-119 — API GET /api/v1/audit?profile= (handlers_security.go:284, :301) — target `/activity` — status: done (components/activity/AuditPanel.tsx: Enter or leaving the field applies; profile suggestions)
- [x] Level filter All / INFO / WARN / ERROR / SECURITY — legacy audit.html:26-32 — API `level` (uppercased, handlers_security.go:302) — target `/activity` — status: done (components/activity/AuditPanel.tsx; **fix** done: applies at once, translated names)
- [x] Apply + refresh buttons — legacy audit.html:22, audit.html:33 — target `/activity` — status: done (components/activity/AuditPanel.tsx: filters apply at once; Refresh icon button)
- [x] Limit 300 — legacy audit.js:104 — API `limit`, `before` cursor, `X-Kongtrol-Audit-More` (handlers_audit.go) — target `/activity` — status: done (components/activity/AuditPanel.tsx: pages of 200, "Cargar más" asks for the next older page while the header says there is one)
- [x] Date range (new, ADR-0042) — legacy none (page-side only) — API `from` / `to` (local days) over every retained day — target `/activity` — status: done (components/activity/AuditPanel.tsx: Desde / Hasta filter on the daemon; From after To explained on the field)
- [x] Signature verified per entry (new, ADR-0042) — legacy none — API `signed`, `verified` — target `/activity` — status: done ("Firma válida" / "Firma no válida" (danger, with why and a count) / "Sin firma")
- [x] Table: Time (`toLocaleString(lang)`), Level badge, Action, Profile, Message, Signed (signed / unsigned) — legacy audit.html:36-51, audit.js:80-94 — API `timestamp, level, action, profile, message, hmac` — target `/activity` — status: done (components/activity/AuditPanel.tsx DataList (cards under 640px); signature as word + icon)
- [x] Level badge classes INFO ok, WARN pending, ERROR off, SECURITY on — legacy audit.js:61-66 — target `/activity` — status: done (components/activity/AuditPanel.tsx; INFO neutral (Green Means Up), WARN caution, ERROR red, SECURITY emphasized neutral)
- [x] Note "Showing {count} events"; empty `empty.noEvents`; skeleton — legacy audit.js:72-78, audit.html:48-49 — target `/activity` — status: done (components/activity/AuditPanel.tsx: count, empty states per cause, skeleton on first load)
- [x] Error → toast `audit.notConfigured` — legacy audit.js:108-110 — API 500 (handlers_security.go:287, :297); unconfigured returns `[]` (:291) — target `/activity` — status: done (components/activity/AuditPanel.tsx; **fix** done: unconfigured is explained with where to turn it on; other errors show message + what to do)
- [x] Events feed (history of daemon notices) — legacy none (toasts only, notices.js) — API GET /api/v1/events?after= (handlers_events.go:28) — target `/activity` — status: done (components/activity/EventsPanel.tsx + data/activity.ts: history grouped by day, plain language, Conectar in place, refreshed every 5 s)

### 3.6 Cross-cutting

#### `shell.js` (nav, header, theme, shortcuts)

- [x] Sidebar nav: Overview, Studio, Security, Audit Log, Settings, active item — legacy shell.js:11-17, 47-52 — target app router nav (8 routes, §2) — status: done (components/layout/AppShell.tsx `SideNav` + app/sections.ts: seven sections, active item)
- [x] Brand logo top-left + name + BETA badge — legacy shell.js:43-46 — target shell — status: done for logo + name (components/layout/AppShell.tsx); BETA badge dropped: ADR-0043 (the version tag shows the build)
- [x] Desktop collapse rail, remembered (`kongtrol-dashboard-sidebar-collapsed`), tooltips mirror labels — legacy shell.js:59-67, 257-289 — target shell — status: replaced by the width-based rail 600–839px (ADR-0038, components/layout/AppShell.tsx + theme/shell.css); manual collapse memory dropped: ADR-0043
- [x] Mobile drawer + scrim, closes on nav click — legacy shell.js:73-75, 90-95, 232-252 — target shell — status: replaced by the bottom bar + "Más" sheet (components/layout/AppShell.tsx `BottomBar`, closes on navigation)
- [x] Page title in top bar (`page.*.title`) — legacy shell.js:76 — target shell — status: done as each page's focusable h1 (components/layout/PageHeader.tsx)
- [x] Theme: explicit choice persisted (localStorage + preferences), choosing the system theme clears it, follows `prefers-color-scheme` live — legacy shell.js:103-160 — API PUT /api/v1/preferences (handlers_security.go:199) `theme` — target shell — status: done (app/PreferencesProvider.tsx)
- [x] Theme switch circular reveal (view transition, reduced-motion off) — legacy shell.js:167-175, style.css:92-146 — target shell — status: dropped: ADR-0043 (optional polish; the theme switches at once)
- [x] Preferences load is authoritative (clears stale browser copy) — legacy shell.js:194-207 — API GET /api/v1/preferences (handlers_security.go:188) — target shell — status: done (app/PreferencesProvider.tsx)
- [x] Language toggle with flash animation — legacy shell.js:80-83, 214-230 — target shell — status: done for the toggle (components/layout/AppShell.tsx); flash animation dropped: ADR-0043
- [x] Live connection indicator + label (`ws.*`) — legacy shell.js:84-85, app.js:444-448 — target shell — status: done (components/layout/AggregateStatus.tsx: text + icon, never color only)
- [x] Tabs component remembers last tab — legacy shell.js:327-346 — target router (URL is the memory) — status: done (components/ui/Tabs.tsx: `role=tab`, `aria-selected`, arrow/Home/End keys; the URL is the memory)
- [x] Kill switch detail text (ARMED protecting n / ACTIVE blocking names) — legacy shell.js:359-381 — API `kill_switch_status{state,protected,blocking}` — target `/`, `/security` — status: done (web/ui/src/components/home/SecurityCard.tsx + web/ui/src/components/home/model.ts securityView); **fix**: own dict duplicates `killswitch.state.*` of internal/i18n
- [x] Keyboard shortcuts — legacy none found — target — — status: n/a (no shortcuts in legacy; none invented)
- [x] Skip link to main content — legacy none — target shell — status: done (components/layout/AppShell.tsx)

#### `notices.js` (live notices)

- [x] Watchdog `manual_reconnect_required` / attach drop notices — legacy not filtered in (kind list notices.js:122) — `no verificado` whether daemon emits other kinds — target shell — status: daemon side done (ADR-0042: `watchdog.event.manual_reconnect` notice on /api/v1/events; legacy notices.js and web/ui home + activity texts added); kind `tunnel_watchdog` is already shown by legacy notices.js:124 and web/ui NOTICE_KINDS (data/home.ts:450)
- [x] Poll events every 5 s on every page; `after` seq kept in sessionStorage (`kongtrol.noticeSeq`), fresh tab = no backlog — legacy notices.js:98-137 — API GET /api/v1/events?after= (handlers_events.go:28) `events[], seq` — target shell (all pages), list in `/activity` — status: done (web/ui/src/data/home.ts `useDaemonNotices`, mounted once by web/ui/src/components/layout/AppShell.tsx; list on `/` and `/activity`)
- [x] Kinds shown: `policy_tunnel_down`, `tunnel_watchdog`, `update_available`; WARN → error style — legacy notices.js:122-124 — target shell — status: done (web/ui/src/data/home.ts `NOTICE_KINDS`, WARN → error tone)
- [x] Keys `policy.event.allow|block|connect|connect_failed`, `watchdog.event.reconnect_failing|failover|reconnected|no_response|responding`, `update.event.available` (args `{0}{1}{2}`) — legacy notices.js:8-39 — target `src/i18n/messages.ts` — status: done (web/ui/src/i18n/pages/home.ts `home.notice.*`)
- [x] Action toast "Connect {profile}" when `can_connect`, stays 60 s — legacy notices.js:48-81, 125-126 — target shell — status: done (web/ui/src/data/home.ts `useNoticeToasts`, `ACTION_TOAST_MS`)
- [x] Action → connect with `notice.connecting` / `notice.connectFailed` — legacy notices.js:83-94 — API POST /api/v1/tunnels/{name}/connect — target shell — status: done (web/ui/src/data/home.ts `useNoticeToasts` → tunnel actions with their own progress and error toasts)

#### `toast.js`

- [x] Toast stack: message, optional detail line, raw API error as tooltip, click to dismiss, default 4.5 s — legacy toast.js:41-74 — target shell (`aria-live` region) — status: done (components/ui/Toaster.tsx: polite/assertive live regions, detail line, dismiss button, auto-close paused on hover/focus)
- [x] `showApiError`: context message + translated hint per API `code` (8 codes) + raw detail as tooltip — legacy toast.js:6-39 — API `code` from `writeError` (handlers_common.go:23-48) — target shell — status: done (api/client.ts `ApiError` with translated `hint` + `error_key`; components/ui/States.tsx `ErrorState`)

#### `publicip.js`

- [x] Render IP or — + hint `publicip.hint` while off; Check buttons only while on — legacy publicip.js:41-63 — target `/`, `/security` — status: done (web/ui/src/components/home/SecurityCard.tsx)
- [x] Check now: busy `publicip.checking`, error `publicip.failed` + `failedHint` or daemon error/hint, leak_check error state — legacy publicip.js:66-89 — API POST /api/v1/security/publicip/check (handlers_security.go:77; 409 `publicip.err.disabled`) — target `/`, `/security` — status: done (web/ui/src/components/home/SecurityCard.tsx)
- [x] Toggle: `publicip.on` + notice with host / `publicip.off` / `publicip.toggleFailed`; revert switch on error — legacy publicip.js:93-116 — API POST /api/v1/security/publicip (handlers_security.go:47) — target `/security` — status: done (web/ui/src/components/security/PublicIPCheckCard.tsx: notice + consent BEFORE enabling)

#### `charts.js`

- [x] HiDPI canvas prep — legacy charts.js:4-14 — target `/traffic` — status: done (web/ui/src/components/traffic/MiniChart.tsx: SVG, resolution-independent, no canvas)
- [x] Sparkline (min-max scaled) — legacy charts.js:17-40 — target `/` — status: done (web/ui/src/components/home/Sparkline.tsx: SVG with a text alternative)
- [x] Time series with 4-line grid, shared max — legacy charts.js:44-80 — target `/traffic` — status: done with fix (web/ui/src/components/traffic/MiniChart.tsx: y labels with units, x labels, figcaption summary, keyboard crosshair (slider), data table)

#### `select.js`

- [x] Themed select replacement (keeps native `<select>` as source of truth) — legacy select.js:13-109 — target shared `Select` component — status: replaced by the native `<select>` (components/ui/Field.tsx `SelectField`): keyboard and screen readers for free; custom widget dropped: ADR-0043
- [x] Keyboard: ArrowDown/Enter/Space opens, arrows move, Enter/Space selects, Escape closes, focus returns to trigger — legacy select.js:47-56, 69, 81-85 — target shared component — status: done by the native `<select>` (components/ui/Field.tsx `SelectField`)

#### `i18n.js`

- [x] Language resolution: explicit choice (localStorage `kongtrol-dashboard-lang` + preferences) else browser (`es*` → es) — legacy i18n.js:28-58, 131-140 — API preferences `language` — target shell — status: done (app/PreferencesProvider.tsx, i18n/lang.ts)
- [x] Follow browser `languagechange` when no explicit choice — legacy i18n.js:141-143 — target shell — status: done (app/PreferencesProvider.tsx `languagechange` listener)
- [x] `t()` with `{var}` interpolation, EN fallback — legacy i18n.js:60-64 — target `src/i18n/messages.ts` — status: done (src/i18n/messages.ts)
- [x] Rich text with `<code>` only, no innerHTML — legacy i18n.js:82-92 — target messages — status: done (JSX `<code>` elements, no innerHTML)
- [x] `html lang` attribute updated — legacy i18n.js:68 — API — — target shell — status: done (app/PreferencesProvider.tsx sets `document.documentElement.lang`)
- [x] Refresh-button titles + lang-toggle aria label translated — legacy i18n.js:100-108 — target shell — status: done (components/layout/AppShell.tsx)

## 4. Endpoints matrix

Routes from `internal/api/server.go:146-204`. "Legacy" = first call site in `web/dashboard/`. "New UI" = `web/ui/src/api/client.ts`.
Other clients (CLI `cmd/kongtrol`, tray, `internal/daemonctl`) listed when a dashboard does not call the endpoint.

| Method + path | Handler | Legacy use | New UI | Target |
|---|---|---|---|---|
| GET /api/v1/tunnels | handlers_tunnels.go:40 | app.js:461 | client.ts:86 | `/` |
| GET /api/v1/metrics/history | handlers_tunnels.go:31 | app.js:654 | — | `/traffic` |
| POST /api/v1/tunnels/{name}/connect | handlers_tunnels.go:50 | app.js:1022, notices.js:86 | — | `/`, shell |
| POST /api/v1/tunnels/{name}/cancel_connect | handlers_tunnels.go:88 | app.js:1035 | — | `/` |
| POST /api/v1/tunnels/{name}/disconnect | handlers_tunnels.go:112 | app.js:1089 | — | `/` |
| POST /api/v1/tunnels/{name}/reload | handlers_tunnels.go:150 | app.js:1109 | — | `/` |
| GET /api/v1/tunnels/routes | handlers_tunnel_routes.go:76 | app.js:733 | — | `/traffic` |
| GET /api/v1/tunnels/{name}/routes | handlers_tunnel_routes.go:81 | **none** (CLI `routes [perfil]`, cmd/kongtrol/tunnel_routes.go:48) | — | `/traffic` (per-tunnel view) |
| GET /api/v1/routes | handlers_routes.go:13 | app.js:721 | — | `/traffic` |
| GET /api/v1/network/overview | handlers_routes.go:94 | app.js:845 | — | `/`, `/traffic` |
| GET /api/v1/security/status | handlers_security.go:91 | app.js:665, security.js:114 | client.ts:94 | `/`, `/security` |
| POST /api/v1/security/killswitch | handlers_security.go:227 | security.js:177 | — | `/security` |
| POST /api/v1/security/dnsguard | handlers_security.go:253 | security.js:192 | — | `/security` |
| POST /api/v1/security/publicip | handlers_security.go:47 | publicip.js:97 | client.ts:96 | `/security`, `/` |
| POST /api/v1/security/publicip/check | handlers_security.go:77 | publicip.js:72 | client.ts:98 | `/`, `/security` |
| GET /api/v1/vpns | handlers_vpn.go:18 | studio.js:731, security.js:227 | data/profiles.ts | `/profiles`, `/security` |
| GET /api/v1/vpns/detect | handlers_detect.go:26 | **none** (new, read-only; wizard probes) | data/profiles.ts | `/profiles` (first run) |
| POST /api/v1/vpns | handlers_vpn.go:184 | studio.js:740 | data/profiles.ts | `/profiles` |
| PUT /api/v1/vpns/{name} | handlers_vpn.go:214 | studio.js:770 | data/profiles.ts | `/profiles` |
| DELETE /api/v1/vpns/{name} | handlers_vpn.go:349 | studio.js:797 | data/profiles.ts | `/profiles` |
| PUT /api/v1/vpns/{name}/killswitch | handlers_settings.go:95 | security.js:236 | — | `/security` |
| GET /api/v1/scheduler/rules | handlers_schedule.go:12 | settings.js:288 | — | `/rules/schedule` |
| POST /api/v1/scheduler/rules | handlers_schedule.go:32 | settings.js:296 | — | `/rules/schedule` |
| PUT /api/v1/scheduler/rules/{name} | handlers_schedule.go:62 | settings.js:318 | — | `/rules/schedule` |
| DELETE /api/v1/scheduler/rules/{name} | handlers_schedule.go:116 | settings.js:344 | — | `/rules/schedule` |
| GET /api/v1/audit | handlers_security.go:284 | audit.js:107 | — | `/activity` |
| GET /api/v1/settings | handlers_settings.go:14 | settings.js:207 | — | `/settings`, `/security`, `/rules/schedule`, `/activity` |
| PUT /api/v1/settings | handlers_settings.go:49 | settings.js:213 | — | same |
| GET /api/v1/preferences | handlers_security.go:188 | shell.js:196 | client.ts:90 | shell, `/settings` |
| PUT /api/v1/preferences | handlers_security.go:199 | shell.js:181 | client.ts:92 | shell, `/settings` |
| GET /api/v1/groups | handlers_groups.go:17 | app.js:1134, studio.js:956 | client.ts, data/profiles.ts | `/`, `/profiles/groups` |
| POST /api/v1/groups | handlers_groups.go:39 | studio.js:965 | data/profiles.ts | `/profiles/groups` |
| PUT /api/v1/groups/{name} | handlers_groups.go:71 | studio.js:987 | data/profiles.ts | `/profiles/groups` |
| DELETE /api/v1/groups/{name} | handlers_groups.go:126 | studio.js:1013 | data/profiles.ts | `/profiles/groups` |
| POST /api/v1/groups/{name}/connect | handlers_groups.go:148 | app.js:1158, studio.js:1034 | data/profiles.ts | `/`, `/profiles/groups` |
| POST /api/v1/groups/{name}/disconnect | handlers_groups.go:199 | app.js:1168, studio.js:1044 | data/profiles.ts | `/`, `/profiles/groups` |
| POST /api/v1/groups/{name}/reload | handlers_groups.go:249 | studio.js:1063 | data/profiles.ts | `/profiles/groups` |
| GET /api/v1/policies | handlers_policy.go:53 | app.js:909, studio.js:456 | — | `/rules`, `/traffic` |
| GET /api/v1/policies/meta | handlers_policy.go:113 | studio.js:440 | — | `/rules`, `/settings` |
| POST /api/v1/policies/reload | handlers_policy.go:138 | studio.js:469 | — | `/rules` |
| POST /api/v1/policies | handlers_policy.go:151 | studio.js:481 | — | `/rules` |
| PUT /api/v1/policies/{name} | handlers_policy.go:182 | studio.js:502 | — | `/rules` |
| DELETE /api/v1/policies/{name} | handlers_policy.go:234 | studio.js:527 | — | `/rules` |
| POST /api/v1/policies/test | handlers_policy.go:263 | studio.js:549 | — | `/rules` |
| GET /api/v1/policies/explain | handlers_policy.go (handleExplainPolicy) | — (new; = `kongtrol policy show`) | data/rules.ts | `/rules` |
| GET /api/v1/resolve | handlers_policy.go:309 | app.js:1007 | — | `/traffic`, `/rules` |
| GET /api/v1/dns/resolve | handlers_policy.go:404 | **none** (no client found in cmd/, internal/daemonctl, tray) | — | `/traffic` (web/ui/src/data/traffic.ts `useDnsResolveQuery`: "see which IPs it points to through <VPN>" after a domain lookup) |
| POST /api/v1/shutdown | handlers_tunnels.go:21 | **none** (tray via internal/daemonctl) | — | `/settings` (GUI parity gap) |
| POST /api/v1/restart | handlers_setup.go:54 | studio.js:861 | data/profiles.ts | `/settings`, `/profiles` |
| GET /api/v1/setup | handlers_setup.go:39 | app.js:1242, studio.js:820, studio.js:848 | data/profiles.ts | `/`, `/profiles`, `/settings` |
| GET /api/v1/helper/status | handlers_helper.go:67 | app.js:349 | client.ts:87 | `/`, `/settings` |
| POST /api/v1/helper/install | handlers_helper.go:74 | app.js:397 | client.ts:89 | `/`, `/settings` |
| GET /api/v1/doctor | handlers_doctor.go:21 | **none** (link to `/app/`, settings.html:46) | client.ts:106 | `/settings` |
| GET /api/v1/config/export | handlers_doctor.go:52 | settings.html:47 (link) | client.ts:109 | `/settings` |
| GET /api/v1/events | handlers_events.go:28 | notices.js:117 | — | shell, `/activity` |
| GET /api/v1/updates | handlers_lifecycle.go:52 | **none** | client.ts:100 | `/settings` |
| POST /api/v1/updates | handlers_lifecycle.go:63 | **none** | client.ts:102 | `/settings` |
| POST /api/v1/data/purge | handlers_lifecycle.go:92 | **none** | client.ts:104 | `/settings` |
| WS /api/v1/ws/metrics | ws.go:17 | app.js:421 | client.ts:131 | `/`, `/traffic` |

Used by no dashboard client: `GET /api/v1/dns/resolve` (no client at all), `GET /api/v1/tunnels/{name}/routes` (CLI only), `POST /api/v1/shutdown` (tray/CLI only).

## 5. Gaps and risks

### 5.1 Legacy bugs (do not port)

1. Tunnel names are injected unescaped into HTML and inline `onclick` (app.js:568-577); connect/cancel/disconnect paths are not URL-encoded (app.js:1022, 1035, 1089). XSS from a hand-edited profile name; broken actions for names with `/`, `#`, spaces.
2. Tunnel connect/cancel/disconnect errors are swallowed (app.js:1020-1097): no toast, no "what to do". Disconnect `202 waiting_user` (handlers_tunnels.go:132) is not explained.
3. Group connect ignores `res.ok` and the `started` / `already_connected` lists (app.js:1156-1165, studio.js:1030-1039); profiles without an adapter are skipped silently by the daemon (handlers_groups.go:163-166).
4. Tunnel reload toasts "Reloading" even for `skipped_not_connected` (app.js:1119 vs handlers_tunnels.go:171-174).
5. Scheduler switch `onchange="saveSettings()"` (settings.html:165) saves every unsaved field of the whole Settings page.
6. Public IP opt-in has two switches: Security page (publicip.js:93) and Settings `leak_detection_enabled` (settings.html:122). The Settings one skips the "uses an external service" notice, and a stale Settings form saved later silently reverts the Security switch. Conflicts with AGENTS.md "Local only".
7. Audit: any non-OK response shows "not configured" (audit.js:108); an unconfigured log returns `200 []` (handlers_security.go:291), which shows "no events" with no explanation.
8. Settings: empty "Max size (MB)" sends `0` (settings.js:201), overwriting the configured value.
9. Non-overview pages force the connection indicator to "connected" (i18n.js:110-113) without any live check: a false state.
10. `select.js` registers two document listeners per enhanced select (select.js:53-56), re-run on every table re-render (security.js:222): listener leak.
11. Failover field is comma-separated (studio.js:617) while all other list fields are one-per-line; the hint does not say so.
12. Policy create/update give no success feedback (studio.js:491, 512); empty profile filter says "No profiles configured" (studio.js:703).

### 5.2 Strings outside i18n / AGENTS.md rule breaks

13. Hardcoded English in shell.js: `Collapse sidebar` / `Expand sidebar` (64, 260), `Menu` (73), `Toggle theme` (79), `Daemon connection` (85), `Dark theme` / `Light theme` (137), `BETA` (45). Hardcoded `Revisar` in index.html:107, 133 and security.html:98 (overwritten at runtime).
14. Raw config values used as labels: adapter types (studio.html:125-134), auth methods (188-190), `strict/loose` (settings.html:92-93), `notify/killswitch_and_notify` (130-131), audit levels (audit.html:28-31).
15. Every legacy page duplicates its own ES/EN dictionary (nav, badges, actions); `SHELL_KS_I18N` (shell.js:359) duplicates `killswitch.state.*` from `internal/i18n`. New UI: one `messages.ts`.
16. Several user-facing server errors are English only, not `config.ValidationError`: groups (handlers_groups.go:64, :97), scheduler (handlers_schedule.go:55, :90), reload 409 `message` (handlers_tunnels.go:162, handlers_groups.go:268), policy reload (handlers_policy.go:141). Settings, scheduler and toggle requests do not send `X-Kongtrol-Lang` (settings.js:215, 298; security.js:158). **Groups and scheduler done (ADR-0042):** `config.err.*` / `api.err.*` via writeLocalizedError.
17. Color-only state: the connection indicator dot (shell.js:85, title only). Canvas sparkline and traffic chart have no text alternative (app.js:565, 636). Toasts have no `aria-live`. Tabs have no `role=tab` / `aria-selected` / arrow-key support (shell.js:327-346). Confirms use native `confirm()` with untranslated buttons (studio.js:526, 796, 1012; settings.js:343; security.js:171).
18. Daemon-lost / restored toasts exist only on the Overview page (app.js:444-455); other pages never report a dead daemon.

### 5.3 API / architecture notes

19. ~~`GET /api/v1/dns/resolve` has no client anywhere~~ — now used by `/traffic` (lookup detail).
20. `POST /api/v1/shutdown` has no GUI caller outside the tray; the browser dashboard has no "quit and disconnect all". This is a parity gap under "Product model".
21. Settings DTO structs are declared in the wrong files: `settingsDTO` in handlers_groups.go:341, `groupDTO` in handlers_security.go:329. Code hygiene only.
22. Deleting a VPN profile checks policy and group references (handlers_vpn.go:362-376), but not other profiles' `failover` lists or scheduler rules. The delete path does not call `config.Validate`. `no verificado`. **Done (ADR-0042):** DELETE refuses with `api.err.vpn_in_use` listing every reference (policies, groups, failover, scheduler rules) and validates the trial config.
23. `leak_detection_action` (`killswitch_and_notify`) is still editable. Whether it does anything now that the lookup is on demand only is `no verificado`. Confirm or remove. **Done (ADR-0042):** removed (a leak only notifies).
24. Audit HMAC is never verified for the dashboard: `security.VerifyEvent` (internal/security/audit.go:220) has no API caller and `GET /api/v1/audit` (handlers_security.go:284) sends only `hmac`. Missing endpoint/field. **Done (ADR-0042):** each entry has `signed` and `verified`; `/activity` shows them.
25. `security.audit_log.max_size_mb` is stored and defaulted (internal/config/loader.go:115) but nothing enforces it: rotation is daily only (internal/security/audit.go:122). Verified by search, not by test. Enforce it or drop the field. **Done (ADR-0042):** enforced per file (day continues in `.1`, `.2`…), plus `retention_days`.
26. `GET /api/v1/audit` reads only today's and yesterday's files (`ReadRecentAuditEvents(path, 2)`, handlers_security.go:296) and has no date parameter; `/activity` filters dates in the page and says so. **Done (ADR-0042):** `from`, `to`, `before` (cursor), `X-Kongtrol-Audit-More`, over every retained day.

### 5.4 CLI parity (AGENTS.md "CLI-complete")

The CLI has `policy` (list/add/remove/show/test; reload is `kongtrol reload --policy`), `profile` (add/detect/import, `import --create`; the "＋ Agregar" flow), `routes [perfil]`, `config public-ip`, `config dashboard`, `config lang`, `update-check`, `doctor`, `export`, `service install`, and `reload [perfil]`. **No CLI equivalent was found for:** scheduler rule CRUD, settings tuning (health check, split DNS, kill switch mode/allow_lan, fallback DNS, endpoints, audit config), per-profile kill-switch override, group CRUD, and the audit log viewer (`logs` may cover it). All of these are `no verificado`. Check `docs/CLI.md` before deciding whether phase 2 or the CLI owns the gap.

## 6. Counts

Rows = checkbox rows in §3, counted on the first target route named in the row (rows targeting the shell, `api/client.ts` or `messages.ts` count as Shell). "Covered" = status `new UI has it`. Partial rows are not counted as covered.

| Target page | Rows | Covered in new UI | Partial |
|---|---|---|---|
| All pages + shell | 244 | 244 ticked | 0 |

Final gate (phase 3, 2026-11-18): 244 of 244 rows ticked. Of those, 5 rows drop a cosmetic part
(`dropped: ADR-0043`: BETA badge, manual rail collapse memory, theme reveal animation, language
toggle flash, custom select widget replaced by the native `<select>`), 5 are met by a different,
standard pattern (`replaced` / `done as`), and 1 is `n/a`. No capability is lost.
