package i18n

// Routes each profile brings by itself: `kongtrol routes vpn` (ADR-0107). The
// panel's strings live in web/ui/src/i18n/pages/rules.ts.
func init() {
	for lang, entries := range map[Lang]map[string]string{
		ES: {
			"help.kongtrol.routes.vpn.short":   "Muestra las rutas que trae cada perfil",
			"help.kongtrol.routes.vpn.long":    "Muestra las rutas que trae cada perfil por su cuenta: las del archivo (.ovpn o .conf) y las que manda el servidor al conectar. Kongtrol no las cambia. También dice qué gana cuando una de tus reglas toca la misma dirección.",
			"help.kongtrol.routes.vpn.args":    "[perfil]",
			"help.kongtrol.routes.vpn.example": "  kongtrol routes vpn\n  kongtrol routes vpn oficina\n  kongtrol routes vpn --json",

			"cli.routes.vpn.none":            "No hay perfiles guardados.",
			"cli.routes.vpn.connected":       "conectada",
			"cli.routes.vpn.disconnected":    "desconectada",
			"cli.routes.vpn.app_all":         "La app decide: túnel completo (todo el tráfico).",
			"cli.routes.vpn.app":             "La app decide qué pasa por ella. Kongtrol no ve sus rutas.",
			"cli.routes.vpn.unreadable":      "No se pudo leer el archivo. Las rutas aparecen al conectar.",
			"cli.routes.vpn.all_traffic":     "Todo el tráfico",
			"cli.routes.vpn.origin.file":     "del archivo",
			"cli.routes.vpn.origin.server":   "del servidor",
			"cli.routes.vpn.no_file_routes":  "El archivo no trae rutas.",
			"cli.routes.vpn.rules_only":      "Kongtrol no usa estas rutas: por este perfil solo sale lo que dicen tus reglas.",
			"cli.routes.vpn.server_pending":  "Las del servidor aparecen al conectar.",
			"cli.routes.vpn.readonly":        "Vienen del archivo o del servidor de la VPN. Para cambiarlas, edita el archivo o pídeselo a tu proveedor.",
			"cli.routes.vpn.precedence":      "Gana la dirección más específica. Si dos piden lo mismo, el sistema elige una.",
			"cli.routes.vpn.all_wins":        "Tus reglas por otro perfil ganan sobre «Todo el tráfico»: son más específicas.",
			"cli.routes.vpn.next":            "Para que algo salga por otro perfil: kongtrol policy add <dirección> --via <perfil>",
			"cli.routes.vpn.conflict.rule":   "Tu regla «%s» gana para %s: es más específica. Sale por «%s» cuando está conectada.",
			"cli.routes.vpn.conflict.vpn":    "Esta ruta gana para %s sobre tu regla «%s»: es más específica.",
			"cli.routes.vpn.conflict.system": "Tu regla «%s» pide lo mismo (%s) por «%s». El sistema elige una; para asegurarlo, usa una dirección más específica.",
		},
		EN: {
			"help.kongtrol.routes.vpn.short":   "Show the routes each profile brings",
			"help.kongtrol.routes.vpn.long":    "Shows the routes each profile brings by itself: the ones in its file (.ovpn or .conf) and the ones its server sends on connect. Kongtrol does not change them. It also says what wins when one of your rules covers the same address.",
			"help.kongtrol.routes.vpn.args":    "[profile]",
			"help.kongtrol.routes.vpn.example": "  kongtrol routes vpn\n  kongtrol routes vpn office\n  kongtrol routes vpn --json",

			"cli.routes.vpn.none":            "No saved profiles.",
			"cli.routes.vpn.connected":       "connected",
			"cli.routes.vpn.disconnected":    "disconnected",
			"cli.routes.vpn.app_all":         "The app decides: full tunnel (all traffic).",
			"cli.routes.vpn.app":             "The app decides what goes through it. Kongtrol can't see its routes.",
			"cli.routes.vpn.unreadable":      "Couldn't read the file. Routes show up once connected.",
			"cli.routes.vpn.all_traffic":     "All traffic",
			"cli.routes.vpn.origin.file":     "from the file",
			"cli.routes.vpn.origin.server":   "from the server",
			"cli.routes.vpn.no_file_routes":  "The file brings no routes.",
			"cli.routes.vpn.rules_only":      "Kongtrol doesn't use these routes: only what your rules say goes through this profile.",
			"cli.routes.vpn.server_pending":  "The server's routes show up once connected.",
			"cli.routes.vpn.readonly":        "They come from the VPN's file or server. To change them, edit the file or ask your provider.",
			"cli.routes.vpn.precedence":      "The most specific address wins. If two ask for the same, the system picks one.",
			"cli.routes.vpn.all_wins":        "Your rules through another profile win over \"All traffic\": they're more specific.",
			"cli.routes.vpn.next":            "To send something through another profile: kongtrol policy add <address> --via <profile>",
			"cli.routes.vpn.conflict.rule":   "Your rule \"%s\" wins for %s: it's more specific. It goes through \"%s\" while connected.",
			"cli.routes.vpn.conflict.vpn":    "This route wins for %s over your rule \"%s\": it's more specific.",
			"cli.routes.vpn.conflict.system": "Your rule \"%s\" asks for the same (%s) through \"%s\". The system picks one; to be sure, use a more specific address.",
		},
	} {
		for key, text := range entries {
			messages[lang][key] = text
		}
	}
}
