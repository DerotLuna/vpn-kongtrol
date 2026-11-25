package i18n

// Program (app:) rules labelled honestly (ADR-0113): they are saved and
// recognised (policy show/test, the panel's checks); only a Linux WireGuard
// profile with app_route_mode routes them (ADR-0111). `kongtrol policy add|list|show|test`
// say so; the panel's strings live in web/ui/src/i18n/pages.
func init() {
	for lang, entries := range map[Lang]map[string]string{
		ES: {
			"cli.policy.apps_not_applied": "Programas: solo salen por la VPN en Linux, con un perfil WireGuard que tenga «Salida por ejecutable». En los demás casos Kongtrol solo los reconoce: usa sitios o IPs.",
			"cli.policy.app_tag":          "aún no se aplica",
		},
		EN: {
			"cli.policy.apps_not_applied": "Programs: they only go through the VPN on Linux, with a WireGuard profile that has Executable egress on. Otherwise Kongtrol only recognises them: use sites or IPs.",
			"cli.policy.app_tag":          "not applied yet",
		},
	} {
		for key, message := range entries {
			messages[lang][key] = message
		}
	}
}
