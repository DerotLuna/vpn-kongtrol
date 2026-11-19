<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0048: Tagline "Todas tus VPN bajo Kongtrol"

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

ADR-0047 made "Kongtrol" the only name and put "VPN" in a tagline, "Kongtrol · tus VPN bajo
control" / "Kongtrol · your VPNs under control". The landing footer and share title still said
"Todas tus VPN en un solo lugar". The name comes from "control"; the owner wants the tagline to
carry that play on words.

## Considered Options

1. "Kongtrol · tus VPN bajo control" (ADR-0047).
2. "Tus VPN bajo Kongtrol".
3. **"Todas tus VPN bajo Kongtrol" / "All your VPNs under Kongtrol".**

## Decision Outcome

Option 3, proposed by the owner: "Kongtrol" stands where "control" would, and "todas" keeps the
multi-VPN promise the old footer line made. It contains the name, so titles use it alone, without
a "Kongtrol ·" prefix. Used in the landing home title (`landing/src/seo.ts`), footer and share
title, the tray tooltip (`tray.tooltip`), the README lead, `PRODUCT.md` and `DESIGN.md`. This
refines point 1 of ADR-0047 (tagline wording only); the rest of ADR-0047 stands.

### Consequences

- Good: one line that names the product and says what it does.
- Neutral: the hero headline keeps "Todas tus VPN en un solo lugar." as plain description.
