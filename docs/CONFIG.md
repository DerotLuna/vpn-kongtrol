<!-- Generado por `go generate ./cmd/kongtrol` (make docs) desde internal/config/schema.go e internal/docgen/text.go. NO EDITAR A MANO. -->

<img src="../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Referencia de configuración (kongtrol.yaml)

Versión 1.13 · 2026-11-25

Todas las claves de `~/.kongtrol/kongtrol.yaml`, leídas del esquema (`internal/config/schema.go`). Los valores por defecto son los que aplica el código. Guía con ejemplos: [CONFIGURATION.md](CONFIGURATION.md) · ejemplo completo: [configs/example.yaml](../configs/example.yaml) · English: [CONFIG.en.md](CONFIG.en.md).

## `vpns`

Tus perfiles VPN, uno por nombre (`vpns.oficina`, `vpns.casa`, …). Al menos uno.

| Clave | Tipo | Por defecto | Descripción |
|---|---|---|---|
| `vpns.<name>.type` | texto | — | **Obligatorio.** Tipo de VPN (adaptador). Valores: `forticlient`, `openvpn`, `protonvpn`, `nordvpn`, `surfshark`, `ciscoanyconnect`, `wireguard`, `globalprotect`, `tailscale`, `cloudflarewarp`, `mullvad`, `pia`, `expressvpn`, `windscribe`, `cyberghost`. |
| `vpns.<name>.version` | texto | — | Versión del cliente, solo informativa. |
| `vpns.<name>.host` | texto | — | Servidor o gateway VPN (clientes corporativos). |
| `vpns.<name>.port` | entero | — | Puerto del servidor. |
| `vpns.<name>.tunnel_name` | texto | — | Nombre de la conexión guardada en el cliente (p. ej. FortiClient). |
| `vpns.<name>.default_egress` | booleano | — | Opción para un solo perfil WireGuard directo en Windows o Linux. En la próxima conexión, el perfil lleva el tráfico IPv4 e IPv6 sin una ruta más específica. |
| `vpns.<name>.app_route_mode` | texto | — | Solo WireGuard directo en Linux. prefer_destinations conserva las rutas de destino; all prioriza este perfil para los ejecutables seleccionados. Aplica en la próxima conexión. |
| `vpns.<name>.allow_insecure_cert` | booleano | — | Solo FortiClient: acepta advertencias de certificado al automatizar la app. Déjalo en `false` (riesgo de MITM). |
| `vpns.<name>.config` | texto | — | Ruta al `.ovpn` (openvpn) o `.conf` (wireguard). |
| `vpns.<name>.server` | texto | — | Servidor exacto (Proton VPN, NordVPN, Mullvad) o nombre/ID de región (PIA, ExpressVPN, Windscribe). |
| `vpns.<name>.protocol` | texto | — | Protocolo del proveedor: `wireguard` u `openvpn` (protonvpn). |
| `vpns.<name>.binary_path` | texto | — | Ruta al programa del cliente VPN si no se detecta solo. |
| `vpns.<name>.auth.method` | texto | — | **Obligatorio.** Cómo se autentica: `certificate`, `credentials` o `certificate+credentials`. Para protonvpn, nordvpn y surfshark vale `credentials` si lo omites. |
| `vpns.<name>.auth.cert` | texto | — | Ruta al certificado de cliente. |
| `vpns.<name>.auth.key` | texto | — | Ruta a la clave privada. |
| `vpns.<name>.auth.username` | texto | — | Usuario en texto plano (poco sensible). |
| `vpns.<name>.auth.username_keychain` | texto | — | Nombre de la entrada del llavero que guarda el usuario. |
| `vpns.<name>.auth.password_keychain` | texto | — | Nombre de la entrada del llavero que guarda la contraseña. |
| `vpns.<name>.priority` | entero | — | Desempate entre políticas que se solapan: las políticas de un perfil con número mayor se evalúan antes (dominios y apps: gana la primera que coincide; IP: siempre gana el rango más específico y la prioridad solo desempata rangos del mismo tamaño). Sin valor (o 0), cuenta la posición de la política (las primeras van antes). De 0 a 1000. Cámbiala con kongtrol profile priority o en el panel (Perfiles › Orden de sus reglas). No conecta ni cambia perfiles. |
| `vpns.<name>.kill_switch` | booleano | — | Activa o desactiva el kill switch solo para este perfil (si falta, usa `security.kill_switch.enabled`). |
| `vpns.<name>.on_tunnel_down` | texto | — | Acción que heredan las reglas de este perfil mientras su VPN está apagada: `allow` (por defecto), `block` o `connect`. |
| `vpns.<name>.failover` | lista de texto | — | Perfiles de respaldo, en orden, que Kongtrol puede conectar si este se cae y no logra reconectarse. Si falta, nunca conecta otro perfil: avisa y sigue reintentando este. Cada uno debe existir, no puede ser el mismo perfil y no puede formar un círculo. |
| `vpns.<name>.auto_connect` | booleano | — | Proveedores con app oficial: intenta conectar solo (CLI oficial o automatización) antes de pedirte que conectes en la app. Si falta: `true`. |
| `vpns.<name>.country` | texto | — | País del servidor (código o nombre), para proveedores que permiten elegir. |
| `vpns.<name>.city` | texto | — | Ciudad del servidor, para proveedores que permiten elegir. |
| `vpns.<name>.group` | texto | — | Solo nordvpn: grupo de servidores (`P2P`, `Dedicated IP`, `Double VPN`, `Onion Over VPN`). |
| `vpns.<name>.interface` | texto | — | Patrones del nombre de la interfaz del túnel (globs separados por coma); reemplaza los del proveedor. |
| `vpns.<name>.wait_timeout` | texto | — | Cuánto espera `up` a que conectes en la app (duración Go; si falta 5m, mínimo 10s). |
| `vpns.<name>.app_path` | texto | — | Ruta desde donde abrir la app del proveedor. |

