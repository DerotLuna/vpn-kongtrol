package i18n

// Strings for the recent destinations list (ADR-0054): `kongtrol traffic
// recent` and `kongtrol config recent-connections`. Merged into the
// catalog at init, so the parity test covers them like every other key.

func init() {
	for lang, m := range recentMessages {
		for k, v := range m {
			messages[lang][k] = v
		}
	}
}

var recentMessages = map[Lang]map[string]string{
	ES: {
		"help.kongtrol.config.recent-connections.args":    "<on|off|status>",
		"help.kongtrol.config.recent-connections.short":   "Activa o apaga la lista de destinos recientes",
		"help.kongtrol.config.recent-connections.long":    "Encendida por defecto. Kongtrol anota a qué direcciones se conectó este equipo en la última hora, por qué perfil salió cada una y qué programa la abrió. La lista vive solo en la memoria de Kongtrol: no se guarda en disco ni en los registros, no sale del equipo y se borra al apagarla o al cerrar Kongtrol.",
		"help.kongtrol.config.recent-connections.example": "  kongtrol config recent-connections status\n  kongtrol config recent-connections off\n  kongtrol config recent-connections on",

		"help.kongtrol.traffic.short":   "Muestra el tráfico de este equipo",
		"help.kongtrol.traffic.long":    "Comandos para ver por dónde sale tu tráfico.",
		"help.kongtrol.traffic.example": "  kongtrol traffic recent",

		"help.kongtrol.traffic.recent.short":   "Muestra a qué destinos se conectó este equipo hace poco",
		"help.kongtrol.traffic.recent.long":    "Lista los destinos de la última hora: nombre (si una regla lo resolvió) o IP, puerto, programa, por qué perfil salió (o directo) y cuántas conexiones hubo. Necesita Kongtrol abierto. Los datos viven solo en memoria y no salen del equipo. Apágalo con: kongtrol config recent-connections off",
		"help.kongtrol.traffic.recent.example": "  kongtrol traffic recent --limit 20\n  kongtrol traffic recent --search github\n  kongtrol traffic recent --via direct",

		"cli.recent.flag.limit":   "Cuántos destinos mostrar (0 = todos)",
		"cli.recent.flag.search":  "Muestra solo los que contienen este texto en el sitio, la IP o el programa",
		"cli.recent.flag.via":     "Muestra solo los que salieron por este perfil (direct = conexión normal, sin VPN)",
		"cli.recent.no_match":     "Ningún destino reciente coincide con el filtro.",
		"cli.recent.no_match_do":  "Prueba con otro texto o quita --search / --via para ver toda la lista.",
		"cli.recent.invalid":      "%q no es válido · usa on, off o status",
		"cli.recent.on":           "Lista de destinos recientes encendida",
		"cli.recent.off":          "Lista de destinos recientes apagada · se borró lo que había",
		"cli.recent.notice":       "Solo en memoria: no se guarda en disco ni sale del equipo; se olvida a la hora.",
		"cli.recent.status_on":    "Lista de destinos recientes: encendida (solo en memoria, se olvida a la hora)",
		"cli.recent.status_off":   "Lista de destinos recientes: apagada · para encenderla: kongtrol config recent-connections on",
		"cli.recent.no_daemon":    "Kongtrol no está abierto: la lista vive en su memoria.",
		"cli.recent.no_daemon_do": "Ábrelo con: kongtrol up (o desde el ícono) y vuelve a intentarlo.",
		"cli.recent.unsupported":  "Este sistema no permite ver las conexiones sin permisos de administrador (macOS por ahora).",
		"cli.recent.disabled":     "La lista de destinos recientes está apagada.",
		"cli.recent.disabled_do":  "Para encenderla: kongtrol config recent-connections on",
		"cli.recent.empty":        "Todavía no hay destinos en la lista (se revisa cada 15 s).",
		"cli.recent.direct":       "directo",
		"cli.recent.col.dest":     "DESTINO",
		"cli.recent.col.port":     "PUERTO",
		"cli.recent.col.process":  "PROGRAMA",
		"cli.recent.col.via":      "SALE POR",
		"cli.recent.col.count":    "VECES",
		"cli.recent.col.last":     "ÚLTIMA VEZ",
	},
	EN: {
		"help.kongtrol.config.recent-connections.args":    "<on|off|status>",
		"help.kongtrol.config.recent-connections.short":   "Turn the recent destinations list on or off",
		"help.kongtrol.config.recent-connections.long":    "On by default. Kongtrol notes which addresses this computer connected to in the last hour, which profile each one went through and which program opened it. The list lives only in Kongtrol's memory: it is never saved to disk or logs, never leaves this computer, and is erased when you turn it off or close Kongtrol.",
		"help.kongtrol.config.recent-connections.example": "  kongtrol config recent-connections status\n  kongtrol config recent-connections off\n  kongtrol config recent-connections on",

		"help.kongtrol.traffic.short":   "Show this computer's traffic",
		"help.kongtrol.traffic.long":    "Commands to see where your traffic goes.",
		"help.kongtrol.traffic.example": "  kongtrol traffic recent",

		"help.kongtrol.traffic.recent.short":   "Show which destinations this computer connected to lately",
		"help.kongtrol.traffic.recent.long":    "Lists the destinations of the last hour: name (when a rule resolved it) or IP, port, program, which profile it went through (or direct) and how many connections there were. Needs Kongtrol open. The data lives only in memory and never leaves this computer. Turn it off with: kongtrol config recent-connections off",
		"help.kongtrol.traffic.recent.example": "  kongtrol traffic recent --limit 20\n  kongtrol traffic recent --search github\n  kongtrol traffic recent --via direct",

		"cli.recent.flag.limit":   "How many destinations to show (0 = all)",
		"cli.recent.flag.search":  "Show only those with this text in the site, IP or program",
		"cli.recent.flag.via":     "Show only those that went through this profile (direct = normal connection, no VPN)",
		"cli.recent.no_match":     "No recent destination matches the filter.",
		"cli.recent.no_match_do":  "Try other text or drop --search / --via to see the whole list.",
		"cli.recent.invalid":      "%q is not valid · use on, off or status",
		"cli.recent.on":           "Recent destinations list turned on",
		"cli.recent.off":          "Recent destinations list turned off · what it held was erased",
		"cli.recent.notice":       "Memory only: never saved to disk or sent off this computer; forgotten after an hour.",
		"cli.recent.status_on":    "Recent destinations list: on (memory only, forgotten after an hour)",
		"cli.recent.status_off":   "Recent destinations list: off · to turn it on: kongtrol config recent-connections on",
		"cli.recent.no_daemon":    "Kongtrol is not open: the list lives in its memory.",
		"cli.recent.no_daemon_do": "Open it with: kongtrol up (or from the icon) and try again.",
		"cli.recent.unsupported":  "This system does not let Kongtrol see connections without admin rights (macOS for now).",
		"cli.recent.disabled":     "The recent destinations list is off.",
		"cli.recent.disabled_do":  "To turn it on: kongtrol config recent-connections on",
		"cli.recent.empty":        "No destinations in the list yet (checked every 15 s).",
		"cli.recent.direct":       "direct",
		"cli.recent.col.dest":     "DESTINATION",
		"cli.recent.col.port":     "PORT",
		"cli.recent.col.process":  "PROGRAM",
		"cli.recent.col.via":      "GOES THROUGH",
		"cli.recent.col.count":    "TIMES",
		"cli.recent.col.last":     "LAST SEEN",
	},
}
