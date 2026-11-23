package i18n

// Strings for the actions a daemon notice offers (monitor.Event.Offer,
// ADR-0102): the one-time kill switch offer after a VPN drops on its own,
// "Recuperar internet" on every kill switch block, and "Agregar <App>".
// The tray's native menu and the CLI live view use them; the panel has its
// own copy (web/ui src/i18n/pages/offers.ts). Merged into the catalog at
// init, so the parity test covers them like every other key.

func init() {
	for lang, m := range killSwitchOfferMessages {
		for k, v := range m {
			messages[lang][k] = v
		}
	}
}

// OfferAppName is the short brand name of a provider adapter type in the
// "Agregar <App>" button ("Agregar Proton"); an unknown type is returned
// as it is.
func OfferAppName(adapterType string) string {
	switch adapterType {
	case "protonvpn":
		return "Proton"
	case "nordvpn":
		return "NordVPN"
	case "surfshark":
		return "Surfshark"
	case "forticlient":
		return "FortiClient"
	}
	return adapterType
}

var killSwitchOfferMessages = map[Lang]map[string]string{
	ES: {
		"killswitch.event.offer":         "%[1]s se cayó sin que lo pidieras · mientras volvía a conectar, tu tráfico salió sin VPN. ¿Quieres que Kongtrol lo impida? Si %[1]s se cae, Kongtrol bloquea su tráfico hasta que vuelva. Si no vuelve, te aviso y con un clic recuperas internet.",
		"offer.action.enable_killswitch": "Activar protección",
		"offer.action.release_block":     "Recuperar internet",
		"offer.action.add_provider":      "Agregar %s",
		"offer.cli.enable_killswitch":    "para activarlo: kongtrol config killswitch on",
		"offer.cli.release_block":        "para recuperar internet: kongtrol down %s",
		"offer.cli.add_provider":         "para agregarla: kongtrol profile add %s",
		"offer.err.enable_killswitch":    "No se pudo activar la protección · actívala en el panel, en Seguridad, o con: kongtrol config killswitch on",
	},
	EN: {
		"killswitch.event.offer":         "%[1]s dropped without you asking · while it was reconnecting, your traffic went out without the VPN. Do you want Kongtrol to prevent that? If %[1]s drops, Kongtrol blocks its traffic until it is back. If it does not come back, you get a notice and one click brings your internet back.",
		"offer.action.enable_killswitch": "Turn on protection",
		"offer.action.release_block":     "Get internet back",
		"offer.action.add_provider":      "Add %s",
		"offer.cli.enable_killswitch":    "to turn it on: kongtrol config killswitch on",
		"offer.cli.release_block":        "to get internet back: kongtrol down %s",
		"offer.cli.add_provider":         "to add it: kongtrol profile add %s",
		"offer.err.enable_killswitch":    "Could not turn on protection · turn it on in the panel, under Security, or with: kongtrol config killswitch on",
	},
}
