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
		"banner.subtitle": "Orquestación Multi-VPN",
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
		"section.policies":    "Políticas de enrutamiento",

		// Profile management
		"profile.header":          "\n── Perfil: %s (%s) ──\n",
		"profile.refresh_any":     "¿Actualizar credenciales de algún perfil existente?",
		"profile.refresh_creds":   "  ¿Actualizar credenciales de este perfil?",
		"profile.add_new":         "¿Agregar un nuevo perfil VPN?",
		"profile.already_exists":  "  El perfil '%s' ya existe.",
		"profile.replace_confirm": "  ¿Reemplazarlo con la nueva configuración?",
		"profile.replace_skipped": "  Perfil no modificado — elige un nombre diferente si quieres agregar uno nuevo.",

		// Routing policies
		"policy.existing":      "Políticas existentes:",
		"policy.no_profiles":   "No hay perfiles configurados — agrega al menos uno antes de crear políticas.",
		"policy.add_new":       "¿Agregar una política de enrutamiento?",
		"policy.name":          "  Nombre de la política (ej: trabajo, streaming)",
		"policy.via":           "  Perfil VPN para esta política",
		"policy.domains_hint":  "    Dominio o sufijo (ej: empresa.com, .internal) — deja en blanco para terminar",
		"policy.domain_prompt": "  Dominio",
		"policy.ips_hint":      "    IP o rango — ejemplos:\n      172.28.152.26       → una sola IP\n      10.0.0.0/24         → 256 IPs (10.0.0.0 a 10.0.0.255)\n      10.0.0.0/16         → 65536 IPs (10.0.x.x)\n      Deja en blanco para terminar",
		"policy.ip_prompt":     "  IP / rango",
		"policy.empty_match":     "  [!] La política necesita al menos un dominio o rango IP. Descartada.",
		"policy.already_exists":  "  La política '%s' ya existe.",
		"policy.replace_confirm": "  ¿Reemplazarla con la nueva configuración?",
		"policy.replace_skipped": "  Política no modificada — elige un nombre diferente para agregar una nueva.",
		"policy.yaml_hint":       "    También puedes editar políticas directamente en kongtrol.yaml → sección 'policies'.",

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
		"nextsteps.init":      "  kongtrol init                    — agregar más perfiles en cualquier momento",
		"nextsteps.status":    "  kongtrol status                  — ver estado de los túneles",
		"nextsteps.up":        "  kongtrol up <perfil>             — conectar un perfil",
		"nextsteps.dashboard": "  kongtrol dashboard               — abrir el panel web",

		// Profile collection — shared
		"collect.profile_name":       "  Nombre del perfil (ej. oficina, aws, wg-casa)",
		"collect.profile_name_empty": "el nombre del perfil no puede estar vacío",
		"collect.type":               "  Tipo de adaptador",
		"collect.priority":           "  Prioridad (menor = preferido, 1–100)",
		"collect.unknown_adapter":    "tipo de adaptador desconocido %q",

		// Adapter type select hints
		"adapter.hint.forticlient":    "FortiClient SSL VPN",
		"adapter.hint.openvpn":        "OpenVPN (archivo .ovpn)",
		"adapter.hint.protonvpn":      "ProtonVPN (cuenta Proton)",
		"adapter.hint.ciscoanyconnect":"Cisco AnyConnect",
		"adapter.hint.wireguard":      "WireGuard (archivo .conf)",
		"adapter.hint.globalprotect":  "Palo Alto GlobalProtect",
		"adapter.hint.tailscale":      "Tailscale mesh / exit node",
		"adapter.hint.cloudflarewarp": "Cloudflare WARP",

		// Select UI
		"select.choose":   "  Elige",
		"select.detected": "detectado",

		// Auth method select
		"collect.auth_method":       "  Método de autenticación",
		"auth.hint.credentials":     "usuario y contraseña",
		"auth.hint.certificate":     "solo certificado cliente",
		"auth.hint.cert+creds":      "certificado + usuario y contraseña (más seguro)",

		// Protocol select
		"collect.proton_proto":      "  Protocolo",
		"proto.hint.wireguard":      "más rápido, recomendado",
		"proto.hint.openvpn":        "más compatible",

		// FortiClient version select
		"collect.forti_ver":         "  Versión de FortiClient",
		"forti.ver.hint.6":          "6.4.x — más común",
		"forti.ver.hint.7":          "7.x",
		"forti.ver.hint.5":          "5.x — legacy",

		// Profile collection — field hints (shown dim before the prompt)
		"hint.host.forti":      "    Encuéntralo en FortiClient > Ajustes o pídelo a IT. Ej: vpn.empresa.com",
		"hint.port":            "    443 para SSL VPN (default). Cambia solo si IT indica otro puerto.",
		"hint.tunnel_name":     "    El nombre exacto de la conexión tal como aparece en la lista de FortiClient GUI.",
		"hint.host.cisco":      "    Hostname del gateway — en la config actual de AnyConnect o lo da IT.",
		"hint.host.gp":         "    Hostname del portal GlobalProtect — lo da IT o aparece en la app.",
		"hint.ovpn_config":     "    Ruta completa al .ovpn. Usa la ubicación donde ya está, no hace falta copiar.",
		"hint.auth.forti.win":  "    En Windows FortiClient conecta por nombre de túnel — sin certificado. Se usará 'credentials'.",
		"hint.auth.openvpn":    "    Si el .ovpn tiene <cert> y <key> embebidas, elige 'certificate' y deja los paths en blanco.",
		"hint.proton_srv":      "    Código ISO (US, NL, DE...) o 'fastest' para el servidor más rápido disponible.",
		"hint.wg_config":       "    Ruta al archivo .conf. Para ProtonVPN: descárgalo en account.protonvpn.com → Downloads → WireGuard configuration.",

		// Profile collection — fields
		"collect.host":        "  Host VPN",
		"collect.port":        "  Puerto",
		"collect.tunnel_name": "  Nombre del túnel",
		"collect.cert":        "  Ruta al certificado cliente",
		"collect.key":         "  Ruta a la clave privada",
		"collect.username":    "  Usuario",
		"collect.ovpn_config": "  Ruta al archivo .ovpn",
		"collect.ovpn_cert":   "  Ruta al certificado cliente (vacío si está dentro del .ovpn)",
		"collect.ovpn_key":    "  Ruta a la clave privada (vacío si está dentro del .ovpn)",
		"collect.proton_srv":  "  Servidor / código de país",
		"collect.proton_user": "  Usuario de ProtonVPN",
		"collect.cisco_host":  "  Host del gateway VPN",
		"collect.cisco_user":  "  Usuario",
		"collect.wg_config":   "  Ruta al archivo .conf de WireGuard",
		"collect.gp_host":     "  Host del gateway GlobalProtect",
		"collect.gp_user":     "  Usuario",
		"collect.ts_exitnode": "  Hostname del nodo de salida (vacío para enrutamiento Tailscale)",
		"collect.ts_usekey":   "  ¿Usar una auth key? (vacío para reutilizar sesión 'tailscale login' existente)",
		"collect.ts_key":      "  Auth key de Tailscale (vacío para omitir)",
		"collect.warp_info1":  "  [i] WARP no requiere credenciales por perfil.",
		"collect.warp_info2":  "      Ejecuta 'warp-cli register' una vez si aún no lo has hecho.",

		// Binary path
		"collect.not_detected":  "  [!] Este cliente VPN no fue detectado automáticamente en tu sistema.",
		"collect.binary_path":   "  Ruta al binario (vacío = autodetectar, ej. C:\\Program Files\\...\\vpn.exe)",

		// Credentials
		"collect.password":      "  Contraseña para %s (guardada en llavero del sistema, no en YAML)",
		"collect.password_warn": "  advertencia: no se pudo guardar la credencial: %v",
	},

	// ── English ───────────────────────────────────────────────────────────────
	EN: {
		// Banner
		"banner.title":    "Kongtrol — Setup Wizard",
		"banner.subtitle": "Multi-VPN Orchestration",
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
		"section.policies":    "Routing policies",

		// Profile management
		"profile.header":          "\n── Profile: %s (%s) ──\n",
		"profile.refresh_any":     "Update credentials for any existing profile?",
		"profile.refresh_creds":   "  Refresh / update credentials for this profile?",
		"profile.add_new":         "Add a new VPN profile?",
		"profile.already_exists":  "  Profile '%s' already exists.",
		"profile.replace_confirm": "  Replace it with the new configuration?",
		"profile.replace_skipped": "  Profile unchanged — choose a different name to add a new one.",

		// Routing policies
		"policy.existing":      "Existing policies:",
		"policy.no_profiles":   "No profiles configured — add at least one before creating policies.",
		"policy.add_new":       "Add a routing policy?",
		"policy.name":          "  Policy name (e.g. work, streaming)",
		"policy.via":           "  VPN profile for this policy",
		"policy.domains_hint":  "    Domain or suffix (e.g. company.com, .internal) — leave blank to finish",
		"policy.domain_prompt": "  Domain",
		"policy.ips_hint":      "    IP or range — examples:\n      172.28.152.26       → single IP\n      10.0.0.0/24         → 256 IPs (10.0.0.0 to 10.0.0.255)\n      10.0.0.0/16         → 65536 IPs (10.0.x.x)\n      Leave blank to finish",
		"policy.ip_prompt":     "  IP / range",
		"policy.empty_match":     "  [!] Policy needs at least one domain or IP range. Discarded.",
		"policy.already_exists":  "  Policy '%s' already exists.",
		"policy.replace_confirm": "  Replace it with the new configuration?",
		"policy.replace_skipped": "  Policy unchanged — choose a different name to add a new one.",
		"policy.yaml_hint":       "    You can also edit policies directly in kongtrol.yaml → 'policies' section.",

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
		"nextsteps.init":      "  kongtrol init                    — add more profiles at any time",
		"nextsteps.status":    "  kongtrol status                  — check tunnel states",
		"nextsteps.up":        "  kongtrol up <profile>            — connect a profile",
		"nextsteps.dashboard": "  kongtrol dashboard               — open the web UI",

		// Profile collection — shared
		"collect.profile_name":       "  Profile name (e.g. office, aws, wg-home)",
		"collect.profile_name_empty": "profile name cannot be empty",
		"collect.type":               "  Adapter type",
		"collect.priority":           "  Priority (lower = preferred, 1–100)",
		"collect.unknown_adapter":    "unknown adapter type %q",

		// Adapter type select hints
		"adapter.hint.forticlient":    "FortiClient SSL VPN",
		"adapter.hint.openvpn":        "OpenVPN (.ovpn file)",
		"adapter.hint.protonvpn":      "ProtonVPN (Proton account)",
		"adapter.hint.ciscoanyconnect":"Cisco AnyConnect",
		"adapter.hint.wireguard":      "WireGuard (.conf file)",
		"adapter.hint.globalprotect":  "Palo Alto GlobalProtect",
		"adapter.hint.tailscale":      "Tailscale mesh / exit node",
		"adapter.hint.cloudflarewarp": "Cloudflare WARP",

		// Select UI
		"select.choose":   "  Choose",
		"select.detected": "detected",

		// Auth method select
		"collect.auth_method":   "  Auth method",
		"auth.hint.credentials": "username and password",
		"auth.hint.certificate": "client certificate only",
		"auth.hint.cert+creds":  "certificate + username and password (more secure)",

		// Protocol select
		"collect.proton_proto": "  Protocol",
		"proto.hint.wireguard": "faster, recommended",
		"proto.hint.openvpn":   "more compatible",

		// FortiClient version select
		"collect.forti_ver":   "  FortiClient version",
		"forti.ver.hint.6":    "6.4.x — most common",
		"forti.ver.hint.7":    "7.x",
		"forti.ver.hint.5":    "5.x — legacy",

		// Profile collection — field hints (shown dim before the prompt)
		"hint.host.forti":     "    Found in FortiClient Settings or ask IT. E.g. vpn.company.com",
		"hint.port":           "    443 for SSL VPN (default). Change only if IT specifies a different port.",
		"hint.tunnel_name":    "    The exact connection name as it appears in the FortiClient GUI list.",
		"hint.host.cisco":     "    Gateway hostname — in your current AnyConnect config or from IT.",
		"hint.host.gp":        "    GlobalProtect portal hostname — from IT or shown in the app.",
		"hint.ovpn_config":    "    Full path to the .ovpn. Point to where it already lives — no need to copy.",
		"hint.auth.forti.win": "    On Windows FortiClient connects by tunnel name — no certificate needed. Using 'credentials'.",
		"hint.auth.openvpn":   "    If the .ovpn has embedded <cert> and <key>, choose 'certificate' and leave paths blank.",
		"hint.proton_srv":     "    ISO country code (US, NL, DE...) or 'fastest' for the fastest available server.",
		"hint.wg_config":      "    Path to the .conf file. For ProtonVPN: download it at account.protonvpn.com → Downloads → WireGuard configuration.",

		// Profile collection — fields
		"collect.host":        "  VPN host",
		"collect.port":        "  Port",
		"collect.tunnel_name": "  Tunnel name",
		"collect.cert":        "  Client cert path",
		"collect.key":         "  Private key path",
		"collect.username":    "  Username",
		"collect.ovpn_config": "  .ovpn file path",
		"collect.ovpn_cert":   "  Client cert path (leave blank if embedded in .ovpn)",
		"collect.ovpn_key":    "  Private key path (leave blank if embedded in .ovpn)",
		"collect.proton_srv":  "  Server / country code",
		"collect.proton_user": "  ProtonVPN username",
		"collect.cisco_host":  "  VPN gateway host",
		"collect.cisco_user":  "  Username",
		"collect.wg_config":   "  WireGuard .conf path",
		"collect.gp_host":     "  GlobalProtect gateway host",
		"collect.gp_user":     "  Username",
		"collect.ts_exitnode": "  Exit node hostname (leave blank to use Tailscale mesh routing)",
		"collect.ts_usekey":   "  Use an auth key? (leave blank to reuse existing 'tailscale login' session)",
		"collect.ts_key":      "  Tailscale auth key (leave blank to skip)",
		"collect.warp_info1":  "  [i] WARP uses no per-profile credentials.",
		"collect.warp_info2":  "      Run 'warp-cli register' once if not already registered.",

		// Binary path
		"collect.not_detected":  "  [!] This VPN client was not auto-detected on your system.",
		"collect.binary_path":   "  Binary path (leave blank to auto-detect, e.g. /usr/bin/vpn)",

		// Credentials
		"collect.password":      "  Password for %s (stored in OS keychain, not in YAML)",
		"collect.password_warn": "  warning: could not store credential: %v",
	},
}
