<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0075: Install scripts install the CLI and the tray, with a CLI-only opt-out

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer

Refines [ADR-0025](0025-desktop-release-channels-first-run.md) (release channels),
[ADR-0034](0034-uninstall-and-update-notice.md) (uninstall from the manifest),
[ADR-0046](0046-installers-detect-existing-install.md) (existing install) and
[ADR-0073](0073-installers-stop-when-the-new-binary-cannot-run.md) (binary must run). None is
superseded. Research:
[2026-11-20-install-scripts-cli-and-tray](../research/2026-11-20-install-scripts-cli-and-tray.md).

## Context and Problem Statement

`install.sh` installed CLI + tray (Linux) or Kongtrol.app (macOS), but `install.ps1` installed only
`kongtrol.exe`: a Windows user who ran the one-liner got no tray, no Start menu entry and no
autostart, so not the GUI-first product (AGENTS.md "Product model"). The owner decided
(2026-11-20) that every install path gives the same product. The tray is a client of the daemon
(`kongtrol daemon` from `kongtrol.exe`), so the valid sets are "CLI + tray" (default) and
"CLI only"; a tray without the CLI does not work.

## Decision Drivers

- Same product on every path; GUI first, CLI complete.
- No interactive question: piped `iwr | iex` / `curl | sh` often has no terminal, and common
  installers (rustup, Docker Desktop) install everything by default with an opt-out flag.
- Machines with no desktop (servers, SSH, Windows Server Core) must still install cleanly.
- An update must not undo the user's choice; nothing stale stays when the choice changes.
- One copy, one updater: never a second Kongtrol next to the Windows installer's.
- Keep ADR-0073: nothing replaced until the new binaries are known not to be blocked.

## Considered Options

1. CLI + tray by default, `--cli-only` / `-CliOnly` / `KONGTROL_CLI_ONLY=1` to opt out, CLI only
   automatically when there is no desktop, choice kept in the manifest (chosen).
2. Ask "Install the tray too? [Y/n]" — rejected: no terminal in a piped install; standard
   installers do not ask.
3. Keep `install.ps1` CLI-only and point to the Windows installer — rejected by the owner: paths
   would give different products.
4. Publish a new raw `kongtrol-tray-windows-<arch>.exe` asset — rejected for now: needs a
   `.goreleaser.yaml` change and would not exist in releases already published; the existing
   `kongtrol_windows_<arch>.zip` (checksummed, both programs) works for all of them.

## Decision Outcome

Chosen option 1.

**Which parts.** Decided in this order: a flag or `KONGTROL_CLI_ONLY` (`1` = CLI only, `0` =
with tray; `--with-tray` / `-WithTray` too; both flags together is an error); else the manifest
of the last install (`components=cli` keeps CLI only; a listed tray keeps the tray); else no
desktop → CLI only, said in one line with how to add the tray; else CLI + tray. "No desktop" is
Linux without `DISPLAY`, `WAYLAND_DISPLAY` and `XDG_CURRENT_DESKTOP`, and Windows Server Core /
Nano Server by Microsoft's documented `ServerLevels` check (`ServerCore=1` without
`Server-Gui-Shell=1`, or `NanoServer=1`). The automatic choice is written as
`components=auto-cli` and is not a user choice: it is decided again next time (an SSH session to
a desktop must not lock out the tray), and an update keeps a tray that is installed. Manifests
from before this change carry no `components=`: an old `install.ps1` manifest (CLI only, there
was no choice) gets the tray on update like a new install. Every run ends with "Installed: …"
naming what is installed and how to get the other part.

**`install.ps1` 1.8.0.** With the tray it downloads `kongtrol_windows_<arch>.zip`, checks it
against `checksums.txt` (and the cosign step) exactly like the CLI, and extracts only
`kongtrol.exe` and `kongtrol-tray.exe` by name; CLI only keeps the plain
`kongtrol-windows-<arch>.exe`. Both go to the same folder (the tray finds `kongtrol.exe` next to
itself). It closes this copy's running tray, moves old files aside (`.old`), probes, writes the
manifest (`file=` for both exes and `Kongtrol.lnk`, `runvalue=<tray>`, `components=`), runs the
existing stop / helper / resume steps, and only after the final probe creates the Start menu
shortcut (`Programs\Kongtrol.lnk`) and the HKCU Run value `Kongtrol` = `"<tray>"` (the same value
name as `kongtrol.iss`; `KONGTROL_NO_AUTOSTART=1` skips it and is remembered like in
`install.sh`), then opens the tray unless `KONGTROL_NO_LAUNCH=1` (first run opens the Profiles
page, ADR-0025). Switching to CLI only removes the tray, shortcut and Run value it installed
before. Windows-only steps are skipped when PowerShell runs on another OS (the offline tests).

