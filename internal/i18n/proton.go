package i18n

// Strings for a direct profile from a Proton account next to Proton's own
// app, and for two direct profiles with the same tunnel address
// (ADR-0103): the import hint, the connect refusals, the handover notice
// when Proton's app connects, and the hint when Proton's kill switch may
// block another VPN. Merged into the catalog at init, so the parity test
// covers them like every other key.

func init() {
	for lang, m := range protonMessages {
		for k, v := range m {
			messages[lang][k] = v
		}
	}
}

var protonMessages = map[Lang]map[string]string{
	ES: {
		"profile.import.proton_hint":               "Este archivo es de tu cuenta de Proton. No lo conectes mientras la app de Proton esté conectada: chocan.",
		"api.err.connect_proton_app":               "Proton VPN ya está conectado en su app y %[1]s usa tu misma cuenta de Proton: los dos a la vez chocan y cortan esas conexiones.",
		"api.err.connect_proton_app.fix":           "Tu tráfico ya sale por Proton. Para usar %[1]s, desconecta Proton en su app y vuelve a conectar %[1]s.",
		"api.err.connect_same_address":             "%[1]s y %[2]s usan la misma dirección interna (%[3]s): a la vez chocan.",
		"api.err.connect_same_address.fix":         "Desconecta %[2]s, o cambia la dirección de uno en su archivo (por ejemplo 10.3.0.2 y DNS 10.3.0.1, como indica Proton).",
		"watchdog.event.proton_handover":           "Conectaste Proton VPN en su app. Desconecté %[1]s para que no choquen; tu tráfico sale por Proton (%[2]s). Para volver a %[1]s, desconecta Proton en su app y pulsa Reintentar.",
		"watchdog.event.proton_handover_add":       "Conectaste Proton VPN en su app. Desconecté %[1]s para que no choquen. Tu tráfico sale por la app de Proton, pero Kongtrol no la ve: agrégala como perfil para verla aquí y usarla en tus reglas.",
		"watchdog.event.proton_blocks_other":       "%[1]s no responde mientras Proton está conectado en su app. Si en Proton tienes el \"kill switch avanzado\", bloquea otras VPN: desactívalo o agrega %[1]s en \"Split tunneling\".",
		"watchdog.event.reconnect_blocked_proton":  "%[1]s no se reconectó: Proton VPN está conectado en su app y %[1]s usa tu misma cuenta de Proton · desconecta Proton en su app y pulsa Reintentar (o: kongtrol up %[1]s)",
		"watchdog.event.reconnect_blocked_address": "%[1]s no se reconectó: %[2]s usa la misma dirección interna (%[3]s) · desconecta %[2]s y pulsa Reintentar (o: kongtrol up %[1]s)",
		"cli.alert.proton_handover_failed":         "No se pudo desconectar %[1]s al conectarse la app de Proton: %[2]v",
	},
	EN: {
		"profile.import.proton_hint":               "This file is from your Proton account. Do not connect it while the Proton app is connected: they clash.",
		"api.err.connect_proton_app":               "Proton VPN is already connected in its app and %[1]s uses the same Proton account: both at once clash and cut those connections.",
		"api.err.connect_proton_app.fix":           "Your traffic already goes through Proton. To use %[1]s, disconnect Proton in its app, then connect %[1]s again.",
		"api.err.connect_same_address":             "%[1]s and %[2]s use the same internal address (%[3]s): both at once clash.",
		"api.err.connect_same_address.fix":         "Disconnect %[2]s, or change the address of one of them in its file (for example 10.3.0.2 and DNS 10.3.0.1, as Proton suggests).",
		"watchdog.event.proton_handover":           "You connected Proton VPN in its app. I disconnected %[1]s so they do not clash; your traffic goes through Proton (%[2]s). To go back to %[1]s, disconnect Proton in its app and press Retry.",
		"watchdog.event.proton_handover_add":       "You connected Proton VPN in its app. I disconnected %[1]s so they do not clash. Your traffic goes through the Proton app, but Kongtrol cannot see it: add it as a profile to see it here and use it in your rules.",
		"watchdog.event.proton_blocks_other":       "%[1]s is not responding while Proton is connected in its app. If Proton's \"advanced kill switch\" is on, it blocks other VPNs: turn it off or add %[1]s under \"Split tunneling\".",
		"watchdog.event.reconnect_blocked_proton":  "%[1]s did not reconnect: Proton VPN is connected in its app and %[1]s uses the same Proton account · disconnect Proton in its app and press Retry (or: kongtrol up %[1]s)",
		"watchdog.event.reconnect_blocked_address": "%[1]s did not reconnect: %[2]s uses the same internal address (%[3]s) · disconnect %[2]s and press Retry (or: kongtrol up %[1]s)",
		"cli.alert.proton_handover_failed":         "Could not disconnect %[1]s when the Proton app connected: %[2]v",
	},
}
