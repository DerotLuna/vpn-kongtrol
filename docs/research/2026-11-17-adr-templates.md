<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: ADR templates (Nygard vs MADR)

Version 1.0.0 · 2026-11-17

## Question

Which Architecture Decision Record format should Kongtrol use to record every decision?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Michael Nygard (Cognitect) — Documenting Architecture Decisions (2011) | https://www.cognitect.com/blog/2011/11/15/documenting-architecture-decisions | 2026-11-17 |
| 2 | MADR — Markdown Architectural Decision Records | https://adr.github.io/madr/ | 2026-11-17 |
| 3 | adr.github.io — ADR templates overview | https://adr.github.io/adr-templates/ | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- Nygard [1]: sections Title, Context, Decision, Status, Consequences. Statuses "proposed",
  "accepted", "deprecated", "superseded". "ADRs will be numbered sequentially and monotonically.
  Numbers will not be reused." Location suggested: `doc/arch/adr-NNN.md`.
- MADR [2]: current release 4.0.0 (2024-09-17). Sections: Context and Problem Statement,
  Decision Drivers, Considered Options, Decision Outcome (with Consequences and Confirmation),
  Pros and Cons of the Options, More Information; several are optional. Filenames follow
  `NNNN-title-with-dashes.md`. License: MIT OR CC0-1.0.
- adr.github.io [3] lists MADR, Nygardian ADRs and Y-Statements as the common templates.

## Confidence

High: both primary pages loaded.

## Choice

MADR 4.0.0. It keeps Nygard's core (context, decision, status, consequences, immutable numbered
records) and adds explicit considered options and confirmation, which the owner's rule
("alternatives", "everything verified") needs. Nygard alone has no alternatives section.
MADR's optional YAML front matter is not used: the org rule puts the logo first in every file,
so status and date go on a plain line instead.

## Open questions

- None.

## Used by

- [ADR-0001](../adr/0001-record-decisions-with-madr.md)
