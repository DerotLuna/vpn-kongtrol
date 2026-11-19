<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: cross-vendor agent instructions (`AGENTS.md`)

Version 1.0.0 · 2026-11-17

## Question

Can one instructions file serve every coding agent (Claude Code, Codex, Copilot, Gemini CLI, …)
instead of one file per vendor?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | AGENTS.md | https://agents.md/ | 2026-11-17 |
| 2 | Claude Code docs — How Claude remembers your project | https://code.claude.com/docs/en/memory | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- [1]: "A simple, open format for guiding coding agents." Lists compatible tools including
  OpenAI Codex, Google Jules, Aider, VS Code, Cursor, GitHub Copilot's coding agent, Windsurf
  and Gemini CLI. "AGENTS.md is now stewarded by the Agentic AI Foundation under the Linux
  Foundation."
- [2]: "CLAUDE.md files can import additional files using `@path/to/import` syntax." With both
  files present, Claude Code reads "Your `CLAUDE.md` files only"; with a `CLAUDE.md` that
  imports `AGENTS.md`, it reads "Your `CLAUDE.md`, with `AGENTS.md` included through the
  import".

## Conclusions (applied)

`AGENTS.md` holds all guidance; root `CLAUDE.md` contains only `@AGENTS.md`, so Claude Code and
the other tools read the same text and nothing drifts.

## Confidence

High (primary pages).

## Open questions

- None.

## Used by

- [ADR-0002](../adr/0002-agents-md-cross-vendor-instructions.md)
