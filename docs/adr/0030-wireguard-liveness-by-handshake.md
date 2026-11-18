<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0030: WireGuard liveness by latest handshake, read through the helper

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer · Refines
[ADR-0005](0005-privileged-helper-service.md), [ADR-0018](0018-failover-only-to-declared-profiles.md)

## Context and Problem Statement

Issue #33: a WireGuard profile stopped working while `kongtrol status` and the dashboard showed
"connected". `Status()` needed `wg show`, which only Administrators/SYSTEM (root on Unix) may
run, so the unprivileged daemon fell back to "the interface exists" — always true for a
WireGuard tunnel whose server is gone. WireGuard has no connection state; the standard liveness
signal is the peer's latest handshake. Research:
[2026-11-18-wireguard-liveness](../research/2026-11-18-wireguard-liveness.md).

## Decision Drivers

- Never report "connected" for a tunnel Kongtrol cannot confirm, and say why.
- No admin terminal (helper does privileged reads); no secrets leave the helper.
- Reuse what exists: `vpn.Status`, watchdog reconnect/failover (ADR-0018), kill switch,
  `on_tunnel_down`, `monitor.Event` notices, the attach-style "message key" rendering.
- No false alarms for an idle tunnel without keepalive (it never handshakes).

## Considered Options

1. Keep the interface check; only show the handshake age (informative, still "connected").
2. Ping a host through the tunnel (needs a target per profile; not WireGuard's own signal).
3. Link `golang.zx2c4.com/wireguard/wgctrl` in the helper (vetted library; Windows WireGuardNT
   support **no verificado**; new dependency).
4. **New read-only helper op `wg.stats` running the trusted `wg show <iface> dump`; a health
   rule on handshake + counters; a dead tunnel reports `StatusError` + `TunnelHealth`.**

## Decision Outcome

Option 4.

- **Op.** `privops.WGStats` (`wg.stats`, `ProtocolVersion` 4): validated tunnel name, only a
  tunnel the helper manages or the caller's own profile tunnel (checks of `wg.remove`, nothing
  adopted). Windows runs `%ProgramFiles%\WireGuard\wg.exe`, Unix the root-owned `wg`.
  `ParseWGDump` keeps only peer public key, latest handshake, rx, tx, keepalive; failures never
  echo the output. Successful calls are not audited (polled every 10 s). An older helper →
  `ErrHelperOutdated` (protocol mismatch or "unknown operation").
- **Rule** (`internal/vpn/wireguard/health.go`): handshake ≤ 200 s old (REJECT_AFTER_TIME 180 s
  + 20 s) → `ok`. Otherwise, after a 30 s grace from connect: `no_response` if
  PersistentKeepalive is set, or tx grew while rx stayed flat for 15 s (3 × REKEY_TIMEOUT), or
  it never handshaked but sent something; else `idle` (unused tunnel without keepalive, not
  flagged). Counters unreadable → `unknown` with the helper guidance key.
- **Status.** `vpn.Status` keeps its four values. `no_response` makes `Status()` return
  `StatusError` (the existing "down" value), so the watchdog reconnects / fails over exactly as
  for a drop, the kill switch treats it as an unexpected drop and `on_tunnel_down` applies; the
  reconnect replaces the tunnel instead of reusing it. `unknown` keeps `StatusConnected` and
  says it could not be verified. New `vpn.TunnelHealth` (`vpn.HealthReporter`,
  `TunnelMetrics.Health`, json `health`) carries the detail; clients render `health.*` keys.
- **Events.** The watchdog emits `no_response` once per outage and `responding` when the
  handshake is fresh again (`watchdog.event.no_response` / `.responding`); while a reconnected
  tunnel has not answered yet, the outage (backoff, one failover) continues.

### Consequences

- Good: issue #33 is visible within ~30 s with keepalive, or as soon as traffic gets no answer;
  status, live view, tray, both dashboards say what to do.
- Good: no new dependency; secrets stay in the helper.
- Bad: the protocol bump means every helper op fails as "outdated" until the user runs
  `kongtrol service install` once after updating (existing rule for wire changes).
- Bad: an idle tunnel without PersistentKeepalive is only judged once traffic flows (documented;
  suggest `PersistentKeepalive = 25`).
- Neutral: one `wg show` per WireGuard tunnel every 10 s in the helper.

### Confirmation

Tests: `internal/privops/wgstats_test.go` (dump parsing, no secrets, validation, old helper),
`internal/vpn/wireguard/health_test.go` (rule table incl. idle-no-keepalive, keepalive-stale,
tx-without-rx, never-handshaked, helper-outdated; `Status()` follows the handshake; reconnect
restarts), `internal/monitor/watchdog_health_test.go`, CLI/tray/web UI rendering tests. Manual:
a real tunnel whose server is blocked (see the issue's PR).

## More Information

Code: `internal/privops/{privops,protocol,dispatch,client,local,wgstats,sys_windows,sys_unix}.go`,
`internal/privhelper/server.go`, `internal/vpn/health.go`, `internal/vpn/wireguard/`,
`internal/monitor/{collector,watchdog}.go`, `cmd/kongtrol/{attach_render,status_report,daemon_watchdog}.go`,
`cmd/kongtrol-tray/tray_attach.go`, `web/dashboard/{app,notices}.js`,
`web/ui/src/components/welcome/TunnelsCard.tsx`. Docs: `docs/CONFIGURATION.md`, `docs/SETUP.md`,
`docs/SECURITY.md`, `AGENTS.md`.
