<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0098: Bounded reconnects, restore at start and a visible sign-in switch

Version 1.0.0 · 2026-11-23

Status: Accepted · Date: 2026-11-23 · Decision-makers: project maintainer

Research: [2026-11-23-bounded-reconnect-and-restore](../research/2026-11-23-bounded-reconnect-and-restore.md).
Consistent with [ADR-0095](0095-save-rules-paused-and-bound-auto-connect.md) (three attempts
for rule auto-connect) and [ADR-0018](0018-failover-only-to-declared-profiles.md) (failover only
to declared backups). Keeps [ADR-0050](0050-installer-resume-tunnels-and-open-request.md)
(`resume.json`) and [ADR-0075](0075-install-scripts-install-cli-and-tray.md) (autostart entries).

## Context

The watchdog retried a dropped profile forever (2 s × 2ⁿ, capped at 5 min) and said "Kongtrol
sigue intentando solo". The owner asked (2026-11-23) for bounded retries that never pester, a
clear notice with a Retry action, and an automatic retry only on a signal that retrying makes
sense. Separately, after a shutdown, restart or crash the VPNs the user had on did not come
back, and "open Kongtrol when you sign in" existed only as an installer task.

## Decision

**Bounded reconnect.** `monitor.DefaultRetryDelays` = one attempt after 2 s, 30 s and 2 min
(three per outage, like ADR-0095). The first is fast for a blip; the last ends ~2.5 min after
the drop. A declared backup is still tried once per outage. When all fail, the watchdog emits
`reconnect_gave_up` once per outage and stops. The daemon turns it into one notice
`watchdog.event.gave_up` with `can_connect`: "Reintentar <perfil>" in the panel (toast, Inicio
notices, Actividad) and the tray, `kongtrol up <perfil>` in the CLI text. NetworkManager uses
the same shape (4 tries, then blocked) and OpenVPN and Mullvad retry forever; three keeps rule
and tunnel retries alike, and the re-arm below covers longer outages.

**Re-arm signals.** A profile that gave up gets one new, silent round when
`NetworkFingerprint` changes or the computer wakes from sleep. The fingerprint is the up,
non-loopback interfaces with their IPv4 addresses and a has-global-IPv6 flag. IPv6 addresses
are left out because temporary ones rotate (RFC 8981). A wake is detected when wall time runs
more than 30 s ahead of Go's monotonic time between two polls. NetworkManager resets its retries
on carrier-on and on wake. Its 300 s timer reset is not copied, because that would retry forever.
A connected status ends the outage, so the user's Retry (or any connect) starts fresh.

**Errors only the user can fix.** `Watchdog.SetWaitForUser` classifies them. One case is another
VPN that carries all traffic (ADR-0099, `app.ErrFullTunnelConflict`). Such an error is not an
attempt. It stops the round, gives one notice `watchdog.event.reconnect_blocked` with Retry,
and re-arms on the same signals. Until ADR-0099 is merged, `reconnectWaitsForUser` in
`cmd/kongtrol/daemon_watchdog.go` matches nothing. The merge sets it to
`errors.Is(err, app.ErrFullTunnelConflict)`.

**Kill switch unchanged.** A protected profile that gave up keeps its block. The notice variant
says how to get traffic back. `gave_up_blocking` covers split destinations and
`gave_up_blocking_full` covers all traffic. In both, the user disconnects the profile. It is
chosen from `KillSwitchService.Status()` (`Blocking`, `FullTunnel`).

**Restore at start.** The daemon keeps the profiles the user wants up in
`~/.kongtrol/run/wanted.json`. The file is `daemonctl.Wanted`, written with fsync and atomic
replace on every change, so a crash or power cut keeps it.

- A user connect that succeeds adds the profile. User connects come from the API (panel, tray,
  `kongtrol up`, groups) and from `daemon <profiles>`.
- A user disconnect removes it.
- `POST /api/v1/shutdown` (`kongtrol down`, "Salir y desconectar todo", installers) and the
  data purge empty it.
