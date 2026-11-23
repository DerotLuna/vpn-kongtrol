# Research: VPN coexistence on one machine

Version 1.1.0 · 2026-11-23

## Question

Why might Proton VPN and TunnelBear fail to connect when Cisco Secure Client,
Tailscale and Cloudflare WARP are installed, and what can Kongtrol verify locally?

## Sources

| Source | Accessed |
|---|---|
| [Cisco Secure Client 5.1 release notes](https://www.cisco.com/c/en/us/td/docs/security/vpn_client/anyconnect/Cisco-Secure-Client-5/release/notes/release-notes-cisco-secure-client-5-1.pdf) | 2026-11-21 |
| [Cloudflare One Client with legacy VPNs](https://developers.cloudflare.com/cloudflare-one/team-and-resources/devices/cloudflare-one-client/deployment/vpn/) | 2026-11-21 |
| [Tailscale alongside other VPNs](https://tailscale.com/docs/reference/faq/other-vpns) | 2026-11-21 |
| [Proton VPN advanced kill switch](https://protonvpn.com/support/advanced-kill-switch) | 2026-11-21 |
| [Proton VPN for Windows release notes](https://protonvpn.com/support/release-notes-windows) | 2026-11-21 |
| [TunnelBear: interaction with other VPN services](https://help.tunnelbear.com/hc/en-us/articles/360006839792-Why-is-my-Bear-slow) | 2026-11-21 |
| [TunnelBear: Windows driver errors and TunnelBear Adapter v9](https://help.tunnelbear.com/hc/en-us/articles/360060998872-Error-Driver-Error-on-Windows) | 2026-11-21 |
| [TunnelBear: Windows 11 24H2 protocol issue](https://help.tunnelbear.com/hc/en-us/articles/29910219458971-TunnelBear-not-working-after-Windows-11-24H2-update) | 2026-11-21 |
| [Microsoft: GetBestRoute2](https://learn.microsoft.com/en-us/windows/win32/api/netioapi/nf-netioapi-getbestroute2) | 2026-11-21 |
| [Microsoft: IPv4/IPv6 interface metric](https://learn.microsoft.com/en-us/windows-hardware/drivers/network/mib-ipinterface-row) | 2026-11-21 |
| [Tailscale issue #20637: Proton VPN and Windows split DNS](https://github.com/tailscale/tailscale/issues/20637) | 2026-11-21 |
| [Tailscale issue #5631: WARP interoperability reports](https://github.com/tailscale/tailscale/issues/5631) | 2026-11-21 |
| [Cisco Community: AnyConnect using a Surfshark-labelled device](https://community.cisco.com/t5/vpn/can-t-connect-to-vpn-using-anyconnect-fails-to-activate-or/td-p/4529139) | 2026-11-21 |
| [AdGuard issue #1192: WinTun coexistence](https://github.com/AdguardTeam/AdGuardVPNForWindows/issues/1192) | 2026-11-21 |
| [r/ProtonVPN: connection restored after removing another VPN](https://www.reddit.com/r/ProtonVPN/comments/1nlsnj1) | 2026-11-21 |
| [Microsoft: Windows Filtering Platform diagnostics](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/netsh-wfp) | 2026-11-21 |
| [WireGuard: Linux routing and network namespaces](https://www.wireguard.com/netns/) | 2026-11-21 |
| [WireGuard for Windows: routing and firewall behavior](https://github.com/WireGuard/wireguard-windows/blob/master/docs/netquirk.md#firewall-considerations-for-0-allowed-ips) | 2026-11-21 |
| [OpenVPN 2.7 manual: redirect-gateway](https://openvpn.net/community-docs/community-articles/openvpn-2-7-manual/) | 2026-11-21 |
| [Cloudflare One Client modes](https://developers.cloudflare.com/cloudflare-one/team-and-resources/devices/cloudflare-one-client/configure/modes/) | 2026-11-21 |
| [Tailscale issue #19128: macOS control-plane reconnection with Cisco Secure Client](https://github.com/tailscale/tailscale/issues/19128) | 2026-11-21 |

## Findings

- Cisco explicitly excludes simultaneous AnyConnect VPN sessions with another
  VPN client. Cloudflare documents conditional coexistence, with separate ownership
  of DNS and mutual routing exclusions for VPN transport. Its recipe does not
  override Cisco's support limits or a managed VPN policy.
- A Tailscale tailnet without an exit node can have narrow routes, but aggressive
  firewalls and overlapping CGNAT ranges can still interfere. Tailscale documents
  userspace/proxy mode as an option for apps that can use it. An exit node behaves
  differently and its split-tunnel coexistence workaround does not apply.
- Proton's advanced kill switch keeps Windows Filtering Platform rules active even
  after manual disconnect and reboot. An inactive adapter does not tell us whether
  such a policy is installed. Proton 5.1.5 release notes also document detection of
  network conflict and a fix for DNS state left behind after a crash.
- TunnelBear documents VPN-service interference, failures in its driver or
  maintenance process and a Windows 11 24H2 issue affecting some protocols. A driver
  or OS failure, missing WireGuard binary, profile failure and unrelated connectivity
  issue are alternative explanations.
- Two tunnel interfaces advertising complete routes for one family are a concrete
  signal of overlapping coverage. An intentional standby path, a scoped route,
  or a more specific route to the VPN endpoint can explain why the overlap is
  harmless. Metrics alone cannot prove the path a particular VPN executable takes.
  `GetBestRoute2` can identify the OS route for a known endpoint without sending a
  packet. That result does not reveal WFP blocking, proxy selection, split DNS or
  user-mode provider policy. Kongtrol's initial check reads the routes it can see.
- For an official WireGuard for Windows tunnel with a single peer, a `/0` in
  `AllowedIPs` activates restrictive firewall rules. Two complementary `/1`s
  can cover the same family with different filtering behavior. OpenVPN's
  `redirect-gateway ipv6` uses `2000::/4` and `3000::/4` to cover the global
  IPv6 unicast space. Kongtrol's `/0` and complementary-`/1` check misses that
  pattern.
- Cloudflare One Client can run in DNS-only, local proxy or posture-only mode.
  DNS-only mode can change resolver behavior with no competing full route. In
  local proxy mode, explicitly configured applications depend on the client's
  MASQUE transport over UDP, even though it does not take over host-wide IP
  routing. Posture-only mode neither routes traffic nor forwards DNS.

## First-hand community reports

- The Tailscale #20637 reporter tested Proton with both WireGuard and OpenVPN on
  Windows 11. The Tailscale tunnel carried IP traffic while queries to its split
  DNS resolver timed out. An IP test and a DNS test can disagree even with the
  same two providers. This report did not reproduce the inability to start Proton.
- Tailscale #5631 collects reports spanning several operating systems and WARP
  modes. The variation makes provider versions and whether WARP owns traffic,
  DNS, or both essential to reproduction.
- In Tailscale #19128, a macOS reporter initially connected to the Tailscale
  control plane over IPv4 alongside Cisco Secure Client in split-exclude mode.
  After the control connection dropped, retries logged unreachable IPv6 addresses
  despite working IPv4 bypass routes for fresh connections. The report shows why
  a reconnect failure can require endpoint addresses, address family and retry
  logs even when there are no competing full routes. The proposed cause involving
  the cached dial plan and fallback is the reporter's analysis.
- A Cisco Community user saw a device named after an uninstalled VPN while the
  driver belonged to Cisco. An AdGuard tracker report describes a different set
  of clients with intermittent routing and DNS while several Wintun interfaces
  were active. These reports justify capturing driver descriptions and stable
  interface identity. Neither establishes a generic Wintun collision.
- A Proton community commenter reported that removing another installed VPN
  restored connectivity. With several changes or no before/after state, that
  observation does not identify the conflicting service, filter or device.

## Limits of the current diagnostic

- The Cisco warning is driven by detected interface presence. The fallback
  accepts an up interface with a matching name or description and a unicast
  address, including private addresses. Registered provider matchers may also
  accept an up interface without an address. OpenVPN's documented `persist-tun`
  behavior illustrates how an interface can retain an address and routes through
  a reconnect. A retained interface or a misleading device name can look like a
  second active VPN. Use provider-reported session state where available before
  asserting that both sessions are established.
- Route entries are grouped by interface name and address family. A standby
  default can trigger the coverage warning, and interface aliases or route scope
  can affect attribution. On Linux, the route listing reads the main tables for
  IPv4 and IPv6, so a default selected through an `ip rule` and a separate table
  can escape the check. Generic interfaces for VPNs outside Kongtrol are also
  missed when the provider cannot be identified.

## Coexistence design

Separate three controls: who owns internet egress, who answers DNS for each
namespace, and who may block traffic. One selected egress and disjoint
split-tunnel routes are a safer default where the vendor supports both.
Keep policy-managed clients in an observe-only mode. When a vendor forbids
simultaneous use, a separate host or VM may be an option if the organization
allows it. Linux network namespaces can isolate direct WireGuard routes
while keeping the transport socket on a designated uplink. That is a
candidate for an isolated-profile mode, not evidence of Windows coexistence.

The next diagnostic can collect two bounded, in-memory snapshots around one
connection attempt. Compare OS route selection to the actual transport endpoint,
provider interface identity, effective DNS and the adapter's handshake/error.
Look for third-party WFP filters only through an explicit diagnostic on Windows.
Report a concrete next action for the detected layer. This approach needs a
controlled VM proof and permission for any elevated capture. It should never
change filters, DNS or vendor policy implicitly.

## Two full tunnels and the kill switch (2026-11-23, ADR-0099)

Question from the owner: what happens when Proton VPN is connected in its app
and the user starts a direct WireGuard full tunnel, or the reverse, and why does
the kill switch sometimes block everything?

| Source | Accessed |
|---|---|
| [WireGuard for Windows: firewall considerations for `/0` Allowed IPs](https://github.com/WireGuard/wireguard-windows/blob/master/docs/netquirk.md#firewall-considerations-for-0-allowed-ips) (read through the GitHub API, raw file) | 2026-11-23 |
| [Tailscale alongside other VPNs](https://tailscale.com/docs/reference/faq/other-vpns) | 2026-11-23 |
| [Mullvad: Tailscale partnership](https://mullvad.net/en/blog/tailscale-has-partnered-with-mullvad) | 2026-11-23 |
| [Proton VPN Android: action needed to connect (one always-on VPN)](https://protonvpn.com/support/action-needed-to-connect-to-vpn) | 2026-11-23 |

- WireGuard for Windows: with one peer whose AllowedIPs hold a `/0`, it enables a
  "kill-switch" whose firewall rules permit the tunnel service, loopback,
  DHCP/NDP and packets going through the WireGuard tunnel; "all other packets
  are blocked". A second VPN's transport and tunnel traffic is "other". Two
  `/1` halves avoid those rules but still take the whole default route.
- Tailscale: "Most VPNs set aggressive firewall rules to ensure all network
  traffic goes through them", which can drop another VPN's traffic; "exit nodes
  only support one VPN at a time". Narrow (split) routes are the documented way
  to run two clients.
- Mullvad documents its combination with Tailscale only through the Tailscale
  app (Mullvad servers as exit nodes), not through two apps side by side.
- Proton documents, for Android, that only one VPN app can be always-on. No
  Proton page found that supports two desktop VPNs both taking all traffic
  (**no verificado** that one does not exist). Proton's advanced kill switch
  (sources above) keeps its own filter rules.

Kongtrol's own kill switch had a matching failure: a full-tunnel profile that
"dropped" (interface gone or `no_response`) got a block of everything except VPN
endpoints. If the drop happened because another VPN took the default route,
that block also covered the other VPN's traffic (Windows Firewall block rules
match remote addresses on every interface), i.e. the whole internet.

Decision inputs: refuse a second full tunnel before it starts (both directions,
managed or provider app), allow split + full, and treat a full-tunnel drop while
another full tunnel is up as a switch, not an outage. Mullvad and Tailscale
choose "one full tunnel at a time" in their own products; Kongtrol enforces the
same rule across apps.

Limits (**no verificado** on live clients): provider app detection uses
interface matchers with an address, so a retained adapter can look active and a
macOS Network Extension service (no address) is not counted; a server-pushed
`redirect-gateway` is only known after the OpenVPN tunnel is up; corporate
clients (FortiClient, Cisco, GlobalProtect) and WireGuard tunnels started
outside Kongtrol are not counted as full tunnels.

## Reproduction plan

Use an isolated Windows VM at the same build and provider versions. Begin with one
working client. Install each additional client individually, record whether it starts
on login, and compare installed, disconnected and connected states. Test Proton and
TunnelBear against that baseline using each app's reported protocol. Change one setting
at a time, with the owner's permission if a protection policy must be changed.

Record the interface and family, effective route to the VPN endpoint, DNS ownership
and local provider error before and after the attempt. Compare a single default VPN and
one split tunnel, two defaults, WARP split exclusions, Tailscale with and without an exit
node, and Cisco's unsupported overlap. Keep diagnostic data local. The reported device's
root cause and these live provider combinations are **no verificado**.
