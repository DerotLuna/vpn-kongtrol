# ADR-0090: Select WireGuard default egress on Windows and Linux

Status: Accepted · Date: 2026-11-21

The system connection remains the default when there is no selected Kongtrol profile.
One compatible direct WireGuard profile can opt in. The selection applies when
Kongtrol next starts the profile. Existing routes with more specific prefixes
continue to take precedence. Independently started VPNs can still manage their own routes.

Keep Linux's native wg-quick handling from [ADR-0085](0085-native-wireguard-default-egress.md).
On Windows, the helper rewrites the two /0 values to complementary /1 pairs in
its private config copy. WireGuard installs and removes the routes with its
tunnel service. This preserves more-specific routes through other interfaces
and keeps WireGuard's /0 firewall from blocking them. The opt-in requires a
dual-stack file with one peer, both /0 prefixes and a DNS server, and leaves
the imported file intact. Existing Kongtrol kill-switch options are independent.
Windows retains its normal DNS resolution behavior in this mode.

Diagnostics query the Windows device inventory for the explicitly named TunnelBear
adapter. The report can show an installed client with a missing named adapter or
a Windows device error code. The dashboard offers the vendor's guide for these
cases. It does not disable devices or infer a connection failure from absence.

Before starting, the helper checks active broadly scoped routes for likely
conflicts. It does not change another VPN's routes or firewall. The check does
not guarantee compatibility with every filter, network compartment or VPN.

Device-level forwarding, DNS behavior and failover with concurrently active VPNs
remain unverified on a live Windows peer. Fixtures exercise both address
families, transformation of the private copy and route conflict detection.

[Research and sources](../research/2026-11-21-windows-vpn-egress.md)
