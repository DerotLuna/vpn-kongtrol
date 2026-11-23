<img src="../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Local API reference

Version 1.16 · 2026-11-23

<!-- feature:api -->
The daemon's local API (IPC on this machine only — nothing leaves the host). The tray, the CLI and
the dashboard (`web/ui/`, served at `/`) are clients of it — everything they can do is
reachable directly too. All endpoints are under `http://127.0.0.1:9741` by default and require
the local capability stored in `~/.kongtrol/api-token`. The daemon writes a new one every time it
starts, so read the file for each request rather than once
([ADR-0062](adr/0062-origin-bound-sessions-token-rotation-listener-owner.md)):

```bash
curl -H "X-Kongtrol-Token: $(cat ~/.kongtrol/api-token)" \
  http://127.0.0.1:9741/api/v1/tunnels
```

Kongtrol's own clients (`internal/daemonctl`) also check that the program listening on that
address belongs to you before sending the token; `curl` does not.

Browsers never get that token ([ADR-0039](adr/0039-browser-sessions-one-time-nonce.md),
[ADR-0062](adr/0062-origin-bound-sessions-token-rotation-listener-owner.md)):

```bash
POST   /api/v1/session/nonce   # token header only → {"nonce","expires_in":60}; single use
POST   /api/v1/session         # {"nonce"} from the page; Origin required → kongtrol_session cookie
                               #   + {"session_token"}: send it as X-Kongtrol-Session on every request
DELETE /api/v1/session         # sign the browser out (every page of that cookie)
POST   /api/v1/ws-ticket       # cookie + X-Kongtrol-Session → {"ticket","expires_in":30}; single use
GET    /api/v1/ws/metrics?ticket=<ticket>   # browser WebSocket: daemon Origin + cookie + ticket
                               #   (native clients: X-Kongtrol-Token header instead)
```

Clients open pages with `daemonctl.(*Client).PanelURL(path)`, which returns
`<base><path>#k=<nonce>` (`/`, `/tray`, `/profiles`…); the page (`web/ui/src/api/auth.ts`) trades
the nonce for the cookie and its page token (kept in `sessionStorage`). The cookie alone gets
401: browsers send it to every port of 127.0.0.1. Pages never set credentials; a 401 means "open
it from Kongtrol".

