<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0003: One product, GUI-first, CLI-complete

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Kongtrol started as a CLI and grew a tray app and a web dashboard. Non-technical users rely on
it; developers want automation. Should there be separate "user" and "dev" products, and which
surface is primary?

## Decision Drivers

- Non-technical users must be able to install, set up and connect without a terminal.
- Automation needs a complete, scriptable interface.
- Features must not fork between surfaces.

## Considered Options

1. CLI-first, GUI as optional extra — excludes non-technical users.
2. Two products (consumer GUI app, dev CLI) — duplicated work, divergent features.
3. One product: GUI-first (installer, tray, dashboard), CLI with full parity.

## Decision Outcome

Chosen: **option 3** (`AGENTS.md`, "Product model"): "One product, GUI-first, CLI-complete.
Everyone (devs included) uses the app… The CLI has full parity for automation. No feature may
exist only in the terminal (or only in the GUI)." Companion rule "User guidance": every command
states what it does and the next step; every error adds one "what to do" line; the daemon sends
phase + i18n key/args and clients render.

### Consequences

- Good: one feature set; docs and tests cover one behaviour.
- Bad: every feature costs work in CLI, tray, both dashboards, wizard, docs and landing
  ("Docs stay in sync" rule).

### Confirmation

Review checks parity across `cmd/kongtrol`, `cmd/kongtrol-tray`, `web/dashboard`, `web/ui`.

## More Information

- `AGENTS.md`: "Product model", "User guidance", "Docs stay in sync"
- Related: [ADR-0004](0004-detached-daemon-lifecycle.md), [ADR-0011](0011-dashboard-vite-react-ts.md)
