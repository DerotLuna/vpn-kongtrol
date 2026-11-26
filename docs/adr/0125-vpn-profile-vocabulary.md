# ADR-0125: VPN profile vocabulary

Status: Accepted · Date: 2026-11-26 · Decision-maker: owner

Research: [VPN profile vocabulary](../research/2026-11-26-vpn-profile-vocabulary.md).

## Decision

Call the saved named configuration a VPN profile; multiple profiles can use
the same provider. Where profiles must be distinguished, show the provider
and app or protocol when known. A protocol alone does not identify a provider;
when the provider is unknown, show only the known technology.

A rule expresses the user's destination and behavior choice. A route is an
actual network path, including those from a file, server or system. Keep
"Agregar regla de salida" for the action established by ADR-0124. The routes
section is grouped by profile, not by provider.
