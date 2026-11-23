# Research: a Proton file next to the Proton app, and shared tunnel addresses

Version 1.0.0 · 2026-11-23

## Question

Why does a direct WireGuard/OpenVPN profile downloaded from a Proton account break when the
Proton VPN app is connected too, even as a split tunnel (issue #51)? How can Kongtrol tell a
Proton file apart without a new config field, and what else makes two direct tunnels clash?

## Sources

| Source | Accessed |
|---|---|
| [OPNsense: WireGuard client with Proton VPN](https://docs.opnsense.org/manual/how-tos/wireguard-client-proton.html) | 2026-11-23 |
| [Proton VPN: WireGuard configurations](https://protonvpn.com/support/wireguard-configurations) | 2026-11-23 |
| [Gist: Proton WireGuard config with IPv6 lines (2026-03-07)](https://gist.github.com/mikaeldui/12127c91ccece42ea375c9f42d94aa8a) | 2026-11-23 |
| [Proton VPN: OpenVPN on Linux](https://protonvpn.com/support/linux-openvpn/) | 2026-11-23 |
| [ArchWiki: ProtonVPN](https://wiki.archlinux.org/title/ProtonVPN) | 2026-11-23 |
| [Netgate forum: two Proton WireGuard tunnels](https://forum.netgate.com/post/1078523) | 2026-11-23 |
| [OPNsense forum: two Proton WireGuard tunnels](https://forum.opnsense.org/index.php?topic=52712.0) | 2026-11-23 |
| [Proton VPN: advanced kill switch](https://protonvpn.com/support/advanced-kill-switch) | 2026-11-23 |
| [Proton VPN: split tunneling](https://protonvpn.com/support/protonvpn-split-tunneling) | 2026-11-23 |
| [Proton VPN: plans](https://protonvpn.com/support/proton-vpn-plans) | 2026-11-23 |

## Findings

### What a Proton file looks like

- WireGuard: Proton writes comments under `[Interface]`: `# Bouncing = 0`, `# NetShield = 1`,
  `# Moderate NAT = off`, `# NAT-PMP (Port Forwarding) = off`, `# VPN Accelerator = on`; then
  `Address = 10.2.0.2/32` and `DNS = 10.2.0.1`. The `[Peer]` has a server comment such as
  `# NO#21` and an `Endpoint` on port 51820 (OPNsense how-to).
- Since 2026-03-07 the files also carry IPv6 values, so `Address` and `DNS` may list several
  values (gist).
- Proton does not document this format: it can change without notice.
- OpenVPN: files are named like `node-ch-02.protonvpn.net.udp.ovpn` (Proton Linux guide) and
  carry `verify-x509-name "us-ny-291.protonvpn.net" name` (ArchWiki). Whether `remote` uses a
  hostname or an IP in current files: **no verificado**; the detector accepts either directive.

### Why the file and the app clash

- Every Proton WireGuard file uses the same tunnel address 10.2.0.2; two tunnels on one machine
  conflict unless the addresses differ (Netgate and OPNsense forums). Proton itself says to
  change a second tunnel to `10.3.0.2/32` with DNS `10.3.0.1` (Proton WireGuard page).
- Whether the Windows app also uses 10.2.0.2 on its own adapter: **no verificado** (likely, as
  it uses the same servers and account).
- The advanced kill switch blocks everything outside Proton's interface with persistent WFP
  rules (Proton kill switch page). Whether it blocks another VPN adapter such as a direct
  WireGuard tunnel: **no verificado**. On Windows, split tunneling can include or exclude apps
  and IPs (Proton split tunneling page).
- Plans: free allows 1 connection, Plus 10 (Proton plans page). A free account with the app
  and a file connected at once uses two connections: **no verificado** what Proton does then.

## What Kongtrol does with it (ADR-0103)

- Detection from the file at runtime, no config field: `profileimport.ProviderForContent`
  (two or more of the comment keys plus a 10.x.0.2 address or 10.x.0.1 DNS; or a host ending in
  `.protonvpn.net` in `verify-x509-name`/`remote`).
- Refuse a direct Proton profile while the Proton app tunnel is up; hand over (disconnect the
  profile) when the app connects; refuse two direct tunnels with one address; hint at the
  advanced kill switch when a split profile stops answering while the app is up.

## Pending

- Real Windows test: Proton app connected + direct Proton `.conf` (addresses in use, which
  side loses traffic) and Proton's advanced kill switch next to a non-Proton WireGuard tunnel
  (issue #51).
