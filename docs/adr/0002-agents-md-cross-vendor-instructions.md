<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0002: `AGENTS.md` as the cross-vendor agent instructions

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Several coding agents from different vendors work on this repository. Each vendor has its own
instructions file (`CLAUDE.md`, etc.). Duplicated files drift. Which file is the source of truth?

## Decision Drivers

- One text for every agent; no drift.
- Claude Code must still load it.

## Considered Options

1. `CLAUDE.md` only — Claude-specific; other tools would not read it.
2. One file per vendor, kept in sync by hand — drift risk.
3. `AGENTS.md` as the single file, `CLAUDE.md` only imports it.

## Decision Outcome

Chosen: **option 3**. `AGENTS.md` holds all guidance; the root `CLAUDE.md` contains only
`@AGENTS.md`. Verified 2026-11-17: AGENTS.md is "a simple, open format for guiding coding
agents" supported by Codex, Jules, Aider, VS Code, Cursor, Copilot's coding agent, Windsurf and
Gemini CLI, and stewarded by the Agentic AI Foundation under the Linux Foundation
(https://agents.md/). Claude Code docs: a `CLAUDE.md` that imports `AGENTS.md` loads both
(https://code.claude.com/docs/en/memory).

### Consequences

- Good: one place to edit; vendor-neutral.
- Bad: Claude-only features (e.g. rule scoping in `.claude/rules/`) are not used for shared rules.

### Confirmation

`CLAUDE.md` stays a one-line import; review rejects guidance added there.

## More Information

- Code: `AGENTS.md` (line 5: "`CLAUDE.md` only imports this file"), `CLAUDE.md`
- Research: [agent instructions](../research/2026-11-17-agent-instructions.md)
