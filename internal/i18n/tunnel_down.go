package i18n

func init() {
	for lang, entries := range tunnelDownMessages {
		for key, message := range entries {
			messages[lang][key] = message
		}
	}
}

var tunnelDownMessages = map[Lang]map[string]string{
	ES: {
		"cli.daemon.down.kept_protection":              "Kongtrol sigue activo para aplicar el bloqueo de las reglas. Para detenerlo, usa kongtrol down --all.",
		"cli.policy.add.flag.on_tunnel_down":           "al desconectarse el perfil: inherit (por defecto), allow, block o connect",
		"cli.profile.add.flag.on_tunnel_down":          "acción de sus reglas al desconectar el perfil: allow (por defecto), block o connect",
		"cli.policy.add.on_tunnel_down_conflict":       "Usa solo una opción: --on-tunnel-down o --block-when-down.",
		"cli.policy.disconnect_protection_failed":      "No se pudo activar el bloqueo antes de desconectar. Revisa el ayudante en Diagnóstico y vuelve a intentarlo.",
		"config.err.profile_on_tunnel_down":            "Perfil %[1]q: on_tunnel_down %[2]q no es válido.",
		"config.err.profile_on_tunnel_down.fix":        "Usa allow para permitir, block para bloquear o connect para bloquear e intentar conectar.",
		"help.kongtrol.profile.on-tunnel-down.short":   "Cambia la acción de las reglas cuando el perfil está desconectado",
		"help.kongtrol.profile.on-tunnel-down.long":    "Cambia la acción que heredan las reglas de este perfil. allow permite el tráfico, block bloquea las IP conocidas y connect las bloquea e intenta conectar si detecta tráfico TCP. Los intentos rechazados pueden pasar inadvertidos. En macOS, conecta manualmente.",
		"help.kongtrol.profile.on-tunnel-down.example": "  kongtrol profile on-tunnel-down oficina block",
		"help.kongtrol.policy.on-tunnel-down.short":    "Cambia la acción de una regla cuando su perfil está desconectado",
		"help.kongtrol.policy.on-tunnel-down.long":     "Usa inherit para seguir la opción del perfil, allow para permitir, block para bloquear las IP conocidas o connect para bloquear e intentar conectar si se detecta tráfico TCP. En macOS, conecta manualmente.",
		"help.kongtrol.policy.on-tunnel-down.example":  "  kongtrol policy on-tunnel-down intranet inherit",
		"policy.on_down.inherit":                       "Usar la opción del perfil VPN",
		"cli.on_tunnel_down.saved":                     "%s ahora usa %s cuando el perfil está desconectado.",
	},
	EN: {
		"cli.daemon.down.kept_protection":              "Kongtrol stays running to enforce rule blocking. To stop it, run kongtrol down --all.",
		"cli.policy.add.flag.on_tunnel_down":           "when the profile is disconnected: inherit (default), allow, block, or connect",
		"cli.profile.add.flag.on_tunnel_down":          "rule action when the profile is disconnected: allow (default), block, or connect",
		"cli.policy.add.on_tunnel_down_conflict":       "Use only one option: --on-tunnel-down or --block-when-down.",
		"cli.policy.disconnect_protection_failed":      "Could not enable blocking before disconnecting. Check the helper in Diagnostics and try again.",
		"config.err.profile_on_tunnel_down":            "Profile %[1]q: on_tunnel_down %[2]q is invalid.",
		"config.err.profile_on_tunnel_down.fix":        "Use allow to permit traffic, block to block it, or connect to block it and try to connect.",
		"help.kongtrol.profile.on-tunnel-down.short":   "Change what rules do while the profile is disconnected",
		"help.kongtrol.profile.on-tunnel-down.long":    "Change the action this profile's rules inherit. allow permits traffic, block blocks known IPs, and connect blocks those IPs and tries to connect when TCP traffic is observed. Rejected attempts may be missed. On macOS, connect manually.",
		"help.kongtrol.profile.on-tunnel-down.example": "  kongtrol profile on-tunnel-down office block",
		"help.kongtrol.policy.on-tunnel-down.short":    "Change what a rule does while its profile is disconnected",
		"help.kongtrol.policy.on-tunnel-down.long":     "Use inherit to follow the profile setting, allow to permit traffic, block to block known IPs, or connect to block and try to connect when TCP traffic is observed. On macOS, connect manually.",
		"help.kongtrol.policy.on-tunnel-down.example":  "  kongtrol policy on-tunnel-down intranet inherit",
		"policy.on_down.inherit":                       "Use the VPN profile setting",
		"cli.on_tunnel_down.saved":                     "%s now uses %s while the profile is disconnected.",
	},
}
