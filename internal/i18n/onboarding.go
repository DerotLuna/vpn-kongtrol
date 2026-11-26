package i18n

// Strings for the guided restore at start (ADR-0101): the question asked
// once after the first profile the user connects, and what the daemon says
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
		"onboarding.restore.title":           "¿Volver a conectar tus perfiles VPN al encender el equipo?",
		"onboarding.restore.body":            "Kongtrol se abrirá al encender el equipo y volverá a conectar los perfiles VPN que dejaste conectados. Los perfiles que desconectes tú no vuelven a conectarse. Puedes cambiarlo en Ajustes › Al iniciar.",
		"onboarding.restore.yes":             "Sí, activar",
		"onboarding.restore.no":              "Ahora no",
		"cli.onboarding.restore.hint":        "Para que Kongtrol vuelva a conectar tus perfiles VPN al encender el equipo: kongtrol config restore-tunnels on y kongtrol config autostart on",
		"cli.onboarding.restore.on":          "Listo: al encender el equipo Kongtrol se abrirá y volverá a conectar los perfiles VPN que dejes conectados",
		"cli.onboarding.restore.off":         "Entendido: Kongtrol no volverá a conectar tus perfiles VPN al iniciar · puedes cambiarlo en Ajustes › Al iniciar o con kongtrol config restore-tunnels on",
		"cli.daemon.restore_waiting_network": "Esperando conexión a internet para volver a conectar %s…",
		"cli.daemon.restore_no_network":      "Sigue sin haber conexión a internet · Kongtrol volverá a conectar %s cuando cambie tu red",
		"cli.daemon.restore_app_skipped":     "%s no se volvió a conectar: al iniciar, Kongtrol no abre la app asociada a este perfil VPN · conéctala en la app y Kongtrol la detecta",
	},
	EN: {
		"onboarding.restore.title":           "Reconnect your VPN profiles when the computer starts?",
		"onboarding.restore.body":            "Kongtrol will open when the computer starts and reconnect the VPN profiles you left connected. Profiles you disconnect yourself stay disconnected. You can change this in Settings › When starting.",
		"onboarding.restore.yes":             "Yes, turn on",
		"onboarding.restore.no":              "Not now",
		"cli.onboarding.restore.hint":        "To have Kongtrol reconnect your VPN profiles when the computer starts: kongtrol config restore-tunnels on and kongtrol config autostart on",
		"cli.onboarding.restore.on":          "Done: when the computer starts, Kongtrol will open and reconnect the VPN profiles you leave connected",
		"cli.onboarding.restore.off":         "Got it: Kongtrol will not reconnect your VPN profiles at startup · change it in Settings › When starting or with kongtrol config restore-tunnels on",
		"cli.daemon.restore_waiting_network": "Waiting for an internet connection to reconnect %s…",
		"cli.daemon.restore_no_network":      "Still no internet connection · Kongtrol will reconnect %s when your network changes",
		"cli.daemon.restore_app_skipped":     "%s was not reconnected: when starting, Kongtrol does not open the app for this VPN profile · connect it in the app and Kongtrol picks it up",
	},
}
