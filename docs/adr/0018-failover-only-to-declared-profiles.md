<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0018: Failover only to explicitly declared profiles

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Before the ADRs existed, the watchdog had an implicit failover: when a profile with
`priority > 0` could not reconnect (or failed a health check), it connected the first other
profile with `priority > 0`, lowest number first. On a real machine, killing `openvpn.exe` of
`ekiipago-ovpn` (OpenVPN) made the watchdog connect `ekiipago` (FortiClient), an unrelated VPN
to a different network, launching the FortiClient window every ~30 s. In Kongtrol every
profile reaches its own networks, and "the user decides which tunnels are up" is settled
(`AGENTS.md`). Separately, `priority` was documented three different ways (wizard: "lower
wins"; watchdog: lower preferred; policy engine: higher evaluated first).

## Decision Drivers

- The user controls which tunnels are up; nothing else is ever brought up implicitly.
- Profiles are never linked by name, type or priority — only by what the user declares.
- Follow established tools: failover targets are explicit and ordered
  ([research](../research/2026-11-17-vpn-failover.md)).
- Existing configs (which all have `priority` values) must not change behavior in a surprising
  way.

## Considered Options

1. Keep implicit failover by `priority`.
2. No failover at all; only retry the same profile.
3. No implicit failover; failover only to profiles declared on the dropped profile
   (`failover: [backup]`), validated.

## Decision Outcome

Chosen: **option 3**.

- `vpns.<name>.failover` (list, `omitempty`): backups tried in order when `<name>` drops and its
  reconnect fails, once per outage. Validation (`config.err.failover_*`, ES+EN with a "what to
  do" line): every entry exists, is not the profile itself, appears once, and the lists form no
  cycle.
- No list → no failover: the watchdog keeps retrying the same profile with backoff
  (2 s × 2^n, max 5 min) and sends one notice per outage (`watchdog.event.reconnect_failing`,
  event kind `tunnel_watchdog` for tray, dashboard and live view).
- A backup that takes over: the watchdog stops retrying the original until the user connects it
  again (`watchdog.event.failover` says so). Health-check degradation only notifies; it never
  starts a failover (a backup next to a still-up tunnel would fight it over routes and DNS).
- The watchdog only reconnects profiles that were brought up (`MarkActive`); the only other
  profiles it may connect are declared backups.
- `priority` keeps only its policy-engine meaning: a higher number evaluates a profile's policies
  first (domains and apps: first match wins; IPs: longest prefix always wins and priority only
  breaks ties of equal length); without it a policy's position counts (earlier first). Wizard
  and docs now say so.
- `down` cancels any reconnect/failover the watchdog has in flight for that profile
  (`MarkIntended`) and any running connect in `ProfileService`, and waits for it, so the status is
  final when `down` returns and no provider window opens afterwards.

### Consequences

- Good: no VPN is ever connected without the user asking for it (directly, by a rule, or by a
  declared backup). Existing configs keep their `priority` values with no failover side effect —
  no migration needed.
- Good: one meaning per field; the wizard no longer contradicts the engine.
- Bad: users who relied on priority failover must add `failover:` (none known).

### Confirmation

Tests: `internal/monitor/watchdog_failover_test.go` (similar names and equal priority never fail
over; declared backup does; `MarkIntended` cancels in-flight reconnects),
`cmd/kongtrol/daemon_watchdog_test.go` (the real config: priorities never become failover),
`internal/config/failover_test.go`, `internal/app/profile_service_test.go`.

## More Information

- Research: [How VPN clients define failover](../research/2026-11-17-vpn-failover.md).
- No earlier ADR described the implicit failover, so none is superseded.
