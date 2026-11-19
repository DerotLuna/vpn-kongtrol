<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: desktop release, installers, package managers, signing and first run

Version 1.0.0 · 2026-11-18

## Question

How can a non-technical user download, install, set up and upgrade Kongtrol (CLI + tray) with no
terminal and nothing lost, on Windows, macOS and Linux, through the usual package managers, with
signed/verifiable artifacts, at $0 (public open-source repo)?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | fyne.io/systray v1.11.0 README + source (module cache: `systray_darwin.go` is the only cgo file) | https://github.com/fyne-io/systray | 2026-11-18 |
| 2 | GoReleaser — checksum (`extra_files`) | https://goreleaser.com/customization/checksum/ | 2026-11-18 |
| 3 | GoReleaser — release (`extra_files`; `templated_extra_files` is Pro) | https://goreleaser.com/customization/release/ | 2026-11-18 |
| 4 | GoReleaser Pro features (prebuilt builder, split/merge) | https://goreleaser.com/pro/ · https://goreleaser.com/blog/goreleaser-v1.12 | 2026-11-18 |
| 5 | GoReleaser docs source: winget, scoop, aur, homebrew_casks, npm, templates, repository/PR includes | https://github.com/goreleaser/goreleaser/tree/main/www/content | 2026-11-18 |
| 6 | Inno Setup license | https://jrsoftware.org/files/is/license.txt | 2026-11-18 |
| 7 | Inno Setup commercial license request | https://jrsoftware.org/isorder.php | 2026-11-18 |
| 8 | Inno Setup downloads (6.7.4, 7.1.1 on GitHub releases) | https://jrsoftware.org/isdl.php | 2026-11-18 |
| 9 | runner-images issue: Inno Setup not on windows-2025 | https://github.com/actions/runner-images/issues/11644 | 2026-11-18 |
| 10 | Chocolatey `innosetup` package (latest 6.7.1) | https://community.chocolatey.org/packages/innosetup | 2026-11-18 |
| 11 | SignPath Foundation (free for OSS) and terms | https://signpath.org/ · https://signpath.org/terms | 2026-11-18 |
| 12 | SignPath GitHub Actions integration (`signpath/github-action-submit-signing-request@v3`) | https://docs.signpath.io/trusted-build-systems/github | 2026-11-18 |
| 13 | Apple Developer ID / notarization | https://developer.apple.com/developer-id/ | 2026-11-18 |
| 14 | Apple Developer Program fee waiver (nonprofits, education, government; not individuals) | https://developer.apple.com/help/account/membership/fee-waivers/ | 2026-11-18 |
| 15 | Apple `LSFileQuarantineEnabled` ("whether the files this app creates are quarantined by default") | https://developer.apple.com/documentation/bundleresources/information-property-list/lsfilequarantineenabled | 2026-11-18 |
| 16 | Homebrew source: every quarantine code path lives under `Library/Homebrew/cask/` | https://github.com/Homebrew/brew/tree/master/Library/Homebrew/cask | 2026-11-18 |
| 17 | GoReleaser homebrew_casks "Signing and Notarizing" (unsigned casks: "damaged" alert; `xattr` bypass warning) | https://goreleaser.com/customization/homebrew_casks/ | 2026-11-18 |
| 18 | npm registry: `@biomejs/biome` (optionalDependencies per os/cpu, no postinstall) and `esbuild` (same + postinstall fallback) | https://registry.npmjs.org/@biomejs/biome/latest · https://registry.npmjs.org/esbuild/latest | 2026-11-18 |
| 19 | npm provenance | https://docs.npmjs.com/generating-provenance-statements | 2026-11-18 |
| 20 | GitHub artifact attestations | https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations | 2026-11-18 |
| 21 | Sigstore cosign `sign-blob --bundle` | https://docs.sigstore.dev/cosign/signing/signing_with_blobs/ | 2026-11-18 |
| 22 | freedesktop Desktop Entry and Autostart specs | https://specifications.freedesktop.org/desktop-entry-spec/latest/ · https://specifications.freedesktop.org/autostart-spec/latest/ | 2026-11-18 |
| 23 | Name availability: `registry.npmjs.org/kongtrol` → 404; AUR RPC `kongtrol`, `kongtrol-bin` → 0 results; `microsoft/winget-pkgs/manifests/k` has no Kongtrol | (lookups) | 2026-11-18 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- **Tray build** [1]: only macOS needs cgo (`systray_darwin.go` is the only file with `import "C"`;
  Linux uses D-Bus StatusNotifierItem, "removing the GTK dependency"). Verified locally:
  `CGO_ENABLED=0 GOOS=linux|windows GOARCH=amd64|arm64 go build ./cmd/kongtrol-tray` succeeds;
  `GOOS=darwin CGO_ENABLED=0` fails. README: on GNOME older trays may need a proxy
  (snixembed); the AppIndicator extension provides SNI on GNOME.
