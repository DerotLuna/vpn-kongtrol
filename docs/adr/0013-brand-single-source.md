<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0013: Brand assets — single source in `assets/brand`

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Logos were copied into `assets/`, `web/dashboard/` and `landing/public/`. Copies drift and Go's
`//go:embed` cannot use `../` paths, which pushed people to duplicate files.

## Decision Drivers

- One source for every logo size and format.
- Respect `//go:embed` path rules.
- Reproducible derived sizes.

## Considered Options

1. Copies per consumer — drift.
2. `assets/brand/` as single source with a generator; consumers embed from it or copy at build
   time; duplicates never committed.

## Decision Outcome

Chosen: **option 2**. `assets/brand/` holds `logo.svg`, `logo-kong.svg`, PNG sizes
(`logo-kong-32/64/192/512.png`, `logo-512.png`), `favicon.svg`, `og-1200x630.jpg`, `fonts/` and
`source/`; regenerate with `go run assets/brand/generate.go`. `assets/embed.go` embeds from
`assets/brand/`; `TrayIcon(size)` in `assets/icon.go`. Web clients get logos served by the Go side
or copied at build time. Rule in `AGENTS.md`: "Never commit duplicate brand files".

### Consequences

- Good: one edit updates every surface.
- Bad: `landing/public/` still holds logo files at the time of writing (`logo.png`, `logo.svg`,
  `logo-kong*.png`) — whether they are build-time copies or committed duplicates was **no
  verificado** here; check `.gitignore` / landing scripts.

### Confirmation

Review rejects new logo files outside `assets/brand/`.

## More Information

- Code: `assets/brand/generate.go`, `assets/embed.go`, `assets/icon.go`
- `AGENTS.md`: "`//go:embed` constraints", Key subsystems (`assets/`)
- No external sources: internal design decision.
