<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0045: Every installer detects an existing install and presents itself as an update

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer

Refines point 7 of [ADR-0037](0037-native-tray-menu-states-and-installer-upgrade.md) (installer
upgrades keep the helper current) and the upgrade paths of
[ADR-0025](0025-desktop-release-channels-first-run.md) /
[ADR-0034](0034-uninstall-and-update-notice.md). Those decisions stand, so none is superseded.
Research: [2026-11-19-installer-upgrade-detection](../research/2026-11-19-installer-upgrade-detection.md).

## Context and Problem Statement

Running an installer over an existing Kongtrol looked like a first install: the Windows wizard
showed the license and tasks again and ended with "Abrir Kongtrol y crear mi primera VPN";
`install.sh` / `install.ps1` printed only "Downloading…"; `kongtrol service install` asked for
UAC / sudo even when the helper was already current. Non-technical users read that as "I am
starting over" and some were asked for things they had already answered. While checking this we
also found that an outdated helper was never updated by `service install --if-outdated`: the
client reports a reply with another protocol as `privops.ErrHelperOutdated`, and that error was
treated as "not installed".

## Decision Drivers

- Owner goal: every install path detects an existing install and says **update** (or
  reinstall / repair, or older version) instead of "install", and never asks for anything
  unnecessary. A first install stays as it was.
- Reuse the installer tool's own mechanisms (Inno Setup's uninstall key and `UsePrevious*`
  directives) instead of our own state.
- Same version rules everywhere; silent / scripted runs never prompt.
- No admin prompt unless something privileged really changes (AGENTS.md "No admin terminal").

## Considered Options

1. Detect in each installer, change texts and skip pages / questions (chosen).
2. Ship a separate "updater" binary or in-app auto-update — rejected: ADR-0034 settled
   "opt-in notice, never auto-update".
3. Leave the installers and only document "running it again updates" — rejected: the screens
   would keep saying "install" and asking again.

## Decision Outcome

Chosen option 1.

**Version order (all installers).** SemVer 2.0.0 precedence (https://semver.org §11): a leading
`v` and `+build` are ignored; `X.Y.Z` numeric (a missing field is 0); a pre-release is lower than
its release; pre-release identifiers compare dot by dot (numbers numerically and below text,
text in ASCII order, fewer identifiers lower). So `0.6.0-rc.3 < 0.6.0-rc.4 < 0.6.0 < 0.10.0`.
Implemented three times, each small and table-tested: Inno Pascal `CompareVersions`
(`packaging/windows/kongtrol.iss`, reviewed only: no `iscc` here), `ver_cmp` in `install.sh`
(awk), `Compare-KongtrolVersion` in `install.ps1`.

**Windows installer** (`packaging/windows/kongtrol.iss` 1.3.0):
- `InitializeSetup` reads `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\{AppId}_is1`
  (`DisplayVersion`, `InstallLocation`) before any page and sets the kind: first / update /
  reinstall (same version) / older version. A key without a version counts as an update.
- Not first: `ShouldSkipPage` skips license, folder and tasks (the folder and the task choices are
  the previous ones: `UsePreviousAppDir`, `UsePreviousTasks`, both default yes), so the wizard
  opens on the Ready page titled "Actualizar Kongtrol <old> → <new>" / "Reinstalar Kongtrol <v>
  (reparar)" / "Instalar una versión anterior (<old> → <new>)", button "Actualizar" /
  "Reinstalar" / "Instalar versión anterior", a "what is kept" line, and a memo with the folder
  only. Older version: one confirmation (default No) on Next; silent installs never ask.
- Finished page: "Kongtrol se actualizó a <v>…" / "…se reinstaló…" and the only run option is
  "Abrir Kongtrol" (no "crear mi primera VPN").
- Unchanged: `PrepareToInstall` still runs `down --all`; the helper task runs on a first install
  only; every non-first install runs `service install --if-outdated` (UAC only if outdated).

**`install.sh` / `install.ps1`** (2.2.0 / 1.2.0): read `~/.kongtrol/install.manifest` (else a
`kongtrol` in the default folders), ask the old binary and the downloaded one for their version
(`kongtrol version` prints only the version when not on a terminal), print "Updating Kongtrol
<old> → <new>" / "Reinstalling…" / an older-version warning (asked only on a terminal / an
interactive session; `--yes`, `-Yes` or `KONGTROL_YES=1` answer for scripts; without a terminal
it proceeds, since the version was chosen explicitly). An update reuses the previous install
folder (and macOS app folder) and the Linux autostart choice from the manifest, so nothing is
asked again and no second copy is left. On update they run `service install --if-outdated`
(install.sh only with a terminal, for sudo; otherwise the tray offers "Actualizar ayudante"),
restart a running daemon, and end with "Kongtrol updated to <v> … Next: …". `install.ps1` now
moves a locked `kongtrol.exe` aside before replacing it and restarts the daemon, like
`install.sh`. The scripts stay English-only, as before (they never had a language switch);
`kongtrol` itself prints its lines in the user's language.

**`kongtrol service install`** (plain, by a user): pings the helper first. Same protocol and same
version and no Homebrew tool to re-copy → "El ayudante ya está instalado y al día · nada que
hacer" and no prompt. Another protocol or version → "Actualizando el ayudante…" / "Ayudante
actualizado". Missing, stopped, refusing this user, or tools changed → installs as before, so
every "Qué hacer: kongtrol service install" hint still repairs. `init`, `up` and `doctor --fix`
say "Actualizando…" when the helper is outdated. `--if-outdated` and the tray now treat
`privops.ErrHelperOutdated` as outdated (bug fix).

**Already fine, unchanged:** `kongtrol init` / `up` offer the helper only when missing or outdated
(`helperState`); the tray hides the helper item when current and says "Actualizar ayudante (una
vez)" when outdated; the dashboard / tray install (`POST /api/v1/helper/install`) returns at once
when the helper answers with this protocol.

**Package managers and other packages:** Homebrew, npm, Scoop, winget, AUR, `.deb` / `.rpm` print
their own upgrade text; the macOS `.app` (`packaging/macos/`) and the Linux `.desktop` have no
installer UI of ours. No change there.

### Consequences

- Good: an update reads as an update on every path, asks nothing already answered, and a current
  helper never triggers UAC / sudo; an outdated helper is updated again on Windows upgrades.
- Good: downgrades are visible and confirmed interactively, never blocked in scripts.
- Bad: three copies of the version compare (Pascal, awk, PowerShell); kept small and tested by
  the same table.
- Bad: the Windows language dialog still appears on every run (`ShowLanguageDialog=yes`,
  pre-selected with the previous language by `UsePreviousLanguage`). `auto` would hide it but
  also changes the first install, so it is left for a separate decision.
- Not verified: the Inno script compiles only in CI (`release.yml`, job `windows`); no `iscc`
  was available here, so the Pascal code was reviewed, not compiled or run.

### Confirmation

- `go test ./cmd/kongtrol -run 'TestOutdatedHelperAction|TestHelperInstallAction'`,
  `go test ./cmd/kongtrol-tray -run TestHelperState`.
- `install.sh`: `sh -n`, the version table and the detection block run in a temp `HOME`;
  `install.ps1`: PowerShell 7 and Windows PowerShell 5.1 parser, the same version table.
- The release workflow compiles the Inno script; check the wizard by hand on the next release
  candidate (first install, update, same version, older version, `/SILENT` update).
