<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0045: Double click on kongtrol.exe shows where to get the installer

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer

## Context and Problem Statement

On Windows the release lists both `Kongtrol-Setup-windows-amd64.exe` (the app installer) and
`kongtrol-windows-amd64.exe` (the terminal command). Opening the second one from Explorer flashes
a console window that closes at once: the user sees nothing and does not know what went wrong.
Research: [2026-11-19-explorer-double-click](../research/2026-11-19-explorer-double-click.md).

## Decision Outcome

- `kongtrol` started from Explorer with no arguments (`mousetrap.StartedByExplorer()`, Windows
  only through build tags) prints `cli.explorer_notice` in the CLI language (KONGTROL_LANG →
  `preferences.json` → OS) and waits for Enter, then exits 0. Code:
  `cmd/kongtrol/explorer_notice*.go`.
- cobra's own mousetrap is turned off (`cobra.MousetrapHelpText = ""`): with
  `MousetrapDisplayDuration = 0` it adds a fixed English line, and it also fires when a shortcut
  passes arguments.
- Every GitHub release body starts with one line that points Windows users to the installer
  (`.goreleaser.yaml` `release.header`).

## Consequences

- Good: the user gets one line saying what the file is and what to download, in their language.
- Neutral: one more direct dependency in `go.mod` (mousetrap was already in the build via cobra).
- Bad: the check depends on the parent process name; another file manager launching the exe is
  not detected and keeps the old behavior.
