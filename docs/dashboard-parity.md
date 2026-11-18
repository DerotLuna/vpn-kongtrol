<img src="../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Dashboard parity: legacy → new UI

Version 1.1.0 · 2026-11-18

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

- [ ] Tunnels card (title, table) — legacy index.html:19 — API GET /api/v1/tunnels (handlers_tunnels.go:40) + WS — target `/` — status: partial (components/welcome/TunnelsCard.tsx: list only)
- [ ] Groups quick-launch card — legacy index.html:49, app.js:1140 — API GET /api/v1/groups (handlers_groups.go:17) — target `/` — status: missing
- [x] Traffic over time card — legacy index.html:59, app.js:609 — API GET /api/v1/metrics/history (handlers_tunnels.go:31) + WS — target `/traffic` — status: done (web/ui/src/components/traffic/TunnelTrafficCard.tsx, web/ui/src/data/traffic.ts)
- [x] Traffic map & policy resolver card — legacy index.html:68 — API GET /api/v1/policies (handlers_policy.go:53), GET /api/v1/resolve (handlers_policy.go:309) — target `/traffic` — status: done (web/ui/src/components/traffic/LookupCard.tsx)
- [ ] Base network egress card — legacy index.html:97, app.js:852 — API GET /api/v1/network/overview (handlers_routes.go:94) — target `/` (same data reused by `/traffic` network overview) — status: missing
- [ ] Security controls summary card + "Manage →" link — legacy index.html:112-116 — API GET /api/v1/security/status (handlers_security.go:91) — target `/` (link to `/security`) — status: partial (components/welcome/PublicIPCard.tsx: public IP only)
- [x] Active routes (live) card — legacy index.html:142, app.js:803 — API GET /api/v1/routes (handlers_routes.go:13) — target `/traffic` — status: done (web/ui/src/components/traffic/SystemRoutes.tsx; collapsed by default, read only while open)
- [ ] Helper install banner (prepended to `main`, `role=status`) — legacy app.js:357-386 — API GET /api/v1/helper/status (handlers_helper.go:67) — target `/` (+ `/settings` helper section) — status: new UI has it (components/welcome/HelperCard.tsx)
- [ ] Aggregate status headline (N of M connected) — legacy none (new) — API GET /api/v1/tunnels — target `/` — status: new UI has it (components/welcome/TunnelsCard.tsx)

#### Tunnels table: fields per row

- [ ] Profile name, rows sorted by name (`localeCompare`) — legacy app.js:523, app.js:577 — API WS snapshot `Name` key — target `/` — status: new UI has it (components/welcome/TunnelsCard.tsx); **fix**: legacy injects `name` unescaped into HTML and `onclick` (app.js:568-577)
- [ ] Status dot + translated label (`status.connected|connecting|disconnected|error`) — legacy app.js:525-530 — API `Status` — target `/` — status: new UI has it (components/welcome/TunnelsCard.tsx, StatusPill)
- [ ] Attach line (provider adapters, `attach.*` 7 phases, `waiting` style when user must act) — legacy app.js:540-543, app.js:1047-1059, app.js:1083 — API `attach {phase,app,location,reason}` — target `/` — status: missing
- [ ] Health line (`health.idle|no_response|unknown` + fix from `reason_key`), caution vs waiting style — legacy app.js:547-551, app.js:1064-1080 — API `health {state,reason_key}` — target `/` — status: partial (components/welcome/TunnelsCard.tsx: no `idle` line, only 2 reason keys mapped)
- [ ] Last error line `tunnel.lastError ({time})` + translated `last_error_key` (8 keys), raw error as tooltip, hidden when connected — legacy app.js:327-341, app.js:535-537 — API `last_error`, `last_error_key`, `last_error_at` — target `/` — status: missing
- [ ] "What goes through this tunnel (n / everything)" toggle (`aria-expanded`) — legacy app.js:552-555, app.js:747 — API GET /api/v1/tunnels/routes (handlers_tunnel_routes.go:76) — target `/` (opens `/traffic` per-tunnel view or inline) — status: missing
- [ ] Assigned IP (— when empty) — legacy app.js:559 — API `AssignedIP` — target `/` — status: missing
- [ ] Uptime (`formatDuration`, only when connected) — legacy app.js:560, app.js:1196 — API `UptimeSeconds` — target `/` — status: missing
- [ ] Bytes sent / received (`formatBytes`, — when 0) — legacy app.js:561-562, app.js:1189 — API `BytesSent`, `BytesReceived` — target `/` — status: missing
- [ ] Latency `{n}ms` — legacy app.js:563 — API `LatencyMS` — target `/` — status: missing
- [ ] Latency trend sparkline (canvas, connected only, redrawn on theme change) — legacy app.js:564-566, app.js:591-596, app.js:1221-1229 — API WS samples — target `/` — status: missing; **fix**: canvas has no text alternative

#### Tunnels: actions

- [ ] Connect (disconnected/error rows) — legacy app.js:568, app.js:1020 — API POST /api/v1/tunnels/{name}/connect (handlers_tunnels.go:50; 202 `connecting`, 200 `already_connected`, 404, 409 in progress, 503) — target `/` — status: missing; **fix**: legacy omits `encodeURIComponent` (app.js:1022) and swallows errors (no toast)
- [ ] Cancel (connecting rows) — legacy app.js:572-573, app.js:1033 — API POST /api/v1/tunnels/{name}/cancel_connect (handlers_tunnels.go:88) — target `/` — status: missing; **fix**: same encoding/error issues (app.js:1035)
- [ ] Disconnect (connected rows) — legacy app.js:569-570, app.js:1087 — API POST /api/v1/tunnels/{name}/disconnect (handlers_tunnels.go:112; 202 `waiting_user` for manual disconnect) — target `/` — status: missing; **fix**: legacy ignores 202 `waiting_user` (must say "disconnect in the app") and errors
- [ ] Reload (connected rows; re-reads yaml, restarts tunnel in place) — legacy app.js:571, app.js:1107-1128 — API POST /api/v1/tunnels/{name}/reload (handlers_tunnels.go:150; 409 `restart_required`, 202 `restarting|skipped_not_connected`) — target `/` — status: missing; **fix**: `skipped_not_connected` still toasts "Reloading"
- [ ] Connect all (non-connected tunnels, parallel) — legacy index.html:23, app.js:1177 — API POST …/connect ×N — target `/` — status: missing
- [ ] Disconnect all (connected tunnels, parallel) — legacy index.html:24, app.js:1182 — API POST …/disconnect ×N — target `/` — status: missing
- [ ] Refresh tunnels button — legacy index.html:25, app.js:459 — API GET /api/v1/tunnels — target `/` — status: partial (components/ui/ErrorNotice.tsx retry only on error)
- [ ] After any tunnel action: refresh routes + network overview after 600 ms — legacy app.js:1026-1029, 1038-1041, 1092-1095, 1123-1126 — API GET /api/v1/routes, /network/overview — target `/` — status: missing

#### Groups quick launch

