<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0016: Legal and contact — individual open-source project, Apache-2.0

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Kongtrol is published publicly and touches security-sensitive areas (VPN, firewall, privileged
helper). Users and researchers need a license, terms, a privacy statement and a way to report
vulnerabilities, but there is no company and no staff behind it.

## Decision Drivers

- Honest expectations: one maintainer, no SLA.
- No personal email or phone published.
- Standard, machine-readable security contact.

## Considered Options

1. Company-style terms with response SLAs — not true for a one-person project.
2. Email contact — exposes a personal address, spam.
3. Individual open-source project, GitHub-only contact, best-effort security response,
   Apache-2.0.

## Decision Outcome

Chosen: **option 3**.

- License Apache-2.0 (`LICENSE`); terms at `/terminos` (EN `/en/terms`): no warranty, beta.
- Contact only through GitHub: vulnerabilities via private vulnerability reporting
  (`/security/advisories/new`), everything else via issues (`SECURITY.md`).
- Best effort: "maintained by one person, so responses are best effort… no guaranteed
  timelines" (`SECURITY.md`); good-faith research safe harbor.
- `/.well-known/security.txt` per RFC 9116 with `Contact`, `Expires` (under one year:
  2027-11-16), `Preferred-Languages: es, en`, `Canonical`, `Policy` (RFC verified 2026-11-17).
- Privacy at `/privacidad` (EN `/en/privacy`), consistent with the "Local only" rule
  ([ADR-0010](0010-public-ip-check-opt-in.md)).

### Consequences

- Good: honest, low-maintenance, standard.
- Bad: `security.txt` `Expires` must be renewed before 2027-11-16.
- Legal adequacy of the terms for any jurisdiction: **no verificado** (not legal advice;
  not reviewed by a lawyer).

### Confirmation

`SECURITY.md`, `landing/public/.well-known/security.txt`, landing legal pages stay in sync.

## More Information

- Code: `LICENSE`, `SECURITY.md`, `landing/public/.well-known/security.txt`,
  `landing/src/pages/{terminos,privacidad,seguridad}`
- Research: [security.txt](../research/2026-11-17-security-txt.md)