**Next to the Windows installer.** When HKCU
`…\Uninstall\{BA277280-D8B2-4309-8620-BCEECBAA4B41}_is1` exists (the key `kongtrol.iss` reads),
`install.ps1` installs nothing, also with `-CliOnly`, and says what to do: run the new
`Kongtrol-Setup-windows-amd64.exe` (keeps VPNs and settings; `/VERYSILENT` for scripts), or
uninstall Kongtrol in Settings > Apps first (data stays) to switch to the script. Why refuse
instead of updating or adding a CLI: the Inno copy owns its folder, shortcut, Run value, PATH entry
and uninstaller; a second copy would overwrite the shortcut and Run value, leave two
`kongtrol.exe` of different versions talking to one daemon, and be updated by two mechanisms.
Running Kongtrol-Setup from the script was not chosen: it is amd64 only, may be missing from a
release, and has its own choices; one clear line is safer.

**`install.sh` 2.8.0.** `--cli-only` / `--with-tray` and `KONGTROL_CLI_ONLY`. Linux CLI only
installs only `kongtrol` from the same tarball (no menu entry, icon or autostart); macOS CLI only
downloads `kongtrol-darwin-<arch>` instead of the app. Switching to CLI only removes the tray
parts the manifest lists (Kongtrol's own names only, like `kongtrol uninstall`). The tray is
opened at the end only when installed.

**Tray probe (ADR-0073).** `kongtrol-tray --version` now prints the version and exits before
logging, the single-instance lock or any daemon start (`cmd/kongtrol-tray/main.go`). A tray from
0.7.0-rc.3 or older ignores the flag and would open (and start a daemon from the temp folder),
so the scripts start it with `--version` only when the new CLI is newer than 0.7.0-rc.3;
otherwise `install.ps1` reads the file back (SHA256 must still match, a blocked read is error
225/226 like a blocked start) and `install.sh` checks it is executable. A tray that fails stops
the install before anything changes (and, after placement, puts the previous files back), with
the CLI-only flag as the way to go on meanwhile.

**Uninstall.** `internal/uninstall` reads the new `runvalue=` lines (only a `kongtrol-tray.exe`
path) into `Plan.RunValues` and accepts `Kongtrol.lnk` as a script file name. `kongtrol
uninstall` deletes HKCU Run `Kongtrol` only while its data still starts that tray, so the Windows
installer's value is never touched. `install.ps1 -Uninstall` already delegates to it.

### Consequences

- Good: the Windows one-liner gives the same app as the installer: tray, Start menu, autostart.
- Good: servers and SSH sessions install the CLI without a question; the choice survives updates.
- Good: no second copy next to the Windows installer.
- Bad: the Windows download is the zip (~30 MB instead of ~20 MB) when the tray is installed.
- Bad: one more probe per install; the read-back check is weaker than starting the tray for
  releases up to 0.7.0-rc.3.
- Not verified: the Server Core check on a real Server Core; a real Defender detection on the
  tray (read or start); the Start menu shortcut and Run value were not created end to end by the
  script on Windows here (this machine has the Windows installer, so `install.ps1` refuses; the
  shortcut snippet was run in a temp folder); macOS CLI-only path not run (no Mac here).

### Confirmation

- `bash packaging/tests/install-scripts.sh`: install.sh headless install (CLI only, one line,
  `components=auto-cli`), desktop install (CLI + tray), headless update keeps the tray,
  `--cli-only` removes the tray and is kept, `KONGTROL_CLI_ONLY=0` adds it back, both flags
  refused; install.ps1 refuses next to the Windows installer, Server Core picks the CLI asset, and
  (PowerShell on Linux/macOS, as in CI) default installs both from the zip with the tray probed by
  `--version`, `KONGTROL_CLI_ONLY=1` installs only the CLI and removes the tray, an update keeps
  that choice. Run here on Windows (Git Bash + PowerShell 7.6) and in the
  `mcr.microsoft.com/powershell` container (Linux, all cases).
- `go test ./internal/uninstall/ ./cmd/kongtrol-tray/ ./cmd/kongtrol/`: manifest `runvalue=` /
  `Kongtrol.lnk`, `runValueIsFor`, tray `--version`.
- PowerShell 7 and Windows PowerShell 5.1 parse `install.ps1`; `sh -n install.sh`.

## More Information

- User docs: README "Install" and "Uninstalling", `docs/SETUP.md` (Windows, Linux), landing guide
  (install notes, ES/EN), `AGENTS.md` distribution paragraph, the scripts' header comments.
