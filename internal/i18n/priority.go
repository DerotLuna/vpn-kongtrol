package i18n

// Profile priority made visible (ADR-0108): `kongtrol profile priority`,
// `kongtrol profile list`, the order column of `kongtrol policy list` and
// the config check. The panel's strings live in web/ui/src/i18n/pages.
func init() {
	for lang, entries := range map[Lang]map[string]string{
		ES: {
			"config.err.profile_priority":     "Perfil %[1]q: la prioridad %[2]v no es válida.",
			"config.err.profile_priority.fix": "Usa un número entero de 1 a %[3]d (gana el mayor), o 0 para que decida la posición de la regla.",

			"help.kongtrol.profile.priority.short":   "Muestra o cambia la prioridad de un perfil",
			"help.kongtrol.profile.priority.long":    "La prioridad solo decide entre tus reglas cuando dos cubren el mismo sitio o el mismo rango de IP: gana el número mayor. Sin prioridad (0 o none) decide la posición de la regla: la primera va antes. No conecta otra VPN y no cambia las rutas que trae cada VPN.",
			"help.kongtrol.profile.priority.args":    "<perfil> [N|none]",
			"help.kongtrol.profile.priority.example": "  kongtrol profile priority oficina\n  kongtrol profile priority oficina 20\n  kongtrol profile priority oficina none",
			"help.kongtrol.profile.list.short":       "Muestra tus perfiles y su prioridad",
			"help.kongtrol.profile.list.long":        "Muestra cada perfil con su tipo y su orden: «Prioridad N» o «por posición». Para cambiarla: kongtrol profile priority <perfil> <N>.",
			"help.kongtrol.profile.list.example":     "  kongtrol profile list\n  kongtrol profile list --json",

			"cli.profile.priority.value":       "Prioridad %d",
			"cli.profile.priority.by_position": "por posición",
			"cli.profile.priority.show":        "%s: %s",
			"cli.profile.priority.saved":       "%s: %s.",
			"cli.profile.priority.next":        "Para ver el orden de tus reglas: kongtrol policy list",
			"cli.profile.priority.hint":        "Solo decide entre tus reglas cuando dos cubren el mismo sitio o el mismo rango de IP: gana el número mayor.",
			"cli.profile.list.none":            "No hay VPN guardadas. Para crear una: kongtrol profile add <tipo>.",
			"cli.profile.list.col.name":        "PERFIL",
			"cli.profile.list.col.type":        "TIPO",
			"cli.profile.list.col.order":       "ORDEN",
			"cli.policy.list.col.order":        "ORDEN",
			"cli.policy.list.order_hint":       "Orden: solo cuenta cuando dos reglas cubren el mismo sitio o el mismo rango de IP. Para cambiarlo: kongtrol profile priority <perfil> <N>.",
			"wizard.priority.placeholder":      "vacío = por posición",
		},
		EN: {
			"config.err.profile_priority":     "Profile %[1]q: priority %[2]v is invalid.",
			"config.err.profile_priority.fix": "Use a whole number from 1 to %[3]d (the higher wins), or 0 to let the rule's position decide.",

			"help.kongtrol.profile.priority.short":   "Show or change a profile's priority",
			"help.kongtrol.profile.priority.long":    "Priority only decides between your rules when two cover the same site or the same IP range: the higher number wins. With no priority (0 or none) the rule's position decides: the first one goes first. It never connects another VPN and never changes the routes each VPN brings.",
			"help.kongtrol.profile.priority.args":    "<profile> [N|none]",
			"help.kongtrol.profile.priority.example": "  kongtrol profile priority office\n  kongtrol profile priority office 20\n  kongtrol profile priority office none",
			"help.kongtrol.profile.list.short":       "Show your profiles and their priority",
			"help.kongtrol.profile.list.long":        "Shows each profile with its type and its order: \"Priority N\" or \"by position\". To change it: kongtrol profile priority <profile> <N>.",
			"help.kongtrol.profile.list.example":     "  kongtrol profile list\n  kongtrol profile list --json",

			"cli.profile.priority.value":       "Priority %d",
			"cli.profile.priority.by_position": "by position",
			"cli.profile.priority.show":        "%s: %s",
			"cli.profile.priority.saved":       "%s: %s.",
			"cli.profile.priority.next":        "To see the order of your rules: kongtrol policy list",
			"cli.profile.priority.hint":        "It only decides between your rules when two cover the same site or the same IP range: the higher number wins.",
			"cli.profile.list.none":            "No saved VPNs. To create one: kongtrol profile add <type>.",
			"cli.profile.list.col.name":        "PROFILE",
			"cli.profile.list.col.type":        "TYPE",
			"cli.profile.list.col.order":       "ORDER",
			"cli.policy.list.col.order":        "ORDER",
			"cli.policy.list.order_hint":       "Order only counts when two rules cover the same site or the same IP range. To change it: kongtrol profile priority <profile> <N>.",
			"wizard.priority.placeholder":      "empty = by position",
		},
	} {
		for key, message := range entries {
			messages[lang][key] = message
		}
	}
}
