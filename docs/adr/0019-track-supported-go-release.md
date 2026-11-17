<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0019: Track a supported Go release (1.27)

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

The CI `security` job (govulncheck) failed on every branch. The module said `go 1.25.0` and CI
used Go 1.25.x. govulncheck reported called standard-library vulnerabilities (GO-2026-6859,
-6809, -6617, -6613, -6612, -6611, -6610, -6609, -6608, -6607 and two more) fixed only in
go1.26.9–go1.26.11, plus `golang.org/x/net` v0.38.0 (fixed in v0.60.0). Go 1.25 no longer gets
fixes: "Each major Go release is supported until there are two newer major releases" (Go release
policy), and the current stable release is go1.27.4.

## Decision Drivers

- Security fixes for the standard library only reach supported releases.
- CI, release, Docker and local builds use the same Go version.
- Contributors with an older Go still build without manual installs.

## Considered Options

1. Stay on Go 1.25 and ignore the findings — unsupported, unpatched.
2. Go 1.26 — supported today, but loses support at the next major release.
3. Go 1.27 (current stable), `toolchain go1.27.4`.

## Decision Outcome

Chosen: **option 3**. `go.mod` has `go 1.27.0` and `toolchain go1.27.4` (written by
`go get go@1.27.0 toolchain@go1.27.4`, the method the toolchain docs recommend). The `go` line is
the minimum version; the `toolchain` line is the suggested toolchain, so with the default
`GOTOOLCHAIN=auto` an older local Go downloads go1.27.4 instead of building with an unpatched
1.27.0. Verified 2026-11-17: Go 1.25.5 here switched to go1.27.4 and built `./cmd/kongtrol`.
CI (`ci.yml`, `docs.yml`, `release.yml`: `go-version: "1.27"`) and `build/docker/Dockerfile`
(`golang:1.27-alpine`, tag present on Docker Hub) use 1.27. `golang.org/x/net` v0.63.0 and
`golang.org/x/image` v0.48.0 (7 uncalled advisories) were updated; govulncheck reports no
vulnerabilities.

Rule: the `go` directive stays on a supported Go release. When a new major Go release ships,
move the `go` line before the old one leaves support.

### Consequences

- Good: stdlib security fixes reach CI and release builds; govulncheck passes.
- Good: Renovate (`renovate.json`) proposes `toolchain` updates by default, so patch releases
  arrive as PRs.
- Bad: Renovate does **not** bump the `go` directive by default (its docs: "Renovate's default
  behavior is not to propose upgrades to it"). The move to each new major release stays manual
  unless a `packageRules` entry with `rangeStrategy: "bump"` is added.
- Bad: building from source needs Go 1.27+ (or `GOTOOLCHAIN=auto` with network access).

### Confirmation

`go.mod` `go`/`toolchain` lines; CI `security` job green;
`go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./internal/...`.

## More Information

- Research note: [2026-11-17-go-release-support](../research/2026-11-17-go-release-support.md).