- [ ] Group chip: name, `{n} profiles`, Connect, Disconnect — legacy app.js:1146-1153 — API GET /api/v1/groups — target `/` — status: missing
- [ ] Connect group — legacy app.js:1156-1165 — API POST /api/v1/groups/{name}/connect (handlers_groups.go:148; 202 `started[]`, `already_connected[]`) — target `/` — status: missing; **fix**: ignores `res.ok` and both lists
- [ ] Disconnect group — legacy app.js:1167-1175 — API POST /api/v1/groups/{name}/disconnect (handlers_groups.go:199; 500 joined errors) — target `/` — status: missing
- [ ] Refresh groups button — legacy index.html:52 — API GET /api/v1/groups — target `/` — status: missing

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

- [ ] Connected tunnels count — legacy index.html:103, app.js:854 — API GET /api/v1/network/overview `connected_tunnels` — target `/` — status: missing
- [ ] Default egress `iface (metric n)` — legacy index.html:104, app.js:856-860 — API `default_routes[0].interface, metric` — target `/` — status: missing
- [ ] Gateway (or `on-link`) — legacy index.html:105, app.js:861 — API `default_routes[0].gateway` — target `/` — status: missing
- [ ] Local IPs — legacy index.html:106, app.js:863-864 — API `local_ips` — target `/` — status: missing
- [ ] Public IP row + Check button — legacy index.html:107, app.js:865-869 — API `public_ip, public_ip_check_enabled` — target `/` — status: new UI has it (components/welcome/PublicIPCard.tsx)
- [ ] Note "Unmatched traffic follows your default route" — legacy index.html:109 — API — — target `/` — status: missing
- [ ] Refresh button — legacy index.html:100 — API GET /api/v1/network/overview — target `/` — status: missing

#### Security summary

- [ ] Kill switch badge (ACTIVE/ARMED/OFF + detail text) — legacy index.html:118-121, app.js:670-671, shell.js:299-306 — API GET /api/v1/security/status `kill_switch, kill_switch_enabled, kill_switch_status` — target `/` — status: missing
- [ ] DNS guard badge (ACTIVE/IDLE/DISABLED) — legacy index.html:122-125, app.js:672 — API `dns_guard, dns_guard_enabled` — target `/` — status: missing
- [ ] Leak check badge (CLEAN/LEAK/ERROR/PENDING/UNAVAILABLE/DISABLED) — legacy index.html:126-129, app.js:673, shell.js:312-318 — API `leak_check, leak_check_available, leak_detection_configured` — target `/` — status: missing
- [ ] Public IP + Check (hidden while lookup off, hint instead) — legacy index.html:130-134, app.js:677-680 — API `public_ip_check_enabled, leak_check.public_ip` — target `/` — status: new UI has it (components/welcome/PublicIPCard.tsx)
- [ ] Last checked time (`toLocaleTimeString(lang)`, — when off) — legacy index.html:135-138, app.js:682-698 — API `leak_check.checked_at` — target `/` — status: missing

#### States

- [ ] Loading skeleton rows (tunnels, policies, routes, base network, groups) — legacy index.html:43-44, 55, 90-91, 103-107, 155-156 — API — — target `/`, `/traffic` — status: partial (components/welcome/TunnelsCard.tsx: text loading, `aria-busy`; `/traffic` done: skeletons for tunnels, rules, routes, base network in components/traffic/)
- [ ] Empty: no tunnels / no policies / no policies for filter / no routes for filter / no groups / chart no data — legacy app.js:517, 959, 970, 822, 1143, 615 — API — — target `/`, `/traffic` — status: partial (components/welcome/TunnelsCard.tsx: no tunnels only; `/traffic` done: no VPN connected / no VPNs, no rules, no rules for query, no routes for filter, chart collecting data)
- [ ] Fetch errors: legacy silently ignores (`catch (_) {}`) — legacy app.js:459-468, 652-659, 718-726 … — API all — target shell (every page) — status: new UI has it (components/ui/ErrorNotice.tsx); **fix**: every card shows an error + "what to do"
- [ ] Button state per tunnel status (connect / cancel / disconnect+reload) — legacy app.js:568-574 — API `Status` — target `/` — status: missing
- [ ] First-run redirect: `needs_setup` → profile editor — legacy app.js:1242-1247 — API GET /api/v1/setup (handlers_setup.go:39) `needs_setup` — target `/` → `/profiles` — status: missing

#### Notices / toasts

- [ ] `toast.daemonLost` (error, 8 s) / `toast.daemonRestored` (success) on WS state change, not on first load — legacy app.js:442-455 — API WS — target shell (every page) — status: missing
- [ ] `toast.groupConnecting` / `toast.groupDisconnecting` / `toast.groupActionFailed` — legacy app.js:1159-1172 — API group connect/disconnect — target `/` — status: missing
- [ ] `toast.tunnelReloading` (info), `toast.tunnelRestartRequired` (409), `alerts.tunnelReloadFailed` via `showApiError` — legacy app.js:1110-1121 — API tunnel reload — target `/` — status: missing
- [ ] `helper.installing` (busy banner), `helper.installed` (success toast), `helper.failed` + `helper.failedHint` / daemon `error`+`hint`+`detail` — legacy app.js:390-415 — API POST /api/v1/helper/install (handlers_helper.go:74; 403 cancelled, 504 timeout, 500 failed) — target `/` + `/settings` — status: new UI has it (components/welcome/HelperCard.tsx)

#### Keyboard / focus

- [x] Enter in resolve input runs Check — legacy app.js:1012-1014 — API GET /api/v1/resolve — target `/traffic` — status: done (web/ui/src/components/traffic/LookupCard.tsx: a real form, Enter submits)
- [ ] Route-drawer toggle is a button with `aria-expanded` — legacy app.js:554 — API — — target `/traffic` / `/` — status: partial (web/ui/src/components/traffic/TunnelTrafficCard.tsx: `/traffic` done; Home toggle pending, deep link `/traffic#<tunnelAnchor(name)>`)

#### Endpoints / cadence

- [ ] WS `/api/v1/ws/metrics` (ws.go:17; push on change + every 5 s), reconnect after 3 s — legacy app.js:419-440 — target `/` — status: new UI has it (api/client.ts `subscribeTunnels`, api/hooks.ts `useTunnels` + 5 s poll fallback)
- [ ] Poll 30 s: security status, routes (+tunnel routes), policies, network overview, metrics history, groups, helper status — legacy app.js:1231-1238 — target `/`, `/traffic` — status: partial (api/hooks.ts: helper + public IP poll 5 s; `/traffic` done in web/ui/src/data/traffic.ts: routes, tunnel routes, policies, network overview, metrics history every 30 s, plus a refresh 600 ms after any tunnel goes up/down)
- [ ] Re-render cached data on language change (policies, routes, charts, helper banner) — legacy app.js:317-322 — API — — target shell (every page) — status: new UI has it (React re-render via app/PreferencesProvider.tsx)

### 3.2 `studio.html` + `studio.js` (Studio)

#### Shared