- A restart (`POST /api/v1/restart`) and a daemon stopped by the OS keep it. On Windows, Go
  delivers logoff and shutdown as SIGTERM.
- Watchdog, scheduler and rule connects never touch it.

On start, with `restore_tunnels` on, each saved profile that still exists gets one connect.
A provider profile attaches or waits for the user, as with `kongtrol up`. A failure is handed to
the watchdog (`MarkActiveAfterFailedConnect`), so the bounded policy and its one notice apply.

The setting lives in `~/.kongtrol/preferences.json` as `restore_tunnels`, default on. It
belongs to the machine: the same `kongtrol.yaml` exported to another computer should not
reconnect there. Default on because the owner expects it, and because Tailscale persists
`WantRunning` and WireGuard for Windows installs active tunnels as automatic services. Mullvad
defaults auto-connect off; we choose differently because Kongtrol only restores what the user
had on.

**Why keep resume.json.** `resume.json` is one-shot and applies whatever the setting is, for
24 h. It serves installers. `wanted.json` is persistent and follows the setting. An upgrade
empties `wanted.json` (`down --all --remember`), and `resume` reconnects the profiles, which
writes them back. Merging the two would make upgrades depend on the user's setting.

**Sign-in start, visible.** `internal/autostart` changes the OS's own per-user entry:

- **Windows:** the HKCU `Run` value "Kongtrol". It is written only when missing. On/off is set
  in `Explorer\StartupApproved\Run`, the switch that Settings › Apps › Startup and Task Manager
  use, so an installer update that rewrites `Run` keeps the choice.
- **Linux:** `Hidden=true` in the user's `kongtrol.desktop`. It also overrides the packages'
  `/etc/xdg/autostart` entry (XDG Autostart §2.2–2.3). `install.sh` 2.9.0 keeps `Hidden=true`
  on update.
- **macOS:** a per-user LaunchAgent with `RunAtLoad`.

A CLI-only install reports unsupported, with what to do. The switch is in the panel (Ajustes ›
Al iniciar, `GET/PUT /api/v1/startup`, audit `config.startup.set`) and in the CLI (`kongtrol
config autostart`, `kongtrol config restore-tunnels`). Neither command needs the daemon.
`kongtrol uninstall` removes entries Kongtrol wrote for its own tray. `kongtrol.iss` 1.8.0
deletes the `StartupApproved` value on uninstall.

## Consequences

- Good: no endless retries or repeated toasts. There is one notice per outage with a Retry
  action everywhere, and longer outages recover by themselves when the network returns.
- Good: after a reboot, crash or power cut, what was on comes back. `down` still means "nothing".
- Good: the sign-in start can be seen and changed without reinstalling, and an update keeps
  the choice.
- Bad: a network that changes all the time (for example, many VPN adapters coming and going)
  starts more silent rounds. Each round is bounded to three attempts.
- Bad: after a reboot the kill switch blocks nothing until a protected profile connects again.
  The kill switch only blocks tunnels that were up in this daemon, as before. This is not a new
  gap.
- Bad: `StartupApproved` is undocumented by Microsoft.

## Not verified (no verificado)

- The `02` (enabled) value of `StartupApproved`. Only `03` + FILETIME was observed on this
  machine.
- Inno `ValueType: none` + `dontcreatekey uninsdeletevalue` was not run end to end.
- Linux and macOS autostart were not run here, because this machine runs Windows. Only the pure
  format helpers were tested, on every OS.
- Sleep detection by monotonic versus wall time was not observed during a real suspend.

## Confirmation

- `go test ./internal/monitor/ -run 'RetryDelay|GivesUp|Fresh|AfterFailed|GaveUp|WaitForUser|Woke|Fingerprint|Failover'`
- `go test ./internal/daemonctl/ -run Wanted`, `./internal/autostart/`,
  `./internal/api/ -run 'Startup|ShutdownCalls'`
- `go test ./cmd/kongtrol/ -run 'RestoreTargets|GaveUp|UserConnect'`
- web/ui: `Settings.test.tsx` (startup switches) and `ActivityPage.test.tsx` (Reintentar).
