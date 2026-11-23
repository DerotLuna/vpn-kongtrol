# ADR-0085: Select native WireGuard default egress on Linux

Status: Superseded by ADR-0090 · Date: 2026-11-21

## Context

A profile can carry the default IPv4 and IPv6 routes while existing LAN and VPN routes continue to handle their destinations. The routing method depends on the tunnel client. Linux `wg-quick` already installs an automatic default in a separate table, marks its transport, and uses more-specific main-table routes first. Removing the live `AllowedIPs` defaults also prevents its ordinary teardown from removing the routing rules. [Research and source notes](../research/2026-11-21-default-egress.md)

## Decision

Allow one direct WireGuard profile to set `default_egress: true`. The saved profile editor and `kongtrol routes default-egress <profile> <on|off>` write the same setting. `kongtrol routes check-default <profile>` examines the saved file without touching the network. The local profile API returns `enabled`, `compatible` and, if incompatible, `reason`. A running daemon refuses changes while a connection is up or coming up. The selection applies the next time Kongtrol starts the tunnel. Existing profiles with the field omitted retain their current connection and policy behavior.

Compatibility requires Linux, a readable direct WireGuard file, one peer with an endpoint, IPv4 and IPv6 tunnel addresses, both `/0` prefixes, automatic routing and a DNS server. The helper rechecks the bytes it will use and gives `wg-quick` its private copy. A private marker keeps both defaults through policy reconciliation and helper restarts. The helper checks for conflicting Linux suppression rules before starting the selected tunnel. Legacy helper clients omit the option. A current client using the option with an older helper gets the existing update guidance. The profile file stays intact. Existing more-specific main-table routes, including routes to other VPNs and LANs, keep their precedence. A rule that allows traffic after a tunnel drops leaves the decision to the remaining OS routes. With the WireGuard default still active, the remaining route can use it.

An option to insert generic default routes through any tunnel was considered. Its correctness depends on separately establishing peer routing, endpoint reachability, DNS and rollback on each platform. Native Linux support makes those responsibilities explicit while retaining the existing route mechanism for policies.

## Limits and verification

Compatibility describes the saved file. Effective routes depend on the host's rules and active VPNs. DNS behavior, remote IPv6 forwarding and recovery after forced termination were **not verified**. Another process can start a conflicting full tunnel or change Linux policy rules after the check. The feature carries no guarantee of kill switch behavior, exclusive traffic coverage or arbitrary direct-internet exceptions. Existing external tunnels keep their existing routes when detected. A tunnel must be disconnected and started by Kongtrol to apply the selection.

Tests use fixture configurations, simulated route tables and the local API. No test changes network settings. The recommended evidence for live compatibility is route lookup for both IP families, with and without more-specific routes, in an isolated environment.