## `groups`

Grupos de perfiles que se conectan o desconectan juntos con `--group <nombre>`.

| Clave | Tipo | Por defecto | Descripción |
|---|---|---|---|
| `groups.<name>.profiles` | lista de texto | — | **Obligatorio.** Perfiles del grupo (nombres de `vpns`). |

## `policies`

Reglas que deciden por qué VPN sale cada destino, en orden.

| Clave | Tipo | Por defecto | Descripción |
|---|---|---|---|
| `policies[].name` | texto | — | **Obligatorio.** Nombre de la regla. |
| `policies[].match.ip_ranges` | lista de texto | — | Rangos de IP en notación CIDR. |
| `policies[].match.domains` | lista de texto | — | Dominios o globs (`*.ejemplo.com`). |
| `policies[].match.apps` | lista de texto | — | Ejecutables o globs (experimental). |
| `policies[].via` | texto | — | **Obligatorio.** Perfil VPN por el que sale el tráfico. |
| `policies[].disabled` | booleano | — | `true` conserva la regla pausada hasta que la actives. Por defecto la regla está activa. |
| `policies[].on_tunnel_down` | texto | — | Vacío: hereda la opción del perfil. `allow` permite y avisa si detecta tráfico. `block` bloquea las IP conocidas. `connect` también las bloquea e intenta conectar si detecta tráfico TCP en Windows o Linux. En macOS, conecta manualmente. |

## `security`

Protecciones: kill switch, DNS guard, chequeos y registro de auditoría.

| Clave | Tipo | Por defecto | Descripción |
|---|---|---|---|
| `security.kill_switch.enabled` | booleano | — | Bloquea los destinos de un túnel protegido si se cae sin que lo apagues. |
| `security.kill_switch.mode` | texto | `strict` | Compatibilidad: `loose` equivale a `allow_lan: true`. Valores: `strict`, `loose`. |
| `security.kill_switch.allow_lan` | booleano | — | Deja la red local (impresoras, NAS, router) accesible mientras un túnel completo está caído; las búsquedas DNS pueden salir entonces por el DNS de tu red. Si es `false`, una caída también corta la red local hasta que vuelva la VPN. |
| `security.dns_guard.enabled` | booleano | — | Fuerza el DNS de la VPN mientras hay túneles conectados (evita fugas de DNS). |
| `security.dns_guard.fallback_dns` | texto | — | DNS de respaldo si la VPN no envía ninguno. |
| `security.integrity_check.enabled` | booleano | — | Comprueba periódicamente que cada túnel sigue sano. |
| `security.integrity_check.interval` | texto | `120s` | Cada cuánto (duración Go). |
| `security.integrity_check.expected_ips` | mapa de texto a texto | — | IP pública esperada por perfil (`perfil: ip`), para detectar fugas. |
| `security.leak_detection.enabled` | booleano | — | Bloque antiguo: `true` cuenta como aceptar la consulta de IP pública. |
| `security.public_ip_check.enabled` | booleano | — | Consulta opcional de tu IP pública a un servicio externo (ese servicio ve tu IP). Apagada si falta. |
| `security.public_ip_check.endpoints` | lista de texto | — | URLs `https://` a probar en orden; vacío = la de por defecto. |
| `security.audit_log.path` | texto | — | Archivo del registro de auditoría. |
| `security.audit_log.max_size_mb` | entero | `100` | Tamaño máximo de un archivo (MB); al pasarlo el mismo día sigue en audit-AAAA-MM-DD.1.log, .2… (por defecto 100). |
| `security.audit_log.retention_days` | entero | `30` | Días que se guardan, hoy incluido (1 a 365; sin poner o 0 = 30). Los más viejos se borran al iniciar, al llenarse un archivo y cada medianoche. |
| `security.audit_log.max_total_mb` | entero | `500` | Espacio máximo de todos los archivos juntos (MB, 1 a 10240; sin poner o 0 = 500, nunca menos que un archivo). Al pasarlo se borran primero los más viejos. |
| `security.audit_log.sign` | booleano | — | Firma cada entrada con HMAC-SHA256 (detecta manipulación). |

