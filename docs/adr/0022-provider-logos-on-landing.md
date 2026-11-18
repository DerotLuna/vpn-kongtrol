<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0022: Provider logos on the landing — only where the brand's rules allow it

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

The landing's VPN section (`landing/src/components/sections/Providers.tsx`) showed each supported
VPN as a two-letter monogram, with green borders marking attach mode. The redesign groups the
cards in two titled groups ("Abres tu app y Kongtrol se adjunta" / "Kongtrol conecta directo"),
shows supported operating systems per card, and asked to use the official logos: monochrome SVGs
from Simple Icons 16.38.0 (CC0), self-hosted, drawn in `currentColor` for light and dark themes.
Can Kongtrol show those logos?

## Decision Drivers

- Simple Icons is CC0 as a project but says not every icon is, and trademark rights stay with
  each owner; each owner's policy decides
  ([research](../research/2026-11-17-provider-logo-guidelines.md)).
- Kongtrol is independent and must not suggest affiliation or endorsement.
- Local only and no new runtime dependencies: nothing is loaded from a third-party host.
- Same card style for every VPN; WCAG contrast in both themes; no layout shift.

## Considered Options

1. Official logo for every VPN.
2. Logo only where the brand's published rules allow this use; our monogram for the rest.
3. Monograms for every VPN (status quo).

## Decision Outcome

Chosen: **option 2**.

- Each brand's policy was read on 2026-11-17 (table in the research note). Only Proton's media kit
  allows its assets freely, on condition that the page refers back to proton.me. NordVPN,
  Surfshark, Fortinet, WireGuard, OpenVPN, Tailscale, Cloudflare, Cisco and Palo Alto Networks
  forbid recolouring or modifying their logos, require written permission, or reject single-colour
  versions, so they keep the monogram.
- The Proton VPN path is copied inline from Simple Icons 16.38.0 (slug `protonvpn`) into the
  component, `fill="currentColor"`, `aria-hidden="true"`, fixed `width`/`height`; the name is
  always visible text. The section's note links to `https://proton.me` (`PROTON_HOME` in
  `landing/src/links.ts`) and keeps the "sin afiliación" line.
- Any future logo follows the same rule: read the brand's policy, record the verdict in a research
  note, and only then add the path.

Option 1 was rejected: it breaks nine of the ten brands' published rules. Option 3 was rejected
because Proton explicitly allows its logo and the request was to use logos where possible.

### Consequences

- Good: no trademark use the owners forbid; the page still names every VPN.
- Good: no third-party request, no new dependency; the SVG is a few hundred bytes.
- Bad: one logo among nine monograms is less uniform; cards keep the same frame and size.
- Neutral: if a brand grants written permission later, its monogram can switch to a logo.

## Confirmation

- `pnpm typecheck`, `pnpm check:i18n` and `pnpm build` pass in `landing/`.
- Reviewers check that every logo in `Providers.tsx` has a row in the research note with a
  verdict that allows it.

## More Information

- Research: [provider logo guidelines](../research/2026-11-17-provider-logo-guidelines.md).
- Grouping follows [ADR-0006](0006-provider-adapters-attach-mode.md) (attach group = adapters on
  `internal/vpn/appattach`); FortiClient OS status follows
  [ADR-0020](0020-forticlient-multi-version-strategies.md) and
  [docs/forticlient-support.md](../forticlient-support.md).