- [ ] Tabs Profiles / Policies / Groups, last tab remembered (localStorage `kongtrol-dashboard-tab-studio`) — legacy studio.html:20-25, shell.js:327-346 — API — — target `/profiles`, `/profiles/groups`, `/rules` (routes replace tabs) — status: missing
- [ ] Active config file path line (read-only, `--config`) — legacy studio.html:27, studio.js:448-451 — API GET /api/v1/policies/meta (handlers_policy.go:113) `config_path` — target `/settings` (also shown on `/profiles`) **IA: revisar** — status: missing
- [ ] Error toast text: localized `error`+`hint` when `error_key` set, else page text + raw detail — legacy studio.js:341-345 — API `writeLocalizedError` body `error, hint, error_key` (errors_i18n.go) — target `/profiles`, `/rules` — status: partial (components/ui/ErrorNotice.tsx)
- [ ] Requests send `X-Kongtrol-Lang` — legacy studio.js:483, 504, 551, 742, 772, 967, 989 — API — — target shell (api/client.ts) — status: new UI has it (api/client.ts `setApiLang`)

#### Tab: VPN Profiles → `/profiles`

Screens / sections

- [ ] Setup banner (`setup.title`, `setup.body`) when `needs_setup`; forces Profiles tab — legacy studio.html:102-107, studio.js:818-830 — API GET /api/v1/setup (handlers_setup.go:39) `setup_mode, needs_setup, restart_pending, config_path` — target `/profiles` **IA: revisar** (spec lists setup under `/settings`) — status: missing
- [ ] Restart banner (`restart.pending`, `restart.pendingHint`, Restart now) after any profile change or `restart_pending` — legacy studio.html:108-114, studio.js:833-839 — API `restart_required` from vpn CRUD — target `/profiles` (action in `/settings`) — status: missing
- [ ] Note "changes update config + keychain; restart to use" — legacy studio.html:115 — API — — target `/profiles` — status: missing
- [ ] Editor card + Existing profiles card — legacy studio.html:117, studio.html:215 — API — — target `/profiles` — status: missing

Fields (client side → server side)

- [ ] Name (text, trimmed) — legacy studio.html:120-122, studio.js:614 — API required (handlers_vpn.go:191-193), duplicate 409 (handlers_vpn.go:200-202) — target `/profiles` — status: missing
- [ ] Type select (10 types; default openvpn) drives visible fields — legacy studio.html:123-136, studio.js:603-609 — API `oneof` (internal/config/schema.go:30) via `config.Validate` (handlers_vpn.go:277) — target `/profiles` — status: missing
- [ ] Priority (number, default 10, `parseInt || 0`) + hint (tie-break only) — legacy studio.html:137-140, studio.js:616 — API `priority` — target `/profiles` — status: missing
- [ ] Backup profiles / failover (comma-separated) + hint — legacy studio.html:141-144, studio.js:617 — API `failover` validated by `config.Validate` (exist, not self, no cycles) — target `/profiles` — status: missing; **fix**: comma list while every other list is one-per-line, hint does not say so
- [ ] Host (forticlient, ciscoanyconnect, globalprotect, tailscale) — legacy studio.html:145-147 — API `host` — target `/profiles` — status: missing
- [ ] Config file path (openvpn, wireguard) + openvpn routes hint — legacy studio.html:148-151 — API `config` — target `/profiles` — status: missing
- [ ] Exact server (protonvpn, nordvpn) — legacy studio.html:152-154 — API `server`; cleared for forticlient/surfshark (studio.js:590, 623) — target `/profiles` — status: missing
- [ ] Protocol (protonvpn; placeholder `wireguard`) — legacy studio.html:155-157 — API `protocol` — target `/profiles` — status: missing
- [ ] Connect automatically yes/no + hint (forticlient, protonvpn, nordvpn; hidden = true) — legacy studio.html:158-164, studio.js:626 — API `auto_connect` — target `/profiles` — status: missing
- [ ] Country, City (protonvpn, nordvpn) + hint — legacy studio.html:165-171 — API `country`, `city`; `validateProvider` rejects on surfshark/forticlient (internal/config/semantics.go) — target `/profiles` — status: missing
- [ ] Server group (nordvpn) — legacy studio.html:172-174 — API `group` — target `/profiles` — status: missing
- [ ] Network connection name / interface (providers) + hint — legacy studio.html:175-178 — API `interface` — target `/profiles` — status: missing
- [ ] Wait time (providers; e.g. 10m) + hint — legacy studio.html:179-182 — API `wait_timeout` (duration, `config.Validate`) — target `/profiles` — status: missing
- [ ] App path (providers) — legacy studio.html:183-185 — API `app_path` — target `/profiles` — status: missing
- [ ] Auth method select (certificate / credentials / certificate+credentials; hidden for protonvpn, nordvpn, surfshark) — legacy studio.html:186-192 — API `auth_method` — target `/profiles` — status: missing; **fix**: option labels are raw values, not i18n
- [ ] Certificate path, Key path — legacy studio.html:193-198 — API `cert`, `key` — target `/profiles` — status: missing
- [ ] Username (`autocomplete=off`) — legacy studio.html:199-201 — API `username` → OS keychain (handlers_vpn.go:285) — target `/profiles` — status: missing
- [ ] Password (`type=password`, `autocomplete=new-password`, blank = keep, never prefilled) — legacy studio.html:202-204, studio.js:652 — API `password` → keychain; env references rejected (handlers_vpn.go:255) — target `/profiles` — status: missing
- [ ] Provider-only fields cleared when hidden; other hidden fields not sent — legacy studio.js:578-588, studio.js:619-624 — API absent = kept server-side (handlers_vpn.go:140) — target `/profiles` — status: missing
- [ ] Create sends only non-empty fields; Update sends the full draft (empty = clear) — legacy studio.js:634-636, 743, 773 — API POST /api/v1/vpns, PUT /api/v1/vpns/{name} — target `/profiles` — status: missing

Actions

- [ ] Create — legacy studio.html:206, studio.js:738-761 — API POST /api/v1/vpns (handlers_vpn.go:184; 201 `restart_required`) — target `/profiles` — status: missing
- [ ] Create during first run: toast `setup.saved`, restart, go to `/` — legacy studio.js:753-756 — API POST /api/v1/restart (handlers_setup.go:54) — target `/profiles` — status: missing
- [ ] Update (key = selected or name; rename supported) — legacy studio.html:207, studio.js:763-784 — API PUT /api/v1/vpns/{name} (handlers_vpn.go:214; 404, localized 400) — target `/profiles` — status: missing
- [ ] Delete (selected or typed name) with confirm — legacy studio.html:208, studio.js:786-807 — API DELETE /api/v1/vpns/{name} (handlers_vpn.go:349; 409 last profile :362, referenced by policy :367, by group :374) — target `/profiles` — status: missing
- [ ] Clear form (defaults: openvpn, priority 10, certificate, auto_connect yes) — legacy studio.html:209, studio.js:666-679 — API — — target `/profiles` — status: missing
- [ ] Row Edit (fills form) / row Delete — legacy studio.js:714-718, 724-728, 897-901 — API — — target `/profiles` — status: missing
- [ ] Filter by name, type, host, server (live) — legacy studio.html:217, studio.js:695-700 — API — — target `/profiles` — status: missing
- [ ] Restart now: disable button, wait old daemon gone (≤240×500 ms) and new one back (≤120×500 ms), then reload lists — legacy studio.js:846-895 — API POST /api/v1/restart (handlers_setup.go:54; 501 unsupported), GET /api/v1/setup as liveness — target `/settings` (button also in `/profiles` banner) — status: missing

