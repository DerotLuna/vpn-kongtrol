package i18n

// Strings for bounded reconnects and what Kongtrol does when the computer
// starts (ADR-0098): the watchdog's give-up notice, `kongtrol config
// autostart` and `kongtrol config restore-tunnels`, and the daemon's
// restore at start. Merged into the catalog at init, so the parity test
// covers them like every other key.

func init() {
	for lang, m := range startupMessages {
		for k, v := range m {
			messages[lang][k] = v
		}
	}
}

var startupMessages = map[Lang]map[string]string{
	ES: {
		"watchdog.event.gave_up":                       "No se pudo reconectar %[1]s tras %[2]s intentos · Kongtrol deja de intentarlo y prueba otra vez solo si cambia tu red. Revisa tu internet o el servidor de esa VPN y pulsa Reintentar (o: kongtrol up %[1]s)",
		"watchdog.event.gave_up_blocking":              "No se pudo reconectar %[1]s tras %[2]s intentos · el kill switch sigue bloqueando sus destinos. Para usarlos sin VPN, desconéctala (kongtrol down %[1]s); para probar otra vez, pulsa Reintentar (o: kongtrol up %[1]s)",
		"watchdog.event.gave_up_blocking_full":         "No se pudo reconectar %[1]s tras %[2]s intentos · el kill switch sigue bloqueando todo tu tráfico. Para volver a tener internet, desconéctala (kongtrol down %[1]s); para probar otra vez, pulsa Reintentar (o: kongtrol up %[1]s)",
		"watchdog.event.reconnect_blocked":             "%[1]s no se reconectó: otra VPN lleva todo tu tráfico ahora · desconecta esa VPN y pulsa Reintentar (o: kongtrol up %[1]s)",
		"watchdog.event.action_retry":                  "Reintentar %s",
		"cli.alert.retry_rearmed":                      "tu red cambió: Kongtrol vuelve a intentar reconectar %s",
		"cli.daemon.restoring":                         "Reconectando las VPN que tenías conectadas: %s",
		"startup.err.no_tray":                          "Esta instalación no tiene el ícono de Kongtrol, así que no hay nada que abrir al iniciar sesión",
		"startup.err.no_tray.fix":                      "Instala Kongtrol con su ícono (el instalador, o install.sh / install.ps1 sin --cli-only) y vuelve a activarlo",
		"startup.err.autostart":                        "No se pudo cambiar el inicio automático: %s",
		"startup.err.autostart.fix":                    "Cámbialo en la configuración de inicio de tu sistema (Windows: Configuración › Aplicaciones › Inicio) o vuelve a intentarlo",
		"cli.startup.invalid":                          "%q no es válido · usa: kongtrol config %s on, off o status",
		"cli.startup.autostart.on":                     "Kongtrol se abrirá cuando inicies sesión",
		"cli.startup.autostart.off":                    "Kongtrol ya no se abrirá al iniciar sesión · ábrelo desde el menú de aplicaciones cuando lo necesites",
		"cli.startup.autostart.status_on":              "Abrir Kongtrol al iniciar sesión: sí · para quitarlo: kongtrol config autostart off",
		"cli.startup.autostart.status_off":             "Abrir Kongtrol al iniciar sesión: no · para activarlo: kongtrol config autostart on",
		"cli.startup.autostart.unsupported":            "Esta instalación no tiene el ícono de Kongtrol: no se abre al iniciar sesión · instálalo con su ícono para poder activarlo",
		"cli.startup.restore.on":                       "Al iniciar, Kongtrol volverá a conectar las VPN que tenías conectadas",
		"cli.startup.restore.off":                      "Al iniciar, Kongtrol no conectará ninguna VPN · conéctalas tú desde su ícono o con kongtrol up",
		"cli.startup.restore.status_on":                "Volver a conectar al iniciar: sí · para quitarlo: kongtrol config restore-tunnels off",
		"cli.startup.restore.status_off":               "Volver a conectar al iniciar: no · para activarlo: kongtrol config restore-tunnels on",
		"help.kongtrol.config.autostart.args":          "<on|off|status>",
		"help.kongtrol.config.autostart.short":         "Abre Kongtrol cuando inicias sesión, o deja de hacerlo",
		"help.kongtrol.config.autostart.long":          "Kongtrol se abre solo cuando inicias sesión en este equipo (lo deja así el instalador). Con off ya no se abre; con on vuelve a abrirse. Usa el mecanismo de tu sistema: en Windows, Configuración › Aplicaciones › Inicio; en Linux, la entrada de inicio automático; en macOS, un agente de inicio de tu usuario. Una actualización respeta lo que elijas.",
		"help.kongtrol.config.autostart.example":       "  kongtrol config autostart status\n  kongtrol config autostart off\n  kongtrol config autostart on",
		"help.kongtrol.config.restore-tunnels.args":    "<on|off|status>",
		"help.kongtrol.config.restore-tunnels.short":   "Al iniciar, vuelve a conectar las VPN que tenías conectadas",
		"help.kongtrol.config.restore-tunnels.long":    "Encendido por defecto. Si apagas o reinicias el equipo, o Kongtrol se cierra de golpe, al volver a abrirse conecta otra vez las VPN que tenías conectadas. Las que desconectaste tú no se conectan, y kongtrol down (o \"Salir y desconectar todo\") hace que no se conecte ninguna. Si una no conecta, Kongtrol lo intenta pocas veces y te avisa una sola vez.",
		"help.kongtrol.config.restore-tunnels.example": "  kongtrol config restore-tunnels status\n  kongtrol config restore-tunnels off\n  kongtrol config restore-tunnels on",
	},
	EN: {
		"watchdog.event.gave_up":                       "Could not reconnect %[1]s after %[2]s attempts · Kongtrol stops trying and tries again on its own only if your network changes. Check your internet or that VPN's server and press Retry (or: kongtrol up %[1]s)",
		"watchdog.event.gave_up_blocking":              "Could not reconnect %[1]s after %[2]s attempts · the kill switch keeps blocking its destinations. To use them without the VPN, disconnect it (kongtrol down %[1]s); to try again, press Retry (or: kongtrol up %[1]s)",
		"watchdog.event.gave_up_blocking_full":         "Could not reconnect %[1]s after %[2]s attempts · the kill switch keeps blocking all your traffic. To get internet back, disconnect it (kongtrol down %[1]s); to try again, press Retry (or: kongtrol up %[1]s)",
		"watchdog.event.reconnect_blocked":             "%[1]s did not reconnect: another VPN carries all your traffic now · disconnect that VPN and press Retry (or: kongtrol up %[1]s)",
		"watchdog.event.action_retry":                  "Retry %s",
		"cli.alert.retry_rearmed":                      "your network changed: Kongtrol tries to reconnect %s again",
		"cli.daemon.restoring":                         "Reconnecting the VPNs you had connected: %s",
		"startup.err.no_tray":                          "This install has no Kongtrol icon, so there is nothing to open when you sign in",
		"startup.err.no_tray.fix":                      "Install Kongtrol with its icon (the installer, or install.sh / install.ps1 without --cli-only) and turn it on again",
		"startup.err.autostart":                        "Could not change the sign-in start: %s",
		"startup.err.autostart.fix":                    "Change it in your system's startup settings (Windows: Settings › Apps › Startup) or try again",
		"cli.startup.invalid":                          "%q is not valid · use: kongtrol config %s on, off or status",
		"cli.startup.autostart.on":                     "Kongtrol will open when you sign in",
		"cli.startup.autostart.off":                    "Kongtrol will no longer open when you sign in · open it from your apps menu when you need it",
		"cli.startup.autostart.status_on":              "Open Kongtrol when you sign in: yes · to stop it: kongtrol config autostart off",
		"cli.startup.autostart.status_off":             "Open Kongtrol when you sign in: no · to turn it on: kongtrol config autostart on",
		"cli.startup.autostart.unsupported":            "This install has no Kongtrol icon: it does not open when you sign in · install it with its icon to turn this on",
		"cli.startup.restore.on":                       "When it starts, Kongtrol will reconnect the VPNs you had connected",
		"cli.startup.restore.off":                      "When it starts, Kongtrol will connect no VPN · connect them yourself from its icon or with kongtrol up",
		"cli.startup.restore.status_on":                "Reconnect when starting: yes · to stop it: kongtrol config restore-tunnels off",
		"cli.startup.restore.status_off":               "Reconnect when starting: no · to turn it on: kongtrol config restore-tunnels on",
		"help.kongtrol.config.autostart.args":          "<on|off|status>",
		"help.kongtrol.config.autostart.short":         "Open Kongtrol when you sign in, or stop doing so",
		"help.kongtrol.config.autostart.long":          "Kongtrol opens on its own when you sign in to this computer (the installer sets it up). off stops that; on brings it back. It uses your system's own mechanism: on Windows, Settings › Apps › Startup; on Linux, the autostart entry; on macOS, a launch agent for your user. An update keeps your choice.",
		"help.kongtrol.config.autostart.example":       "  kongtrol config autostart status\n  kongtrol config autostart off\n  kongtrol config autostart on",
		"help.kongtrol.config.restore-tunnels.args":    "<on|off|status>",
		"help.kongtrol.config.restore-tunnels.short":   "When starting, reconnect the VPNs you had connected",
		"help.kongtrol.config.restore-tunnels.long":    "On by default. If you shut down or restart the computer, or Kongtrol closes suddenly, it connects the VPNs you had connected again when it opens. The ones you disconnected yourself stay off, and kongtrol down (or \"Quit and disconnect all\") means none is connected. If one does not connect, Kongtrol tries a few times and tells you once.",
		"help.kongtrol.config.restore-tunnels.example": "  kongtrol config restore-tunnels status\n  kongtrol config restore-tunnels off\n  kongtrol config restore-tunnels on",
	},
}
