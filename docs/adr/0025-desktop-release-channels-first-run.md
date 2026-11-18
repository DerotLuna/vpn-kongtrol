<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0025: Desktop release, package managers, signing, safe upgrades and first run without a terminal

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

Releases shipped only the CLI (`.goreleaser.yaml`, `release.yml`); the tray was never published,
and on first run it told the user to open a terminal and run `kongtrol init`. Goal: a
non-technical user downloads, installs, sets up and upgrades Kongtrol (CLI + tray) with no
terminal and without losing anything, also through winget, Scoop, Homebrew, AUR, apt/dnf and
npm, with verifiable artifacts, at $0 (public open-source repo). Research:
[2026-11-18-desktop-release-channels](../research/2026-11-18-desktop-release-channels.md).

## Decision Drivers

- GUI-first, CLI-complete ([ADR-0003](0003-product-model-gui-first-cli-complete.md)); user guidance in
  ES+EN; local only (no new outbound call).
- $0: free runners for public repos, free OSS programs only.
- Standard tools over our own; every publisher optional; nothing lost on upgrade/uninstall.

## Considered Options

1. GoReleaser Pro `prebuilt`/split-merge for the native builds — Pro only (paid).
2. Per-OS jobs that each upload to the release, plus a job that recomputes checksums.
3. **Native jobs only where needed, GoReleaser OSS for the rest, joined with `extra_files`.**

Installers: Inno Setup vs WiX/MSI (heavier, admin-oriented) → **Inno Setup** per-user.
macOS: cask vs **formula + curl script + npm** (casks are quarantined, the app is not notarized).
Updates: in-app check (outbound call, opt-in) vs **package managers / re-run the installer**.

## Decision Outcome

Chosen: **option 3**, with:

- **Builds.** The tray is pure Go on Windows/Linux, so GoReleaser builds it there (Windows:
  `-H=windowsgui`, icon via `akavel/rsrc@v0.10.2`). macOS: `packaging/macos/build-app.sh` on a
  macOS runner builds universal (`lipo`) CLI + tray, `Kongtrol.app` (`Info.plist` with
  `LSUIElement`), ad-hoc `codesign -s -`, `ditto` zip. Windows: `packaging/windows/kongtrol.iss`
  (Inno Setup 6.7.4, official signed installer, portable) → `Kongtrol-Setup-windows-amd64.exe`.
  Both go to `dist-extra/`, which GoReleaser attaches and checksums (`extra_files`, OSS).
- **Release** published directly (`draft: false`, `prerelease: auto`): package managers need
  public URLs. Tags with `-` are prereleases and skip every publisher (`skip_upload: auto`).
- **Channels** (each skips itself while its secret is missing): Scoop bucket, winget (PR from
  the `DerotLuna/winget-pkgs` fork to `microsoft/winget-pkgs`), AUR `kongtrol-bin`, `.deb`/`.rpm`
  (nfpm, `.desktop` + XDG autostart + icon; suggests `gnome-shell-extension-appindicator`),
  Homebrew **formula** in `DerotLuna/homebrew-tap` (`packaging/homebrew/`; macOS installs the
  app bundle into the keg; never the official `homebrew/cask`), npm `kongtrol` +
  `@kongtrol/cli-<os>-<cpu>` (Biome/esbuild pattern, no postinstall, `--provenance`), and
  `install.sh` (macOS: `~/Applications/Kongtrol.app`; Linux: binaries + menu/autostart entry).
- **Signing and verification.** SBOM per archive (syft), keyless cosign bundle for
  `checksums.txt`, GitHub build provenance attestations for every artifact, npm provenance.
  Windows Authenticode via **SignPath Foundation** (free for OSS); the steps run only when
  `SIGNPATH_API_TOKEN` exists. macOS: **no free notarization** (Apple Developer Program,
  $99/year; the waiver excludes individuals) → ad-hoc signature and channels that do not set
  `com.apple.quarantine` (curl script, formula, npm), plus "Open Anyway" guidance for a browser
  download. Paid notarization can be added later (e.g. funded by GitHub Sponsors) as a
  `codesign`/`notarytool` step that skips while its secrets are absent.
- **Upgrades keep everything.** No channel touches `~/.kongtrol` or the keychain; uninstall never
  deletes user config (Homebrew/npm/AUR/deb leave the home alone; Inno Setup only removes its
  own files, the Run key, its PATH entry and the helper service it installed). The installer
  upgrades in place (same `AppId`): `PrepareToInstall` runs the old `kongtrol down --all`
  (tunnels down, DNS restored, kill switch lifted — the user stays online), closes the tray,
  replaces files, reopens the tray on silent upgrades, and installs the helper only on the first
  install. A helper with an older `privops.ProtocolVersion` is already reported as
  `needs_install`, so the tray/dashboard offer to reinstall it. `install.sh` and
  `kongtrol restart` restart a running daemon on the new binary with its connected tunnels.
  Config stays backward compatible (non-strict YAML loader; no schema change here).
  **No in-app update check** (it would be an outbound call).
- **First run without terminal.** With no `kongtrol.yaml`, `kongtrol daemon` starts in setup
  mode (`config.NewEmpty()`, `GET /api/v1/setup`). The tray shows "Set up now…", opens the
  dashboard Studio once automatically (marker `~/.kongtrol/run/setup-opened`) and the profile
  editor shows a welcome guide; the first valid profile writes the file and the page restarts
  the daemon (`POST /api/v1/restart`) and lands on the overview. The tray's "open a terminal
  with `kongtrol init`" (Windows only) is removed; `kongtrol init` stays.
- **Restart** solves `restart_required`: the dashboard button and `kongtrol restart` (CLI parity)
  start a fresh daemon (`daemon --restart <connected profiles>`) that waits for the old one and
  reconnects the tunnels that were up.

### Consequences

- Good: installers and package managers for every OS at $0; verifiable artifacts; first run and
  profile changes need no terminal; upgrades keep settings, credentials and the helper.
- Bad: Windows binaries stay unsigned until SignPath approves; the macOS app is not notarized;
  an installer upgrade disconnects the VPNs briefly (internet stays up); winget/Scoop upgrades
  need Kongtrol closed first (Windows file locks, **no verificado** in practice); the release is
  no longer a draft (bad tags must be deleted by hand).
- Neutral: Windows CLI+tray are built twice (installer job and GoReleaser); Inno Setup asks
  commercial users (> USD 5,000 revenue) to buy a license — not the case today.

### Confirmation

`go build ./...`, `go vet`, `go test ./...`; `actionlint`; `goreleaser check` and
`goreleaser release --snapshot` (v2.20.2) produce archives, `.deb`/`.rpm`, Scoop/winget/AUR
manifests and checksums with the extra files; `ISCC` 6.7.4 compiles the installer; the npm
launcher runs the packaged binary. Tests: `internal/api/handlers_setup_test.go`,
`cmd/kongtrol/daemon_restart_test.go`, `internal/config` `TestNewEmpty_DefaultsButInvalid`.
CI job `packaging` re-checks config/scripts/npm on changes. Still to confirm on the first tag:
macOS job and SignPath flow on GitHub runners.

## More Information

User setup steps (secrets, repos, SignPath application): [docs/DEVELOPMENT.md](../DEVELOPMENT.md#release).
Verification commands: [docs/SECURITY.md](../SECURITY.md#verifying-releases).
