# Research: routes a VPN brings by itself, and what wins against a rule

Version 1.0.0 · 2026-11-24

## Question

Which routes does a VPN install on its own (not owned by Kongtrol), where do they come from, and
when one of the user's rules sends an overlapping destination through another VPN, which one does
the operating system actually use? Can the Reglas page say "your rule wins" truthfully?

## Sources

| Source | Accessed |
|---|---|
| [OpenVPN 2.6 manual (`--route`, `net_gateway`, `--route-metric`, `--route-nopull`, `--pull-filter`)](https://openvpn.net/community-docs/community-articles/openvpn-2-6-manual.html) | 2026-11-24 |
| [Microsoft Learn: Route metric (unattend) — longest prefix first, then lower metric](https://learn.microsoft.com/en-us/windows-hardware/customize/desktop/unattend/microsoft-windows-tcpip-interfaces-interface-routes-route-metric) | 2026-11-24 |
| [Microsoft Learn: Configure the order of network interfaces — route metric + interface metric](https://learn.microsoft.com/en-us/windows-server/networking/technologies/network-subsystem/net-sub-interface-metric) | 2026-11-24 |
| [Microsoft Learn: Automatic Metric for IPv4 routes (KB 299540)](https://learn.microsoft.com/en-us/troubleshoot/windows-server/networking/automatic-metric-for-ipv4-routes) | 2026-11-24 |
| [Microsoft Learn: Routes and the best route (RRAS)](https://learn.microsoft.com/en-us/windows/win32/rras/routes-and-the-best-route) | 2026-11-24 |
| [ip-route(8)](https://man7.org/linux/man-pages/man8/ip-route.8.html) | 2026-11-24 |
| [ip-rule(8) — rule order, `suppress_prefixlength`](https://man7.org/linux/man-pages/man8/ip-rule.8.html) | 2026-11-24 |
| [wg-quick for Linux (`add_default`)](https://raw.githubusercontent.com/WireGuard/wireguard-tools/master/src/wg-quick/linux.bash) | 2026-11-24 |

Code read (this repository, 2026-11-24): `internal/policy/engine.go` (`New`, `resolveIPMatch`),
`internal/monitor/policyresolver.go` (`syncRoutes`: `routeMetric = 10`, /0 and /1 never installed,
equal routes already on the tunnel interface left alone), `internal/app/profile_service.go`
(`policyAllowedIPs`, `patchWireGuardAllowedIPs`), `internal/vpn/openvpn/process.go` (no
`--route-nopull`), `internal/vpn/openvpn/logparse.go` (`parsePushedRoutes`),
`internal/vpn/wireguard/routes.go`, `internal/app/coexist.go` (`fullTunnelApps`).

## Findings

1. **OpenVPN file routes.** `route network [netmask] [gateway] [metric]` adds a route after the
   connection is up; netmask defaults to 255.255.255.255; gateway `net_gateway` is "the
   pre-existing IP default gateway", i.e. that destination stays OFF the tunnel (manual). Kongtrol
   starts openvpn with the user's file, so these are applied as written.
2. **OpenVPN server routes.** Kongtrol does not pass `--route-nopull` (process.go), so routes in
   the server's `PUSH_REPLY` are applied; they are only known once connected (logparse.go). A
   `route-nopull` or `pull-filter ignore "route"` written in the user's own file would drop them;
   the panel would still list what the PUSH_REPLY line says — **no verificado** whether openvpn
   logs the PUSH_REPLY before or after filtering.
3. **WireGuard AllowedIPs are NOT used as written** unless `default_egress` is on: on connect
   `policyAllowedIPs` rewrites the peer's AllowedIPs to the tunnel /24, its DNS and the rule
   destinations (PolicyResolver keeps them in sync). The file's AllowedIPs are therefore shown
   as "Kongtrol no las usa".
4. **Provider apps** (Proton, NordVPN, Surfshark, FortiClient, …) install their own routes;
   Kongtrol does not read them. Proton/NordVPN/Surfshark are treated as full tunnels elsewhere
   (`fullTunnelApps`, ADR-0099). Shown as "la app decide".
5. **OS choice: longest prefix first.** Windows: "If multiple route prefixes match a packet
   destination, the route with the longest prefix length is preferred"; with the same prefix
   length "the route with the lower metric is used", and the metric compared is route metric +
   interface metric (Microsoft Learn, above). Linux: route lookup within a table is longest prefix
   match (standard FIB behaviour; ip-route(8) does not spell it out — **reasoned, not verified
   against a primary page**); `preference`/metric: lower is preferred (ip-route(8)). macOS: longest
   prefix match (**no verificado**, no primary source read).
6. **Policy routes vs VPN routes.** PolicyResolver installs each rule destination (IP ranges, and
   every resolved domain IP as a /32) as its own route on the rule's tunnel interface with metric
   10, only while that tunnel is up, never a /0 or /1. So a rule narrower than a VPN route wins for
   its addresses; a VPN route narrower than the rule wins for its own; a full tunnel (/0, or two /1
   as OpenVPN `def1` and Kongtrol's Windows `default_egress` use) loses to every rule route.
7. **Equal prefixes on two interfaces** are decided by metric. Kongtrol sets 10 on its route, but
   on Windows the interface metric is added and is set by the OS (Automatic Metric) or by the VPN
   client; OpenVPN's own route metric comes from `--route-metric` or its defaults. Which one wins
   for a given pair of adapters is **no verificado** (Windows and Linux), so the panel says "el
   sistema elige una" and suggests a more specific address.
8. **wg-quick full tunnel on Linux** adds `ip rule add not fwmark $table table $table` and
   `ip rule add table main suppress_prefixlength 0`; the second rule is consulted first and rejects
   only the main table's default route, so specific routes in the main table (Kongtrol's rule
   routes) still win over the WireGuard table's default route.
9. **`priority` does not take part.** It only orders rules in the policy engine (domain/app rules
   and equal-length IP prefixes; IP rules otherwise pick the longest prefix, `resolveIPMatch`). The
   OS never sees it; it does not decide anything against a VPN's own routes.

## Open questions

- Equal prefix, Windows: does an OpenVPN (TAP/DCO/wintun) route beat Kongtrol's metric-10 route
  on a WireGuard interface? Needs a test machine (**no verificado**).
- Proton's advanced kill switch may block traffic through another tunnel regardless of the route
  table (ADR-0103, **no verificado**); the panel does not claim a rule wins over a provider app.
