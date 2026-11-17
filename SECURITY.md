<img src="assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Security Policy

Version 1.1.0 · 2026-11-17

The full policy (Spanish and English) lives at **https://kongtrol.app/seguridad**
(English: https://kongtrol.app/en/security). Machine-readable contact:
https://kongtrol.app/.well-known/security.txt ([RFC 9116](https://www.rfc-editor.org/rfc/rfc9116)).

## Reporting a vulnerability

Report privately through GitHub's private vulnerability reporting:
**https://github.com/DerotLuna/vpn-kongtrol/security/advisories/new**

Do not open a public issue. Include the Kongtrol version (`kongtrol version`), your OS, the
component (CLI, background service, tray, dashboard, privileged helper, website), steps to
reproduce and the impact. Never include real passwords, your full `kongtrol.yaml` or
`~/.kongtrol/api-token`.

## What to expect

Kongtrol is a free, open-source, non-profit project maintained by one person, so responses are
best effort: I will try to answer as soon as possible, but there are no guaranteed timelines.

- I will confirm receipt and, later, whether the issue is confirmed and how severe it is.
- The fix and its release are coordinated with you; details are published once the fix is
  available. Credit in the advisory if you want it.

General questions (not vulnerabilities): [GitHub issues](https://github.com/DerotLuna/vpn-kongtrol/issues).

## Good-faith research

If you research in good faith, on your own machines, without accessing other people's data and
following this policy, I will not take action against you and will work with you to understand and
fix the issue.

## How Kongtrol is secured

Network connections Kongtrol makes, the privileged helper and its leases, kill switch and DNS guard
internals, and recovery procedures: [docs/SECURITY.md](docs/SECURITY.md).

## Supported versions

Kongtrol is in beta: only the latest release receives security fixes.

## Scope

In scope: this repository and the latest release binaries — traffic/DNS leaks the kill switch,
DNS guard or rules should prevent; privilege escalation via the helper or background service;
access to the local dashboard/API without the token or from another machine; credentials
exposed outside the OS credential store; undetected audit-log tampering; vulnerabilities in
kongtrol.app.

Out of scope: VPN providers' own clients and services, attacks that already require
administrator control of the machine, volumetric DoS, social engineering, missing headers with
no demonstrable impact.