## `monitor`

Monitoreo, dashboard, programador y alertas.

| Clave | Tipo | Por defecto | Descripción |
|---|---|---|---|
| `monitor.enabled` | booleano | — | Activa el monitoreo. |
| `monitor.dashboard.port` | entero | `9741` | Puerto local del dashboard (`kongtrol config dashboard set-port` lo cambia solo en este equipo). |
| `monitor.dashboard.bind` | texto | `127.0.0.1` | Dirección donde escucha el dashboard. |
| `monitor.health_check.interval` | texto | `30s` | Cada cuánto se revisa cada túnel (duración Go). |
| `monitor.health_check.timeout` | texto | `10s` | Tiempo máximo de cada revisión. |
| `monitor.history.path` | texto | — | Archivo con el historial de estabilidad por perfil. |
| `monitor.history.flush_interval` | texto | `30s` | Cada cuánto se guarda el historial. |
| `monitor.scheduler.enabled` | booleano | — | Conecta perfiles según horarios. |
| `monitor.scheduler.interval` | texto | `1m` | Cada cuánto se evalúan los horarios. |
| `monitor.scheduler.rules[].name` | texto | — | Nombre de la regla de horario. |
| `monitor.scheduler.rules[].profiles` | lista de texto | — | Perfiles que deben estar conectados en ese horario. |
| `monitor.scheduler.rules[].weekdays` | lista de texto | — | Días (`mon` … `sun`), opcional. |
| `monitor.scheduler.rules[].start` | texto | — | Hora de inicio `HH:MM` (24 h), opcional. |
| `monitor.scheduler.rules[].end` | texto | — | Hora de fin `HH:MM` (24 h), opcional. |
| `monitor.scheduler.rules[].enabled` | booleano | — | `false` pausa la regla sin borrarla (por defecto encendida). Los cambios de reglas, intervalo y encendido se aplican sin reiniciar. |
| `monitor.split_dns.enabled` | booleano | — | Resuelve cada dominio con el DNS de su túnel (split DNS). |
| `monitor.split_dns.interval` | texto | `60s` | Cada cuánto se refresca el split DNS. |
| `monitor.alerts.on_vpn_drop.actions` | lista de texto | — | Acciones al caerse una VPN: `notify`, `log`, `kill_switch`. |
| `monitor.alerts.on_leak_detected.actions` | lista de texto | — | Acciones al detectar una fuga: `notify`, `log`, `kill_switch`. |
| `monitor.alerts.on_high_latency_ms` | entero | — | Alerta cuando la latencia supera estos milisegundos. |
| `monitor.alerts.on_reconnect_attempts` | entero | — | Alerta tras este número de intentos de reconexión. |
| `monitor.recent_connections.enabled` | booleano | — | Lista de destinos recientes (`kongtrol traffic recent`): solo en memoria, nunca en disco ni fuera del equipo; 500 destinos, cada uno se olvida 1 h después de verlo. Encendida por defecto (Windows y Linux); `false` la apaga y la borra. |

## `updates`

Aviso de versión nueva (opcional).

| Clave | Tipo | Por defecto | Descripción |
|---|---|---|---|
| `updates.check_enabled` | booleano | — | Avisa cuando hay una versión nueva: consulta api.github.com como máximo una vez al día (solo envía tu IP y `kongtrol/<versión>`). Nunca descarga ni instala. Apagado si falta. |

Las rutas `~/` se expanden a tu carpeta de usuario y `${VAR}` a variables de entorno. Las contraseñas nunca van aquí: viven en el llavero del sistema.
