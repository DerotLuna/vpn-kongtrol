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

Every release also attaches `install.sh` and `install.ps1` (listed in `checksums.txt`), so
`releases/latest/download/install.sh` always matches the latest assets; `https://kongtrol.app/install.sh`
and `/install.ps1` redirect there (`landing/public/_redirects`,
[ADR-0035](adr/0035-install-scripts-pinned-and-release-assets.md)).

Release checklist:

- [ ] Tag pushed, workflow green, assets and `checksums.txt` on the release page.
- [ ] **TODO after v0.6.0 final:** switch `INSTALL_SCRIPT_BASE` and `UNINSTALL_SCRIPT_BASE` in
  `landing/src/links.ts` to `https://kongtrol.app/` (redirect to latest), check
  `curl -sI https://kongtrol.app/install.sh` answers 302 → the release asset, and replace the
  `raw.githubusercontent.com/…/install.sh` commands in `README.md` (pinned to `v0.5.0`) and
  `docs/SETUP.md` (`main`) with it. In `README.md` also drop the "since v0.6.0 (pre-release)"
  marks, the v0.5.0 column of the commands table and the rc.3 links, and list the package managers
  once they are published.
  Until then the landing stays pinned to the v0.5.0 tag (the scripts on `main` need v0.6.0 assets).
  Also remove the "desde v0.6.0" note (`sinceV060` in `landing/src/content/guide.*.tsx`, its two
  uses in `Guide.tsx` and the `.guide-since` rule in `landing/src/index.css`).
  And switch `WINDOWS_INSTALLER_URL` in `landing/src/links.ts` (pinned to the
  `v0.6.0-rc.5` pre-release installer, the "Descargar para Windows" buttons of the hero,
  download section and footer) to `releases/latest/download/Kongtrol-Setup-windows-amd64.exe`.

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

The public marketing site under `landing/` is independent from the embedded dashboard (`web/ui`):

```bash
make build-all-cli          # produce CLI binaries in build/dist/
make landing-sync-binaries  # copy + refresh checksums in landing/_binaries/
cd landing && pnpm install && pnpm run dev
```

### Regenerate README media

The GIFs in `README.md` (`assets/readme/hero.gif`, `routing.gif`, `attach.gif`) are recorded from
the English landing page in dark theme, so they follow its design. After a visual change to the
hero terminal, the routing diagram or the attach demo, run from the repo root:

```bash
node scripts/readme-media.mjs              # builds landing/, records, converts
node scripts/readme-media.mjs --no-build   # reuse landing/dist
node scripts/readme-media.mjs --only=hero  # hero | routing | attach (comma list)
```

Needs Node 20+, pnpm and ffmpeg on `PATH`. Playwright is **not** a project dependency: the script
installs `playwright@1.65.1` (override with `PLAYWRIGHT_VERSION`) and Chromium once into
`<os tmp>/kongtrol-readme-media`, outside the repo. It serves `landing/dist` on `127.0.0.1` only
while recording and stops it before exiting. Each GIF is one animation cycle (loops cleanly),
15 fps, at most 720 px wide, and must stay under ~2 MB (the script warns); check one frame before
committing, e.g. `ffmpeg -ss 4 -i assets/readme/hero.gif -frames:v 1 frame.png`.

## Docker

```bash
make docker-build
make docker-up
```

See [DOCKER.md](DOCKER.md) for deployment details.
