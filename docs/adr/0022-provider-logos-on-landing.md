<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0022: No third-party logos on the landing — our own monograms for every VPN

Version 1.1.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

The landing's VPN section (`landing/src/components/sections/Providers.tsx`) showed each supported
VPN as a two-letter monogram, with green borders marking attach mode. The redesign groups the
cards in two titled groups ("Abres tu app y Kongtrol se adjunta" / "Kongtrol conecta directo"),
shows supported operating systems per card, and considered using the official logos: monochrome
SVGs from Simple Icons 16.38.0, self-hosted, drawn in `currentColor` for light and dark themes.
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
3. Our own monogram for every VPN, no third-party logos.

## Decision Outcome

Chosen: **option 3**, decided by the maintainer on 2026-11-18.

- Each brand's policy was read on 2026-11-17 (table in the research note). Nine of the ten
  (NordVPN, Surfshark, Fortinet, WireGuard, OpenVPN, Tailscale, Cloudflare, Cisco, Palo Alto
  Networks) forbid recolouring or modifying their logos, require written permission, or reject
  single-colour versions. Only Proton's media kit allows its assets, if the page refers back to
  proton.me.
- Every VPN gets the same monogram tile: two letters in the display font, `--text` colour, fixed
  44×44 box, `aria-hidden="true"` with the name as visible text. No brand colours: the tile is
  tinted by group with existing tokens (`--live` for attach, `--steel` for direct) in its border,
  a faint wash and a corner dot only. Hover lifts the tile 1px; off with `prefers-reduced-motion`.
- The "sin afiliación" note stays. No third-party SVG, link or asset is stored in the repo.

Option 1 was rejected: it breaks nine of the ten brands' published rules. Option 2 was rejected:
one logo among nine monograms breaks the uniform look, and Proton's condition adds an outbound link
for one card only.

### Consequences

- Good: no trademark use any owner could object to; one consistent look in both themes.
- Good: no third-party files or requests; nothing to re-check when Simple Icons or a brand changes.
- Bad: visitors recognise brands by name only, not by logo.
- Neutral: a brand's written permission would not change this by itself; that needs a new ADR.

## Confirmation

- `pnpm typecheck`, `pnpm check:i18n` and `pnpm build` pass in `landing/`.
- Reviewers check that `Providers.tsx` and `providers.css` carry no third-party SVG or brand colour.

## More Information

- Research: [provider logo guidelines](../research/2026-11-17-provider-logo-guidelines.md).
- Grouping follows [ADR-0006](0006-provider-adapters-attach-mode.md) (attach group = adapters on
  `internal/vpn/appattach`); FortiClient OS status follows
  [ADR-0020](0020-forticlient-multi-version-strategies.md) and
  [docs/forticlient-support.md](../forticlient-support.md).
