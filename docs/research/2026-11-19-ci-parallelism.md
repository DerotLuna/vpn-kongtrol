<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# CI: what runs in parallel and what must wait

Version 1.0.0 · 2026-11-19

Backs [ADR-0059](../adr/0059-ci-parallel-jobs-and-superseded-runs.md). Sources accessed 2026-11-19.

## Question

Which jobs of `ci.yml`, `release.yml`, `landing.yml` and `docs.yml` wait for something they do not
use, and how much time does that cost, without dropping a check, a signature or a published file?

## Measured before (GitHub API, `gh run view <id> --json jobs`)

Durations are job run time (start to end, queue excluded) unless "end" (seconds from run creation).

### ci.yml (12 push runs on main, 2026-11-19 14:17–16:37)

| Job | Typical | Notes |
|---|---|---|
| `changes` | 5–10 s | paths filter |
| `ui` | 33–46 s | `pnpm run test` alone 19–20 s; lint 1–2 s; build 1–2 s |
| `test (windows-latest)` | 123–169 s | setup-go (cache restore) **38–48 s**, `go test` 75–78 s |
| `test (ubuntu-latest)` / `(macos-latest)` | 92–110 s / 88–137 s | macOS queued up to 600 s in 2 runs (runner availability) |
| `race` | 100–121 s | `go test -race` 94–95 s |
| `security` | 17–22 s | |
| `packaging` | 97–124 s | six sequential cross-compiles 102 s |

Dependency graph: `changes → ui → test(3 OS) → ci-ok`; `changes → race | security | packaging → ci-ok`.
Critical path when Go and UI change: `changes → ui → test (windows)` = run end **185–199 s**
(runs 44568192067, 44548473778, 44565423659 without the macOS stall). Go only: **144–150 s**.
Packaging only: **139 s** (run 44549975563).

Cache finding (verified in the log of run 44435646995): the `security` job finished first and
saved the shared Linux setup-go key `setup-go-Linux-x64-ubuntu26-go-1.27.4-<hash>`; `race` then
logged "Unable to reserve cache … another job may be creating this cache". Cache keys are
immutable, so test/race/packaging restored the cache that a 20-second govulncheck job left until
`go.sum` changed; `-race` objects were never cached.

### release.yml (v0.6.0-rc.5 44544452229, rc.6 44561169986; rc.7 44572810069 failed because the macOS runner never started)

| Job | rc.5 | rc.6 |
|---|---|---|
| `ui` | 15 s | 12 s |
| `windows` (needs ui) | 90 s | 97 s |
| `macos` (needs ui) | 91 s | 116 s |
| `release` (needs all) | 244 s (tests **88 s**, GoReleaser 127 s) | 268 s (tests **97 s**, GoReleaser 138 s) |
| run end | 358 s | 403 s |

Critical path: `ui → macos → release[tests → GoReleaser → attest → brew → npm]`. The race tests
inside `release` waited for the macOS/Windows builds although they use none of their output.

### landing.yml (10 runs) and docs.yml (10 runs)

- landing: `build-landing` 38–44 s, then `redirect` 7–13 s → end 54–64 s. `redirect` writes fixed
  stubs and never reads `landing/`. The workflow-wide group `pages` (no cancel) also queued every
  PR build behind deploys from main.
- docs: one job, 29–176 s; the external link check (lychee, network) is 6–132 s of it. Go docs
  check + impact + landing build together are ~25–35 s.

## Standards used

- Concurrency: a newer run in the same group cancels the pending one; `cancel-in-progress` accepts
  an expression; GitHub's own example cancels only on `pull_request`.
  https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/control-the-concurrency-of-workflows-and-jobs
- `needs`: "If a job fails or is skipped, all jobs that need it are skipped unless the jobs use a
  conditional expression that causes the job to continue" (`always()`, `!cancelled()`).
  https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax
- Caching: a run restores caches from its branch or the default branch (PRs also the base branch;
  tags never from other tags); "You cannot change the contents of an existing cache";
  `restore-keys` are prefix matches, newest first.
  https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching
