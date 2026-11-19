<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: OpenVPN interactive service and non-admin options

Version 1.0.0 · 2026-11-17

## Question

How does OpenVPN on Windows let a non-admin user start tunnels, and can Kongtrol reuse it?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | OpenVPN community wiki — OpenVPNInteractiveService | https://community.openvpn.net/openvpn/wiki/OpenVPNInteractiveService | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- [1]: the interactive service "removes the need to always run OpenVPN as administrator".
- [1]: a config "must reside in the configuration folder defined by `config_dir` registry
  value… For all other folders the invoking user must be a member of local Administrators
  group" — or of `ovpn_admin_group` (default "OpenVPN Administrators").
- The page does not mention a per-user config directory being allowed for non-admins.

## Conclusions (reasoned)

- Reusing the interactive service would force users to copy `.ovpn` files into the
  admin-owned `config_dir` or join "OpenVPN Administrators" — both need an admin step per
  profile or a group change, and it exists only on Windows.
- Kongtrol's own privileged helper (one-time `kongtrol service install`) covers OpenVPN,
  WireGuard, routes, DNS and blocking on all three OSes with one model (ADR-0005), so it was
  preferred. Reasoned, not benchmarked.

## Confidence

High for the interactive service rules (official wiki).

## Open questions

- None for now.

## Used by

- [ADR-0005](../adr/0005-privileged-helper-service.md), [ADR-0007](../adr/0007-openvpn-pushed-routes-binary-cd.md)
