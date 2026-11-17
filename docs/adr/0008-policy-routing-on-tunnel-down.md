<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0008: Policy routing for all adapters, `on_tunnel_down`, notices

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Policies map domains/IPs to a VPN profile. Originally only some adapter types got routes. What
happens to a policy's traffic when its tunnel is off, and how does the user learn about it?

## Decision Drivers

- The user decides which tunnels are up.
- Same policy behaviour for every adapter type (direct, provider attach, Tailscale, WARP).
- Tell the user when traffic would go the "wrong" way, without spamming.

## Considered Options

1. Routes only for WireGuard (AllowedIPs) — other adapters ignore policies.
2. OS routes through the tunnel interface for every adapter; per-policy `on_tunnel_down`.
3. Always block policy traffic when its tunnel is down — surprising; breaks normal browsing.

## Decision Outcome

Chosen: **option 2**.

- `monitor.PolicyResolver` resolves policy domains (tunnel up or not), installs OS routes through
  the tunnel interface for every adapter type (WireGuard also gets AllowedIPs), follows tunnels
  via `TunnelProbe` and removes routes when the tunnel goes down. Overlaps follow policy engine
  order. Resolved IPs expire after 6 h unseen.
- `on_tunnel_down` per policy: `allow` (normal internet, default behaviour), `block` (adds its
  destinations to the kill switch blocked set, [ADR-0009](0009-kill-switch-split-tunneling.md)),
  `connect` (bring the tunnel up; rate-limited 3 min).
- `monitor.PolicyWatch` checks open TCP connections every 15 s (Windows GetExtendedTcpTable,
  Linux `/proc/net/tcp`; macOS unsupported) and emits i18n events, throttled 30 min per policy,
  served by `GET /api/v1/events?after=<seq>` to tray, dashboard and live view.

### Consequences

- Good: consistent policies across adapters; user stays in control.
- Bad: domain policies cover only IPs resolved so far; macOS has no connection watch.

### Confirmation

`internal/monitor/policyresolver_test.go`, `internal/policy` tests.

## More Information

- Code: `internal/monitor/policyresolver.go`, `policywatch.go`, `events.go`,
  `internal/config/schema.go` (`OnTunnelDown`)
- `AGENTS.md`: "Policy routing, kill switch, on_tunnel_down"
- No external sources: internal design decision.
