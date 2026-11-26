package i18n

func init() {
	for lang, entries := range policyStateMessages {
		for key, message := range entries {
			messages[lang][key] = message
		}
	}
}

var policyStateMessages = map[Lang]map[string]string{
	ES: {
		"cli.policy.add.flag.paused":         "guarda la regla pausada hasta que la actives",
		"cli.policy.add.paused_next":         "Quedó pausada. Actívala en Reglas o con kongtrol policy state %q on.",
		"cli.policy.state.on":                "%s está activa.",
		"cli.policy.state.off":               "%s está pausada.",
		"cli.policy.state.usage":             "Usa on para activar la regla u off para pausarla.",
		"help.kongtrol.policy.state.short":   "Consulta, activa o pausa una regla",
		"help.kongtrol.policy.state.long":    "Una regla pausada se conserva y no enruta, bloquea ni intenta conectar su perfil VPN. Activarla aplica su comportamiento de inmediato.",
		"help.kongtrol.policy.state.example": "  kongtrol policy state intranet on",
	},
	EN: {
		"cli.policy.add.flag.paused":         "save the rule paused until you activate it",
		"cli.policy.add.paused_next":         "Saved paused. Activate it in Rules or with kongtrol policy state %q on.",
		"cli.policy.state.on":                "%s is active.",
		"cli.policy.state.off":               "%s is paused.",
		"cli.policy.state.usage":             "Use on to activate the rule or off to pause it.",
		"help.kongtrol.policy.state.short":   "Show, activate, or pause a rule",
		"help.kongtrol.policy.state.long":    "A paused rule remains saved without routing, blocking, or attempting to connect its VPN profile. Activating it applies its behavior immediately.",
		"help.kongtrol.policy.state.example": "  kongtrol policy state intranet on",
	},
}
