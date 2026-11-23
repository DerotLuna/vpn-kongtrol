<!-- Generado por `go generate ./cmd/kongtrol` (make docs) desde el árbol de comandos y internal/i18n. NO EDITAR A MANO. -->

<img src="../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Referencia de la CLI

Versión 2.18 · 2026-11-23

Todos los comandos de `kongtrol`, en los mismos grupos y orden que `kongtrol --help`. Cada comando trae su ayuda con ejemplos: `kongtrol <comando> --help`. English: [CLI.en.md](CLI.en.md). Primera vez: [SETUP.md](SETUP.md).

## Conexión

### `kongtrol up`

Conecta los perfiles indicados y abre la vista en vivo; las VPN siguen conectadas al cerrarla. Después: kongtrol status.

```text
kongtrol up [perfil...] [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--alert-bell` | emite un beep en alertas críticas |
| `--all` | conecta todos los perfiles configurados |
| `--dry-run` | simula el flujo de conexión y verificaciones sin cambiar estado del sistema |
| `--fav` | conecta perfiles favoritos guardados |
| `--group grupo` | conecta todos los perfiles del grupo |

**Ejemplos**

```bash
kongtrol up oficina
kongtrol up --group trabajo
kongtrol up oficina --dry-run
```

### `kongtrol down`

Desconecta los perfiles indicados y restaura tu DNS al caer el último túnel. Si quedan reglas con bloqueo, la desconexión de un perfil mantiene Kongtrol activo. Usa kongtrol down --all para detenerlo.

```text
kongtrol down [perfil...] [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--all` | desconecta todos los túneles activos |
| `--group grupo` | desconecta todos los perfiles del grupo |

**Ejemplos**

```bash
kongtrol down oficina
kongtrol down --group trabajo
kongtrol down --all
```

### `kongtrol status`

Muestra cada túnel con su estado, IP y tráfico. Si alguno falla: kongtrol doctor.

```text
kongtrol status [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--alert-bell` | emite un beep en alertas críticas |
| `-w, --watch` | auto-refresh cada 2 segundos |

**Ejemplos**

```bash
kongtrol status
kongtrol status --watch
kongtrol status --json
```

### `kongtrol routes`

Muestra qué destinos salen por cada túnel, o por uno solo si indicas el perfil. Con --system: las rutas del sistema que agregó Kongtrol.

```text
kongtrol routes [perfil] [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--system` | muestra las rutas del sistema que agregó Kongtrol (si falta alguna: kongtrol reload) |

**Ejemplos**

```bash
kongtrol routes
kongtrol routes oficina
kongtrol routes --system
```

### `kongtrol routes check-default`

Examina el archivo local de un perfil WireGuard en Windows o Linux sin cambiar la red.

```text
kongtrol routes check-default <perfil> [opciones]
```

**Ejemplos**

```bash
kongtrol routes check-default oficina
```

### `kongtrol routes default-egress`

Guarda la opción para la próxima conexión de un perfil WireGuard compatible en Windows o Linux. Desconecta el perfil antes de cambiarla.

```text
kongtrol routes default-egress <perfil> <on|off> [opciones]
```

**Ejemplos**

```bash
kongtrol routes default-egress oficina on
```

### `kongtrol dashboard`

Abre el dashboard para configurar y conectar tus VPN desde el navegador; arranca Kongtrol en segundo plano si hace falta. Si no se abre, copia la dirección que aparece.

```text
kongtrol dashboard [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--diagnostics` | abre directamente el diagnóstico |

**Ejemplos**

```bash
kongtrol dashboard
```

### `kongtrol tray`

Abre la aplicación de bandeja instalada. Úsala desde tu sesión de escritorio para ver y gestionar las VPN.

```text
kongtrol tray [opciones]
```

**Ejemplos**

```bash
kongtrol tray
```

## Configuración

### `kongtrol init`

Te guía paso a paso para agregar tus VPN, políticas y seguridad en kongtrol.yaml. Al terminar revisa el ayudante (te ofrece instalarlo) y te pregunta si conectas ya.

