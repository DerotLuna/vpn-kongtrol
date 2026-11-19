<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0059: CI runs independent jobs in parallel and cancels superseded runs

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

The owner asked for faster pipelines without losing checks or security. Measurements of the last
runs ([research note](../research/2026-11-19-ci-parallelism.md)) show jobs waiting for work they do
not use: Go tests wait for the UI unit tests, the release tests wait for the macOS and Windows
builds, the Pages redirect waits for the landing build, and the workflow-wide `pages` group queues
landing PR builds behind deploys. Nothing cancels a run made stale by a newer push.

## Decision Drivers

- Never drop a check, a signature or a published file (ADR-0025 stays as is).
- Least privilege and per-step secrets stay; prefer less code under write tokens.
- Build only what changed (path filters and the `ci-ok` gate keep working for skipped jobs).
- Standard GitHub Actions features only; no AI in tooling.

## Considered Options

1. Keep the pipelines as they are.
2. **Remove waits on unused output, split mixed jobs, cancel superseded runs on PRs and branches
   (never tags/releases).**
3. Also drop the real embedded UI from the Go tests, or split GoReleaser (Pro-only split/merge).

## Decision Outcome

Option 2:

- `ci.yml`: `ui` becomes `ui-build` (build, handed to the Go tests) and `ui-check` (lint + unit
  tests), in parallel; `test` waits only for `ui-build`; `ci-ok` requires both. `security` stops
  saving the shared setup-go cache; `race` gets its own cache key. The six packaging
  cross-compiles run at once. `timeout-minutes` on every job that runs code. Workflow
  `concurrency` per PR (cancel superseded runs) and per commit on `main` (never cancelled), so every commit on `main` keeps its own result.
- `release.yml`: the `go test -race` step moves to a `test` job (needs `ui`, read-only token) that
  runs beside `macos` and `windows`; `release` needs it, so nothing is published unless it passes.
  No concurrency group: a release is never cancelled.
- `landing.yml`: concurrency per PR/ref, cancel only on PRs; the Pages deploy keeps its own
  non-cancelling `pages` group at job level and no longer waits for `build-landing` (its stubs never
  read the build).
- `docs.yml`: same job layout (non-blocking, link check dominates).
- All `actions/*` references pinned by commit SHA at the version their tag already resolved to;
  the new jobs check out with `persist-credentials: false`.

Option 3 was rejected: the first loses a check, the second needs GoReleaser Pro.

### Consequences

- Good: estimated critical paths ci ~185–199 s → ~160–170 s (UI changes), release ~358–403 s →
  ~270–305 s, landing on main ~60 s → ~45 s; release tests no longer share a job with
  `contents: write` / `id-token: write`.
- Good: runs on `main` are never cancelled, so a failure always points at its own commit. One more runner per UI change (`ui-check`) and per release (`test`).
- Not verified: the estimates come from past durations; the first runs after the merge measure
  the real numbers.

### Confirmation

actionlint 1.7.7 (with shellcheck 0.11.0) clean; zizmor 1.32.0 offline: no new finding class;
`unpinned-uses` 40 → 0; `cache-poisoning` 6 → 7 (the new release `test` job keeps the cache the
tests used inside `release`).

## More Information

- Research: [CI parallelism](../research/2026-11-19-ci-parallelism.md).
- Code: `.github/workflows/ci.yml`, `release.yml`, `landing.yml`, `docs.yml`.
- Related: ADR-0022 (landing hosting), ADR-0025 (release channels).
