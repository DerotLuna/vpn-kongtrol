<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0101: Guided restore at start and waiting for the network

Version 1.0.0 · 2026-11-23

Status: Accepted · Date: 2026-11-23 · Decision-makers: project maintainer

Research: [2026-11-23-network-ready-at-logon](../research/2026-11-23-network-ready-at-logon.md).
Refines [ADR-0098](0098-bounded-reconnect-and-restore-at-start.md) and supersedes only its
default of `restore_tunnels`. Keeps [ADR-0033](0033-cli-command-restructure.md) (questions only
on an interactive terminal) and the provider rules in AGENTS.md (never GUI-automate or launch a
consumer VPN app without the user).

## Context

ADR-0098 turned restore on by default. The owner chose (2026-11-23) to ask instead: users who
did not expect Kongtrol to reconnect VPNs at start should not be surprised. Restore also had
two problems:

- At sign-in the daemon starts before Wi-Fi or DHCP is ready, so the one restore connect failed
  and used the watchdog's bounded attempts on nothing.
- A provider profile (Proton, NordVPN, Surfshark, FortiClient) launched its app at boot and
  waited for the user.

## Decision

**One question, after the first VPN the user connects.** `restore_tunnels` unset now means off.
The first successful user connect (`userConnectProfile`: panel, tray, `kongtrol up`, groups,
`daemon <profiles>`) sets `restore_ask: true` in `~/.kongtrol/preferences.json`, unless the user
already chose. The daemon then announces a change (`api.Server.AnnounceChange`, WebSocket
`changed`). `GET /api/v1/startup` reports `ask: true`. The panel (Inicio banner) and the tray
flyout show:

- "¿Volver a conectar tus VPN al encender el equipo?"
- "Kongtrol se abrirá al encender el equipo y volverá a conectar las VPN que dejaste
  conectadas. Las que desconectes tú no vuelven. Puedes cambiarlo en Ajustes › Al iniciar."
- "Sí, activar" / "Ahora no"

`PUT /api/v1/startup {"restore_answer": bool}` answers it. Yes turns on `restore_tunnels` and,
when the install has the tray, the start at sign-in. "Ahora no" saves `restore_tunnels: false`
and leaves the start at sign-in as the installer set it (Inno task, `install.ps1` HKCU Run);
installers are unchanged. Any answer, the Ajustes switch or `kongtrol config restore-tunnels`
clears `restore_ask` (`Preferences.SetRestoreTunnels`), so the question is asked once. Users who
already set `restore_tunnels` keep it and are never asked.

`kongtrol up` asks the same question after a successful connect, only through the ADR-0033
rules (interactive terminal, never with `--json` or `-q`). Without a terminal it prints one line
with `kongtrol config restore-tunnels on` and `kongtrol config autostart on`.

**Wait for the network before restoring.** `monitor.NetworkReady` asks the routing table which
local address would reach a documentation address (`192.0.2.1`, `2001:db8::1`; UDP connect,
nothing sent). The network is ready when that address is not loopback, link-local or
unspecified, and its interface is up and is not one of Kongtrol's own tunnels. Pure Go, no
admin. `monitor.NetworkWait` polls every 2 s for up to 90 s. While waiting, the daemon output
and the audit log say "Esperando conexión a internet para volver a conectar X…". The wait is
not a connect, so it spends no watchdog attempt. With no network after 90 s there is no notice
(one INFO line only). `monitor.WaitNetworkChange` then waits for the next change of
`NetworkFingerprint`, the watchdog's re-arm signal, and restores when the network is ready.
Before each connect the daemon checks that the profile is still wanted and restore is still on.

**Provider apps are never launched at start.** A profile whose adapter type has an appattach
matcher is attached when `appattach.ActiveTunnel` says its tunnel is already up. Otherwise it is
skipped with an INFO line and stays in `wanted.json`.

## Consequences

- Good: nobody gets VPNs reconnected at start without having chosen it, and the choice is
  offered at the moment it makes sense.
- Good: restores at sign-in no longer fail because the Wi-Fi was a few seconds late, and the
  bounded attempts stay for real outages.
- Good: no consumer VPN app opens at boot asking for the user.
- Bad: ADR-0098 users who never touched the switch (restore was on by default) now have it off
  until they answer the question after their next connect.
- Bad: a provider app that brings its tunnel up a little after Kongtrol starts is not
  attached by the restore. The user connects it in the app and Kongtrol attaches as usual.
- Bad: on a network where the routing table has a default route but the internet is not
  reachable (captive portal), the wait ends at once and the connect may fail. The watchdog's
  bounded policy then applies, as before.

## Not verified (no verificado)

- The probe without a network, and on Linux and macOS (only checked online on Windows 11).
- nm-online and Apple `NWPathMonitor` pages (see the research note).

## Confirmation

- `go test ./internal/config/ -run RestoreQuestion`, `./internal/api/ -run Startup`,
  `./internal/monitor/ -run 'NetworkWait|WaitNetworkChange|UsableSource|NetworkReady'`
- `go test ./cmd/kongtrol/ -run 'PlanRestore|RunRestore|MarkRestore|OfferRestore'`
- web/ui: `src/components/onboarding/RestoreQuestion.test.tsx`.
