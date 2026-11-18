<!-- Generado por `go generate ./cmd/kongtrol` (make docs) desde el árbol de comandos y internal/i18n. NO EDITAR A MANO. -->

<img src="../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Referencia de la CLI

Versión 2.4 · 2026-11-18

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

Desconecta los perfiles indicados y restaura tu DNS al caer el último túnel. Después: kongtrol status para confirmarlo.

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

### `kongtrol dashboard`

Abre el dashboard para configurar y conectar tus VPN desde el navegador; arranca Kongtrol en segundo plano si hace falta. Si no se abre, copia la dirección que aparece.

```text
kongtrol dashboard [opciones]
```

**Ejemplos**

```bash
kongtrol dashboard
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

### `kongtrol policy`

Muestra tus políticas, qué VPN usa cada destino y por qué, y prueba listas de destinos. Empieza con: kongtrol policy list.

```text
kongtrol policy [comando]
```

**Ejemplos**

```bash
kongtrol policy list
kongtrol policy show github.com
kongtrol policy test destinos.txt
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

Imprime una plantilla de kongtrol.yaml sin contraseñas para compartir con tu equipo. Para guardarla: kongtrol export > equipo.yaml.

```text
kongtrol export [opciones]
```

**Ejemplos**

```bash
kongtrol export
kongtrol export > equipo.yaml
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
| `--profile perfil` | filtrar por perfil |
| `--tail N` | mostrar los últimos N eventos (por defecto `100`) |

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
