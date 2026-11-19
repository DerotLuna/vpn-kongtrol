<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: provider CLIs and manual configs (NordVPN, Proton, Surfshark, Cloudflare WARP)

Version 1.1.0 · 2026-11-19

## Question

Which consumer VPN providers offer an official command line to connect/disconnect (so Kongtrol
can automate without GUI automation), and which offer downloadable WireGuard/OpenVPN configs
(the direct way)?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | NordVPN — Connect to NordVPN (Windows) with Command Prompt | https://support.nordvpn.com/hc/en-us/articles/19919384880145-Connect-to-NordVPN-Windows-with-Command-Prompt | 2026-11-17 |
| 2 | NordVPN — Installing NordVPN on Linux distributions | https://support.nordvpn.com/hc/en-us/articles/20196094470929-Installing-NordVPN-on-Linux-distributions | 2026-11-17 |
| 3 | Proton VPN — Use the Linux CLI | https://protonvpn.com/support/use-linux-cli | 2026-11-17 |
| 4 | Proton VPN — Linux CLI release notes | https://protonvpn.com/support/release-notes-linux-cli | 2026-11-17 |
| 5 | Proton VPN — WireGuard configurations | https://protonvpn.com/support/wireguard-configurations | 2026-11-17 |
| 6 | Surfshark — Linux app guide | https://support.surfshark.com/hc/en-us/articles/5067279648146 | 2026-11-17 |
| 7 | Surfshark — Linux (Legacy version) | https://support.surfshark.com/hc/en-us/articles/360017418334-How-to-set-up-Surfshark-VPN-on-Linux-Legacy-version | 2026-11-17 |
| 8 | Surfshark — Ethernet sharing guide (DCO adapter) | https://support.surfshark.com/hc/en-us/articles/360008180073 | 2026-11-17 |
| 9 | Surfshark — Manual WireGuard connection on Windows | https://support.surfshark.com/hc/en-us/articles/6586553044498 | 2026-11-17 |
| 10 | Cloudflare — WARP client, Linux get started | https://developers.cloudflare.com/warp-client/get-started/linux/ | 2026-11-19 |
| 11 | Cloudflare One Client changelog | https://developers.cloudflare.com/cloudflare-one/changelog/cloudflare-one-client/ | 2026-11-19 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- NordVPN Windows [1]: run from the install folder, `cd "C:\Program Files\NordVPN\"`, then
  `nordvpn -c -g "United States"`; disconnect `nordvpn -d`.
- NordVPN Linux [2]: `nordvpn connect` / `nordvpn c`, `nordvpn connect <country_code server_number>`,
  `nordvpn disconnect` / `nordvpn d`.
- Proton Linux [3][4]: the CLI command is `protonvpn` (`protonvpn connect --country US`,
  `protonvpn disconnect`, `protonvpn signout`). First release 0.1.2 on 2025-11-14; 1.0.0 on
  2026-04-07 ("first fully-stable Proton CLI release"); latest 1.1.0 on 2026-11-02. That it
  replaces the older `protonvpn-cli` is **no verificado** (no page names the old tool); Kongtrol
  still falls back to `protonvpn-cli` on PATH (`internal/vpn/protonvpn/cli.go`).
- Proton configs [5]: account.protonvpn.com → Downloads → WireGuard configuration → `.conf`.
  Free plan note: "the only VPN option available to you is VPN Accelerator" (as fetched).
- Surfshark Linux [6][7]: the current app is a GUI; Terminal only for install/updates (package
  `surfshark`). The legacy guide documents `sudo surfshark-vpn` (`down`, `status`, `multi`).
  "No official CLI on Windows/macOS" — **no verificado**: no page states it; it is an absence
  of evidence.
- Surfshark interface [8]: "The 2nd adapter is the Surfshark Data Channel Offload adapter, or the
  DCO adapter for short."
- Surfshark manual configs [9]: VPN → Manual setup → WireGuard → generate or reuse a key pair →
  choose location → download. "For any manual WireGuard® connection, you first need an active
  Surfshark subscription."
- Cloudflare WARP [10][11] (added 2026-11-19): register once with "Register the client
  `warp-cli registration new`", then `warp-cli connect`. The changelog says a 2024 GA release was
  "the last GA release that will be supporting older, deprecated warp-cli commands" (they ran with
  a deprecation warning) and later releases removed them. That `warp-cli register` is one of those
  removed verbs is reasoned from the new `registration` subcommand, **no verificado**: no page
  names it, and the exact removal version per OS is not confirmed. Kongtrol only prints the hint
  (`internal/vpn/cloudflarewarp/cli.go`, i18n `collect.warp_info2`); it never runs a registration
  command, and names `warp-cli register` only as the verb of clients before 2024.

## Confidence

High for NordVPN and Proton (primary docs). Medium for Surfshark "no CLI" (absence of evidence).
High for `warp-cli registration new` (primary docs); medium for when `register` went away.

## Open questions

- NordVPN macOS CLI: not researched.
- Whether Proton's Windows app gets a CLI: not researched.

## Used by

- [ADR-0006](../adr/0006-provider-adapters-attach-mode.md)