```text
kongtrol init [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `-y, --yes` | al terminar, instala el ayudante y conecta sin preguntar |

**Ejemplos**

```bash
kongtrol init
kongtrol init --yes
KONGTROL_LANG=en kongtrol init
```

### `kongtrol config`

Agrupa los comandos para validar kongtrol.yaml, encender o apagar la protección (kill switch, DNS guard, consulta de IP pública) y ajustar idioma, favoritos, grupo por defecto y panel en este equipo. Empieza con: kongtrol config validate.

```text
kongtrol config [comando]
```

**Ejemplos**

```bash
kongtrol config validate
kongtrol config killswitch on
kongtrol config lang es
```

### `kongtrol config validate`

Revisa kongtrol.yaml y explica cada error con cómo corregirlo, sin conectar nada. Si está bien: kongtrol up.

```text
kongtrol config validate [opciones]
```

**Ejemplos**

```bash
kongtrol config validate
kongtrol config validate --json
```

### `kongtrol config favorites`

Guarda los perfiles que más usas en este equipo para conectarlos juntos. Después: kongtrol up --fav.

```text
kongtrol config favorites [comando]
```

Alias: `fav`

**Ejemplos**

```bash
kongtrol config favorites add oficina
kongtrol config fav list
```

### `kongtrol config favorites list`

Muestra los perfiles favoritos guardados en este equipo. Para conectarlos: kongtrol up --fav.

```text
kongtrol config favorites list [opciones]
```

**Ejemplos**

```bash
kongtrol config favorites list
```

### `kongtrol config favorites add`

Marca el perfil como favorito en este equipo. Después: kongtrol up --fav.

```text
kongtrol config favorites add <perfil> [opciones]
```

**Ejemplos**

```bash
kongtrol config favorites add oficina
```

### `kongtrol config favorites remove`

Quita el perfil de tus favoritos; el perfil sigue configurado. Para revisar: kongtrol config favorites list.

```text
kongtrol config favorites remove <perfil> [opciones]
```

**Ejemplos**

```bash
kongtrol config favorites remove oficina
```

### `kongtrol config default-group`

Sin argumento muestra el grupo que kongtrol up conecta en este equipo cuando no indicas perfiles; con un grupo, lo guarda. Después: kongtrol up.

```text
kongtrol config default-group [grupo] [opciones]
```

**Ejemplos**

```bash
kongtrol config default-group
kongtrol config default-group trabajo
```

### `kongtrol config lang`

Guarda el idioma de los mensajes en este equipo; KONGTROL_LANG tiene prioridad. Para probarlo: kongtrol --help.

```text
kongtrol config lang <es|en> [opciones]
```

**Ejemplos**

```bash
kongtrol config lang es
kongtrol config lang en
```

### `kongtrol config dashboard`

Cambia dónde escucha el dashboard solo en este equipo, sin tocar kongtrol.yaml. Para verlo: kongtrol config dashboard show.

```text
kongtrol config dashboard [comando]
```

**Ejemplos**

```bash
kongtrol config dashboard show
kongtrol config dashboard set-port 9090
```

### `kongtrol config dashboard show`

Muestra la dirección y el puerto del dashboard en este equipo y de dónde salen. Para cambiarlos: kongtrol config dashboard set-port.

```text
kongtrol config dashboard show [opciones]
```

**Ejemplos**

```bash
kongtrol config dashboard show
```

### `kongtrol config dashboard set-port`

Guarda el puerto del dashboard en este equipo; aplica la próxima vez que Kongtrol arranque. Después: kongtrol down --all y kongtrol up.

```text
kongtrol config dashboard set-port <puerto> [opciones]
```

**Ejemplos**

```bash
kongtrol config dashboard set-port 9090
```

### `kongtrol config dashboard set-bind`

Guarda la dirección en la que escucha el dashboard; 127.0.0.1 deja que solo este equipo lo vea. Aplica cuando Kongtrol vuelva a arrancar.

```text
kongtrol config dashboard set-bind <dirección> [opciones]
```

**Ejemplos**

```bash
kongtrol config dashboard set-bind 127.0.0.1
```

### `kongtrol config dashboard reset`

Borra la dirección y el puerto guardados en este equipo y vuelve a los de kongtrol.yaml. Aplica cuando Kongtrol vuelva a arrancar.

```text
kongtrol config dashboard reset [opciones]
```

**Ejemplos**

```bash
kongtrol config dashboard reset
```

### `kongtrol config audit-log`

Sin opciones muestra dónde está el registro, cuántos días se guardan, el tamaño de cada archivo, el espacio total y cuánto ocupa ahora. Los días más viejos se borran solos al iniciar, al llenarse un archivo y cada medianoche; si se pasa del espacio total se borran primero los más viejos. Con Kongtrol abierto el cambio se aplica al momento; si no, se guarda en kongtrol.yaml.

```text
kongtrol config audit-log [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--days int` | días que se guardan, hoy incluido (1 a 365; 0 = 30) |
| `--max-size-mb int` | tamaño máximo de un archivo en MB (1 a 1024; 0 = 100) |
| `--max-total-mb int` | espacio máximo de todos los archivos juntos en MB (1 a 10240; 0 = 500) |

**Ejemplos**

```bash
kongtrol config audit-log
kongtrol config audit-log --days 30
kongtrol config audit-log --max-total-mb 500 --max-size-mb 100
```

### `kongtrol config public-ip`

Viene apagada. Encendida, Kongtrol pregunta tu IP pública a un servicio externo (por defecto api.ipify.org), que ve esa IP; solo al conectar o desconectar un túnel y cuando lo pides (kongtrol ip). El kill switch y el DNS guard no la necesitan.

```text
kongtrol config public-ip <on|off> [opciones]
```

**Ejemplos**

```bash
kongtrol config public-ip on
kongtrol config public-ip off
```

### `kongtrol config killswitch`

El kill switch evita fugas: si una VPN protegida se cae sin que la desconectes, Kongtrol bloquea lo que iba por ella hasta que vuelva. Con Kongtrol en marcha se aplica al instante; si no, se guarda en kongtrol.yaml.

```text
kongtrol config killswitch <on|off|status> [opciones]
```

**Ejemplos**

```bash
kongtrol config killswitch on
kongtrol config killswitch status
```

### `kongtrol config dns-guard`

El DNS guard evita que tus consultas DNS salgan fuera de la VPN: mientras hay VPN conectadas, Kongtrol usa solo sus DNS. Con Kongtrol en marcha se aplica al instante; si no, se guarda en kongtrol.yaml.

```text
kongtrol config dns-guard <on|off|status> [opciones]
```

**Ejemplos**

```bash
kongtrol config dns-guard on
kongtrol config dns-guard status
```

### `kongtrol config autostart`

Kongtrol se abre solo cuando inicias sesión en este equipo (lo deja así el instalador). Con off ya no se abre; con on vuelve a abrirse. Usa el mecanismo de tu sistema: en Windows, Configuración › Aplicaciones › Inicio; en Linux, la entrada de inicio automático; en macOS, un agente de inicio de tu usuario. Una actualización respeta lo que elijas.

```text
kongtrol config autostart <on|off|status> [opciones]
```

**Ejemplos**

```bash
kongtrol config autostart status
kongtrol config autostart off
kongtrol config autostart on
```

### `kongtrol config restore-tunnels`

Encendido por defecto. Si apagas o reinicias el equipo, o Kongtrol se cierra de golpe, al volver a abrirse conecta otra vez las VPN que tenías conectadas. Las que desconectaste tú no se conectan, y kongtrol down (o "Salir y desconectar todo") hace que no se conecte ninguna. Si una no conecta, Kongtrol lo intenta pocas veces y te avisa una sola vez.

```text
kongtrol config restore-tunnels <on|off|status> [opciones]
```

**Ejemplos**

```bash
kongtrol config restore-tunnels status
kongtrol config restore-tunnels off
kongtrol config restore-tunnels on
```

### `kongtrol config update-check`

Viene apagado. Encendido, Kongtrol pregunta a api.github.com como máximo una vez al día cuál es la última versión publicada; GitHub ve tu IP y «kongtrol/<versión>», nada más. Si hay una nueva te avisa (bandeja, panel, kongtrol version y status). Nunca descarga ni instala: actualizas con tu instalador o tu gestor de paquetes.

```text
kongtrol config update-check <on|off|status> [opciones]
```

**Ejemplos**

```bash
kongtrol config update-check on
kongtrol config update-check off
kongtrol config update-check status
```

### `kongtrol config recent-connections`

Encendida por defecto. Kongtrol anota a qué direcciones se conectó este equipo en la última hora, por qué VPN salió cada una y qué programa la abrió. La lista vive solo en la memoria de Kongtrol: no se guarda en disco ni en los registros, no sale del equipo y se borra al apagarla o al cerrar Kongtrol.

```text
kongtrol config recent-connections <on|off|status> [opciones]
```

**Ejemplos**

```bash
kongtrol config recent-connections status
kongtrol config recent-connections off
kongtrol config recent-connections on
```

### `kongtrol profile`

Crea perfiles VPN sin preguntas (para scripts), importa archivos de WireGuard u OpenVPN y busca las apps de VPN de este equipo. Con preguntas paso a paso: kongtrol init o el panel (kongtrol dashboard).

```text
kongtrol profile [comando]
```

**Ejemplos**

```bash
kongtrol profile detect
kongtrol profile add nordvpn --name casa --country Spain
kongtrol profile import oficina.conf --create
```

### `kongtrol profile import`

Revisa el archivo con las mismas reglas que el panel (sin scripts ni comandos; un .ovpn con los certificados dentro) y lo guarda en ~/.kongtrol/configs/ solo para tu usuario. No reemplaza un archivo con el mismo nombre salvo con --overwrite. Con --create también crea el perfil que lo usa (nombre: el del archivo, o --name). Después: kongtrol up <nombre>.

```text
kongtrol profile import <archivo.conf|archivo.ovpn> [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--create` | crea también el perfil VPN que usa el archivo |
| `--name string` | nombre del archivo guardado, sin extensión (por defecto, el del archivo) |
| `--overwrite` | reemplaza un archivo importado antes con el mismo nombre |
| `-y, --yes` | con --create, reinicia Kongtrol sin preguntar para usar el perfil nuevo |

**Ejemplos**

```bash
kongtrol profile import ~/Descargas/oficina.conf --create
kongtrol profile import proveedor.ovpn --name casa
kongtrol profile import oficina.conf --overwrite
```

### `kongtrol profile add`

Crea un perfil como "＋ Agregar" en el panel, con las mismas reglas. <tipo>: forticlient, openvpn, protonvpn, nordvpn, surfshark, ciscoanyconnect, wireguard, globalprotect, tailscale o cloudflarewarp. La contraseña nunca va en la línea de comandos: si el tipo la usa, se pide sin mostrarla en una terminal; si no, agrégala en el panel. Si Kongtrol está en marcha, ofrece reiniciarlo para usar el perfil. Después: kongtrol up <nombre>.

```text
kongtrol profile add <tipo> [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--app-path string` | ruta de la app oficial, si no está en su lugar habitual |
| `--auth-method string` | certificate, credentials o certificate+credentials (por defecto, el habitual del tipo) |
| `--cert string` | archivo del certificado de cliente |
| `--city string` | ciudad del servidor (NordVPN, Proton VPN) |
| `--config-file string` | archivo .conf de WireGuard, o .ovpn / .conf de OpenVPN, del perfil |
| `--country string` | país del servidor (NordVPN, Proton VPN) |
| `--group string` | grupo de servidores de NordVPN |
| `--host string` | dirección del servidor VPN (FortiClient, Cisco, GlobalProtect) o nodo de salida (Tailscale) |
| `--interface string` | nombre de la interfaz de red de la VPN, si Kongtrol no la reconoce sola |
| `--key string` | archivo de la clave privada del certificado |
| `--name string` | nombre del perfil (obligatorio), por ejemplo oficina |
| `--no-auto-connect` | no conectar la app automáticamente: Kongtrol la abre y espera a que conectes |
| `--on-tunnel-down string` | acción de sus reglas al apagarse la VPN: allow (por defecto), block o connect |
| `--port int` | puerto del servidor VPN |
| `--protocol string` | protocolo (Proton VPN) |
| `--server string` | servidor exacto (NordVPN, Proton VPN) |
| `--username string` | usuario de la VPN (la contraseña se pide aparte, nunca aquí) |
| `--wait-timeout string` | cuánto esperar a que conectes en la app (por ejemplo 5m) |
| `-y, --yes` | reinicia Kongtrol sin preguntar para usar el perfil nuevo |

**Ejemplos**

```bash
kongtrol profile add nordvpn --name casa --country Spain
kongtrol profile add forticlient --name trabajo --host vpn.miempresa.com --username ana
kongtrol profile add wireguard --name oficina --config-file ~/.kongtrol/configs/oficina.conf
```

### `kongtrol profile detect`

Lista las apps de VPN que encuentra (lo mismo que el panel y kongtrol init) y el tipo de perfil de cada una. Solo busca archivos: no abre ninguna app ni se conecta a nada. Después: kongtrol profile add <tipo> --name <nombre>.

```text
kongtrol profile detect [opciones]
```

**Ejemplos**

```bash
kongtrol profile detect
kongtrol profile detect --json
```

### `kongtrol profile on-tunnel-down`

Cambia la acción que heredan las reglas de este perfil. allow permite el tráfico, block bloquea las IP conocidas y connect las bloquea e intenta conectar si detecta tráfico TCP. Los intentos rechazados pueden pasar inadvertidos. En macOS, conecta manualmente.

```text
kongtrol profile on-tunnel-down <name> <allow|block|connect> [opciones]
```

**Ejemplos**

```bash
kongtrol profile on-tunnel-down oficina block
```

### `kongtrol policy`

Agrega o borra políticas (qué sitios, IP o apps van por qué VPN), muestra qué VPN usa cada destino y por qué, y prueba listas de destinos. Empieza con: kongtrol policy list.

```text
kongtrol policy [comando]
```

**Ejemplos**

```bash
kongtrol policy add github.com --via oficina
kongtrol policy show github.com
kongtrol policy test destinos.txt
```

### `kongtrol policy add`

Crea una política como "＋ Agregar" en el panel: un sitio o enlace (y sus subdominios), una IP o un rango (10.0.0.0/8) o una app (app:chrome.exe) van por la VPN de --via. El nombre se genera solo (por ejemplo "github.com por oficina") salvo con --name. Con --block-when-down ese tráfico no sale por internet normal mientras la VPN esté apagada. Después: kongtrol policy show <sitio>.

```text
kongtrol policy add <sitio|ip|rango|app:<exe>>... [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--block-when-down` | si esa VPN está apagada, bloquea el tráfico en vez de usar internet normal |
| `--name string` | nombre de la política (por defecto: "<primer destino> por <vpn>") |
| `--no-subdomains` | solo el sitio exacto, sin sus subdominios (*.sitio) |
| `--on-tunnel-down string` | al apagarse la VPN: inherit (por defecto), allow, block o connect |
| `--paused` | guarda la regla pausada hasta que la actives |
| `--via string` | perfil de VPN por el que va el tráfico (obligatorio) |

**Ejemplos**

```bash
kongtrol policy add claude.com "*.anthropic.com" --via oficina
kongtrol policy add api.anthropic.com 192.0.2.42 192.0.2.0/24 --via trabajo --block-when-down
kongtrol policy add app:teams.exe --via trabajo --name teams
```

### `kongtrol policy remove`

Borra la política con ese nombre (kongtrol policy list los muestra). Pregunta antes; para scripts: --yes. Ese tráfico vuelve a usar internet normal.

```text
kongtrol policy remove <nombre> [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `-y, --yes` | borra sin preguntar (para scripts) |

**Ejemplos**

```bash
kongtrol policy remove "github.com por oficina"
kongtrol policy remove teams --yes
```

### `kongtrol policy state`

Una regla pausada se conserva y no enruta, bloquea ni intenta conectar su VPN. Activarla aplica su comportamiento de inmediato.

```text
kongtrol policy state <name> [on|off] [opciones]
```

**Ejemplos**

```bash
kongtrol policy state intranet on
```

### `kongtrol policy on-tunnel-down`

Usa inherit para seguir la opción del perfil, allow para permitir, block para bloquear las IP conocidas o connect para bloquear e intentar conectar si se detecta tráfico TCP. En macOS, conecta manualmente.

```text
kongtrol policy on-tunnel-down <name> <inherit|allow|block|connect> [opciones]
```

**Ejemplos**

```bash
kongtrol policy on-tunnel-down intranet inherit
```

### `kongtrol policy list`

Muestra tus políticas con su VPN y sus IP resueltas. Para un destino concreto: kongtrol policy show <destino>.

```text
kongtrol policy list [opciones]
```

**Ejemplos**

```bash
kongtrol policy list
kongtrol policy list --json
```

### `kongtrol policy show`

Muestra qué política y qué VPN usa un destino (dominio o IP) o una app, y por qué. Para verlas todas: kongtrol policy list.

```text
kongtrol policy show <destino|app:<exe>> [opciones]
```

**Ejemplos**

```bash
kongtrol policy show github.com
kongtrol policy show 10.20.30.40
kongtrol policy show app:chrome.exe
```

### `kongtrol policy test`

Lee un archivo con un destino (IP o dominio) por línea y muestra qué VPN usaría cada uno. Si algo no cuadra: kongtrol policy show <destino>.

```text
kongtrol policy test <archivo> [opciones]
```

**Ejemplos**

```bash
kongtrol policy test destinos.txt
kongtrol policy test destinos.txt --json
```

### `kongtrol reload`

Carga kongtrol.yaml en Kongtrol en marcha: políticas y túneles conectados, o solo un perfil o un grupo. Si agregaste perfiles, te ofrece reiniciar Kongtrol. Después: kongtrol status.

```text
kongtrol reload [perfil] [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--group grupo` | recarga solo los túneles activos de este grupo |
| `--policy` | recarga solo el motor de políticas (omite grupos) |
| `-y, --yes` | si hace falta reiniciar Kongtrol, reinicia sin preguntar |

**Ejemplos**

```bash
kongtrol reload
kongtrol reload oficina
kongtrol reload --yes
```

### `kongtrol restart`

Detiene Kongtrol y lo vuelve a iniciar con kongtrol.yaml releído; reconecta las VPN que estaban conectadas. Es lo mismo que el botón Reiniciar del panel. Después: kongtrol status.

```text
kongtrol restart [opciones]
```

**Ejemplos**

```bash
kongtrol restart
```

### `kongtrol export`

Muestra en la terminal una plantilla de kongtrol.yaml sin contraseñas, para usarla en otro equipo o pasársela a tus compañeros. Kongtrol no la envía a ningún lado: tú decides dónde la guardas. Para guardarla en un archivo: kongtrol export > plantilla.yaml.

```text
kongtrol export [opciones]
```

**Ejemplos**

```bash
kongtrol export
kongtrol export > plantilla.yaml
```

## Diagnóstico

### `kongtrol doctor`

Revisa todo lo que Kongtrol necesita sin conectar ninguna VPN y dice cómo arreglar cada fallo. Para corregir lo seguro e instalar el ayudante si falta: kongtrol doctor --fix.

```text
kongtrol doctor [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--fix` | aplica las correcciones seguras e instala o actualiza el ayudante si falta |

**Ejemplos**

```bash
kongtrol doctor
kongtrol doctor --fix
kongtrol doctor --json
```

### `kongtrol ip`

Consulta una vez tu IP pública con un servicio externo (por defecto api.ipify.org) y la compara con las IP esperadas. Solo funciona si la activaste: kongtrol config public-ip on. Si encuentra una fuga: kongtrol doctor.

```text
kongtrol ip [opciones]
```

**Ejemplos**

```bash
kongtrol ip
kongtrol ip --json
```

### `kongtrol traffic`

Comandos para ver por dónde sale tu tráfico.

```text
kongtrol traffic [comando]
```

**Ejemplos**

```bash
kongtrol traffic recent
```

### `kongtrol traffic recent`

Lista los destinos de la última hora: nombre (si una política lo resolvió) o IP, puerto, programa, por qué VPN salió (o directo) y cuántas conexiones hubo. Necesita Kongtrol abierto. Los datos viven solo en memoria y no salen del equipo. Apágalo con: kongtrol config recent-connections off

```text
kongtrol traffic recent [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--limit int` | Cuántos destinos mostrar (0 = todos) |

**Ejemplos**

```bash
kongtrol traffic recent
kongtrol traffic recent --limit 20
kongtrol traffic recent --json
```

### `kongtrol logs`

Muestra los últimos eventos del registro de auditoría, o del proceso en segundo plano con --daemon. Para seguirlos en vivo: kongtrol logs -f.

```text
kongtrol logs [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--daemon` | muestra el registro del daemon en segundo plano (daemon.log) en lugar del de auditoría |
| `-f, --follow` | seguir eventos en vivo |
| `--level nivel` | filtrar por nivel (INFO\|WARN\|ERROR\|SECURITY) |
| `--profile perfiles` | filtrar por perfiles (varios: repite la opción o sepáralos con comas) |
| `--since día` | desde este día (AAAA-MM-DD) |
| `--tail N` | mostrar los últimos N eventos (por defecto `100`) |
| `--until día` | hasta este día incluido (AAAA-MM-DD) |

**Ejemplos**

```bash
kongtrol logs
kongtrol logs -f --profile oficina
kongtrol logs --daemon
```

## Sistema

### `kongtrol service`

El ayudante hace los cambios de red que piden permisos de administrador, así no abres una terminal de administrador. Instálalo una vez: kongtrol service install.

```text
kongtrol service [comando]
```

**Ejemplos**

```bash
kongtrol service install
kongtrol service status
kongtrol service uninstall
```

### `kongtrol service install`

Instala el ayudante como servicio del sistema; pide permiso de administrador una sola vez. Después: kongtrol service status.

```text
kongtrol service install [opciones]
```

**Ejemplos**

```bash
kongtrol service install
```

### `kongtrol service uninstall`

Quita el ayudante; sin él Kongtrol no puede cambiar rutas ni DNS. Para volver a instalarlo: kongtrol service install.

```text
kongtrol service uninstall [opciones]
```

**Ejemplos**

```bash
kongtrol service uninstall
```

### `kongtrol service status`

Muestra si el ayudante está instalado, responde y es de esta versión. Si falta o está desactualizado: kongtrol service install.

```text
kongtrol service status [opciones]
```

**Ejemplos**

```bash
kongtrol service status
```

### `kongtrol uninstall`

Desconecta tus VPN, cierra Kongtrol, quita el ayudante de red, el inicio automático y el PATH que agregó su instalador. Borra la app solo si la instaló install.sh o install.ps1; si vino de un gestor de paquetes (Homebrew, npm…) o del instalador de Windows, te dice cómo terminar. Tus datos están solo en este equipo: Kongtrol no guarda nada en internet ni en otro lugar. Sin opciones se quedan aquí, para que todo vuelva si reinstalas: la carpeta ~/.kongtrol (VPN, ajustes, archivos importados, registros) y tus contraseñas en el llavero del sistema de este equipo. --purge también los borra: la carpeta ~/.kongtrol, las contraseñas de Kongtrol del llavero y la clave que firma el registro de auditoría (y el registro de auditoría o el --config si los tienes en otra carpeta); no se puede deshacer. --data-only borra lo mismo, pero deja la app instalada.

```text
kongtrol uninstall [opciones]
```

**Opciones**

| Opciones | Descripción |
|---|---|
| `--data-only` | borra lo mismo que --purge, pero la app queda instalada |
| `--dry-run` | muestra qué haría, sin cambiar nada |
| `--purge` | también borra de este equipo la carpeta ~/.kongtrol y las contraseñas guardadas en el llavero (no se puede deshacer) |
| `-y, --yes` | no pregunta (para scripts) |

**Ejemplos**

```bash
kongtrol uninstall --dry-run      # muestra qué haría, sin cambiar nada
kongtrol uninstall                # quita la app; tus datos siguen en este equipo
kongtrol uninstall --purge        # quita la app y borra tus datos y contraseñas de este equipo
```

### `kongtrol version`

Muestra la versión instalada de Kongtrol. Para revisar el resto del equipo: kongtrol doctor.

```text
kongtrol version [opciones]
```

**Ejemplos**

```bash
kongtrol version
kongtrol --version
kongtrol version --plain
```

### `kongtrol help`

Muestra la ayuda de cualquier comando, con ejemplos. Prueba: kongtrol help up.

```text
kongtrol help [comando] [opciones]
```

**Ejemplos**

```bash
kongtrol help
kongtrol help up
```

## Opciones globales

| Opciones | Descripción |
|---|---|
| `-c, --config ruta` | ruta del archivo de configuración (por defecto: ~/.kongtrol/kongtrol.yaml) |
| `-h, --help` | muestra la ayuda de kongtrol |
| `--json` | emite salida JSON para automatización |
| `--plain` | salida plana ASCII (sin color ni símbolos Unicode) |
| `-q, --quiet` | reduce la salida no esencial (sin banner ni spinner) |
| `-v, --version` | muestra la versión instalada y sale |

## Idioma de la salida

Mensajes y ayuda siguen `KONGTROL_LANG`, luego el idioma guardado con `kongtrol config lang`, luego el del sistema (español si el sistema está en español, inglés si no). Para una sola ejecución: `KONGTROL_LANG=en kongtrol --help`. `kongtrol config favorites|defaults|lang|dashboard` guardan en `~/.kongtrol/preferences.json` (solo este equipo), no en `kongtrol.yaml`.

## Más

- Todas las claves de `kongtrol.yaml`: [CONFIG.md](CONFIG.md)
- API local que usan la bandeja y el dashboard: [API.md](API.md)
- Guía completa: https://kongtrol.app/guia
