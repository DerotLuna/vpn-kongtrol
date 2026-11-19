<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0050: Updates reconnect the tunnels that were up; the installer's "open" checkbox always opens

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Two bugs reported by the owner on v0.6.0-rc.4 (Windows installer), re-checked against the code
after ADR-0046:

1. **"Abrir Kongtrol y crear mi primera VPN" opened nothing.** The [Run] entry itself was fine
   (`postinstall nowait skipifsilent`; a postinstall entry already runs as the original user and
   its working directory defaults to the program's folder — research note). The root cause was in
   what the tray does when started:
   - it opens the Perfiles page by itself only when **no `kongtrol.yaml` exists** and only
     **once** (marker `~/.kongtrol/run/setup-opened`, ADR-0025). Both survive an uninstall that
     keeps data, and the installer chose the checkbox by "were the program files there"
     (`IsFirstInstall`), not by "do profiles exist". After an earlier install (the owner's case:
     `~/.kongtrol/kongtrol.yaml` present, program folder gone) the checkbox promised "create my
     first VPN" and the tray only added an icon to the notification area;
   - when a tray was already running, the new one exits at once (single-instance mutex), again
     showing nothing (`tray.log`: "another instance is already running, exiting").
2. **An update left every tunnel down.** `PrepareToInstall` runs `kongtrol down --all --quiet`
   (needed: the running daemon locks `kongtrol.exe`) and nothing reconnected afterwards, also
   on silent updates (winget). `install.ps1`/`install.sh` used `kongtrol restart` after the
   helper update, so a helper restart that dropped a tunnel first made it look "not connected"
   and it was not brought back.

## Considered Options

For 2:

1. Reconnect every profile on update (wrong: connects what the user had off).
2. Keep `restart` only, run the helper update after it (the new daemon would reconnect through an
   outdated helper and fail).
3. **Remember, stop, update, resume**: before stopping, save the connected profiles; after the
   files and the helper are updated, start the daemon with exactly those, reusing the restart
   machinery (`daemon --restart <profiles>`, ADR-0025).

For 1:

1. Make the tray always open the dashboard on start (wrong for autostart at sign-in).
2. **An explicit request**: the checkbox passes `--open /profiles`; the tray opens it no matter
   the one-time marker or existing profiles, and a second tray instance opens it before exiting.
   The checkbox label follows whether profiles exist.

## Decision Outcome

Option 3 for updates and option 2 for the checkbox.

- `kongtrol down --all --remember` (hidden flag, `cmd/kongtrol/cmd_down.go`) asks the running
  daemon for the statuses and writes the **connected** ones to `~/.kongtrol/run/resume.json`
  (`internal/daemonctl/resume.go`: version 1, sorted, deduplicated, 0600; no daemon or none
  connected removes an older file). Then `down` stops everything as before.
- `kongtrol resume` (hidden command, `cmd/kongtrol/cmd_resume.go`) reads and deletes the file
  (one-shot; ignored after 24 h, broken or unknown version), loads the config only when there
  is something to resume (a first install has no config and must not fail), drops profiles that
  no longer exist (`cli.restart.profile_gone`) and, with no daemon, starts
  `daemon --restart <profiles>` from its own executable — the same replacement `kongtrol restart`
  starts — waiting only until it answers; with a daemon already running (the tray was faster) it
  asks that daemon to connect them. It says `cli.resume.reconnecting` ("Reconectando las VPN que
  tenías: a, b").
- **Inno Setup**: `PrepareToInstall` extracts the NEW `kongtrol.exe` to `{tmp}`
  (`ExtractTemporaryFile`) and runs `down --all --quiet --remember` with it, because an older
  installed copy does not know `--remember` (it would fail and not stop the daemon). If that
  cannot run, the installed copy stops Kongtrol as before (no resume). [Run] order:
  `service install --if-outdated` → `resume --quiet` (both `runhidden waituntilterminated`, also
  on silent installs) → tray. So tunnels come back after a helper update, through the current
  helper.
- `install.ps1` / `install.sh` (update/repair/downgrade): `down --all --quiet --remember` with the
  new binary **before** the helper update, then `resume --quiet` (replaces `restart`).
- Provider apps (FortiClient, Proton, …) follow the `up` rules: `resume` connects through the same
  `ProfileService.Connect`, so a tunnel still up is attached, otherwise the normal path (official
  CLI, the corporate-client automation `up` already does, or the wait/notice). Nothing new is
  automated.
- **Checkbox**: `HasProfiles` (`%USERPROFILE%\.kongtrol\kongtrol.yaml` or `config.yaml`, the paths
  `config.DefaultPaths()` checks first) picks the entry: no profiles → "Abrir Kongtrol y crear mi
  primera VPN" = `kongtrol-tray.exe --open /profiles`; profiles → "Abrir Kongtrol" (tray only,
  ADR-0046). The tray accepts only `--open /` and `--open /profiles`
  (`cmd/kongtrol-tray/open_request.go`); first-run claims the marker anyway, so a later start
  does not open it again by itself.

### Consequences

- Good: after any update (wizard, silent, winget, scripts) the user is where they were.
- Good: the checkbox does what it says, every time it is checked.
- Neutral: the script installers now stop Kongtrol before the helper check instead of after it;
  with no daemon running, `down --all` still runs (as the Inno installer always did).
- Not covered: Scoop/winget-portable/Homebrew/npm/deb/rpm do not stop Kongtrol themselves; their
  guidance stays "quit Kongtrol, update, `kongtrol restart`" (docs/SETUP.md).
- Not covered, **no verificado**: `down --all` of a provider profile without an official CLI asks
  the user to disconnect in the app and waits (existing behavior, bounded by its timeout); during
  a hidden installer step that wait is invisible. Untested on a real FortiClient update.
- **No verificado** until a release candidate is installed: the Inno parts compile only in CI
  (`release.yml`), and the update/resume flow was not run end to end on Windows.

## Links

- Research: [2026-11-19 installer resume and postinstall](../research/2026-11-19-installer-resume-and-postinstall.md)
- Builds on ADR-0025 (restart, first run), ADR-0046 (update detection), ADR-0032 (helper).
- Code: `packaging/windows/kongtrol.iss`, `install.ps1`, `install.sh`,
  `cmd/kongtrol/cmd_resume.go`, `cmd/kongtrol/cmd_down.go`, `internal/daemonctl/resume.go`,
  `cmd/kongtrol-tray/open_request.go`, `cmd/kongtrol-tray/tray_menu.go`.
