# Research: Native WireGuard default egress

Accessed 2026-11-21.

| Source | Finding |
|---|---|
| [WireGuard routing and namespaces](https://www.wireguard.com/netns/) | The automatic routing pattern consults more-specific main-table routes and uses a mark to keep WireGuard's own UDP transport outside the fallback VPN route. |
| [WireGuard Linux wg-quick implementation](https://git.zx2c4.com/wireguard-tools/tree/src/wg-quick/linux.bash) | Each `/0` installs an unmarked-traffic routing rule and a main-table suppression rule. The `down` path checks the live peer prefixes before rule cleanup and removes matching main-table suppression rules for both families. That cleanup can affect another `wg-quick` default, so competing rules must be detected. The DNS configuration invokes the system's `resolvconf`. |
| [Kongtrol Linux routing implementation](../../internal/privops/sys_linux.go) | Policy routes use ordinary Linux routes without an alternate table. Paired with the upstream `wg-quick` ordering, more-specific policy and LAN routes retain priority over the native fallback default. |
| [Kongtrol policy resolver](../../internal/monitor/policyresolver.go) | The resolver can install a more-specific OS route without replacing the WireGuard peer's defaults. The existing overlap and endpoint rules still select the policy routes. |

The sources support using the delegated Linux `wg-quick` path and preserving `AllowedIPs` until teardown. They do not establish runtime behavior for arbitrary existing rules, custom DNS resolvers, other VPN clients, endpoint changes, or a process killed during configuration. Those cases remain **not verified**. This investigation did not modify the network.