- setup-go: caching on by default, `cache: false` disables it; modules and build outputs cached,
  key from the dependency file hash. https://github.com/actions/setup-go (README now describes v6;
  the v5 key with the `go.sum` hash is the one seen in our cache list).
- Job default timeout 360 min (the step limit 360 min is on the same page; the job default is
  **no verificado** in this session).
- zizmor audits (`unpinned-uses`, `artipacked`, `cache-poisoning`): https://docs.zizmor.sh/audits/

## Options per workflow

| Change | Saves on critical path | Check kept? |
|---|---|---|
| ci: split `ui` into `ui-build` (build → Go tests) and `ui-check` (lint + vitest), tests wait only for the build | ~20–30 s when the UI changes | Yes: same commands; Go tests still embed the real UI on all 3 OS; ci-ok requires both |
| ci: skip waiting for the UI at all | ~35–45 s | **No**: Go tests would lose the real embedded UI — rejected |
| ci: `security` with `cache: false`, `race` with its own `actions/cache` key | `race` faster (not critical); Linux test jobs get a real build cache | Yes |
| ci: six cross-compiles in background with `wait` per PID | packaging-only runs (est. 139 → ~80–100 s, **no verificado**) | Yes: any failed build fails the step |
| ci: `timeout-minutes` (15–20) | a hung job fails in 20 min instead of 6 h | Yes |
| ci/landing: concurrency per PR/branch, cancel superseded | frees runners, newest result sooner | ci: cancel only on PRs, one group per commit on main; landing: cancel only on PRs |
| release: race tests in their own `test` job (needs `ui`, read-only token), `release` needs it | ~90 s (rc.5/rc.6: 358/403 → ~270/305 s est.) | Yes, and the tests no longer run in the job holding `contents: write` + `id-token: write` |
| release: GoReleaser split/merge to start before macOS/Windows | more | **Not possible**: split/merge is GoReleaser Pro; ADR-0025 is OSS |
| landing: `redirect` without `needs: build-landing`; Pages group on the job | ~45 s on main pushes; PRs no longer wait for deploys | Yes (stubs never read the build) |
| docs: split into Go / links / comment jobs | ≤ 30 s, non-blocking check | Not done: docs never blocks a merge, the sticky comment would need artifact hand-offs; lean by design |

## Expected after

| Workflow | Before (critical path) | After (estimate) |
|---|---|---|
| ci, Go + UI change | 185–199 s | ~160–170 s |
| ci, Go only | 144–150 s | ~140–150 s (Windows setup-go 40 s cache restore stays) |
| ci, packaging only | 139 s | ~80–100 s (**no verificado**) |
| release | 358–403 s | ~270–305 s |
| landing, main push | 54–64 s | ~40–45 s |
| docs | 29–176 s | unchanged |

macOS runner queue time (0–600 s, and rc.7's runner that never started) is outside our control.

## Also done

GitHub-owned actions (`actions/*`) were referenced by tag (`@v4`), against AGENTS.md's
SHA-pinning rule; zizmor 1.32.0 reported 40 `unpinned-uses`. They are now pinned to the commit the
same tag resolved to on 2026-11-19 (same versions: checkout v4.4.0, setup-go v5.6.0, setup-node
v4.4.0, upload-artifact v4.6.2, download-artifact v4.3.0, cache v4.3.0, attest-build-provenance
v4.2.2, upload-pages-artifact v3.0.1, deploy-pages v4.0.5); Renovate's
`helpers:pinGitHubActionDigestsToSemver` keeps them updated.

## Open

- `release.yml` jobs restore setup-go/pnpm caches written on main (zizmor `cache-poisoning`, 6
  findings before this change, 7 after: the new `test` job keeps the cache the tests had inside
  `release`). Disabling caches in release.yml is safer but slower; a separate decision.
- Windows setup-go cache restore (38–48 s for 160 MiB) is the largest single wait left in ci.yml.
