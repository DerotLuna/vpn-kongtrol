<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: RFC 9116 `security.txt`

Version 1.0.0 · 2026-11-17

## Question

What must Kongtrol's machine-readable security contact contain, and where?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | RFC 9116 — A File Format to Aid in Security Vulnerability Disclosure | https://www.rfc-editor.org/rfc/rfc9116 | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- `Contact`: "This field MUST always be present".
- `Expires`: "MUST always be present and MUST NOT appear more than once"; "It is RECOMMENDED
  that the value of this field be less than a year into the future".
- Location: "MUST place the "security.txt" file under the "/.well-known/" path".

## Applied

`landing/public/.well-known/security.txt`: `Contact` = GitHub private vulnerability reporting,
`Expires: 2027-11-16T23:59:59.000Z` (under one year), `Preferred-Languages: es, en`,
`Canonical`, `Policy: https://kongtrol.app/seguridad`. The file must be renewed before it expires.

## Confidence

High (the RFC).

## Open questions

- Who renews `Expires` and when (no reminder automation exists — **no verificado** whether one
  is planned).

## Used by

- [ADR-0016](../adr/0016-legal-contact-model.md)
