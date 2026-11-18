<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0036: Native tray menu with translated states, stopped-daemon state, versions, and installer upgrades that keep the helper current

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

On v0.6.0-rc.3 (Windows 11) the tray's "Conexiones VPN" mixed `brainbox · respuesta sin
verificar` with `ekiipago [disconnected]` (the raw English status whenever a tunnel had no
health info), the menu had no version, the tray icon was the 512 px raster downscaled onto an
opaque background, `kongtrol --version` ran an older manual copy (the installer's PATH task was
off by default), and after upgrading the helper stayed outdated: the installer only installed it
on a first install. After `kongtrol down` the tray only said "Kongtrol no responde". Research:
[2026-11-18-native-tray-menu](../research/2026-11-18-native-tray-menu.md).

## Decision Drivers

- GUI-first: the tray must say the state in the user's language, like the CLI and dashboards.
- DESIGN.md: green means up, Four States Rule (connected / connecting / no answer / error), a word
  with every color; amber is never a state.
- Use what `fyne.io/systray` already supports per OS; no new UI toolkit in this change (a custom
  flyout is a separate effort).
- Never ask for UAC needlessly, but never leave the helper outdated after an upgrade.

## Decision

1. **Menu layout:** status header ("N VPN conectada(s)" / "Ninguna VPN conectada" /
   "Kongtrol detenido" / "Iniciando Kongtrol…"), kill switch and public IP lines, then the lines
   that ask for one action (Iniciar Kongtrol, Instalar/Actualizar ayudante, Conectar <perfil>);
   primary actions (Abrir panel, Conexiones VPN ▸, Grupos ▸); "Kongtrol vX" (disabled); exits.
2. **VPN lines** are always `name · <state word>` (`tray.state.*`), with `· sin verificar`
   when the server can't be checked, `· sin respuesta` (caution) when it does not answer, or the
   attach / error message. Connected VPNs are check items listed first (slots relabelled each
   refresh, since systray cannot reorder). Each line has a state dot icon (live / steel /
   caution / danger; a grey ring for off) on a transparent background.
3. **Tray icon** is rendered from the vector mark (`generate.go` → `brand/tray-<px>.png`,
   transparent), tinted for the taskbar (Windows registry `SystemUsesLightTheme`), a template
   image on macOS, light ink on Linux; the aggregate state is a small corner badge (green = up,
   caution = trouble, grey ring = stopped), never a recolor. Set only when it changes.
4. **Stopped daemon:** the tray keeps polling. A refused connection is "stopped" (shows
   "Iniciar Kongtrol", which runs `ensureDaemon`), a timeout is "no responde"; it recovers as
   soon as any client starts the daemon. "Salir y desconectar todo" still closes the tray.
5. **Version:** `GET /api/v1/version` (the daemon's build version). Tray, classic dashboard
   (sidebar foot) and the new UI (footer) show it; the tray falls back to its own `-X
   main.version` (now set in every tray build) while the daemon is stopped.
6. **Helper:** `GET /api/v1/helper/status` adds `outdated`. Clients say "Actualizar ayudante"
   and the classic dashboard shows it once as a page notice (rows stay `conectado · sin
   verificar`). After a successful install the tray notifies "Ayudante listo · ya no hace falta
   volver a instalarlo" and hides the item at once.
7. **Installer:** the PATH task is checked by default and prepends `{app}` to the user PATH
   (removed on uninstall). Every upgrade (also silent) runs `kongtrol service install
   --if-outdated`: UAC only when the installed helper speaks an older protocol.
8. **Doctor:** "kongtrol en la terminal" warns when the first `kongtrol` in PATH is another copy,
   or when there are several, with what to do.

## Consequences

- Good: one vocabulary across CLI, dashboards and tray; the menu reads at a glance; upgrades keep
  the helper working; the stale-copy confusion is visible in `kongtrol doctor`.
- Bad: Windows menus have no item tooltips, so detail stays in the dashboard; on Windows a
  checked item shows both the check and the dot (no `MNS_CHECKORBMP`); at 16 px the ringed mark
  is blurry; macOS badges are monochrome (template). Inno Setup keeps a previous "PATH off"
  choice on upgrade.
- Not verified on screen: Windows menu rendering, macOS and Linux tray (no native build here).

## Links

- Code: `cmd/kongtrol-tray/tray_state.go`, `tray_view.go`, `tray_menu.go`, `trayicon_<os>.go`,
  `assets/icon.go`, `assets/state_icons.go`, `assets/brand/generate.go`,
  `internal/api/handlers_helper.go`, `internal/doctor/path.go`, `cmd/kongtrol/cmd_service.go`,
  `packaging/windows/kongtrol.iss`.
- Research: [2026-11-18-native-tray-menu](../research/2026-11-18-native-tray-menu.md).
