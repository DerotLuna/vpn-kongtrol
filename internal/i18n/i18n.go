// Package i18n provides minimal two-language (ES/EN) string lookup for the
// Kongtrol wizard. Add keys to the messages map; call T/F from wizard code.
package i18n

import "fmt"

// Lang is a supported UI language code.
type Lang string

const (
	ES Lang = "es"
	EN Lang = "en"
)

// T returns the translated string for lang and key.
// Falls back to ES when a key is missing in the requested language.
func T(lang Lang, key string) string {
	if m, ok := messages[lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if s, ok := messages[ES][key]; ok {
		return s
	}
	return key // last resort: return the key itself
}

// F returns the translated format string with args applied via fmt.Sprintf.
func F(lang Lang, key string, args ...any) string {
	return fmt.Sprintf(T(lang, key), args...)
}

// IsYes reports whether s is an affirmative answer in the given language.
func IsYes(lang Lang, s string) bool {
	switch lang {
	case ES:
		return s == "s" || s == "si" || s == "sí"
	default:
		return s == "y" || s == "yes"
	}
}

// YesNo returns the [S/n] or [Y/n] hint string appropriate for lang.
func YesNo(lang Lang, defYes bool) string {
	switch lang {
	case ES:
		if defYes {
			return "S/n"
		}
		return "s/N"
	default:
		if defYes {
			return "Y/n"
		}
		return "y/N"
	}
}

var messages = map[Lang]map[string]string{
	// ── Spanish ───────────────────────────────────────────────────────────────
	ES: {
		// Banner
		"banner.title":    "Kongtrol — Asistente de Configuración",
		"banner.yaml":     "Este asistente crea o actualiza ~/.kongtrol/kongtrol.yaml.",
		"banner.keychain": "Las contraseñas se guardan en el llavero del sistema — nunca en el YAML.",

		// VPN detection
		"detected.scanning": "Buscando clientes VPN instalados",
		"detected.header":   "Clientes VPN detectados en este sistema:",
		"detected.none":     "No se detectaron clientes VPN (pueden funcionar si están instalados en otra ruta).",

		// Existing config
		"existing.header": "\nConfiguración existente en %s con %d perfil(es):\n",

		// Section headers
		"section.security":    "Seguridad",
		"section.new_profile": "Nuevo perfil",

		// Profile management
		"profile.header":        "\n── Perfil: %s (%s) ──\n",
		"profile.refresh_creds": "  ¿Actualizar credenciales de este perfil?",
		"profile.add_new":       "¿Agregar un nuevo perfil VPN?",

		// Security prompts
		"security.kill_switch": "¿Activar kill switch? (bloquea todo el tráfico si cae la VPN)",
		"security.dns_guard":   "¿Activar DNS guard? (previene fugas de DNS)",
		"security.audit_log":   "¿Activar log de auditoría firmado?",
		"monitor.dashboard":    "¿Activar panel web? (http://127.0.0.1:9741)",

		// Write / validate
		"write.confirm":          "\n¿Escribir configuración en %s? ",
		"write.aborted":          "Cancelado — no se escribió nada.",
		"write.success":          "[✓] Configuración escrita en %s\n",
		"write.validation_warn":  "\n[!] Advertencia de validación: %v\n",
		"write.validation_hint":  "    Edita el archivo y ejecuta 'kongtrol config validate' cuando esté listo.",
		"write.valid":            "[✓] Configuración válida.",

		// Next steps
		"nextsteps.header":    "\nPróximos pasos:",
		"nextsteps.status":    "  kongtrol status                  — ver estado de los túneles",
		"nextsteps.up":        "  kongtrol up <perfil>             — conectar un perfil",
		"nextsteps.dashboard": "  kongtrol dashboard               — abrir el panel web",

		// Profile collection — shared
		"collect.profile_name":       "  Nombre del perfil (ej. oficina, aws, wg-casa)",
		"collect.profile_name_empty": "el nombre del perfil no puede estar vacío",
		"collect.adapter_line1":      "  Tipos de adaptador: forticlient | openvpn | protonvpn | ciscoanyconnect |",
		"collect.adapter_line2":      "                      wireguard | globalprotect | tailscale | cloudflarewarp",
		"collect.type":               "  Tipo",
		"collect.priority":           "  Prioridad (menor = preferido, 1–100)",
		"collect.unknown_adapter":    "tipo de adaptador desconocido %q",

		// Profile collection — fields
		"collect.host":         "  Host VPN (ej. vpn.empresa.com)",
		"collect.port":         "  Puerto",
		"collect.tunnel_name":  "  Nombre del túnel (como aparece en la UI de FortiClient)",
		"collect.forti_ver":    "  Versión mayor de FortiClient",
		"collect.auth_method":  "  Método de autenticación (certificate | credentials | certificate+credentials)",
		"collect.cert":         "  Ruta al certificado cliente (ej. ~/.kongtrol/certs/oficina.crt)",
		"collect.key":          "  Ruta a la clave privada (ej. ~/.kongtrol/certs/oficina.key)",
		"collect.username":     "  Usuario",
		"collect.ovpn_config":  "  Ruta al archivo .ovpn (ej. ~/.kongtrol/configs/servidor.ovpn)",
		"collect.ovpn_cert":    "  Ruta al certificado cliente (vacío si está dentro del .ovpn)",
		"collect.ovpn_key":     "  Ruta a la clave privada (vacío si está dentro del .ovpn)",
		"collect.proton_srv":   "  Servidor / código de país (ej. US, NL, fastest)",
		"collect.proton_proto": "  Protocolo (wireguard | openvpn)",
		"collect.proton_user":  "  Usuario de ProtonVPN",
		"collect.cisco_host":   "  Host del gateway VPN",
		"collect.cisco_user":   "  Usuario",
		"collect.wg_config":    "  Ruta al archivo .conf de WireGuard (ej. ~/.kongtrol/configs/wg0.conf)",
		"collect.gp_host":      "  Host del gateway GlobalProtect",
		"collect.gp_user":      "  Usuario",
		"collect.ts_exitnode":  "  Hostname del nodo de salida (vacío para enrutamiento Tailscale)",
		"collect.ts_usekey":    "  ¿Usar una auth key? (vacío para reutilizar sesión 'tailscale login' existente)",
		"collect.ts_key":       "  Auth key de Tailscale (vacío para omitir)",
		"collect.warp_info1":   "  [i] WARP no requiere credenciales por perfil.",
		"collect.warp_info2":   "      Ejecuta 'warp-cli register' una vez si aún no lo has hecho.",

		// Credentials
		"collect.password":      "  Contraseña para %s (guardada en llavero del sistema, no en YAML)",
		"collect.password_warn": "  advertencia: no se pudo guardar la credencial: %v",
	},

	// ── English ───────────────────────────────────────────────────────────────
	EN: {
		// Banner
		"banner.title":    "Kongtrol — Setup Wizard",
		"banner.yaml":     "This wizard creates or updates ~/.kongtrol/kongtrol.yaml.",
		"banner.keychain": "Passwords are stored in your OS keychain — never in the YAML file.",

		// VPN detection
		"detected.scanning": "Scanning for installed VPN clients",
		"detected.header":   "Detected VPN clients on this system:",
		"detected.none":     "No VPN clients auto-detected (they may still work if installed elsewhere).",

		// Existing config
		"existing.header": "\nExisting config found at %s with %d profile(s):\n",

		// Section headers
		"section.security":    "Security",
		"section.new_profile": "New profile",

		// Profile management
		"profile.header":        "\n── Profile: %s (%s) ──\n",
		"profile.refresh_creds": "  Refresh / update credentials for this profile?",
		"profile.add_new":       "Add a new VPN profile?",

		// Security prompts
		"security.kill_switch": "Enable kill switch? (blocks all traffic if VPN drops)",
		"security.dns_guard":   "Enable DNS guard? (prevents DNS leaks)",
		"security.audit_log":   "Enable signed audit log?",
		"monitor.dashboard":    "Enable web dashboard? (http://127.0.0.1:9741)",

		// Write / validate
		"write.confirm":         "\nWrite config to %s? ",
		"write.aborted":         "Aborted — nothing written.",
		"write.success":         "[✓] Config written to %s\n",
		"write.validation_warn": "\n[!] Validation warning: %v\n",
		"write.validation_hint": "    Edit the file and run 'kongtrol config validate' when ready.",
		"write.valid":           "[✓] Config is valid.",

		// Next steps
		"nextsteps.header":    "\nNext steps:",
		"nextsteps.status":    "  kongtrol status                  — check tunnel states",
		"nextsteps.up":        "  kongtrol up <profile>            — connect a profile",
		"nextsteps.dashboard": "  kongtrol dashboard               — open the web UI",

		// Profile collection — shared
		"collect.profile_name":       "  Profile name (e.g. office, aws, wg-home)",
		"collect.profile_name_empty": "profile name cannot be empty",
		"collect.adapter_line1":      "  Adapter types: forticlient | openvpn | protonvpn | ciscoanyconnect |",
		"collect.adapter_line2":      "                 wireguard | globalprotect | tailscale | cloudflarewarp",
		"collect.type":               "  Type",
		"collect.priority":           "  Priority (lower = preferred, 1–100)",
		"collect.unknown_adapter":    "unknown adapter type %q",

		// Profile collection — fields
		"collect.host":         "  VPN host (e.g. vpn.empresa.com)",
		"collect.port":         "  Port",
		"collect.tunnel_name":  "  Tunnel name (as shown in FortiClient UI)",
		"collect.forti_ver":    "  FortiClient major version",
		"collect.auth_method":  "  Auth method (certificate | credentials | certificate+credentials)",
		"collect.cert":         "  Client cert path (e.g. ~/.kongtrol/certs/office.crt)",
		"collect.key":          "  Private key path (e.g. ~/.kongtrol/certs/office.key)",
		"collect.username":     "  Username",
		"collect.ovpn_config":  "  .ovpn config path (e.g. ~/.kongtrol/configs/server.ovpn)",
		"collect.ovpn_cert":    "  Client cert path (leave blank if embedded in .ovpn)",
		"collect.ovpn_key":     "  Private key path (leave blank if embedded in .ovpn)",
		"collect.proton_srv":   "  Server / country code (e.g. US, NL, fastest)",
		"collect.proton_proto": "  Protocol (wireguard | openvpn)",
		"collect.proton_user":  "  ProtonVPN username",
		"collect.cisco_host":   "  VPN gateway host",
		"collect.cisco_user":   "  Username",
		"collect.wg_config":    "  WireGuard .conf path (e.g. ~/.kongtrol/configs/wg0.conf)",
		"collect.gp_host":      "  GlobalProtect gateway host",
		"collect.gp_user":      "  Username",
		"collect.ts_exitnode":  "  Exit node hostname (leave blank to use Tailscale mesh routing)",
		"collect.ts_usekey":    "  Use an auth key? (leave blank to reuse existing 'tailscale login' session)",
		"collect.ts_key":       "  Tailscale auth key (leave blank to skip)",
		"collect.warp_info1":   "  [i] WARP uses no per-profile credentials.",
		"collect.warp_info2":   "      Run 'warp-cli register' once if not already registered.",

		// Credentials
		"collect.password":      "  Password for %s (stored in OS keychain, not in YAML)",
		"collect.password_warn": "  warning: could not store credential: %v",
	},
}