Table fields

- [ ] Name, Type, Host/Server, Credentials (`username + password` / `username only` / `password only` / —) — legacy studio.html:218-232, studio.js:681-686, 707-721 — API GET /api/v1/vpns (handlers_vpn.go:18) `name, type, host, server, has_username_credential, has_password_credential` — target `/profiles` — status: missing
- [ ] Kill-switch override shown read-only + link to `/security` — legacy none (decision §2) — API `kill_switch_override` — target `/profiles` — status: missing

States / notices

- [ ] Skeleton rows; empty `empty.noProfiles` (also when filter matches nothing) — legacy studio.html:229-230, studio.js:690-705 — target `/profiles` — status: missing; **fix**: empty-filter state reuses "no profiles configured"
- [ ] Toasts `toast.profileCreated|Updated` (7 s), `toast.profileDeleted`, `alerts.profileCreateFailed|UpdateFailed|DeleteFailed`, `alerts.profileSelectFirst|SelectDelete` — legacy studio.js:747-803 — target `/profiles` — status: missing
- [ ] Confirm `confirm.deleteProfile` — legacy studio.js:796 — target `/profiles` — status: missing; **fix**: native `confirm()` (untranslated buttons) → app dialog
- [ ] Restart toasts `restart.running` (10 s), `restart.done`, `restart.failed` (apiErrorText), `restart.timeout` (15 s) — legacy studio.js:864-884 — target `/settings` — status: missing
- [ ] Hints `pEditor.tip`, field hints (priority, failover, autoConnect, country, interface, waitTimeout, ovpnRoutes) — legacy studio.html:139, 143, 151, 163, 167, 177, 181, 212 — target `/profiles` — status: missing

Cadence

- [ ] Load once on page load (profiles, setup state); no polling — legacy studio.js:1078-1086 — target `/profiles` — status: missing

#### Tab: Policies → `/rules`

Fields

- [ ] Name (text) — legacy studio.html:34-36, studio.js:358 — API required (handlers_policy.go:476), duplicate 409 (handlers_policy.go:169, :216) — target `/rules` — status: missing
- [ ] Via VPN profile (select from meta profiles) — legacy studio.html:37-39, studio.js:439-446 — API GET /api/v1/policies/meta (handlers_policy.go:113) `profiles`; via required + must exist (handlers_policy.go:479-484) — target `/rules` — status: missing
- [ ] If the tunnel is off: allow (default) / block / connect + live hint `editor.onDownHint.*` — legacy studio.html:40-47, studio.js:365, 371-381 — API `on_tunnel_down`, `config.ValidateOnTunnelDown` (handlers_policy.go:488) — target `/rules` — status: missing
- [ ] Domains (one per line) — legacy studio.html:48-50, studio.js:361 — API `match.domains`; at least one match (handlers_policy.go:485); `policy.ParseRule` (handlers_policy.go:491) — target `/rules` — status: missing
- [ ] IP ranges (one per line) — legacy studio.html:51-53, studio.js:362 — API `match.ip_ranges` (CIDR parse, handlers_policy.go:491) — target `/rules` — status: missing
- [ ] Apps (one per line) — legacy studio.html:54-56, studio.js:363 — API `match.apps` — target `/rules` — status: missing
- [ ] Tip + template text (`editor.tip`, `editor.template` with `<code>`) — legacy studio.html:64-65 — target `/rules` — status: missing

Actions

- [ ] Create — legacy studio.html:58, studio.js:479-493 — API POST /api/v1/policies (handlers_policy.go:151; 201; localized 400 :164) — target `/rules` — status: missing; **fix**: no success toast
- [ ] Update (selected or name; rename) — legacy studio.html:59, studio.js:495-514 — API PUT /api/v1/policies/{name} (handlers_policy.go:182; 404 :206) — target `/rules` — status: missing; **fix**: no success toast
- [ ] Delete with confirm `confirm.delete` — legacy studio.html:60, studio.js:516-535 — API DELETE /api/v1/policies/{name} (handlers_policy.go:234) — target `/rules` — status: missing
- [ ] Clear (also clears test inputs/result) — legacy studio.html:61, studio.js:394-404 — target `/rules` — status: missing
- [ ] Row Edit / Delete — legacy studio.js:425-426, 433-437, 537-541 — target `/rules` — status: missing
- [ ] Test before saving: target (domain/IP/URL) or app → matched / no match (reason) / error — legacy studio.html:69-75, studio.js:543-571 — API POST /api/v1/policies/test (handlers_policy.go:263; body `rule, target, app`; resp `matched, via, rule, reason`; 400 "target or app is required" :305) — target `/rules` — status: missing
- [ ] Reload from disk + tip — legacy studio.html:79-81, studio.js:468-477 — API POST /api/v1/policies/reload (handlers_policy.go:138; `policies` count) — target `/rules` — status: missing
- [ ] Resolve shortcut (same resolver as `/traffic`) — legacy none on Studio — API GET /api/v1/resolve — target `/rules` **IA: revisar** (spec lists "resolve" under `/rules`; legacy resolver is on Overview) — status: missing

Table / states / notices

- [ ] Existing policies: Name, Match (domains, ranges, `app:x`), Via, actions; sorted by name — legacy studio.html:82-95, studio.js:406-431, 459 — API GET /api/v1/policies — target `/rules` — status: missing
- [ ] Empty `empty.noPolicies`; skeleton rows — legacy studio.js:409, studio.html:92-93 — target `/rules` — status: missing
- [ ] Toasts `alerts.createFailed|updateFailed|deleteFailed|selectFirst|selectDelete|policyReloadFailed`, `toast.policiesReloaded` — legacy studio.js:472-530 — target `/rules` — status: missing
- [ ] Test result texts `result.matched`, `result.noMatch`, `result.noMatchReason`, `alerts.testFailed` — legacy studio.js:555-570 — target `/rules` — status: missing
- [ ] `on_tunnel_down` shown per row (legacy table omits it) — legacy studio.js:419-429 — API `on_tunnel_down` — target `/rules` — status: missing; **fix**: show it

#### Tab: Groups → `/profiles/groups`

