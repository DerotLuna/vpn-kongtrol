<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0084 · Protection while a VPN profile is disconnected

Version 1.0.0 · 2026-11-21

Status: Accepted · Date: 2026-11-21 · Decision-makers: project maintainer
Supersedes [ADR-0008](0008-policy-routing-on-tunnel-down.md).

## Context and Problem Statement

A VPN profile often serves several rules. Users need a shared default for the time its tunnel is off and a way to make individual exceptions. The TCP table may miss blocked attempts, so it cannot be the source of firewall decisions.

## Considered Options

- Configure each rule separately and keep automatic connection separate from blocking.
- Inherit the profile action and enforce protected destinations from the policy resolver. Use the existing TCP observer for notices and best-effort connection.
- Trigger a connection from a kernel firewall event. The available mechanisms differ across platforms and require verification against the helper's actual firewall rules.

## Decision Outcome

- Set `vpns.<name>.on_tunnel_down` to `allow`, `block` or `connect`. An omitted profile action defaults to `allow`. A rule with no action inherits its profile. Explicit `allow` persists as an override.
- `allow` leaves traffic to the available system routes. Another VPN may provide a system default route. `block` and `connect` request the same firewall rules for known destinations. A rule that only matches an executable needs explicit `allow` under a protective profile, or a domain or IP destination.
- The resolver caches static prefixes and resolved hosts until the configuration or resolved addresses change. Longest prefix wins, with profile priority and then configuration order breaking ties. The firewall uses that order to decide which addresses belong to each rule and keeps reserved destinations and known VPN transport addresses reachable. The independent kill switch can still block traffic.
- Install policy blocks before a user-requested individual disconnect removes routes. Keep the tunnel up if applying those blocks fails. A targeted CLI disconnect keeps the daemon running while any rule requires protection. Full daemon shutdown and helper lease expiry release the blocks.
- On Windows and Linux, a visible TCP sample can trigger notices or a best-effort connection for `connect`. A rejected socket can disappear between samples or before it enters the table. The firewall continues to enforce its current block set with an empty sample, a sampling error or a failed connection. macOS uses manual connection for `connect`.

## Consequences

Domain coverage follows learned IPs. New addresses and abrupt tunnel loss are reconciled asynchronously. Helper errors and lease expiry can prevent enforcement. UDP does not trigger a connection. An older explicit `connect` now requests blocking, and an executable-only `connect` needs a configuration edit before it can load.

## Confirmation

Configuration and service tests cover inheritance, explicit `allow`, prefix precedence and the disconnect transition with the destination firewall driven independently of TCP detection. Native firewall-event observation after a reject remains **unverified**.

## More Information

[TCP observation and firewall events](../research/2026-11-21-tunnel-down-detection.md)
