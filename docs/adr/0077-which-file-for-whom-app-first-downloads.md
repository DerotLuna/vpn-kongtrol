<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0077: Which file is for whom: the app first on every download surface

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

The owner asked whether `Kongtrol-Setup-windows-amd64.exe` and `kongtrol-windows-amd64.exe` contain
the same thing, and what `install.ps1` / `install.sh` install (2026-11-20). They do not: the
installer, the macOS app, the archives and the Linux packages carry the app (tray + panel) **and**
the `kongtrol` command; the bare binaries are the command only. The release notes did not say so,
and the landing's main button on macOS and Linux downloaded the bare binary — the command without
the tray — against the product model (GUI-first, ADR-0025; issue #32).

The tray needs the command (it starts `kongtrol daemon`); the command works alone. So the two sets
are "app (with the command)" and "command only"; "tray only" cannot work.

## Decision Outcome

- **Release notes** (`.goreleaser.yaml` `release.header`): a "Which file do I download?" table —
  the app, command only, archives — per system, with the scripts' command-only switch (`$env:KONGTROL_CLI_ONLY=1` before `iwr | iex`, `| sh -s -- --cli-only`)
  (ADR-0075), a link to the guide in both languages, how to verify, and "all your data stays on
  your computer".
- **Landing** (`landing/src/components/sections/Download.tsx`, `components/ui/downloadCta.tsx`,
  `links.ts`):
  - macOS: the main button downloads the universal app (`Kongtrol-macos-universal.zip`, in every
    release since v0.6.0), so the visitor no longer picks a chip; the bare binaries are listed as
    "Solo el comando".
  - Linux: no single file fits every distribution and the `.deb`/`.rpm` names carry the version, so
    `releases/latest/download/<name>` cannot link them; the card shows the one-line install (app +
    command) and points to the packages on the release page.
  - Windows: unchanged (installer); bare binaries labelled "Solo el comando".
  - A "¿Qué descargo?" block (three definitions) before "Otras formas de instalar".
- No asset is renamed (scripts, Scoop, winget and AUR depend on the names).

### Consequences

- Good: every primary button gives the app; every file says who it is for.
- Neutral: Linux packages are one click further (release page) until they have version-less names
  or an APT/RPM repository (issue #42).

## More Information

- Refines: [ADR-0025](0025-desktop-release-channels-first-run.md), [ADR-0066](0066-landing-gold-accent-footer-and-fixed-hero-terminal.md).
- Related: ADR-0075 (scripts install the app by default; `KONGTROL_CLI_ONLY=1` / `--cli-only` for the command only).
- Docs: `DESIGN.md` (Download section).
