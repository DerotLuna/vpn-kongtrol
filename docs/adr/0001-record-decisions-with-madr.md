<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0001: Record decisions as MADR ADRs, research as dated notes

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Product and architecture decisions were scattered across `AGENTS.md` ("Product decisions
(settled — do not re-ask)"), commit messages and chat history. The owner's rules: everything
verified, nothing invented; every decision recorded as an ADR; every investigation recorded as
research with sources. Which format and where?

## Decision Drivers

- Alternatives and their trade-offs must be recorded, not only the outcome.
- Plain Markdown in the repo, reviewable in PRs, no tooling required.
- External facts must carry a URL and an access date.

## Considered Options

1. Nygard ADR (Title, Context, Decision, Status, Consequences) — minimal, widely known; no
   section for alternatives.
2. MADR 4.0.0 — Nygard's core plus Decision Drivers, Considered Options, Confirmation, Pros and
   Cons; optional sections; MIT OR CC0-1.0.
3. Keep decisions only in `AGENTS.md` — already loaded by agents, but no history, no
   alternatives, no status.

## Decision Outcome

Chosen: **option 2, MADR 4.0.0**, in `docs/adr/NNNN-title-with-dashes.md`, plus research notes in
`docs/research/YYYY-MM-DD-topic.md` using the template in `docs/research/README.md`.
`AGENTS.md` keeps the short operational rules and may link to ADRs.

Adaptations: no YAML front matter (the org rule puts the logo first in every file), so status,
date and decision-makers go on one plain line; every file carries `Version X.Y.Z · date`.

### Consequences

- Good: decisions are traceable to code and sources; superseding is explicit.
- Bad: one more place to keep in sync; backfilled ADRs (0002–0017) record decisions after the
  fact, so their "considered options" are reconstructed from code and `AGENTS.md`.

### Confirmation

Every new decision PR adds or supersedes an ADR and updates the index in `docs/adr/README.md`.

## More Information

- Research: [ADR templates](../research/2026-11-17-adr-templates.md)
- https://www.cognitect.com/blog/2011/11/15/documenting-architecture-decisions
- https://adr.github.io/madr/
