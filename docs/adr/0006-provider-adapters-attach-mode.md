<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0006: Provider adapters with attach mode; direct WireGuard/OpenVPN way

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Users already run official VPN apps (FortiClient, Proton VPN, NordVPN, Surfshark). Kongtrol must
manage those tunnels (status, watchdog, DNS, policies) without replacing the apps or breaking
their terms, and must survive app updates.

## Decision Drivers

- Work with the official app; no reverse-engineered protocols.
- Bringing the connection up is separate from managing the tunnel.
- Automation is best effort and never the only path.

## Considered Options

1. Re-implement each provider's protocol/login — fragile, against provider terms (not researched).
2. GUI-automate every app — breaks on every app update.
3. Attach mode: detect the tunnel by network interface and attach; automate only via an
   official CLI (or GUI automation for corporate clients with no alternative); otherwise ask the
   user and wait. Keep a direct `wireguard` / `openvpn` way with downloaded configs.

## Decision Outcome

Chosen: **option 3** (`AGENTS.md`, "Provider adapters vs direct configs").

- `up`: tunnel already up → attach; else if `auto_connect` (default true) → official CLI (or GUI
  automation for corporate clients); any failure falls through to: launch the app, tell the user
  what to do, wait (`wait_timeout`, default 5m) and attach when the interface appears.
- Never GUI-automate a consumer VPN app.
- `down` without a CLI asks the user to disconnect in the app and waits; the watchdog notifies
  (`manual_reconnect_required`) instead of reconnecting.
- Per provider (verified 2026-11-17, see research):
  - NordVPN: official CLI — Windows `nordvpn -c -g "<country>"` / `nordvpn -d` from the install
    folder; Linux `nordvpn connect` / `nordvpn disconnect`.
  - Proton VPN: Linux CLI `protonvpn` (`connect --country`, `disconnect`; 1.0.0 released
    2026-04-07); Kongtrol falls back to `protonvpn-cli` if present. On Windows use the direct
    way (`type: wireguard` with Proton's downloadable `.conf`).
  - Surfshark: current Linux app is GUI-only; legacy `surfshark-vpn` exists. No CLI found for
    Windows/macOS — **no verificado** as an explicit vendor statement. So `AutoConnect` /
    `Disconnect` are nil: attach + manual. Manual WireGuard configs available with a subscription.

### Consequences

- Good: resilient to app updates; legal; one attach engine for all providers.
- Bad: some flows need a user click; interface names for Surfshark and macOS Network Extension
  are best effort, so `interface:` can override.

### Confirmation

`internal/vpn/appattach/session_test.go`, provider `adapter_test.go` files.

## More Information

- Code: `internal/vpn/appattach/`, `internal/vpn/{nordvpn,protonvpn,surfshark,forticlient}/`,
  `internal/config/semantics.go` (`validateProvider`)
- Research: [provider CLIs](../research/2026-11-17-provider-clis.md)
