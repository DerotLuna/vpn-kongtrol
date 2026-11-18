# Development

See [ARCHITECTURE.md](ARCHITECTURE.md) for system design and project layout, and [CLAUDE.md](../CLAUDE.md) for coding conventions used by contributors and AI assistants working in this repo.

## Build

```bash
make build             # current platform → build/dist/kongtrol
make build-all-cli      # cross-compile CLI for all platforms (CGO_ENABLED=0)
make build-tray-native  # tray app, native only (requires CGO)
```

> **Windows:** `make` targets use `SHELL := /usr/bin/bash` and Unix env-var syntax — run them from **Git Bash**, not PowerShell or cmd.exe.

## Test

```bash
go test ./cmd/... ./internal/... ./assets ./web    # all product packages
go test ./internal/policy/                          # single package
go test -run TestEngine_ResolveIP ./internal/policy/ # single test
make test-race                                       # with the race detector (requires a C toolchain)
go test -tags e2e ./internal/vpn/...                 # E2E tests against live daemons (opt-in)
```

## Lint

```bash
go vet ./cmd/... ./internal/... ./assets ./web
golangci-lint run ./cmd/... ./internal/... ./assets ./web
```

<!-- feature:distribution -->
## Release

Push a `v*` tag; `.github/workflows/release.yml` does the rest
([ADR-0025](adr/0025-desktop-release-channels-first-run.md)): macOS app (native macOS runner),
Windows installer (Inno Setup, native Windows runner), then GoReleaser (CLI for every OS, Linux and
Windows tray, archives, `.deb`/`.rpm`, SBOMs, `checksums.txt` + Sigstore signature, GitHub
release, Scoop, winget, AUR), build provenance attestations, the Homebrew formula and the npm
packages. Tags with a `-` (`v1.2.0-rc.1`) are prereleases: no package manager is updated and npm
gets the `next` tag. `make release-snapshot` builds everything locally without publishing.

Every channel is optional: its step skips itself while its secret is missing. To turn them on
(all free; create once):

| Channel | What to create | Secret / variable (repo settings → Secrets and variables → Actions) |
|---|---|---|
| Homebrew | public repo `DerotLuna/homebrew-tap` (empty, with a `Formula/` folder created on first publish) | `HOMEBREW_TAP_TOKEN`: fine-grained PAT, *Contents: read and write* on that repo |
| Scoop | public repo `DerotLuna/scoop-bucket` | `SCOOP_BUCKET_TOKEN`: fine-grained PAT, *Contents: read and write* on that repo |
| winget | fork of `microsoft/winget-pkgs` under `DerotLuna`; sign the Microsoft CLA on the first PR | `WINGET_GITHUB_TOKEN`: classic PAT with `public_repo` (pushes to the fork, opens the PR upstream) |
| AUR | AUR account with an SSH key; package `kongtrol-bin` is created by the first push | `AUR_KEY`: the private SSH key (no passphrase) |
| npm | npm account with 2FA and the organization `kongtrol` (free, public packages) | `NPM_TOKEN`: granular access token, *read and write*, packages `kongtrol` and `@kongtrol/*` |
| Windows signing | apply at https://signpath.org/apply (free for OSI-licensed projects); after approval, install the SignPath GitHub app and create the project, artifact configuration and signing policy they describe | secret `SIGNPATH_API_TOKEN`; variables `SIGNPATH_ORGANIZATION_ID`, `SIGNPATH_PROJECT_SLUG`, `SIGNPATH_POLICY_SLUG` |

SignPath Foundation's terms also ask for a code signing policy on the project's home page, MFA for
every team member on GitHub and SignPath, and a manual approval of each signing request.
macOS notarization needs the paid Apple Developer Program ($99/year; the fee waiver excludes
individuals): the build then needs a `codesign`/`notarytool` step added to
`packaging/macos/build-app.sh`, guarded so it skips while its secrets are missing.

## Landing site

The public marketing site under `landing/` is independent from the embedded dashboard (`web/dashboard`):

```bash
make build-all-cli          # produce CLI binaries in build/dist/
make landing-sync-binaries  # copy + refresh checksums in landing/_binaries/
cd landing && pnpm install && pnpm run dev
```

## Docker

```bash
make docker-build
make docker-up
```

See [DOCKER.md](DOCKER.md) for deployment details.
