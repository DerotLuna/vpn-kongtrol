# ADR-0122: Rule suggestions and notices

Status: Accepted · Date: 2026-11-26 · Decision-maker: owner

Research: [rule suggestions and notices](../research/2026-11-26-rule-suggestions-and-notices.md).
Refines ADR-0055 and ADR-0102.

## Decision

- Both profile creation paths show the existing per-profile action when its
  VPN is off. The default stays `allow`; choosing `connect` blocks known rule
  destinations and attempts connection under the existing detection policy.
- The rule sheet offers Claude and ChatGPT suggestions. They append editable
  destinations, preserve existing inputs and materialize previously selected
  subdomains. They neither activate nor save a rule, and they warn that they
  are incomplete. CLI users can supply the same destinations to `policy add`.
- A header bell opens Activity's in-memory event history. Informational
  toasts expire; warnings remain until dismissed or a known recovery resolves
  them. Historical retry actions are hidden when the profile is connected or
  connecting. Existing offer actions also depend on current security state.

## Confirmation and limitations

Automated tests cover both profile creation paths, editable suggestions and
notice resolution. Provider domain lists and routing coverage can change;
the suggestions are not verified as complete.
