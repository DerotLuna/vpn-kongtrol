package i18n

// Moving a direct profile's rules to the provider-app profile of the same
// account (ADR-0104): `kongtrol policy move`, the offer after
// `kongtrol profile add <provider>`, and the API refusal.

func init() {
	for lang, entries := range ruleMoveMessages {
		for key, message := range entries {
			messages[lang][key] = message
		}
	}
}

var ruleMoveMessages = map[Lang]map[string]string{
	ES: {
		"api.err.policy_move_same":          "Las reglas de %[1]q ya usan %[1]q.",
		"api.err.policy_move_same.fix":      "Elige otro perfil de destino.",
		"cli.policy.move.done.one":          "Listo: 1 regla ahora usa %[2]s.",
		"cli.policy.move.done.other":        "Listo: %[1]d reglas ahora usan %[2]s.",
		"cli.policy.move.none":              "%[1]s no tiene reglas: no había nada que pasar.",
		"cli.policy.move.next":              "Revisa tus reglas con kongtrol policy list.",
		"cli.policy.move_offer.ask":         "¿Usar %[2]s en tus reglas de %[1]s?",
		"cli.policy.move_offer.body.one":    "1 regla usa %[2]s, tu archivo de %[4]s. Si la pasas a %[3]s, esos sitios saldrán por la app de %[4]s. %[2]s queda guardado por si lo vuelves a usar.",
		"cli.policy.move_offer.body.other":  "%[1]d reglas usan %[2]s, tu archivo de %[4]s. Si las pasas a %[3]s, esos sitios saldrán por la app de %[4]s. %[2]s queda guardado por si lo vuelves a usar.",
		"cli.policy.move_offer.hint":        "Para pasarlas más tarde: kongtrol policy move %[1]q %[2]q",
		"help.kongtrol.policy.move.args":    "<desde> <hacia>",
		"help.kongtrol.policy.move.short":   "Pasa las reglas de un perfil a otro",
		"help.kongtrol.policy.move.long":    "Hace que todas las reglas que usan <desde> usen <hacia>, en un solo cambio. Sirve, por ejemplo, para pasar a la app de Proton las reglas de un perfil hecho con un archivo de Proton. El perfil <desde> queda guardado.",
		"help.kongtrol.policy.move.example": "  kongtrol policy move proton-ch \"Proton VPN\"",
	},
	EN: {
		"api.err.policy_move_same":          "the rules of %[1]q already use %[1]q",
		"api.err.policy_move_same.fix":      "Pick another profile to move them to.",
		"cli.policy.move.done.one":          "Done: 1 rule now uses %[2]s.",
		"cli.policy.move.done.other":        "Done: %[1]d rules now use %[2]s.",
		"cli.policy.move.none":              "%[1]s has no rules: there was nothing to move.",
		"cli.policy.move.next":              "Check your rules with kongtrol policy list.",
		"cli.policy.move_offer.ask":         "Use %[2]s for your %[1]s rules?",
		"cli.policy.move_offer.body.one":    "1 rule uses %[2]s, your %[4]s file. If you move it to %[3]s, those sites will go through the %[4]s app. %[2]s stays saved in case you use it again.",
		"cli.policy.move_offer.body.other":  "%[1]d rules use %[2]s, your %[4]s file. If you move them to %[3]s, those sites will go through the %[4]s app. %[2]s stays saved in case you use it again.",
		"cli.policy.move_offer.hint":        "To move them later: kongtrol policy move %[1]q %[2]q",
		"help.kongtrol.policy.move.args":    "<from> <to>",
		"help.kongtrol.policy.move.short":   "Move a profile's rules to another profile",
		"help.kongtrol.policy.move.long":    "Makes every rule that uses <from> use <to>, in one change. For example, to move to the Proton app the rules of a profile made from a Proton file. The <from> profile stays saved.",
		"help.kongtrol.policy.move.example": "  kongtrol policy move proton-ch \"Proton VPN\"",
	},
}
