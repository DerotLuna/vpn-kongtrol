<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0021: Renovate for dependency updates; Dependabot only for security alerts

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer · Supersedes: [ADR-0017](0017-dependabot-and-github-security.md)

## Context and Problem Statement

Renovate (GitHub app) was already opening dependency PRs when ADR-0017 added a Dependabot
version-updates config, so updates would arrive twice. Renovate also ran with no repo config,
opening one PR per dependency.

## Considered Options

1. Dependabot for everything (uninstall Renovate).
2. Renovate for version updates, Dependabot alerts only (no Dependabot PRs).

## Decision Outcome

Chosen: **option 2** — Renovate was already in use and covers Go modules, both pnpm projects,
GitHub Actions and Docker; Dependabot alerts stay because they are GitHub-native.

- `renovate.json`: `config:recommended` (includes the Dependency Dashboard), `group:allNonMajor`
  (one PR for all minor/patch updates), `schedule:weekly`, `prConcurrentLimit: 5`, label
  `dependencies`; majors stay separate PRs.
- The `go` directive is bumped via a `packageRules` entry for the `golang-version` datasource
  with `rangeStrategy: "bump"` — Renovate does not update it by default (Renovate Go docs;
  [ADR-0019](0019-track-supported-go-release.md)).
- `.github/dependabot.yml` removed. In repo settings: Dependabot alerts on; Dependabot security
  updates and grouped security updates off (they would duplicate Renovate PRs).
- Config validated with `renovate-config-validator` on 2026-11-17.

### Consequences

- Good: one weekly grouped PR instead of one per dependency; no duplicates.
- Bad: security fixes arrive through Renovate's schedule unless triggered from the Dependency
  Dashboard; alerts still notify immediately.

## More Information

- https://docs.renovatebot.com/modules/manager/gomod/ (Go support; `go` directive not bumped by
  default)
- https://docs.renovatebot.com/modules/versioning/go-mod-directive
- Comparisons (secondary, accessed 2026-11-17): https://www.aikido.dev/blog/dependabot-vs-renovate,
  https://appsecsanta.com/sca-tools/dependabot-vs-renovate. The official Renovate bot-comparison
  page could not be fetched: **no verificado**.
