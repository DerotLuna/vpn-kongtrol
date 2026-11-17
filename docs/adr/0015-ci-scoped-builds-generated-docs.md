<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0015: CI — build only what changed, generated docs, non-blocking doc checks

Version 1.1.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

The repo holds a Go binary (with embedded `web/` and `assets/`), a React UI (`web/ui`) and the
landing. Running everything on every change wastes CI and deploys, and docs drift from code.

## Decision Drivers

- "Build only what changed" and "use resources with judgment: cut waste, not checks"
  (`AGENTS.md`).
- Docs that can be generated must be generated.
- Doc checks must never block a merge (a change may legitimately need no doc update).

## Considered Options

1. One workflow running everything — wasteful.
2. Path-scoped workflows + job-level change detection; generated docs checked in CI;
   non-blocking doc checks and a change-impact map.

## Decision Outcome

Chosen: **option 2**.

- Path filters: workflow-level `paths` plus a `changes` job (`dorny/paths-filter@v3`) gating the
  `go` and `ui` jobs (`.github/workflows/ci.yml`); `landing.yml` runs only for `landing/**`,
  `assets/brand/**`, `docs/cli.json`. `web/**` and `assets/**` trigger the Go pipeline (embedded).
- Pull requests run `ci.yml` without a workflow-level `paths` filter: a workflow skipped by
  `paths` reports no status, so a required check would stay pending and block path-scoped PRs
  (GitHub docs "Handling skipped but required checks", archived copy:
  https://docs.github.com/en/enterprise-server@3.0/github/administering-a-repository/defining-the-mergeability-of-pull-requests/troubleshooting-required-status-checks;
  https://github.com/orgs/community/discussions/44490). The tiny `changes` job gates the heavy
  jobs, and one aggregator job `ci-ok` (passes when every job succeeded or was skipped) is the
  single required check for the `main` ruleset.
- Cloudflare Pages build watch paths ([ADR-0012](0012-landing-cloudflare-pages-multilingual.md)).
- Generated docs: hidden `kongtrol gen-docs` (`go generate`, `make docs`) writes CLI/config
  reference docs from code (`internal/docgen`); `make docs-check` / `gen-docs --check` verifies.
- Blocking tests only for code correctness (i18n parity, `configs/example.yaml` loads, impact map
  covers every package). Doc freshness, links (lychee) and the change-impact map
  (`docs/impact.yml`, `internal/impact`) only warn: annotations + one sticky PR comment
  (`.github/workflows/docs.yml`).

### Consequences

- Good: fast CI, no wasted deploys; generated docs cannot drift; the impact map points at prose
  that may need an update without blocking.
- Bad: prose accuracy still depends on the author following the change-impact check in
  `AGENTS.md`.

### Confirmation

Implemented 2026-11-17: `ci.yml`, `landing.yml`, `docs.yml`, `cmd/kongtrol/gendocs.go`,
`internal/docgen/`, `internal/impact/`, `docs/impact.yml`, `.github/pull_request_template.md`.

## More Information

- Code: `.github/workflows/ci.yml`, `.github/workflows/landing.yml`, `.github/workflows/docs.yml`,
  `cmd/kongtrol/gendocs.go`, `internal/docgen/`, `internal/impact/`
- Research: [Cloudflare Pages](../research/2026-11-17-cloudflare-pages.md)
- https://docs.github.com/en/actions/writing-workflows/workflow-syntax-for-github-actions#onpushpull_requestpull_request_targetpathspaths-ignore