- **GoReleaser OSS vs Pro** [2–5]: `checksum.extra_files` and `release.extra_files` are OSS (no
  Pro marker); `prebuilt` builder and split/merge are Pro; `npms` is Pro and uses a
  **postinstall download**; `homebrew_casks.app` is Pro. `scoops`, `winget`, `aurs`, `nfpms`,
  `sboms`, `signs` are OSS. `isEnvSet "NAME"`: "returns true if the env is set and not empty";
  `envOrDefault` exists; `skip_upload` is templated and accepts `auto` (skip on prerelease).
  winget can push to a fork and open a PR to another repo (`pull_request.base`). Verified with
  `goreleaser check` and `goreleaser release --snapshot` (v2.20.2): archives, .deb/.rpm, Scoop
  manifest, winget manifests (zip, `NestedInstallerType: portable`, both commands) and the AUR
  PKGBUILD are generated; `checksums.txt` covers the `dist-extra/` files.
- **Inno Setup** [6–10]: free license, "any purpose, including commercial"; commercial users
  (revenue > USD 5,000) are *asked* to buy a license, "not strictly required"; non-commercial use
  is not asked to. Not preinstalled on current GitHub Windows images [9] (issue, reasoned from
  it); Chocolatey only has 6.7.1 [10]. The official 6.7.4 installer is Authenticode-signed by
  Pyrsys B.V. and supports `/PORTABLE=1`; our script compiles with it locally ("Successful
  compile", no warnings). Per-user install with `PrivilegesRequired=lowest`; HKCU Run key and
  HKCU\Environment PATH are the standard per-user locations (Inno Setup docs, reasoned).
- **Windows signing** [11, 12]: SignPath Foundation: "For OSS projects, our services are free of
  charge." Conditions include an OSI license without commercial dual-licensing, maintained and
  already released, MFA for all team members, reviewed changes, a manual approver per request,
  a code signing policy on the home page, and uninstall instructions. CI: upload the files with
  `actions/upload-artifact`, then `signpath/github-action-submit-signing-request@v3` with
  `api-token`, `organization-id`, `project-slug`, `signing-policy-slug`, `github-artifact-id`,
  `wait-for-completion`, `output-artifact-directory`.
- **macOS** [13–17]: Developer ID and notarization require Apple Developer Program membership
  (Account Holder); the fee waiver needs a nonprofit / education / government legal entity and
  excludes individuals and single-person businesses → **no free path** for this project.
  Quarantine is set by the app that writes the file when it opts in (`LSFileQuarantineEnabled`
  [15]); curl does not (reasoned: command-line tools do not set it; **no verificado** with an
  Apple statement naming curl). Homebrew's quarantine handling is all in `cask/` [16], so
  formulas are not quarantined (reasoned from the source tree). Coordinator-reported (not
  re-verified here): Homebrew 5 disables casks failing Gatekeeper in the official tap on
  2026-09-01 and `--no-quarantine` was removed in brew 4.7 (Homebrew/brew#20755).
- **npm** [18, 19]: Biome's `@biomejs/biome` lists `@biomejs/cli-<os>-<cpu>` as
  `optionalDependencies`, each with `os`/`cpu` (and `libc` for Linux), and a `bin` launcher; no
  install scripts. esbuild uses the same plus a `postinstall` fallback. Go static binaries need no
  libc split. Verified locally: the generated package runs `kongtrol version` through the
  launcher and `npm pack --dry-run` packs 4 files.
- **Supply chain** [20, 21]: attestations are free for public repos and need `id-token: write` +
  `attestations: write`; cosign keyless `sign-blob --bundle` with the workflow's OIDC identity.
- **Upgrades** (code): the helper is copied to a root/admin-owned location by
  `service install` and reported as needing reinstall when `privops.ProtocolVersion` differs
  (`internal/api/handlers_helper.go`), so channel upgrades never leave an old helper silently;
  the loader is non-strict YAML (only `configs/example.yaml` is tested with `KnownFields`), so old
  config files keep loading. On Linux/macOS a running binary can be replaced on disk; Go's
  `os.Executable` gives the path, so a restart picks the new file.

## Confidence

High for build facts, GoReleaser behaviour, Inno compile and npm layout (verified locally). Medium
for curl/Homebrew-formula quarantine (reasoned from Apple docs and Homebrew source) and for
winget/Scoop upgrade behaviour with a running process (**no verificado**: both replace files that
Windows locks while Kongtrol runs, so the docs tell users to quit first).

## Open questions

- First real tag: confirm the macOS job (`lipo`, `codesign -s -`, `iconutil`) and SignPath flow on
  GitHub runners.
- npm organization `kongtrol` availability (npmjs.com blocks automated lookups): **no verificado**.
- Inno Setup 7 compatibility of the script.

## Used by

- ADR-0025
