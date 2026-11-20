<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0080: A .conf is imported as WireGuard or OpenVPN by its content

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

The owner noticed (2026-11-20) that the panel says ".conf (WireGuard) o .ovpn (OpenVPN)", but
OpenVPN GUI on Windows keeps its profiles as `.conf` in `%USERPROFILE%\OpenVPN\config` (often next
to separate `ca.crt` / `.key` files). `profileimport.TypeForFile` mapped every `.conf` to WireGuard,
so importing such a file failed with "not a WireGuard file". OpenVPN's own documentation names
`client.ovpn` on Windows and `client.conf` on Linux for the same profile (no verificado as a single
quotable source; it is the OpenVPN 2.x convention, e.g. https://openvpn.net/community-resources/how-to/ ,
section "Creating configuration files for server and clients", accessed 2026-11-20).

## Decision Outcome

- `profileimport.TypeForContent(path, content)` (Go) and `typeForContent` (`web/ui/src/components/add/model.ts`)
  decide once the file is read: `.ovpn` → OpenVPN; a `.conf` with an `[Interface]` line → WireGuard;
  a `.conf` with OpenVPN directives (a line starting `client`, `remote`, `proto`, `tls-client`,
  `<ca>`, or `dev tun…` / `dev tap…`) → OpenVPN; otherwise WireGuard, and the validators say what is
  wrong. `TypeForFile` stays the pre-read hint (accepted extensions).
- Used by the panel's file step, `POST /api/v1/vpns/import` with `"path"` (only when the caller left
  `type` empty), `kongtrol profile import` and the root-only read (`ReadProtected`, before validating).
- An OpenVPN file that points to separate certificate files is still not imported (a copy would
  lose them, ADR-0042); the "what to do" line now leads with the better option: create the OpenVPN
  profile with the file where it is — openvpn runs with `--cd <its folder>`, so relative `ca` /
  `cert` / `key` paths resolve like OpenVPN GUI. The profile field is "Archivo de OpenVPN (.ovpn o
  .conf)" and its hint no longer asks to move the file.
- WireGuard for Windows (the GUI) and Kongtrol use the same mechanism, `wireguard.exe` tunnel
  services (`WireGuardTunnel$<name>`, `internal/privops/sys_windows.go`); nothing changes there.

## More Information

- Tests: `internal/profileimport/profileimport_test.go` (`TestTypeForContent`),
  `web/ui/src/components/add/model.test.ts`.
- Refines: [ADR-0042](0042-dashboard-api-completeness.md).
