package i18n

// Strings for the guided restore at start (ADR-0101): the question asked
// once after the first VPN the user connects, and what the daemon says
// while it waits for the network before restoring. Merged into the
// catalog at init, so the parity test covers them like every other key.

func init() {
	for lang, m := range onboardingMessages {
		for k, v := range m {
			messages[lang][k] = v
		}
	}
}

var onboardingMessages = map[Lang]map[string]string{
	ES: {
		"onboarding.restore.title":           "¿Volver a conectar tus VPN al encender el equipo?",
		"onboarding.restore.body":            "Kongtrol se abrirá al encender el equipo y volverá a conectar las VPN que dejaste conectadas. Las que desconectes tú no vuelven. Puedes cambiarlo en Ajustes › Al iniciar.",
		"onboarding.restore.yes":             "Sí, activar",
		"onboarding.restore.no":              "Ahora no",
		"cli.onboarding.restore.hint":        "Para que Kongtrol vuelva a conectar tus VPN al encender el equipo: kongtrol config restore-tunnels on y kongtrol config autostart on",
		"cli.onboarding.restore.on":          "Listo: al encender el equipo Kongtrol se abrirá y volverá a conectar las VPN que dejes conectadas",
		"cli.onboarding.restore.off":         "Entendido: Kongtrol no volverá a conectar tus VPN al iniciar · puedes cambiarlo en Ajustes › Al iniciar o con kongtrol config restore-tunnels on",
		"cli.daemon.restore_waiting_network": "Esperando conexión a internet para volver a conectar %s…",
		"cli.daemon.restore_no_network":      "Sigue sin haber conexión a internet · Kongtrol volverá a conectar %s cuando cambie tu red",
		"cli.daemon.restore_app_skipped":     "%s no se volvió a conectar: al encender el equipo Kongtrol no abre la app de esa VPN · conéctala en su app y Kongtrol la detecta",
	},
	EN: {
		"onboarding.restore.title":           "Reconnect your VPNs when the computer starts?",
		"onboarding.restore.body":            "Kongtrol will open when the computer starts and reconnect the VPNs you left connected. The ones you disconnect yourself stay off. You can change this in Settings › When starting.",
		"onboarding.restore.yes":             "Yes, turn on",
		"onboarding.restore.no":              "Not now",
		"cli.onboarding.restore.hint":        "To have Kongtrol reconnect your VPNs when the computer starts: kongtrol config restore-tunnels on and kongtrol config autostart on",
		"cli.onboarding.restore.on":          "Done: when the computer starts, Kongtrol will open and reconnect the VPNs you leave connected",
		"cli.onboarding.restore.off":         "Got it: Kongtrol will not reconnect your VPNs at startup · change it in Settings › When starting or with kongtrol config restore-tunnels on",
		"cli.daemon.restore_waiting_network": "Waiting for an internet connection to reconnect %s…",
		"cli.daemon.restore_no_network":      "Still no internet connection · Kongtrol will reconnect %s when your network changes",
		"cli.daemon.restore_app_skipped":     "%s was not reconnected: when the computer starts, Kongtrol does not open that VPN's app · connect it in its app and Kongtrol picks it up",
	},
}
