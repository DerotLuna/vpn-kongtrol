<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: VPN kill switch mechanisms and semantics

Version 1.0.0 · 2026-11-17

## Question

How do OSes and mature VPN clients implement a kill switch, and what should it mean when only
some traffic goes through a tunnel (split tunneling) and when the user disconnects on purpose?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Microsoft Learn — Windows Firewall rules | https://learn.microsoft.com/en-us/windows/security/operating-system-security/network-security/windows-firewall/rules | 2026-11-17 |
| 2 | Microsoft Learn — WFP Filter Arbitration | https://learn.microsoft.com/en-us/windows/win32/fwp/filter-arbitration | 2026-11-17 |
| 3 | wireguard-windows — netquirk.md | https://git.zx2c4.com/wireguard-windows/plain/docs/netquirk.md | 2026-11-17 |
| 4 | Tailscale docs — Exit nodes | https://tailscale.com/docs/features/exit-nodes | 2026-11-17 |
| 5 | Tailscale GitHub issue 19781 (contributor comment, not docs) | https://github.com/tailscale/tailscale/issues/19781 | 2026-11-17 |
| 6 | Proton VPN — What is a kill switch | https://protonvpn.com/support/what-is-kill-switch | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- Windows Defender Firewall [1]: "Explicit block rules take precedence over any conflicting
  allow rules." "Outbound rules follow the same precedence behaviors." So "block everything +
  allow exceptions" cannot be built from firewall rules: the block wins. An "authenticated
  bypass" that overrides block rules exists per a search snippet only — **no verificado**
  (primary page not fetched); Kongtrol does not use IPsec authentication anyway.
- WFP [2]: sub-layers are evaluated from highest to lowest priority, filters by weight inside
  each. "Block" overrides "Permit" and is final; a hard permit can only be overturned by a
  callout "Veto". A WFP-based kill switch can therefore express "permit tunnel, block rest" by
  weights inside its own sub-layer, which Firewall rules cannot.
- wireguard-windows [3]: "If an interface has only one peer, and that peer contains an Allowed
  IP in `/0`, then WireGuard enables a so-called "kill-switch", which adds firewall rules…";
  "All other packets are blocked" (tunnel, loopback, DHCP/NDP and DNS to configured servers
  allowed); "The UI's editor has a checkbox that toggles this." The page does not name WFP nor
  the checkbox label — label **no verificado**.
- Tailscale [4]: the exit node docs have no kill switch feature. `--exit-node-allow-lan-access`
  keeps LAN traffic local while "all remaining internet traffic continues to route through the
  exit node". Fail-closed when the exit node is offline is only a contributor statement in
  [5] ("traffic is expected to stop flowing") — not official docs.
- Proton [6]: "Standard kill switch is only activated when your connection drops by accident. So
  it doesn't block your internet connection if you deliberately disconnect." Advanced kill
  switch "only allows internet access when connected to Proton VPN, no exceptions" and survives
  restarts.

## Conclusions for Kongtrol (reasoned)

- Distinguish intentional disconnect (no block) from unexpected drop (block), like Proton's
  standard kill switch [6].
- With split tunneling, block only what was protected by the dropped tunnel (its routes and
  policy destinations), not the whole internet.
- On Windows, using Firewall rules, only block rules are usable [1]; a full-tunnel block must be
  expressed as "block the complement of the exceptions". A WFP implementation [2] would allow
  permit/block weights but is more code; not chosen now.

## Confidence

High for [1], [3], [6] (primary docs). Medium for Tailscale (docs silent; issue comment only).

## Open questions

- Whether to move Windows blocking to a WFP sub-layer later (dynamic session filters remove
  themselves if the process dies — **no verificado**, not researched here).

## Used by

- [ADR-0009](../adr/0009-kill-switch-split-tunneling.md), [ADR-0005](../adr/0005-privileged-helper-service.md)
