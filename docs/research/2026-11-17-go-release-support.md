<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: Go release support and the `toolchain` line

Version 1.0.0 · 2026-11-17

## Question

Which Go release must Kongtrol target to receive security fixes, and how should `go.mod` pin it?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Go release history — release policy | https://go.dev/doc/devel/release | 2026-11-17 |
| 2 | Go downloads JSON | https://go.dev/dl/?mode=json | 2026-11-17 |
| 3 | Go toolchains | https://go.dev/doc/toolchain | 2026-11-17 |
| 4 | Renovate gomod manager | https://docs.renovatebot.com/modules/manager/gomod/ | 2026-11-17 |
| 5 | Docker Hub `golang:1.27-alpine` tag | https://hub.docker.com/v2/repositories/library/golang/tags/1.27-alpine | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- [1] "Each major Go release is supported until there are two newer major releases."
- [2] Current stable: go1.27.4. Go 1.25 is therefore unsupported.
- [3] "The go line declares the minimum required Go version"; "The toolchain line declares a
  suggested toolchain", used "if the default toolchain's version is less than the suggested
  toolchain's version". The go/toolchain lines are "most conveniently and safely modified" with
  `go get`. `GOTOOLCHAIN=auto` is the default (`$GOROOT/go.env`).
- [4] Renovate proposes `toolchain` updates by default; for the `go` directive "Renovate's
  default behavior is not to propose upgrades to it" unless a `packageRules` entry sets
  `rangeStrategy: "bump"`.
- [5] Tag `golang:1.27-alpine` exists (amd64 image listed).
- Local check: Go 1.25.5 with `GOTOOLCHAIN=auto` downloaded go1.27.4 and used it after
  `go.mod` gained `toolchain go1.27.4`.
