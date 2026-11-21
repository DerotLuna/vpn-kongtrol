# ADR-0087: Normalize DNS names in VPN rules

Version 1.0.0 · 2026-11-21

Status: Accepted · Date: 2026-11-21

## Decision

Accept exact hostnames and one-label `*.` patterns in lowercase, uppercase or mixed
case. Compare the optional root dot as equivalent and preserve it in direct DNS
lookups. Convert international names to their ASCII spelling using the IDNA lookup
profile in Go and URL host parsing in the UI. Share the existing classifier fixtures.

Use lowercase, real-looking examples for exact domains, exact subdomains and
wildcards. Use documented address prefixes for IPv4 and IPv6 examples. Keep the
existing resolver's common-name probes and tell users to enter exact names for
required destinations.

See the [supporting research](../research/2026-11-21-policy-domain-inputs.md).