- [ ] Name (text) — legacy studio.html:242-244, studio.js:910 — API required (handlers_groups.go:46-48), duplicate 409 (handlers_groups.go:55-57) — target `/profiles/groups` — status: missing
- [ ] Profiles (one per line) — legacy studio.html:245-247, studio.js:911 — API unknown profile / at least one (handlers_groups.go:103-117) — target `/profiles/groups` — status: missing; **fix**: free text → multi-select of existing profiles
- [ ] Create / Update / Delete (confirm `confirm.deleteGroup`) / Clear — legacy studio.html:248-253, studio.js:963-1022 — API POST /api/v1/groups (handlers_groups.go:39), PUT /api/v1/groups/{name} (:71), DELETE (:126) — target `/profiles/groups` — status: missing
- [ ] Row actions: Connect, Disconnect, Reload, Edit, Delete — legacy studio.js:938-944 — API POST /api/v1/groups/{name}/connect (:148), /disconnect (:199), /reload (:249) — target `/profiles/groups` — status: missing
- [ ] Connect toast `toast.groupConnecting` — legacy studio.js:1030-1039 — API group connect — target `/profiles/groups` — status: missing; **fix**: ignores `res.ok`
- [ ] Disconnect toast `toast.groupDisconnecting` / error — legacy studio.js:1041-1051 — target `/profiles/groups` — status: missing
- [ ] Reload: 409 `toast.groupRestartRequired` with `missing_profiles`, else `toast.groupReloading`, error `alerts.groupReloadFailed` — legacy studio.js:1060-1076 — API POST /api/v1/groups/{name}/reload (handlers_groups.go:249; 409 :264) — target `/profiles/groups` — status: missing
- [ ] Reload explanation text `studio.reloadGroupTip` — legacy studio.html:260 — target `/profiles/groups` — status: missing
- [ ] Table Name, Profiles, Actions; sorted; empty `empty.noGroups`; skeleton — legacy studio.html:261-273, studio.js:927-947 — API GET /api/v1/groups (handlers_groups.go:17) `name, profiles` — target `/profiles/groups` — status: missing
- [ ] Toasts `toast.groupCreated|Updated|Deleted`, `alerts.groupCreateFailed|UpdateFailed|DeleteFailed|SelectFirst|SelectDelete` — legacy studio.js:972-1019 — target `/profiles/groups` — status: missing

### 3.3 `security.html` + `security.js` → `/security`

#### Controls

- [ ] Kill switch row: title, description, runtime badge (with detail text), switch — legacy security.html:25-37, security.js:118-124 — API GET /api/v1/security/status (handlers_security.go:91) — target `/security` — status: missing
- [ ] Toggle kill switch; confirm only when turning off (`confirm.disableKillSwitch`); revert on error — legacy security.js:169-186 — API POST /api/v1/security/killswitch (handlers_security.go:227; body `enabled`; resp `kill_switch_enabled`) — target `/security` — status: missing
- [ ] DNS guard row + toggle (no confirm) — legacy security.html:39-51, security.js:188-201 — API POST /api/v1/security/dnsguard (handlers_security.go:253) — target `/security` — status: missing
- [ ] Public IP lookup row: switch + description with host — legacy security.html:52-63, security.js:130-133 — API POST /api/v1/security/publicip (handlers_security.go:47; resp `public_ip_check_host`) — target `/security` — status: partial (components/welcome/PublicIPCard.tsx: on Home)
- [ ] Switch not overwritten while focused during refresh — legacy security.js:119, 121, 131 — target `/security` — status: missing
- [ ] Switch disabled while request runs — legacy security.js:175, 190 — target `/security` — status: partial (components/welcome/PublicIPCard.tsx `busy`)
- [ ] Refresh button — legacy security.html:22 — target `/security` — status: missing

#### Per-profile kill switch override (decision §2)

- [ ] Table Profile + Override select (Inherit (global) / Force on / Force off), sorted — legacy security.html:66-84, security.js:205-233 — API GET /api/v1/vpns (handlers_vpn.go:18) `kill_switch_override` — target `/security` — status: missing
- [ ] Change override → toast `toast.overrideSaved` / `toast.overrideFailed` + reload table — legacy security.js:235-248 — API PUT /api/v1/vpns/{name}/killswitch (handlers_settings.go:95; `override` must be inherit|on|off :113-125; 404 :111) — target `/security` — status: missing
- [ ] Note `security.perProfileNote`; empty `empty.noProfiles`; refresh button — legacy security.html:69-71, security.js:207-209 — target `/security` — status: missing

#### Leak detection

- [ ] Leak check badge, public IP + Check, last checked — legacy security.html:86-105, security.js:126, 134-151 — API GET /api/v1/security/status `leak_check{state,has_leak,public_ip,reason,checked_at}` — target `/security` — status: partial (components/welcome/PublicIPCard.tsx: IP + Check only)

#### Notices / cadence

- [ ] Toasts `toast.killSwitchOn|Off`, `toast.dnsGuardOn|Off`, `toast.toggleFailed` via `showApiError` — legacy security.js:178-196 — target `/security` — status: missing
- [ ] Poll security status every 15 s — legacy security.js:255 — target `/security` — status: missing
- [ ] Badge states identical to `kongtrol status` (shellSecurityBadges) — legacy shell.js:299-321 — target `/security`, `/` — status: missing

### 3.4 `settings.html` + `settings.js`

#### Section: General

- [ ] Dashboard bind + port (read-only) + note + CLI commands `kongtrol config dashboard set-port|set-bind` — legacy settings.html:28-38, settings.js:165-166 — API GET /api/v1/settings (handlers_settings.go:14) `dashboard_bind, dashboard_port` — target `/settings` — status: missing
- [ ] Diagnostics link "Open Diagnostics" (to `/app/`) — legacy settings.html:46 — API GET /api/v1/doctor (handlers_doctor.go:21) — target `/settings` — status: new UI has it (components/welcome/DiagnosticsCard.tsx)
- [ ] Export configuration (download `kongtrol-export.yaml`, no secrets) — legacy settings.html:47 — API GET /api/v1/config/export (handlers_doctor.go:52; 404 `export.err.no_config`) — target `/settings` — status: new UI has it (components/welcome/DiagnosticsCard.tsx)
- [ ] Health check interval (text, e.g. 30s) + timeout (e.g. 10s) — legacy settings.html:52-64, settings.js:167-168, 188-189 — API PUT /api/v1/settings (handlers_settings.go:49) `health_check_interval, health_check_timeout`; `time.ParseDuration` (:57-64), `config.Validate` (:74-77) — target `/settings` — status: missing
- [ ] Split DNS enabled + interval (e.g. 60s) — legacy settings.html:66-79, settings.js:171-172, 192-193 — API `split_dns_enabled, split_dns_interval` (duration check :57-64) — target `/security` — status: missing

#### Section: Security tuning

