<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0009: Kill switch for split tunneling — block-only, per destination

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

A classic kill switch blocks all internet when "the VPN" drops. Kongtrol runs several tunnels at
once with split tunneling, and users disconnect on purpose. Blocking everything on any drop would
cut unrelated traffic and punish intentional disconnects.

## Decision Drivers

- Protect exactly what a dropped tunnel protected.
- Intentional disconnect must not block (`kongtrol down`, tray, dashboard).
- Must be expressible with each OS's firewall through the helper.
- A crashed daemon must not leave the user offline ([ADR-0005](0005-privileged-helper-service.md)).

## Considered Options

1. Global block on any drop — breaks split tunneling and intentional disconnects.
2. "Block all + allow exceptions" with Windows Firewall rules — impossible: "Explicit block
   rules take precedence over any conflicting allow rules" (Microsoft Learn, verified
   2026-11-17).
3. WFP sub-layer with permit/block weights (like wireguard-windows' kill switch) — expressive but
   more native code; not chosen now.
4. Block-only rules for a computed set of destinations, distinguishing intentional vs unexpected
   drop (Proton "standard" kill switch semantics).

## Decision Outcome

Chosen: **option 4**.

- `app.KillSwitchService` owns the blocked set. A protected tunnel that dropped unexpectedly
  (was up; user did not run `down` — `ProfileService` reports intent via `OnIntent`) blocks only
  its captured routes + policy destinations. A full-tunnel profile blocks everything except
  endpoints (+ LAN with `allow_lan` / legacy `mode: loose`), sent as the complement of its
  exceptions. Policies with `on_tunnel_down: block` add theirs. A connected tunnel's
  destinations are never blocked.
- `block.set` replaces the whole set: Windows Firewall BLOCK rules `KongtrolBlock`, Linux
  `KONGTROL_BLOCK` iptables/ip6tables chains (REJECT), macOS pf anchor `com.kongtrol` table.
- States `off` / `armed` / `active` (`security.KillSwitchStatus`).
- Semantics match Proton's standard kill switch: "only activated when your connection drops by
  accident… doesn't block your internet connection if you deliberately disconnect" (verified
  2026-11-17). No "advanced/permanent" mode.

### Consequences

- Good: unrelated traffic keeps working; same model on three OSes; fail-open via lease.
- Bad: domain blocks cover only IPs resolved so far; complement sets can be large.
- Not a permanent kill switch: protection ends if the daemon dies (lease expiry, deliberate).

### Confirmation

`internal/app/killswitch_service_test.go`, `internal/privops/block_test.go`,
`internal/privops/pfrules_test.go`.

## More Information

- Code: `internal/app/killswitch_service.go`, `internal/app/prefixset.go`,
  `internal/privops/block_{windows,linux,darwin}.go`, `internal/security/killswitch_priv.go`
- Research: [kill switch mechanisms](../research/2026-11-17-kill-switch-mechanisms.md)