The WebSocket sends the whole tunnel map on every state change and on a 5 s heartbeat. Without
parameters each frame is that map, keyed by profile name (`kongtrol up`'s live view). With
`?events=1` (the panel and the tray flyout, ADR-0072) every frame is an envelope:
`{"type":"metrics","tunnels":{…}}` for the map, and `{"type":"changed","rev":N}` after any
successful mutating request (profiles, groups, rules, security, settings, preferences, reload),
so clients refetch what they show. A change frame carries no data.

Commands: [CLI.en.md](CLI.en.md) (Español: [CLI.md](CLI.md)).

## Endpoints

```bash
# Tunnels & monitoring
GET  /api/v1/tunnels                         # live snapshot of all tunnels
GET  /api/v1/metrics/history                 # persistent per-profile history (reconnects, jitter, samples)
GET  /api/v1/routes                          # active OS routes with resolved policy/profile
GET  /api/v1/network/overview                # default egress, local IPs, last public IP (only if the lookup is on; never fetched here), connected count
POST /api/v1/tunnels/{name}/connect          # 202 {"status":"connecting"}; 409 {error,hint,error_key} before 202 when it would clash:
                                             #   api.err.connect_full_conflict[_app] (second full tunnel, ADR-0099), api.err.connect_proton_app
                                             #   (a Proton file while the Proton app is connected), api.err.connect_same_address (same tunnel address, ADR-0103)
POST /api/v1/tunnels/{name}/cancel_connect
POST /api/v1/tunnels/{name}/disconnect
POST /api/v1/tunnels/{name}/reload           # reload kongtrol.yaml + restart this tunnel in place if connected
GET  /api/v1/connections/recent?limit=n     # recent destinations, memory only (ADR-0054): {"supported","enabled","max":500,"ttl_seconds":3600,
                                             #   "connections":[{"host","remote_ip","port","process","via","interface","first_seen","last_seen","count"}]};
                                             #   newest first; via "" = direct; bad limit → 400 api.err.recent_limit
GET  /api/v1/dns/resolve?domain=...&via=...  # split-DNS resolution for a specific tunnel
GET  /api/v1/resolve?target=...&app=...      # flow-aware policy resolution (app + target)

# Policies
GET/POST  /api/v1/policies                  # GET: each policy + `resolved` [{cidr, host, last_seen, expires_at}] (resolved IPs expire 6h unseen)
PUT/DELETE /api/v1/policies/{name}          # refusals: {error,hint,error_key} in X-Kongtrol-Lang — 409 api.err.policy_exists,
                                             #   404 api.err.policy_not_found, 400 api.err.policy_* | name_required (same check as `kongtrol policy add|remove`)
POST /api/v1/policies/reload                 # reload kongtrol.yaml from disk, hot-swap the policy engine and apply profiles live:
                                             #   {"status","policies","restart_required":bool,"reconnect_required":[profile]}
POST /api/v1/policies/move                   # {"from","to"}: every rule of `from` now goes through `to`, one validated save (ADR-0104);
                                             #   {"status":"moved","from","to","moved":[rule]}; 400 api.err.policy_via_unknown | policy_move_same
POST /api/v1/policies/test                   # dry-run a policy against a target/app before saving
GET  /api/v1/policies/explain?target=...     # which policy/VPN a target (IP, domain, URL, app:exe) uses and why; same JSON as `kongtrol policy show --json`
                                             # GET policy items expose effective on_tunnel_down and raw on_tunnel_down_override.
                                             # GET includes disabled rules. POST/PUT accepts disabled: true to pause or false to activate.
                                             # POST/PUT accepts on_tunnel_down: "" to inherit, or allow, block, connect.

# VPN profiles (config + OS keychain; applied live, ADR-0053 — see "Profile changes" below)
GET/POST  /api/v1/vpns                      # POST → 201 {"status":"created","profile","restart_required":false,"reconnect_required":false}
                                             # Profile GET exposes on_tunnel_down; POST/PUT accepts allow, block, connect.
                                             # GET adds "provider" ("protonvpn") for a direct profile made from that provider's file (ADR-0103/0104).
GET  /api/v1/vpns/detect                     # VPN clients installed here (same local probes as `kongtrol init`; read-only, runs nothing):
                                             #   [{"label","adapter_type","path","active":true|false|null,"interface"}]; active = the provider app's
                                             #   tunnel is up now (interface listing / macOS scutil, no admin, no automation); null = cannot tell
PUT/DELETE /api/v1/vpns/{name}              # PUT with a different "name" renames (→ {"status":"renamed","profile","previous"}): policies, groups,
                                             #   failover lists, scheduler rules, expected_ips, keychain entries and favorites follow;
                                             #   409 api.err.vpn_exists | api.err.vpn_rename_busy (tunnel up or connecting).
                                             # DELETE: 409 api.err.vpn_in_use listing policies, groups, failover and schedules at once;
                                             #   409 api.err.vpn_last; its expected IP is dropped; trial config validated first.
                                             #   409 api.err.vpn_delete_busy while its tunnel is up or connecting.
                                             # PUT ?reconnect=true: "save and reconnect" (see below) → adds "reconnected":true.
POST /api/v1/vpns/import                     # {"name","type":"wireguard"|"openvpn","content_base64","overwrite":false} → 201
                                             #   {"status":"imported","path","type","overwritten"}: stored 0600 as ~/.kongtrol/configs/<name>.conf|.ovpn;
                                             #   a file from a Proton account adds "provider":"protonvpn", "hint_key":"profile.import.proton_hint"
                                             #   and "hint" (translated: do not connect it while the Proton app is connected; ADR-0103);
                                             #   validated like the helper does (no PostUp/…; no up/down/plugin/script-security>1/log/config…;
                                             #   an .ovpn must inline its certs/keys). 409 api.err.import_exists unless overwrite; 400 api.err.import_* (hook, directive,
                                             #   script_security, file_ref, not_wireguard, invalid…; same rules as `kongtrol profile import`)
                                             #   Or {"path":"/abs/file.conf","elevate":false} instead of content: a file only root can read
                                             #   answers 403 config.err.file_root_only; "elevate":true reads it after the OS permission
                                             #   window (pkexec, Linux) → 201, or 403 api.err.import_elevated_failed (ADR-0058)
                                             #   The path must be absolute (Windows: drive letter only, no \\.\, \??\ or UNC share), without "..", a regular file (no link, folder, device);
                                             #   else 400 api.err.import_protected_path. The read (and the permission window, up to 3 min) runs
                                             #   without the config lock; only storing the file takes it (ADR-0062)
PUT  /api/v1/vpns/{name}/killswitch          # {"override": "inherit"|"on"|"off"}
GET/PUT /api/v1/vpns/{name}/default-egress    # PUT {"enabled":bool}, 409 while connected/connecting
                                             # Both return {enabled,compatible,reason?}, see ADR-0085

# Groups
GET/POST  /api/v1/groups
PUT/DELETE /api/v1/groups/{name}
POST /api/v1/groups/{name}/connect
POST /api/v1/groups/{name}/disconnect
POST /api/v1/groups/{name}/reload            # reload kongtrol.yaml + restart the group's active tunnels in place

# Security — live toggles, immediately re-applied (not just persisted)
GET  /api/v1/security/status                # + dns_guard_servers [ip], dns_guard_interface, dns_guard_profiles {profile:[ip]},
                                             #   kill_switch_error {message,hint,error_key,detail} in X-Kongtrol-Lang (absent when the last apply worked)
GET  /api/v1/security/blocked                # read-only: {"state","prefixes":[cidr],"count","blocking":[profile],"full_tunnel":[profile],"policy_blocking":[policy]}
POST /api/v1/security/killswitch             # {"enabled": true|false}
POST /api/v1/security/publicip               # {"enabled": true|false} — opt-in public IP lookup (off by default)
POST /api/v1/security/publicip/check         # one lookup now ("Revisar"); 409 publicip.err.disabled while off
POST /api/v1/security/dnsguard               # {"enabled": true|false}

# Settings & scheduler
GET/PUT   /api/v1/settings                   # health check, scheduler, split DNS, kill switch/DNS guard tuning, public IP lookup (public_ip_check_enabled, public_ip_endpoints, expected_ips {profile: ip} — replaced whole, empty value drops it, validated: existing profile + one IP), audit log (audit_log_path, audit_log_max_size_mb, audit_log_retention_days, audit_log_max_total_mb, audit_log_sign; the GET answers the values in use, never 0; applied live, ADR-0064), recent_connections_enabled (default true; off empties the list at once)
GET/POST  /api/v1/scheduler/rules
PUT/DELETE /api/v1/scheduler/rules/{name}    # rule {name,profiles,weekdays,start,end,"enabled":bool (false = paused)}; changes apply live (no restart)

# Audit log (every retained day; same reader as `kongtrol logs`)
GET  /api/v1/audit?profile=&level=&limit=&from=&to=&before=
#   newest first; profile may repeat (profile=a&profile=b: an entry of any of them, case-insensitive; ADR-0068);
#   limit 1–1000 (default 200); from/to = local day YYYY-MM-DD (to inclusive) or RFC 3339;
#   before = RFC 3339 cursor (exclusive): pass the last entry's timestamp for the next page.
#   Body: [{id,timestamp,level,action,profile,message,hmac,"signed":bool,"verified":bool}];
#   verified = signed and its HMAC matches the key in the OS keychain (the key never leaves the daemon).
#   Header X-Kongtrol-Audit-More: true|false. Bad limit/date → 400 api.err.audit_limit|audit_date|audit_range.

POST /api/v1/shutdown                        # ask a running daemon to shut down gracefully

# Diagnostics, helper and export
GET  /api/v1/doctor                          # same checks as `kongtrol doctor`, translated (X-Kongtrol-Lang): {"checks":[{section,label,status ok|warn|fail,detail,hint,fix,help}],"failures","warnings"}; fix: "helper_install"; help: "tunnelbear_driver"
GET  /api/v1/version                         # {"version"}: the daemon build version (what `kongtrol version` prints); tray menu and the dashboard show it
GET  /api/v1/helper/status                   # {"installed","running","available","elevated","needs_install","outdated",…}; outdated = answers with an older protocol ("Actualizar ayudante")
POST /api/v1/helper/install                  # installs the privileged helper (one OS permission prompt)
GET  /api/v1/config/export                   # kongtrol.yaml as a download (application/yaml, kongtrol-export.yaml; no passwords: they stay in the keychain); 404 export.err.no_config before the first profile

# First run and restart (ADR-0025)
GET  /api/v1/setup                           # {"setup_mode","needs_setup","restart_pending","config_path"}: daemon started with no kongtrol.yaml
POST /api/v1/restart                         # 202 {"status":"restarting"}: starts a fresh daemon (re-reads kongtrol.yaml, reconnects the tunnels that were up), then this one stops; 500 if the replacement could not start (this daemon keeps running)

# Update notice and "delete all my data" (ADR-0034)
GET  /api/v1/updates                         # {"check_enabled","host","current","latest","url","newer","checked_at"} from the remembered last check; never contacts GitHub
POST /api/v1/updates                         # {"enabled": true|false}: saves updates.check_enabled (opt-in, off by default); 409 cli.update.no_config before the first profile
GET  /api/v1/startup                         # {"start_at_login","start_at_login_supported","restore_tunnels","ask"} (ADR-0098); restore_tunnels is off until chosen, ask = the question after the first connect is pending (ADR-0101)
PUT  /api/v1/startup                         # {"start_at_login"?: bool, "restore_tunnels"?: bool, "restore_answer"?: bool}: the OS sign-in entry and preferences.json restore_tunnels; restore_answer answers the question (true also turns on the sign-in start when the tray is installed); 409 startup.err.no_tray on a CLI-only install
POST /api/v1/data/purge                      # {"confirm": true} (required, else 400 purge.err.confirm): deletes profiles, preferences and Kongtrol credentials while retaining in-use daemon/tray files, then restarts into setup mode → {"status":"purged","restarting":bool}
```

<!-- feature:first_run -->
The replacement process requires a new browser session. After a 401 on the restart probe,
the panel tells the user to reopen it from Kongtrol to exchange a fresh local nonce.

<!-- feature:update_check feature:uninstall -->
**Update notice and purge.** Reading `/api/v1/updates` never makes a request; with
`check_enabled` the daemon asks GitHub at most once per 24 h and emits one `update_available`
event (`update.event.available`, args latest, current, release page) per new version on
`/api/v1/events`. `/api/v1/data/purge` uses the same auth as every route (token or browser session:
cookie + X-Kongtrol-Session, same origin, `application/json`) plus the explicit `confirm`; the running daemon keeps
`api-token`, `run/` and its logs until it restarts.

<!-- feature:killswitch -->
**Notices and their actions.** `GET /api/v1/events?after=<seq>` returns `{"events":[…],"seq":N}`;
each event is `{seq, time, kind, level, key, args, profile, policy, action, can_connect, offer}`
(`internal/monitor/events.go`). `key` + `args` are an `internal/i18n` key the client translates.
`can_connect` asks for a "Conectar" / "Reintentar" button (connect `profile`). `offer` (ADR-0102)
asks for one more button:

| `offer` | Button | What the client calls |
|---|---|---|
| `enable_killswitch` | "Activar protección" + "No, gracias" | `POST /api/v1/security/killswitch {"enabled":true}`; "No, gracias" only closes it (the daemon already recorded that it asked) |
| `release_block` | "Recuperar internet" | `POST /api/v1/tunnels/{profile}/disconnect` (a user disconnect lifts the kill switch block) |
| `add_provider` | "Agregar <App>" | opens `/?add=vpn&type=<args[last]>` (the "＋ Agregar" sheet with that provider) |

Clients ignore an `offer` value they do not know. No new endpoint backs these actions.

<!-- feature:first_run -->
**Setup mode.** A daemon started with no config file (the tray spawns it on first run) serves the
dashboard with no profiles; `GET /api/v1/vpns` returns `[]` and the first valid
`POST /api/v1/vpns` writes `~/.kongtrol/kongtrol.yaml` and applies the first profile to the live
adapter set. On success it returns `restart_required: false` and leaves setup mode in the current
session (ADR-0092). A server without live adapter sync still returns `restart_required: true`.

<!-- feature:profile_live -->
**Profile changes (ADR-0053).** Creating, editing, renaming or deleting a
profile (and `POST /api/v1/policies/reload` after a hand edit) takes effect in the running daemon
at once: `restart_required` is `false` (`true` only if the daemon cannot build that profile, e.g.
an unknown type). A profile whose tunnel is up is never renamed or deleted (409
`api.err.vpn_rename_busy` / `api.err.vpn_delete_busy`). Editing it is saved, but the running tunnel
keeps its old settings: the answer says `reconnect_required: true` and nothing is reconnected
behind the user's back. `PUT /api/v1/vpns/{name}?reconnect=true` ("Guardar y reconectar") saves and
then disconnects and reconnects only that tunnel (same path as `/tunnels/{name}/reload`: the kill
switch knows the drop is intended); the answer adds `"reconnected": true` (the connect finishes in
the background: follow it in `/api/v1/tunnels` or the live feed) or `"reconnect_error"`. Priority,
failover and the kill switch override apply without a reconnect. `POST /api/v1/restart` keeps
working.

<!-- feature:default_egress -->
**WireGuard default egress (ADR-0090).** Compatibility checks the saved profile file locally on Windows and Linux.
A compatible direct WireGuard profile can select the option for its next connection on Linux.
The response's `enabled` field describes the saved selection. It can remain enabled if
the file later becomes incompatible, in which case the option can still be turned off.

<!-- feature:recent_connections -->
**Recent destinations (ADR-0054).** `GET /api/v1/connections/recent` lists the destinations
this computer connected to in the last hour, read from the OS TCP table without admin (Windows
`GetExtendedTcpTable`, Linux `/proc/net/tcp`; macOS `supported: false`). Kept only in the daemon's
memory (500 entries, forgotten 1 h after last seen, emptied when turned off or on shutdown),
never written to disk or logs and never sent off the machine; only this authenticated local API
serves it. Turn it off with `recent_connections_enabled: false` in `/api/v1/settings` or
`kongtrol config recent-connections off`.
