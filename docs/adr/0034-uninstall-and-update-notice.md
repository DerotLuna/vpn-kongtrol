<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0034: Uninstall keeps data unless asked; opt-in update notice

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer · Supersedes the
"uninstall never deletes user config" and "No in-app update check" points of
[ADR-0025](0025-desktop-release-channels-first-run.md)

## Context and Problem Statement

Users asked for two things ADR-0025 ruled out: a way to remove everything Kongtrol stored
(settings and saved passwords), and being told when a new version is out. Both touch settled
rules: keeping data on upgrade/reinstall, "Local only" (any outbound call opt-in, documented on
`/privacidad` and in `docs/SECURITY.md`) and GUI-first with CLI parity.

## Decision Drivers

- Reinstalling must bring everything back: keeping data is the default everywhere.
- Delete exactly what is ours: go-keyring cannot list entries; package managers own their files.
- Local only: no request unless the user turns it on; never download or install.
- Same behavior from GUI, CLI, install scripts and the Windows uninstaller.

## Considered Options

1. Keep ADR-0025 as is (no purge, no notice).
2. Always delete data on uninstall — loses data on every reinstall; against apt/Homebrew practice.
3. **Keep data by default, purge on explicit request; opt-in notice from GitHub Releases at
   most once a day, never auto-update.**
4. Auto-update — downloads and runs code; conflicts with package managers and signing.

## Decision Outcome

Chosen: **option 3** (sources in the [research note](../research/2026-11-18-uninstall-and-update-notice.md)).

Uninstall (`internal/uninstall`):
- `kongtrol uninstall` stops tray, daemon and tunnels (DNS restored, kill switch lifted),
  removes the helper, and removes files only when `install.sh`/`install.ps1` listed them in
  `~/.kongtrol/install.manifest` (plus the legacy `%LOCALAPPDATA%\kongtrol\bin` of
  `install.ps1`). Package-manager installs get their manager's command; the Windows installer
  gets its own uninstaller. `--purge` also deletes data; `--data-only` deletes only data;
  y/N confirmation (default N), `--yes`, `--dry-run`.
- Purge = `~/.kongtrol` (and a `--config` file or custom audit log outside it) + keychain
  entries `<profile>.username`/`.password` for every profile named in `kongtrol.yaml`,
  `config.yaml` and their backups, plus `_system.audit_hmac_key`.
- Windows uninstaller: asks once ("¿Borrar también tu configuración y las contraseñas
  guardadas? Si piensas reinstalar, elige No.", default No via `MB_DEFBUTTON2`); silent never
  deletes data; on Yes it runs `kongtrol uninstall --data-only --yes` before files go.
- `install.sh --uninstall [--purge]`, `install.ps1 -Uninstall [-Purge]` delegate to the CLI.
- Dashboard (web/ui Ajustes): "Borrar todos mis datos" with a confirmation listing what goes →
  `POST /api/v1/data/purge {"confirm":true}` (normal API auth) → daemon restarts in setup mode.
- Tray: no uninstall item; the OS uninstaller or `kongtrol uninstall` covers it.

Update notice (`internal/update`):
- `updates.check_enabled` (default false): `kongtrol config update-check on|off|status`,
  web/ui Ajustes, one question in `kongtrol init` (default No). Tray has no settings area: it
  shows the notice and opens the dashboard.
- When on: `GET https://api.github.com/repos/DerotLuna/vpn-kongtrol/releases/latest`, HTTPS
  only, `User-Agent: kongtrol/<version>` + `Accept`, nothing else; at most once per 24 h
  (attempt time kept in `~/.kongtrol/update-check.json`, failures included); 5 s timeout; 1 MiB
  body limit; pre-releases ignored; semver via `golang.org/x/mod/semver`.
- Newer version → `monitor.Event` `update_available` (key `update.event.available`, args
  latest/current/release page) once per version for tray, dashboards and live view; a line in
  `kongtrol version` and `status`. It never downloads or installs.

### Consequences

- Good: reinstall keeps everything; full removal is one explicit step on every surface.
- Good: no request leaves the machine for users who do not opt in.
- Bad: a profile deleted from `kongtrol.yaml` without a backup leaves its keychain entries
  behind (go-keyring cannot list them).
- Bad: Inno `[UninstallRun]` vs `usUninstall` order is not documented (**no verificado**); the
  purge stops Kongtrol itself so it does not depend on it.

### Confirmation

`internal/update/update_test.go` (newer/same/older/prerelease/timeout/24 h/disabled/https),
`internal/uninstall/*_test.go` (per-channel plan, exact credential list, keep-runtime and
keep-manifest), `internal/api/handlers_lifecycle_data_test.go` (auth, confirm, toggle),
`cmd/kongtrol/cmd_config_update_test.go`, `web/ui/src/components/settings/Settings.test.tsx`.
ISCC compile and a manual uninstall on Windows/macOS/Linux are pending.

## More Information

Code: `internal/update/`, `internal/uninstall/`, `internal/api/handlers_lifecycle.go`,
`cmd/kongtrol/cmd_uninstall*.go`, `cmd/kongtrol/cmd_config_update.go`,
`cmd/kongtrol/daemon_update.go`, `packaging/windows/kongtrol.iss`, `install.sh`, `install.ps1`,
`web/ui/src/components/settings/`.
