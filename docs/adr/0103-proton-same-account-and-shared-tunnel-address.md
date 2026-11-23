# ADR-0103: A Proton file never runs next to the Proton app; one tunnel address, one tunnel

Version 1.0.0 · 2026-11-23

Status: Accepted · Date: 2026-11-23 · Decision-makers: Ruben Luna (owner), implemented by agent

Refines [ADR-0099](0099-refuse-second-full-tunnel-and-switch-aware-kill-switch.md).

## Context and Problem Statement

Issue #51: a direct WireGuard profile made from a Proton account's `.conf` breaks when the
Proton VPN app is connected too. ADR-0099 only refuses a second **full** tunnel, and a direct
WireGuard profile without `default_egress` is a split tunnel (the daemon replaces AllowedIPs
with the rule destinations), so this case was not detected. The file and the app use the same
account and, for WireGuard, the same tunnel address 10.2.0.2: up together they cut each other's
connections, split tunnel or not. Two direct Proton files have the same problem with each
other ([research note](../research/2026-11-23-proton-same-account.md)).

## Decision Drivers

- Never leave the user with tunnels that silently do not work; one notice with what to do.
- No new config field: the user should not have to know which provider issued a file.
- Proton's file format is undocumented; the detection is a hint, never a security decision.
- Reuse ADR-0099's refusal path (409, watchdog waits for the user).

## Considered Options

1. Only document it (guide, SETUP).
2. A config field `provider: protonvpn` on direct profiles.
3. Detect the provider from the file at runtime and refuse / hand over.

## Decision Outcome

Chosen: option 3.

- Detection (`internal/profileimport/provider.go`, `ProviderForContent`): a `.conf` with two or
  more of Proton's comment keys (NetShield, VPN Accelerator, Moderate NAT, NAT-PMP, Bouncing)
  and an address 10.x.0.2 or DNS 10.x.0.1; an `.ovpn` with a host ending in `.protonvpn.net` in
  `verify-x509-name` or `remote`. `app.DirectProfileProvider(v)` reads the profile's file (cached
  by path, size and mtime). `WireGuardAddresses` lists the `[Interface]` addresses.
- Import (API, `kongtrol profile import`, `profile add --import`, wizard, panel Add sheet): the
  result carries `provider`; the API adds `hint`/`hint_key` (`profile.import.proton_hint`) and
  the sheet's Done step shows it.
- Connect (`ProfileService.CheckFullTunnelConflict`, so every path and `POST .../connect` 409):
  - a direct Proton profile while Proton's app tunnel is up on an interface that is not one of
    Kongtrol's own → `api.err.connect_proton_app` (+ `.fix`);
  - a direct WireGuard profile whose address a connected profile already uses →
    `api.err.connect_same_address` (+ `.fix`, names the address and Proton's 10.3.0.2 advice).
  Both are `*app.FullTunnelConflictError` wrapping `app.ErrFullTunnelConflict`, so the
  watchdog waits for the user; its `reconnect_blocked` notice now names the reason
  (`watchdog.event.reconnect_blocked_proton|_address`).
- Handover (`cmd/kongtrol/daemon_proton.go`, every 5 s, the provider tunnels are listed only
  while a direct Proton profile is connected): when Proton's app connects, the daemon
  disconnects the direct profile itself through `ProfileService.Disconnect` (marked intended:
  no watchdog retry, no kill switch block) but **not** as a user disconnect, so it stays in
  `wanted.json`. One notice, `CanConnect` ("Reintentar"):
  `watchdog.event.proton_handover` naming the `protonvpn` profile when one exists, otherwise
  `watchdog.event.proton_handover_add` with `Offer: "add_provider"` (last arg `protonvpn`;
  the button is rendered by ADR-0102's work).
- Hint: a split direct non-Proton profile that goes `no_response` or fails a reconnect while
  Proton's app is up gets `watchdog.event.proton_blocks_other` once per outage (Proton's
  "advanced kill switch" may block other VPNs: turn it off or add the profile in "Split
  tunneling"). That the kill switch blocks another VPN adapter is **no verificado**.

## Consequences

- Good: the Proton file and app never fight silently; the user is told which one carries the
  traffic and how to switch back.
- Bad: detection depends on Proton's undocumented comments; a file stripped of them is not
  detected (then only the shared-address check applies, for WireGuard).
- Bad: the handover disconnects a profile the user turned on; it is the only way both keep
  working, and "Reintentar" plus `wanted.json` keep the way back.

## Confirmation

- `internal/profileimport/provider_test.go`: detection table and address parsing.
- `internal/app/proton_test.go`: provider from file, refusal while the app is up (not on
  Kongtrol's own interface), shared address refusal, handover list, kill switch hint.
- `cmd/kongtrol/daemon_proton_test.go`: handover disconnects once with notice A/B; the
  `reconnect_blocked` notice keys.
- `internal/api/handlers_proton_test.go`: import hint, 409 keys.
- `web/ui/src/components/add/AddSheet.test.tsx`: Done step shows the Proton hint.
- Real Windows test (10.2.0.2 used by the app, kill switch next to WireGuard): **no verificado**
  (issue #51).

## More Information

[Research note](../research/2026-11-23-proton-same-account.md) ·
[ADR-0099](0099-refuse-second-full-tunnel-and-switch-aware-kill-switch.md) ·
code: `internal/app/proton.go`, `internal/profileimport/provider.go`,
`cmd/kongtrol/daemon_proton.go`, `internal/i18n/proton.go`, `web/ui/src/i18n/pages/proton.ts`.
