<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0005: Privileged helper service — closed op list, IPC auth, lease, fail-open

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Routes, DNS, firewall rules, WireGuard and OpenVPN need admin/root. Asking users for an elevated
terminal is unacceptable for non-technical users and "some commands need admin" is confusing
(`AGENTS.md`, "No admin terminal"). User-session parts (tray, keychain, GUI automation) cannot
run as a service. How do unprivileged clients get privileged network work done safely?

## Decision Drivers

- All or nothing: never "some commands need admin".
- Least privilege: the privileged side must not be a generic "run as root" proxy.
- A crashed daemon must never leave the user offline.
- One model on Windows, Linux and macOS.

## Considered Options

1. Elevated terminal / run everything as admin — rejected by the product rule.
2. OpenVPN interactive service (Windows) — OpenVPN only; configs must sit in the admin
   `config_dir` or the user must join "OpenVPN Administrators"
   ([research](../research/2026-11-17-openvpn-non-admin.md)).
3. Add the user to "Network Configuration Operators" (Windows) — documented rights cover TCP/IP
   and DNS settings only; firewall/services not granted per the page (limits **no verificado**)
   ([research](../research/2026-11-17-windows-network-configuration-operators.md)).
4. Own helper service installed once, running a closed list of operations.

## Decision Outcome

Chosen: **option 4**. `kongtrol service install` installs `internal/privhelper`; all privileged
work goes through `privops.Default()` (`Local` when elevated, `Client` otherwise).

- Closed op list (`internal/privops/protocol.go`): `ping`, `route.add|delete|flush`,
  `dns.set|restore`, `killswitch.enable|disable`, `wg.install|remove|allowed_ips`,
  `openvpn.start|stop|status`, `hosts.write`, `lease.renew`, `block.set`. Inputs validated in
  `internal/privops/validate.go`.
- IPC auth: Windows named pipe with SDDL "owner Administrators; deny network logons; full access
  for SYSTEM and Administrators; read/write for the installing user SID"
  (`internal/privhelper/platform_windows.go`); Unix socket with kernel peer credentials
  (`SO_PEERCRED` / `LOCAL_PEERCRED`). Rule (`server.go` `authorize`): the installing user or an
  elevated admin/root.
- Only root/admin-owned binaries run; Homebrew binaries refused (user-swappable dylibs).
- Versioned protocol (`privops.ProtocolVersion = 3`); an older helper is reported as outdated.
- Lease per client process: renew every 5 s, expire after 20 s (`internal/privops/lease.go`).
  Expiry undoes OpenVPN, DNS, hosts, blocked destinations and kill switch — **fail-open**, so a
  crashed daemon never leaves the user offline. WireGuard and routes are not leased.

### Consequences

- Good: no elevated terminal; small, auditable privileged surface; audit log (`privhelper/audit.go`).
- Bad: one admin prompt at install; a helper/daemon version skew needs re-install.
- Fail-open trades kill-switch strictness for availability when the daemon dies (deliberate).
- macOS OpenVPN via the helper needs a bundled signed `openvpn` (pending).

### Confirmation

`internal/privops/*_test.go` (protocol, lease, validate), `internal/privhelper/server_test.go`.

## More Information

- `AGENTS.md`: "No admin terminal (target)", "Privileged helper internals"
- Research: [OpenVPN non-admin](../research/2026-11-17-openvpn-non-admin.md),
  [Network Configuration Operators](../research/2026-11-17-windows-network-configuration-operators.md),
  [kill switch](../research/2026-11-17-kill-switch-mechanisms.md)
