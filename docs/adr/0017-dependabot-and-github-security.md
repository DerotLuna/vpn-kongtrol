<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0017: Dependabot grouped weekly; GitHub security features on

Version 1.0.0 · 2026-11-17

Status: Superseded by [ADR-0021](0021-renovate-for-dependency-updates.md) · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

The repo has four dependency ecosystems (Go modules, `landing` npm/pnpm, `web/ui` npm/pnpm,
GitHub Actions). One PR per dependency per day would flood a one-person project; no updates at
all is a security risk.

## Decision Drivers

- At most one version-update PR per ecosystem per week.
- Security fixes not delayed by the weekly batch.
- Free GitHub features used.

## Considered Options

1. No automation — stale, vulnerable dependencies.
2. Dependabot default (one PR per dependency) — noise.
3. Dependabot weekly with one group per ecosystem; security updates separate.
4. Renovate — more configurable; extra app (not researched further).

## Decision Outcome

Chosen: **option 3** (`.github/dependabot.yml`): `gomod` (`/`), `npm` (`/landing`, `/web/ui`; pnpm
lockfiles are read by the npm ecosystem), `github-actions` (`/`), each `schedule.interval: weekly`
with a `groups` rule `patterns: ["*"]`. Dependabot docs confirm `groups` and `weekly`
(verified 2026-11-17). Security updates come from the repo setting "Dependabot security
updates", outside the weekly batch. Other GitHub security features expected on: private
vulnerability reporting (used by `SECURITY.md`), Dependabot alerts and security updates.

### Consequences

- Good: ≤ 4 grouped PRs per week; security fixes immediate.
- Bad: one failing update blocks its whole group until split.
- Whether the repo settings (private vulnerability reporting, Dependabot alerts/security
  updates, secret scanning) are actually enabled is **no verificado**: the GitHub API needs an
  authenticated admin token, not available when this ADR was written.

### Confirmation

`.github/dependabot.yml`; check Settings → Code security on GitHub.

## More Information

- https://docs.github.com/code-security/dependabot/working-with-dependabot/dependabot-options-reference
- No separate research note (single documentation source).
