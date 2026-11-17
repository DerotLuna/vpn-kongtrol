<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0007: OpenVPN — accept pushed routes, binary resolution, `--cd`

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

`.ovpn` files from corporate servers and providers rely on server-pushed routes and often
reference `ca`/`cert`/`key`/`tls-crypt` by relative path. Users install OpenVPN in different
places. How should the OpenVPN adapter launch `openvpn` so configs behave like in OpenVPN GUI?

## Decision Drivers

- A `.ovpn` that works in OpenVPN GUI must work in Kongtrol unchanged.
- Policy routing must add to, not replace, what the server pushes.
- Find the binary without asking the user.

## Considered Options

1. `--route-nopull` and install only Kongtrol's policy routes — tunnels without policies would
   have zero routes; breaks server intent.
2. Accept pushed routes; add policy routes on top.
3. For relative paths: rewrite the config vs run with `--cd <config dir>`.

## Decision Outcome

Chosen: **option 2** and **`--cd`**.

- Pushed routes are always accepted; `--route-nopull` is never passed (asserted in
  `internal/vpn/openvpn/process_test.go`). Policy routes are added on top, skipping a
  destination the server already routes on the tunnel interface
  ([ADR-0008](0008-policy-routing-on-tunnel-down.md)).
- `--cd <config dir>` (`process.go`, `privops/local.go`) so relative paths resolve like OpenVPN
  GUI; self-contained configs need no `auth.cert` / `auth.key`.
- Binary resolution (`FindBinary`, `binary*.go`): profile `binary_path` (must exist) → `PATH` →
  Windows registry `HKLM\SOFTWARE\OpenVPN\exe_path` / `%ProgramFiles%\OpenVPN\bin\openvpn.exe`;
  Unix `/usr/sbin`, `/usr/local/sbin`, `/opt/homebrew/sbin`.
- Not elevated: runs through the helper (`privops.OpenVPNStart`,
  [ADR-0005](0005-privileged-helper-service.md)); Kongtrol answers the management password
  prompt (`--management-query-passwords`).

### Consequences

- Good: configs behave as in OpenVPN GUI; no config rewriting.
- Bad: a server that pushes `redirect-gateway` makes the tunnel full-tunnel regardless of
  policies (user's server decides).
- Note: the code comment above the args in `internal/vpn/openvpn/process.go` says "Kongtrol
  installs no policy routes for OpenVPN tunnels", which contradicts `AGENTS.md` and
  `internal/monitor/policyresolver.go` (policy routes for every adapter). The comment looks
  stale — to fix in code (not part of this ADR change).

### Confirmation

`internal/vpn/openvpn/process_test.go`, `binary_test.go`, `logparse_routes_test.go`.

## More Information

- `AGENTS.md`: "OpenVPN adapter internals"
- Research: [OpenVPN non-admin](../research/2026-11-17-openvpn-non-admin.md)
