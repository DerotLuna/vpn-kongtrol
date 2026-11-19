<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0053: VPN profiles are created, edited, renamed and deleted without restarting the daemon

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

The daemon built its adapters once at boot into a plain `map[string]vpn.VPNAdapter` shared,
unsynchronized, by the collector, watchdog, kill switch, ProfileService and API handlers. Any
profile change from the dashboard or CLI therefore answered `restart_required: true` and the
user had to restart Kongtrol ("Reiniciar ahora", ADR-0025), which disconnects every tunnel. With
the dashboard's "＋ Agregar" and `kongtrol profile add` (`internal/configedit`) adding
profiles became frequent, so the restart became the main friction.

Also asked: when the dashboard detects a VPN app, say whether its tunnel is up right now.

## Decision Drivers

- No restart for a profile change; never touch a tunnel the user did not ask to touch.
- Readers poll every few seconds; writes are rare (a profile save).
- A tunnel that is up is owned by its adapter instance (process, helper session, attach state).
- No admin, no GUI automation for detection (AGENTS.md product decisions).

## Considered Options

1. Keep the restart (`POST /api/v1/restart`).
2. `sync.RWMutex` around the map, held by every reader.
3. **Copy-on-write map behind `atomic.Pointer` (`vpn.Set`), plus a reconciler (`app.AdapterSync`)
   run on every config save.**

## Decision Outcome

Option 3.

- `internal/vpn/set.go` `vpn.Set`: readers load the pointer (`Get`, `Has`, `Names`, `Snapshot`
  copy) and never lock; writers (`Put`, `Delete`) copy the map under a mutex and swap it. This is
  the standard copy-on-write pattern for read-mostly data (Go `sync/atomic` docs, "atomic.Pointer";
  https://pkg.go.dev/sync/atomic#Pointer, accessed 2026-11-19). Used by ProfileService,
  KillSwitchService, Collector, Watchdog, the API and the CLI.
- `internal/app/adapter_sync.go` `AdapterSync.Sync(cfg)` after every save (`commitRuntimeConfig`
  in the API → `configedit.Save`, and `POST /api/v1/policies/reload` after a hand edit): new
  profiles get an adapter (`NewProfileAdapter`, the same constructor boot uses); profiles gone
  from the config are dropped (`Watchdog.Forget` clears their state and stops their watcher);
  an edited profile gets a new adapter only when its adapter seed changed (type, host, port,
  tunnel name, cert/key, config file, username, `AdapterExtra`) **and** it is disconnected.
- A busy profile (status not `disconnected`) is never replaced or removed. Rename and delete are
  refused while busy (409 `api.err.vpn_rename_busy`, new `api.err.vpn_delete_busy`). An edit of a
  connected profile is saved; the running tunnel keeps its old settings and the answer says
  `reconnect_required: true`. The pending adapter is applied by `Resync()` after the next
  disconnect.
- Owner decision (2026-11-19): **reconnect only when the client asks**. `PUT
  /api/v1/vpns/{name}?reconnect=true` ("Guardar y reconectar" in the dashboard) saves, applies and
  then runs disconnect → connect for that one profile through the same path as
  `/tunnels/{name}/reload` (`restartProfilesInPlace`: ProfileService marks the drop intended, so
  the watchdog does not reconnect and the kill switch does not block; `MarkActive` after the
  connect). Answer `reconnected: true` (connect continues in the background) or
  `reconnect_error`. A query parameter, not a body field: it is an action on the request, not a
  profile setting, and the API already uses query parameters for options (`?after=`, `?target=`).
  CLI parity: `kongtrol reload` (the CLI's edit path is the hand edit + reload) now asks
  "¿Reconectar <vpn> ahora? (se corta unos segundos)" for each connected profile the daemon
  reports as changed (`confirmAction`, `--yes`), and touches no other tunnel. An older daemon
  (no `restart_required` in the reload answer) keeps the old path.
- Priority, failover and the kill switch override are read when used: they never need a
  reconnect (`tunnelSettingsChanged`). Failover lists are re-applied on every sync.
- The watchdog looks the adapter up on every poll, so a reconnect loop follows a replaced adapter.
- Setup mode keeps `restart_required: true` (the restart leaves setup mode, ADR-0025);
  `POST /api/v1/restart` and `kongtrol restart` keep working. A profile whose adapter cannot be
  built (unknown type in a hand edit) answers `restart_required: true` and an alert.
- Detection: provider adapters register their interface `Matcher`
  (`appattach.RegisterMatcher`); `GET /api/v1/vpns/detect` and `kongtrol profile detect` add
  `active` (true/false) and `interface`, using `appattach.ActiveTunnel` — the same interface listing
  (and macOS `scutil --nc list`) attach mode already uses, no admin, nothing launched. Types without
  a matcher (wireguard, openvpn, tailscale, cloudflarewarp, ciscoanyconnect, globalprotect,
  unsupported apps) answer `active: null`: Kongtrol cannot tell without a profile. On macOS the
  Network Extension services are best effort, as for attach mode.

### Consequences

- Good: adding, renaming or deleting a profile never cuts the other tunnels; the dashboard and
  `profile add` no longer offer a restart.
- Good: no lock on the hot read path; `-race`-oriented tests cover set + sync + readers.
- Bad: a snapshot taken by a reader may be one save old for one poll (harmless: next poll sees it).
- Bad: global `cfg` is still swapped without synchronization by `onPolicyUpdate` (unchanged,
  pre-existing); AdapterSync keeps its own copy of the last config.

### Confirmation

Tests: `internal/vpn/set_test.go`, `internal/app/adapter_sync_test.go`,
`internal/monitor/watchdog_live_test.go`, `internal/api/profile_sync_test.go`,
`internal/api/handlers_connections_test.go` (detect), `cmd/kongtrol/cmd_traffic_test.go`
(reload asks), `cmd/kongtrol/cmd_profile_add_test.go` (detect active). `go test -race` needs cgo
(a C compiler); not run on the authoring machine (no gcc): **no verificado** with the race
detector, the concurrency tests run without it.

## More Information

Supersedes nothing: ADR-0025's restart stays for setup mode and as the manual path.