- [ ] Kill switch mode strict/loose (legacy; `allow_lan` preferred) — legacy settings.html:90-95, settings.js:173, 194 — API `kill_switch_mode` (`oneof=strict loose`, internal/config/schema.go:177) — target `/security` — status: missing; **fix**: raw option labels
- [ ] Allow LAN traffic switch — legacy settings.html:96-99 — API `kill_switch_allow_lan` — target `/security` — status: missing
- [ ] Note "enable/disable kill switch on Security page" — legacy settings.html:101 — target `/security` (obsolete once co-located) — status: missing
- [ ] DNS guard fallback DNS (text, e.g. 1.1.1.1) — legacy settings.html:109-111 — API `dns_guard_fallback_dns` (`config.Validate`) — target `/security` — status: missing
- [ ] Public IP lookup enabled (duplicate of the Security switch) — legacy settings.html:120-123, settings.js:176, 197 — API `leak_detection_enabled` (legacy alias, handlers_settings.go:188-190) — target `/security` (single switch) — status: missing; **fix**: merge with the opt-in switch; must show the "uses external service" notice
- [ ] Public IP services (one https:// URL per line; default collapsed to empty) — legacy settings.html:124-126, settings.js:177, 198 — API `public_ip_endpoints` (handlers_settings.go:194-205; `config.ValidatePublicIPEndpoint` via `config.Validate`) — target `/security` — status: missing
- [ ] Public IP note `settings.publicIpNote` — legacy settings.html:127 — target `/security` — status: new UI has it (components/welcome/PublicIPCard.tsx `publicip.notice`)
- [ ] Leak action notify / killswitch_and_notify — legacy settings.html:128-133 — API `leak_detection_action` (`oneof`, internal/config/schema.go:208) — target `/security` — status: missing; **fix**: raw labels; effect with on-demand lookup `no verificado`
- [ ] Audit log path, max size MB (number), HMAC sign switch — legacy settings.html:137-153, settings.js:179-181, 200-202 — API `audit_log_path, audit_log_max_size_mb, audit_log_sign` — target `/activity` **IA: revisar** (config of the log shown there; could stay in `/settings`) — status: missing; **fix**: empty max size sends 0 (settings.js:201)

#### Section: Scheduler → `/rules/schedule`

- [ ] Scheduler enabled switch (saves immediately) — legacy settings.html:163-166 — API PUT /api/v1/settings `scheduler_enabled` — target `/rules/schedule` — status: missing; **fix**: legacy `onchange=saveSettings()` also saves every other unsaved field on the page
- [ ] Scheduler interval (e.g. 1m) — legacy settings.html:167-169, settings.js:170, 191 — API `scheduler_interval` (duration check handlers_settings.go:57-64) — target `/rules/schedule` — status: missing
- [ ] Rule name — legacy settings.html:175-177, settings.js:234 — API required (handlers_schedule.go:39-41), duplicate 409 (:49-51) — target `/rules/schedule` — status: missing
- [ ] Rule profiles (one per line) — legacy settings.html:178-180, settings.js:235 — API `profiles`, `config.Validate` (handlers_schedule.go:96-110) — target `/rules/schedule` — status: missing
- [ ] Weekdays (comma-separated, optional, e.g. mon,tue) — legacy settings.html:181-183, settings.js:236 — API `weekdays` (`config.Validate`) — target `/rules/schedule` — status: missing; **fix**: free text → day checkboxes
- [ ] Start / End HH:MM (optional) — legacy settings.html:184-189, settings.js:237-238 — API `start`, `end` (`config.Validate`) — target `/rules/schedule` — status: missing
- [ ] Create / Update / Delete (confirm `confirm.deleteRule`) / Clear — legacy settings.html:190-195, settings.js:294-359 — API POST /api/v1/scheduler/rules (handlers_schedule.go:32), PUT …/{name} (:62), DELETE …/{name} (:116) — target `/rules/schedule` — status: missing
- [ ] Rules table: Name, Profiles, Window (`days · start-end` or —), Edit/Delete — legacy settings.html:199-215, settings.js:258-279 — API GET /api/v1/scheduler/rules (handlers_schedule.go:12) — target `/rules/schedule` — status: missing
- [ ] Refresh rules button — legacy settings.html:161 — target `/rules/schedule` — status: missing
- [ ] Toasts `toast.ruleCreated|Updated|Deleted`, `alerts.ruleCreateFailed|UpdateFailed|DeleteFailed|SelectFirst|SelectDelete`; empty `empty.noRules` — legacy settings.js:261, 303-350 — target `/rules/schedule` — status: missing

#### Page actions / states

- [ ] Tabs General / Security / Scheduler (remembered) — legacy settings.html:20-25 — target split across `/settings`, `/security`, `/rules/schedule` — status: missing
- [ ] Save settings (one PUT for all fields; response refills form) + `toast.saved` / `alerts.saveFailed` via `showApiError` — legacy settings.html:222, settings.js:212-225 — API PUT /api/v1/settings (handlers_settings.go:49) — target `/settings`, `/security`, `/rules/schedule` (per-section save sending only that section's fields) — status: missing
- [ ] Revert (reload from server) — legacy settings.html:223, settings.js:206-210 — API GET /api/v1/settings — target `/settings` (also `/security`, `/rules/schedule`) — status: missing
- [ ] Unsaved changes warning when leaving — legacy none — API — — target `/settings` (also `/security`, `/rules/schedule`) — status: missing (new requirement, optional) **IA: revisar**
- [ ] Load once; no polling — legacy settings.js:361-365 — API GET /api/v1/settings, GET /api/v1/scheduler/rules — target `/settings` — status: missing

#### Settings items the spec lists for `/settings` that legacy lacks (new UI already ahead)

- [ ] Update notice opt-in — legacy none (only `update.event.available` notice, notices.js:22) — API GET/POST /api/v1/updates (handlers_lifecycle.go:52, :63) — target `/settings` — status: new UI has it (components/settings/UpdatesCard.tsx)
- [ ] Delete all my data (confirm dialog) — legacy none — API POST /api/v1/data/purge (handlers_lifecycle.go:92; `confirm:true` required) — target `/settings` — status: new UI has it (components/settings/DataCard.tsx)
- [ ] Helper status + install — legacy app.js:346-415 (banner) — API GET /api/v1/helper/status, POST /api/v1/helper/install — target `/settings` — status: new UI has it (components/welcome/HelperCard.tsx)
- [ ] Language + theme controls — legacy shell.js:79-83 — API GET/PUT /api/v1/preferences — target `/settings` (also header) — status: new UI has it (components/layout/AppShell.tsx)
- [ ] Shutdown ("quit and disconnect all") — legacy none — API POST /api/v1/shutdown (handlers_tunnels.go:21) — target `/settings` — status: missing (tray-only today; GUI parity gap)

### 3.5 `audit.html` + `audit.js` → `/activity`

- [ ] Filter by profile (text, Enter applies) — legacy audit.html:25, audit.js:117-119 — API GET /api/v1/audit?profile= (handlers_security.go:284, :301) — target `/activity` — status: missing
- [ ] Level filter All / INFO / WARN / ERROR / SECURITY — legacy audit.html:26-32 — API `level` (uppercased, handlers_security.go:302) — target `/activity` — status: missing; **fix**: changing the level needs Apply; level names untranslated
- [ ] Apply + refresh buttons — legacy audit.html:22, audit.html:33 — target `/activity` — status: missing
- [ ] Limit 300 — legacy audit.js:104 — API `limit` (handlers_security.go:317) — target `/activity` — status: missing; **fix**: no paging / "load more"
- [ ] Table: Time (`toLocaleString(lang)`), Level badge, Action, Profile, Message, Signed (signed / unsigned) — legacy audit.html:36-51, audit.js:80-94 — API `timestamp, level, action, profile, message, hmac` — target `/activity` — status: missing
- [ ] Level badge classes INFO ok, WARN pending, ERROR off, SECURITY on — legacy audit.js:61-66 — target `/activity` — status: missing
- [ ] Note "Showing {count} events"; empty `empty.noEvents`; skeleton — legacy audit.js:72-78, audit.html:48-49 — target `/activity` — status: missing
- [ ] Error → toast `audit.notConfigured` — legacy audit.js:108-110 — API 500 (handlers_security.go:287, :297); unconfigured returns `[]` (:291) — target `/activity` — status: missing; **fix**: any error is reported as "not configured"; empty-because-unconfigured is never explained
- [ ] Events feed (history of daemon notices) — legacy none (toasts only, notices.js) — API GET /api/v1/events?after= (handlers_events.go:28) — target `/activity` — status: missing

### 3.6 Cross-cutting

#### `shell.js` (nav, header, theme, shortcuts)

- [ ] Sidebar nav: Overview, Studio, Security, Audit Log, Settings, active item — legacy shell.js:11-17, 47-52 — target app router nav (8 routes, §2) — status: missing (components/layout/AppShell.tsx has header links only)
- [ ] Brand logo top-left + name + BETA badge — legacy shell.js:43-46 — target shell — status: partial (components/layout/AppShell.tsx: logo + name; no BETA)
- [ ] Desktop collapse rail, remembered (`kongtrol-dashboard-sidebar-collapsed`), tooltips mirror labels — legacy shell.js:59-67, 257-289 — target shell — status: missing
- [ ] Mobile drawer + scrim, closes on nav click — legacy shell.js:73-75, 90-95, 232-252 — target shell — status: missing
- [ ] Page title in top bar (`page.*.title`) — legacy shell.js:76 — target shell — status: missing
- [ ] Theme: explicit choice persisted (localStorage + preferences), choosing the system theme clears it, follows `prefers-color-scheme` live — legacy shell.js:103-160 — API PUT /api/v1/preferences (handlers_security.go:199) `theme` — target shell — status: new UI has it (app/PreferencesProvider.tsx)
- [ ] Theme switch circular reveal (view transition, reduced-motion off) — legacy shell.js:167-175, style.css:92-146 — target shell — status: missing (optional polish)
- [ ] Preferences load is authoritative (clears stale browser copy) — legacy shell.js:194-207 — API GET /api/v1/preferences (handlers_security.go:188) — target shell — status: new UI has it (app/PreferencesProvider.tsx)
- [ ] Language toggle with flash animation — legacy shell.js:80-83, 214-230 — target shell — status: partial (components/layout/AppShell.tsx: toggle, no animation)
- [ ] Live connection indicator + label (`ws.*`) — legacy shell.js:84-85, app.js:444-448 — target shell — status: partial (components/welcome/TunnelsCard.tsx "live" text); **fix**: dot is color-only; other legacy pages force it "connected" (i18n.js:110-113)
- [ ] Tabs component remembers last tab — legacy shell.js:327-346 — target router (URL is the memory) — status: missing; **fix**: no `role=tab`/`aria-selected`/arrow keys
- [ ] Kill switch detail text (ARMED protecting n / ACTIVE blocking names) — legacy shell.js:359-381 — API `kill_switch_status{state,protected,blocking}` — target `/`, `/security` — status: missing; **fix**: own dict duplicates `killswitch.state.*` of internal/i18n
- [ ] Keyboard shortcuts — legacy none found — target — — status: n/a (do not invent without ADR)
- [ ] Skip link to main content — legacy none — target shell — status: new UI has it (components/layout/AppShell.tsx)

#### `notices.js` (live notices)

- [ ] Poll events every 5 s on every page; `after` seq kept in sessionStorage (`kongtrol.noticeSeq`), fresh tab = no backlog — legacy notices.js:98-137 — API GET /api/v1/events?after= (handlers_events.go:28) `events[], seq` — target shell (all pages), list in `/activity` — status: missing
- [ ] Kinds shown: `policy_tunnel_down`, `tunnel_watchdog`, `update_available`; WARN → error style — legacy notices.js:122-124 — target shell — status: missing
- [ ] Keys `policy.event.allow|block|connect|connect_failed`, `watchdog.event.reconnect_failing|failover|reconnected|no_response|responding`, `update.event.available` (args `{0}{1}{2}`) — legacy notices.js:8-39 — target `src/i18n/messages.ts` — status: missing
- [ ] Action toast "Connect {profile}" when `can_connect`, stays 60 s — legacy notices.js:48-81, 125-126 — target shell — status: missing
- [ ] Action → connect with `notice.connecting` / `notice.connectFailed` — legacy notices.js:83-94 — API POST /api/v1/tunnels/{name}/connect — target shell — status: missing
- [ ] Watchdog `manual_reconnect_required` / attach drop notices — legacy not filtered in (kind list notices.js:122) — `no verificado` whether daemon emits other kinds — target shell — status: missing

#### `toast.js`

- [ ] Toast stack: message, optional detail line, raw API error as tooltip, click to dismiss, default 4.5 s — legacy toast.js:41-74 — target shell (`aria-live` region) — status: missing; **fix**: no `role=status`/`aria-live`
- [ ] `showApiError`: context message + translated hint per API `code` (8 codes) + raw detail as tooltip — legacy toast.js:6-39 — API `code` from `writeError` (handlers_common.go:23-48) — target shell — status: partial (components/ui/ErrorNotice.tsx)

#### `publicip.js`

- [ ] Render IP or — + hint `publicip.hint` while off; Check buttons only while on — legacy publicip.js:41-63 — target `/`, `/security` — status: new UI has it (components/welcome/PublicIPCard.tsx)
- [ ] Check now: busy `publicip.checking`, error `publicip.failed` + `failedHint` or daemon error/hint, leak_check error state — legacy publicip.js:66-89 — API POST /api/v1/security/publicip/check (handlers_security.go:77; 409 `publicip.err.disabled`) — target `/`, `/security` — status: new UI has it (components/welcome/PublicIPCard.tsx)
- [ ] Toggle: `publicip.on` + notice with host / `publicip.off` / `publicip.toggleFailed`; revert switch on error — legacy publicip.js:93-116 — API POST /api/v1/security/publicip (handlers_security.go:47) — target `/security` — status: new UI has it (components/welcome/PublicIPCard.tsx)

#### `charts.js`

- [x] HiDPI canvas prep — legacy charts.js:4-14 — target `/traffic` — status: done (web/ui/src/components/traffic/MiniChart.tsx: SVG, resolution-independent, no canvas)
- [ ] Sparkline (min-max scaled) — legacy charts.js:17-40 — target `/` — status: missing
- [x] Time series with 4-line grid, shared max — legacy charts.js:44-80 — target `/traffic` — status: done with fix (web/ui/src/components/traffic/MiniChart.tsx: y labels with units, x labels, figcaption summary, keyboard crosshair (slider), data table)

#### `select.js`

- [ ] Themed select replacement (keeps native `<select>` as source of truth) — legacy select.js:13-109 — target shared `Select` component — status: missing; **fix**: trigger lacks `aria-haspopup/expanded`, options lack `aria-selected`, adds global listeners per select (leak)
- [ ] Keyboard: ArrowDown/Enter/Space opens, arrows move, Enter/Space selects, Escape closes, focus returns to trigger — legacy select.js:47-56, 69, 81-85 — target shared component — status: missing

#### `i18n.js`

- [ ] Language resolution: explicit choice (localStorage `kongtrol-dashboard-lang` + preferences) else browser (`es*` → es) — legacy i18n.js:28-58, 131-140 — API preferences `language` — target shell — status: new UI has it (app/PreferencesProvider.tsx, i18n/lang.ts)
- [ ] Follow browser `languagechange` when no explicit choice — legacy i18n.js:141-143 — target shell — status: missing (`no verificado` in PreferencesProvider)
- [ ] `t()` with `{var}` interpolation, EN fallback — legacy i18n.js:60-64 — target `src/i18n/messages.ts` — status: new UI has it
- [ ] Rich text with `<code>` only, no innerHTML — legacy i18n.js:82-92 — target messages — status: missing (needed for `editor.template`)
- [ ] `html lang` attribute updated — legacy i18n.js:68 — API — — target shell — status: `no verificado` in new UI
- [ ] Refresh-button titles + lang-toggle aria label translated — legacy i18n.js:100-108 — target shell — status: new UI has it (components/layout/AppShell.tsx)

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
| GET /api/v1/vpns | handlers_vpn.go:18 | studio.js:731, security.js:227 | — | `/profiles`, `/security` |
| POST /api/v1/vpns | handlers_vpn.go:184 | studio.js:740 | — | `/profiles` |
| PUT /api/v1/vpns/{name} | handlers_vpn.go:214 | studio.js:770 | — | `/profiles` |
| DELETE /api/v1/vpns/{name} | handlers_vpn.go:349 | studio.js:797 | — | `/profiles` |
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
| GET /api/v1/groups | handlers_groups.go:17 | app.js:1134, studio.js:956 | — | `/`, `/profiles/groups` |
| POST /api/v1/groups | handlers_groups.go:39 | studio.js:965 | — | `/profiles/groups` |
| PUT /api/v1/groups/{name} | handlers_groups.go:71 | studio.js:987 | — | `/profiles/groups` |
| DELETE /api/v1/groups/{name} | handlers_groups.go:126 | studio.js:1013 | — | `/profiles/groups` |
| POST /api/v1/groups/{name}/connect | handlers_groups.go:148 | app.js:1158, studio.js:1034 | — | `/`, `/profiles/groups` |
| POST /api/v1/groups/{name}/disconnect | handlers_groups.go:199 | app.js:1168, studio.js:1044 | — | `/`, `/profiles/groups` |
| POST /api/v1/groups/{name}/reload | handlers_groups.go:249 | studio.js:1063 | — | `/profiles/groups` |
| GET /api/v1/policies | handlers_policy.go:53 | app.js:909, studio.js:456 | — | `/rules`, `/traffic` |
| GET /api/v1/policies/meta | handlers_policy.go:113 | studio.js:440 | — | `/rules`, `/settings` |
| POST /api/v1/policies/reload | handlers_policy.go:138 | studio.js:469 | — | `/rules` |
| POST /api/v1/policies | handlers_policy.go:151 | studio.js:481 | — | `/rules` |
| PUT /api/v1/policies/{name} | handlers_policy.go:182 | studio.js:502 | — | `/rules` |
| DELETE /api/v1/policies/{name} | handlers_policy.go:234 | studio.js:527 | — | `/rules` |
| POST /api/v1/policies/test | handlers_policy.go:263 | studio.js:549 | — | `/rules` |
| GET /api/v1/resolve | handlers_policy.go:309 | app.js:1007 | — | `/traffic`, `/rules` |
| GET /api/v1/dns/resolve | handlers_policy.go:404 | **none** (no client found in cmd/, internal/daemonctl, tray) | — | `/traffic` (web/ui/src/data/traffic.ts `useDnsResolveQuery`: "see which IPs it points to through <VPN>" after a domain lookup) |
| POST /api/v1/shutdown | handlers_tunnels.go:21 | **none** (tray via internal/daemonctl) | — | `/settings` (GUI parity gap) |
| POST /api/v1/restart | handlers_setup.go:54 | studio.js:861 | — | `/settings`, `/profiles` |
| GET /api/v1/setup | handlers_setup.go:39 | app.js:1242, studio.js:820, studio.js:848 | — | `/`, `/profiles`, `/settings` |
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
16. Several user-facing server errors are English only, not `config.ValidationError`: groups (handlers_groups.go:64, :97), scheduler (handlers_schedule.go:55, :90), reload 409 `message` (handlers_tunnels.go:162, handlers_groups.go:268), policy reload (handlers_policy.go:141). Settings, scheduler and toggle requests do not send `X-Kongtrol-Lang` (settings.js:215, 298; security.js:158).
17. Color-only state: the connection indicator dot (shell.js:85, title only). Canvas sparkline and traffic chart have no text alternative (app.js:565, 636). Toasts have no `aria-live`. Tabs have no `role=tab` / `aria-selected` / arrow-key support (shell.js:327-346). Confirms use native `confirm()` with untranslated buttons (studio.js:526, 796, 1012; settings.js:343; security.js:171).
18. Daemon-lost / restored toasts exist only on the Overview page (app.js:444-455); other pages never report a dead daemon.

### 5.3 API / architecture notes

19. ~~`GET /api/v1/dns/resolve` has no client anywhere~~ — now used by `/traffic` (lookup detail).
20. `POST /api/v1/shutdown` has no GUI caller outside the tray; the browser dashboard has no "quit and disconnect all". This is a parity gap under "Product model".
21. Settings DTO structs are declared in the wrong files: `settingsDTO` in handlers_groups.go:341, `groupDTO` in handlers_security.go:329. Code hygiene only.
22. Deleting a VPN profile checks policy and group references (handlers_vpn.go:362-376), but not other profiles' `failover` lists or scheduler rules. The delete path does not call `config.Validate`. `no verificado`.
23. `leak_detection_action` (`killswitch_and_notify`) is still editable. Whether it does anything now that the lookup is on demand only is `no verificado`. Confirm or remove.

### 5.4 CLI parity (AGENTS.md "CLI-complete")

The CLI has `policy` (list/add/remove/show/test/reload), `routes [perfil]`, `config public-ip`, `config dashboard`, `config lang`, `update-check`, `doctor`, `export`, `service install`, and `reload [perfil]`. **No CLI equivalent was found for:** scheduler rule CRUD, settings tuning (health check, split DNS, kill switch mode/allow_lan, fallback DNS, endpoints, audit config), per-profile kill-switch override, group CRUD, and the audit log viewer (`logs` may cover it). All of these are `no verificado`. Check `docs/CLI.md` before deciding whether phase 2 or the CLI owns the gap.

## 6. Counts

Rows = checkbox rows in §3, counted on the first target route named in the row (rows targeting the shell, `api/client.ts` or `messages.ts` count as Shell). "Covered" = status `new UI has it`. Partial rows are not counted as covered.

| Target page | Rows | Covered in new UI | Partial |
|---|---|---|---|
| `/` Home | 54 | 10 | 7 |
| `/profiles` | 40 | 0 | 1 |
| `/profiles/groups` | 10 | 0 | 0 |
| `/rules` | 20 | 0 | 0 |
| `/rules/schedule` | 10 | 0 | 0 |
| `/traffic` | 26 | 0 | 0 |
| `/security` | 24 | 2 | 3 |
| `/activity` | 10 | 0 | 0 |
| `/settings` | 16 | 6 | 0 |
| Shell / cross-cutting | 34 | 9 | 4 |
| **Total** | **244** | **27** | **15** |
