# Guía de configuración — Kongtrol

Versión 1.3 · 2026-11-18

Esta guía te lleva desde cero hasta tener todas tus VPNs corriendo bajo Kongtrol.  
Tiempo estimado: **15–20 minutos**.

---

## Lo que vas a lograr

```
kongtrol up --group work
# [+] office connected     → tráfico 10.10.x.x va por FortiClient
# [+] dev-server connected → tráfico del servidor va por OpenVPN
# [+] aws connected        → *.amazonaws.com va por OpenVPN (AWS)

kongtrol up us-content
# [+] us-content connected → Netflix, Hulu por ProtonVPN
```

Un solo comando. Sin tocar cada cliente VPN por separado.

---

## Índice

1. [Prerequisitos](#1-prerequisitos)
2. [Instalar Kongtrol](#2-instalar-kongtrol)
3. [Organizar tus certificados](#3-organizar-tus-certificados)
4. [Ejecutar el wizard](#4-ejecutar-el-wizard-kongtrol-init)
5. [Configurar grupos](#5-configurar-grupos)
6. [Verificar con doctor](#6-verificar-con-doctor)
7. [Primera conexión](#7-primera-conexión)
8. [Dashboard web](#8-dashboard-web)
9. [Uso diario](#9-uso-diario)
10. [Para compañeros de equipo](#10-para-compañeros-de-equipo)
11. [Solución de problemas](#11-solución-de-problemas)
12. [Actualizar y desinstalar](#12-actualizar-y-desinstalar)

---

## 1. Prerequisitos

### Estado de las VPNs antes de empezar

> **TL;DR:** Para configurar (`init`, `doctor`) el estado no importa. Para conectar (`up`), cierra la GUI del cliente primero.

| Acción | Estado requerido de los clientes VPN |
|---|---|
| `kongtrol init` | Cualquiera — solo lee archivos y keychain |
| `kongtrol doctor` | Cualquiera — solo valida, no conecta |
| `kongtrol up office` | FortiClient GUI **cerrado** |
| `kongtrol up dev-server` | OpenVPN GUI **cerrado** |
| `kongtrol up us-content` | ProtonVPN GUI **cerrado** |

**¿Por qué?** Kongtrol llama directamente al CLI del cliente. Si la GUI ya tiene una sesión abierta, dos instancias intentan controlar el mismo túnel y chocan. A partir del momento en que uses Kongtrol, las GUIs quedan cerradas — Kongtrol es quien las maneja.

---

### Clientes VPN requeridos

Asegúrate de tener instalados los clientes VPN que vayas a usar:

| VPN | Cliente requerido | Verificar (Windows) | Verificar (Linux / macOS) |
|---|---|---|---|
| FortiClient (oficina) | [FortiClient](https://www.fortinet.com/support/product-downloads) (cualquier versión; ver [matriz de soporte](forticlient-support.md)) | Abre FortiClient — debe mostrar el túnel configurado | `FortiClient --version` o abre la app |
| OpenVPN (servidor / AWS) | [OpenVPN Community](https://openvpn.net/community-downloads/) | Busca `openvpn.exe` en `C:\Program Files\OpenVPN\bin\` | `openvpn --version` en terminal |
| ProtonVPN (contenido US) | [ProtonVPN](https://protonvpn.com/download) (app de escritorio) | Abre ProtonVPN — debe aparecer el botón de conexión | `protonvpn-cli --version` |

> **Nota Windows:** Los binarios de OpenVPN y ProtonVPN no están en el PATH del sistema — son instalaciones GUI. No corras `openvpn --version` desde PowerShell; Kongtrol los detecta directamente en sus rutas de instalación (`C:\Program Files\OpenVPN`, `C:\Program Files\Proton`). Si el wizard los muestra como detectados (✓), están listos.

> **Importante:** Kongtrol no reemplaza estos clientes — los *orquesta*. Deben estar instalados.

---

## 2. Instalar Kongtrol

<!-- feature:distribution -->
Elige una opción; todas instalan el programa `kongtrol` y el ícono de bandeja, y se actualizan
sin perder nada (tu configuración en `~/.kongtrol`, las contraseñas en el llavero del sistema y
el ayudante de red se conservan; desinstalar tampoco borra tu configuración).

### Windows

1. Descarga `Kongtrol-Setup-windows-amd64.exe` de
   [Releases](https://github.com/DerotLuna/vpn-kongtrol/releases/latest) y ábrelo. Se instala solo
   para tu usuario, sin permisos de administrador.
2. Deja marcadas las opciones recomendadas: abrir Kongtrol al iniciar sesión e instalar el
   ayudante de red (Windows pide permiso **una sola vez**).
3. Al terminar se abre Kongtrol: sigue en [Primer uso sin terminal](#primer-uso-sin-terminal).

Para actualizar, ejecuta el instalador nuevo encima: cierra Kongtrol (las VPN se desconectan
un momento; tu internet normal sigue funcionando), actualiza y lo vuelve a abrir.

Otras vías: `winget install Kongtrol.Kongtrol` (actualiza con `winget upgrade Kongtrol.Kongtrol`)
o Scoop (`scoop bucket add kongtrol https://github.com/DerotLuna/scoop-bucket`,
`scoop install kongtrol`, `scoop update kongtrol`). Con winget/Scoop cierra Kongtrol ("Salir y
desconectar todo") antes de actualizar: Windows no deja reemplazar un programa en uso.

### macOS

```bash
curl -fsSL https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.sh | sh
```

Instala `~/Applications/Kongtrol.app` y el comando `kongtrol`, y abre Kongtrol. Para actualizar,
vuelve a ejecutarlo: reinicia Kongtrol en la versión nueva y reconecta tus VPN. Alternativa:
`brew install DerotLuna/tap/kongtrol` (luego `brew upgrade kongtrol` y `kongtrol restart`).

La app no está notarizada por Apple (cuesta 99 USD al año); el script, Homebrew y npm no marcan
la descarga como "en cuarentena", así que abre sin aviso. Si descargas el `.zip` con el
navegador, la primera vez haz clic derecho → **Abrir**, o ve a Ajustes del Sistema → Privacidad
y seguridad → **Abrir igualmente**.

### Linux

Descarga el `.deb` (Ubuntu, Debian) o el `.rpm` (Fedora) de
[Releases](https://github.com/DerotLuna/vpn-kongtrol/releases/latest) y ábrelo con doble clic,
o usa el mismo `install.sh` de macOS. Arch: `yay -S kongtrol-bin`. Se actualiza con tu gestor
(`sudo apt upgrade`, `sudo dnf upgrade`) y luego `kongtrol restart` aplica la versión nueva.

En GNOME el ícono de bandeja necesita la extensión AppIndicator (Ubuntu ya la trae; en otras
distribuciones instala `gnome-shell-extension-appindicator`).

### npm (solo el programa `kongtrol`)

`npx kongtrol init`, o `npm i -g kongtrol` / `pnpm add -g kongtrol` / `bun add -g kongtrol`.

<!-- feature:first_run -->
### Primer uso sin terminal

1. Abre Kongtrol (menú de inicio, Launchpad o el menú de aplicaciones). Aparece el ícono en la
   bandeja y se abre el panel en tu navegador.
2. En **Perfiles VPN**, elige el tipo, completa los datos y pulsa **Crear**.
3. Kongtrol se reinicia solo y te lleva al resumen: ya puedes conectar desde el panel o el ícono.

¿Cerraste el navegador? En el ícono, **Configurar ahora…** lo abre de nuevo. Si prefieres la
terminal, `kongtrol init` hace lo mismo con un asistente (sección 4).

Si falta el ayudante (lo normal en una instalación nueva de macOS y Linux), el ícono avisa una
vez y su menú muestra **Instalar ayudante (una vez)**, incluso antes de crear la primera VPN:
pide el permiso de administrador una sola vez.

### Desde el código fuente (si tienes Go 1.27+)

```bash
git clone https://github.com/DerotLuna/vpn-kongtrol
cd vpn-kongtrol
make build
# Binario en: build/dist/kongtrol (o kongtrol.exe en Windows)
```

---

## 3. Archivos de conexión

**Si ya tienes tus VPNs configuradas, esta sección no requiere ninguna acción.** Kongtrol apunta a los archivos donde ya están — en el wizard simplemente escribes la ruta actual de cada archivo.

Solo necesitas hacer algo aquí si:
- Recibes archivos de un compañero y necesitas guardarlos en algún lugar
- Tu empresa te entrega un `.crt` / `.key` nuevos y no tienes dónde ponerlos
- Quieres reorganizar todo en un solo directorio para tener orden

### Qué necesita cada adaptador

| Adaptador | Qué pide el wizard | ¿Tienes que hacer algo? |
|---|---|---|
| FortiClient (Windows) | Nombre del túnel + usuario/contraseña | **No.** El túnel ya está en la GUI — Kongtrol lo activa por nombre. Sin certs. |
| FortiClient (Linux/macOS) | Ruta al `.crt`, ruta a la `.key`, credenciales | Tener los archivos en alguna ruta accesible y anotarla. |
| OpenVPN | Ruta al `.ovpn` | **No**, si ya tienes el `.ovpn`. Apunta a donde ya está (ej. `C:\Users\TU\OpenVPN\config\server.ovpn`). Si tiene `<cert>` y `<key>` embebidas, no necesitas campos extra. |
| ProtonVPN | Usuario + contraseña de cuenta Proton | **No.** Sin archivos. |
| WireGuard / otros | Ruta al archivo de config | **No**, si ya tienes el archivo. Solo anotar su ruta. |

> **Organización opcional:** si quieres centralizar, puedes crear `~/.kongtrol/certs/` y copiar los archivos ahí. Útil al incorporar un compañero nuevo o al recibir certs frescos. No es requerido si ya tienes todo en su lugar.

---

<!-- feature:wizard -->
## 4. Registrar tus VPNs: `kongtrol init`

> **¿Ya tienes tus VPNs configuradas?** Perfecto — `kongtrol init` no las toca. Lo que hace es decirle a Kongtrol dónde están y cómo hablarles: qué cliente usar, qué tunnel name tiene, dónde está el `.ovpn`. Si FortiClient ya conecta y OpenVPN ya tiene su `.ovpn`, este paso solo recopila esa información.

El wizard detecta tus clientes VPN instalados, te guía por cada perfil, y guarda las contraseñas en el **keychain del OS** (nunca en el archivo YAML).

```bash
kongtrol init
```

Lo primero que verás es la selección de idioma — presiona Enter para español o `n` para inglés:

```
¿Continuar en español? [S/n]  (Press n for English):
```

Luego el wizard muestra el logo animado y continúa en español:

```
  ▄▄▄▄▄▄▄▄▄▄▄▄▄
  █ ◉       ◉ █  K O N G T R O L
  █   ▄▄▄▄▄▄▄   █  Orquestación Multi-VPN
  ▀▀▀▀▀▀▀▀▀▀▀▀▀

Este asistente crea o actualiza ~/.kongtrol/kongtrol.yaml.
Las contraseñas se guardan en el llavero del sistema — nunca en el YAML.

▸  Clientes VPN detectados en este sistema:
    ✓  FortiClient             6.4.10.1821
    ✓  OpenVPN                 OpenVPN 2.6.8
    ✓  ProtonVPN               4.3.14

¿Agregar un nuevo perfil VPN? [s/N]: s
```

> **Detección automática:** el wizard busca clientes VPN en rutas estándar *y no estándar* de Windows, Linux y macOS (búsqueda recursiva hasta 3 niveles). Si un cliente está instalado pero no aparece en la lista, aún puedes configurarlo — el wizard te pedirá la ruta al binario cuando elijas ese tipo.

### Perfil 1: FortiClient (oficina)

```
  Nombre del perfil (ej. oficina, aws, wg-casa): office
  Tipos de adaptador: forticlient | openvpn | protonvpn | ciscoanyconnect |
                      wireguard | globalprotect | tailscale | cloudflarewarp
  Tipo [openvpn]: forticlient

  (Si el cliente no fue detectado automáticamente):
  !  Este cliente VPN no fue detectado automáticamente en tu sistema.
  Ruta al binario (vacío = autodetectar): ← Enter para omitir

  Host VPN (ej. vpn.empresa.com): vpn.tuempresa.com
  Puerto [443]:
  Nombre del túnel (como aparece en la UI de FortiClient) [Office]:
  Versión mayor de FortiClient [6]:
  Método de autenticación (certificate | credentials | certificate+credentials) [certificate+credentials]:
  Ruta al certificado cliente (ej. ~/.kongtrol/certs/oficina.crt): ~/.kongtrol/certs/office.crt
  Ruta a la clave privada (ej. ~/.kongtrol/certs/oficina.key): ~/.kongtrol/certs/office.key
  Usuario: tu_usuario
  Prioridad entre reglas que se solapan (mayor = gana, 1–100) [10]:
  Contraseña para office (guardada en llavero del sistema, no en YAML): ****
```

<!-- feature:openvpn -->
### Perfil 2: OpenVPN servidor

```
¿Agregar un nuevo perfil VPN? [s/N]: s
  Nombre del perfil (ej. oficina, aws, wg-casa): dev-server
  Tipo [openvpn]:
  Ruta al archivo .ovpn (ej. ~/.kongtrol/configs/servidor.ovpn): ~/.kongtrol/configs/server.ovpn
  Método de autenticación (certificate | credentials | certificate+credentials) [certificate]:
  Ruta al certificado cliente (vacío si está dentro del .ovpn):
  Ruta a la clave privada (vacío si está dentro del .ovpn):
  Prioridad entre reglas que se solapan (mayor = gana, 1–100) [10]: 20
```

> Si el `.ovpn` ya tiene `<cert>` y `<key>` embebidas, deja los paths en blanco.

### Perfil 3: OpenVPN AWS

```
¿Agregar un nuevo perfil VPN? [s/N]: s
  Nombre del perfil (ej. oficina, aws, wg-casa): aws
  Tipo [openvpn]:
  Ruta al archivo .ovpn (ej. ~/.kongtrol/configs/servidor.ovpn): ~/.kongtrol/configs/aws.ovpn
  Método de autenticación (certificate | credentials | certificate+credentials) [certificate]:
  Prioridad entre reglas que se solapan (mayor = gana, 1–100) [10]: 20
```

### Perfil 4: ProtonVPN

```
¿Agregar un nuevo perfil VPN? [s/N]: s
  Nombre del perfil (ej. oficina, aws, wg-casa): us-content
  Tipo [openvpn]: protonvpn
  Servidor / código de país (ej. US, NL, fastest) [fastest]: US
  Protocolo (wireguard | openvpn) [wireguard]:
  Usuario de ProtonVPN: tu_usuario_proton
  Prioridad entre reglas que se solapan (mayor = gana, 1–100) [10]: 5
  Contraseña para us-content (guardada en llavero del sistema, no en YAML): ****

¿Agregar un nuevo perfil VPN? [s/N]:
```

<!-- feature:killswitch feature:dns_guard -->
### Opciones de seguridad y políticas

```
¿Activar kill switch? (bloquea todo el tráfico si cae la VPN) [S/n]:    ← Enter = sí
¿Activar DNS guard? (previene fugas de DNS) [S/n]:
¿Activar log de auditoría firmado? [S/n]:
¿Activar panel web? (http://127.0.0.1:9741) [S/n]:

── Políticas de enrutamiento ──────────────────────────────────────────────

¿Agregar una política de enrutamiento? [S/n]: s

  Nombre de la política (ej: trabajo, streaming): Claude AI
  Perfil VPN para esta política:
    1) office
    2) us-content
  Elige [1]: 2

    Dominio o sufijo — deja en blanco para terminar
  Dominio: claude.ai
  Dominio: *.anthropic.com
  Dominio:

    Rango IP o IP — deja en blanco para terminar
  IP / rango:

¿Agregar una política de enrutamiento? [S/n]:

    También puedes editar políticas directamente en kongtrol.yaml → sección 'policies'.

¿Escribir configuración en ~/.kongtrol/kongtrol.yaml? [S/n]: s

[✓] Configuración escrita en ~/.kongtrol/kongtrol.yaml
[✓] Configuración válida.

!  Falta el ayudante de Kongtrol.
>  ¿Instalarlo ahora? (el sistema te pide permiso una vez) [S/n]:      ← Enter = sí
✓  Ayudante instalado y en marcha.
>  ¿Conectar ahora office, us-content? [S/n]:
```

Al terminar, `init` revisa el **ayudante** (el servicio que hace los cambios de red, así nunca necesitas una terminal de administrador) y te ofrece instalarlo; luego conecta lo que acabas de crear, igual que `kongtrol up`. En scripts: `kongtrol init --yes`. Kill switch y DNS guard se cambian después con `kongtrol config killswitch on|off|status` y `kongtrol config dns-guard on|off|status`, o en el panel → Seguridad.

> **¿Quieres agregar otro perfil o política después?** Vuelve a ejecutar `kongtrol init`. El wizard detecta el config existente, muestra los perfiles y políticas actuales, y no sobreescribe nada sin confirmación. Para más detalle sobre políticas ve a la [sección 5b](#5b-políticas-de-routing).

---

## 5. Configurar grupos

Los grupos te permiten conectar varios perfiles con un solo comando. Edita `~/.kongtrol/kongtrol.yaml` y agrega al final:

```yaml
groups:
  work:
    profiles: [office, dev-server, aws]

  travel:
    profiles: [us-content]

  full:
    profiles: [office, dev-server, aws, us-content]
```

Guarda el archivo. Verifica que sigue siendo válido:

```bash
kongtrol config validate
# [OK] Config is valid.
```

Si Kongtrol ya está funcionando en segundo plano (el daemon que arranca `kongtrol up`), no hace falta reiniciarlo para que tome el grupo nuevo — `kongtrol reload --group work` recarga el `kongtrol.yaml` recién editado y reinicia solo los túneles activos de ese grupo. Para un solo túnel, `kongtrol reload <perfil>`; para políticas, `kongtrol reload --policy`; sin argumentos, `kongtrol reload` aplica todo y, si agregaste perfiles, ofrece reiniciar Kongtrol (`--yes` en scripts). Ver [CLI.md → reload](CLI.md#kongtrol-reload).

---

<!-- feature:policies -->
## 5b. Políticas de routing

### ¿Para qué sirven?

Cuando tienes varios túneles activos al mismo tiempo, las políticas le dicen a Kongtrol qué tráfico va por qué VPN.

**Sin políticas:** cada VPN instala sus propias rutas al conectarse. El último en conectar suele ganar el tráfico general. No puedes controlar qué va por dónde.

**Con políticas:**
- `claude.ai` y `*.anthropic.com` → van por `us-content` (ProtonVPN)
- `10.10.0.0/16` (red interna) → van por `office` (FortiClient)
- `*.tuempresa.com` → van por `office`
- Todo lo demás → conexión directa, sin VPN

### Opción A: desde el wizard

Al final de `kongtrol init` (después de seguridad), el wizard pregunta:

```
── Políticas de enrutamiento ──────────────────────────────

¿Agregar una política de enrutamiento? [S/n]: s

  Nombre de la política (ej: trabajo, streaming): Claude AI
  Perfil VPN para esta política:
    1) office
    2) us-content
  Elige [1]: 2

    Dominio o sufijo — deja en blanco para terminar
  Dominio: claude.ai
  Dominio: *.anthropic.com
  Dominio:

    Rango IP o IP — deja en blanco para terminar
  IP / rango:

¿Agregar una política de enrutamiento? [S/n]:

    También puedes editar políticas directamente en kongtrol.yaml → sección 'policies'.
```

> **Tip:** puedes escribir varios dominios separados por coma en un solo prompt — `claude.ai, *.anthropic.com` genera dos entradas separadas.

### Opción B: editar el YAML directamente

Abre `~/.kongtrol/kongtrol.yaml` y añade la sección `policies:`. Estructura:

```yaml
policies:

  # ── Contenido US / Claude AI ────────────────────────────────────────
  - name: Claude AI
    match:
      domains:
        - claude.ai
        - "*.anthropic.com"
    via: us-content

  - name: Contenido geo-restringido US
    match:
      domains:
        - "*.netflix.com"
        - "*.hulu.com"
        - "*.disneyplus.com"
    via: us-content

  # ── Oficina ─────────────────────────────────────────────────────────
  - name: Red interna de oficina
    match:
      ip_ranges:
        - 10.10.0.0/16          # ajusta a la red de tu oficina
        - 192.168.50.0/24
    via: office

  - name: Dominios internos
    match:
      domains:
        - "*.tuempresa.com"     # cambia al dominio de tu empresa
        - intranet.local
    via: office

  # ── Servidor de desarrollo ───────────────────────────────────────────
  - name: Servidor dev
    match:
      ip_ranges:
        - 185.0.0.0/32          # reemplaza con la IP real de tu servidor
    via: dev-server

  # ── AWS ─────────────────────────────────────────────────────────────
  - name: AWS workloads
    match:
      ip_ranges:
        - 172.31.0.0/16         # VPC default de AWS — ajusta a tus CIDRs
      domains:
        - "*.amazonaws.com"
    via: aws

  # El tráfico que no coincide va por conexión directa (sin VPN).
  # Para forzar TODO el tráfico por una VPN descomenta esto:
  # - name: Default
  #   match:
  #     ip_ranges: [0.0.0.0/0]
  #   via: office
```

### Cómo funciona el matching

#### Dominios

| Regla | Cubre | No cubre |
|---|---|---|
| `claude.ai` | Solo `claude.ai` exacto | `api.claude.ai` |
| `*.anthropic.com` | `api.anthropic.com`, `docs.anthropic.com` | `anthropic.com` solo |

> **Tip:** Para cubrir dominio raíz Y subdominios, agrega dos entradas: `anthropic.com` y `*.anthropic.com`.

#### IPs y rangos

No necesitas ser experto en redes. Aquí la guía rápida:

| Quiero enrutar... | Escribo | Significado |
|---|---|---|
| **Una sola IP** | `172.28.152.26` | Solo esa máquina. Kongtrol agrega `/32` automáticamente |
| **Una red pequeña** (256 IPs) | `10.0.0.0/24` | Todo `10.0.0.x` (de `.0` a `.255`) |
| **Una red mediana** (65,536 IPs) | `10.0.0.0/16` | Todo `10.0.x.x` |
| **Una red grande** (16 millones) | `10.0.0.0/8` | Todo `10.x.x.x` |

**¿Qué significa el `/número`?** Es cuántos bits de la dirección son fijos. Mientras más grande el número, más específica (menos IPs):

```
/32  = 1 IP exacta          (ej: 172.28.152.26/32)
/24  = 256 IPs              (ej: 172.28.152.0/24 → todo 172.28.152.*)
/16  = 65,536 IPs           (ej: 172.28.0.0/16  → todo 172.28.*.*)
/8   = 16,777,216 IPs       (ej: 10.0.0.0/8     → todo 10.*.*.*)
```

> **No te preocupes:** si escribes solo la IP (sin `/`), Kongtrol la trata como una sola IP automáticamente. La notación con `/` solo hace falta cuando quieres cubrir un rango.

#### Prioridad (longest-prefix match)

Si dos reglas cubren la misma IP, gana la más específica:

| Regla | Cubre |
|---|---|
| `10.10.0.0/16` vía `office` | Todo `10.10.x.x` |
| `10.10.1.0/24` vía `dev-server` | Solo `10.10.1.x` — gana sobre `/16` |

Con la misma longitud de prefijo (o con dominios y apps, donde gana la primera regla que coincide), desempata la `priority` del perfil: **gana el número mayor**. Si el perfil no tiene `priority`, cuenta la posición de la política (las primeras van antes). `priority` no conecta ni cambia perfiles.

<!-- feature:watchdog -->
#### Si una VPN se cae y no vuelve (failover)

Kongtrol reintenta solo **esa misma** VPN (espera 2 s, 4 s, 8 s… hasta 5 min) y te avisa una vez en la bandeja, el dashboard y la vista en vivo. Nunca conecta otra VPN por su cuenta: cada perfil llega a redes distintas, aunque los nombres se parezcan.

Si tienes un perfil de respaldo que llega a **las mismas redes**, decláralo en el perfil principal:

```yaml
vpns:
  oficina:
    type: openvpn
    config: ~/.kongtrol/configs/oficina.ovpn
    failover: [oficina-respaldo]   # se prueba en orden si oficina no logra reconectarse
  oficina-respaldo:
    type: openvpn
    config: ~/.kongtrol/configs/oficina-b.ovpn
```

Cuando el respaldo entra, Kongtrol deja de reintentar `oficina` hasta que la conectes otra vez. También se edita en el dashboard (Perfiles → "Perfiles de respaldo").

<!-- feature:wireguard feature:watchdog -->
#### WireGuard: ¿el servidor responde?

En WireGuard la interfaz sigue arriba aunque el servidor ya no conteste. Por eso Kongtrol mira cada 10 s el **último handshake** (el saludo cifrado con el servidor) y lo muestra bajo el túnel en `kongtrol status`, la vista en vivo, la bandeja y el dashboard:

- **Nada extra:** el servidor respondió hace menos de ~3 minutos.
- **"Sin tráfico reciente · normal si no la estás usando":** no se envió nada, así que no hacía falta respuesta.
- **Estado `error` + "Sin respuesta del servidor · …":** Kongtrol siguió enviando y el servidor no contestó. Se trata como una caída: se reconecta sola, se usa tu `failover` y aplican kill switch y `on_tunnel_down`. Te avisa una vez y otra cuando vuelve a responder.
- **"No se puede comprobar si el servidor responde · …":** no se pudieron leer los contadores (por ejemplo, el ayudante falta o está desactualizado: `kongtrol service install`).

Si quieres que detecte un servidor caído aunque no uses la VPN, agrega `PersistentKeepalive = 25` en la sección `[Peer]` de tu `.conf`.

### Validar y aplicar

```bash
kongtrol config validate
# [OK] Config is valid.
```

Si no hay un daemon corriendo todavía, las políticas se leen al ejecutar `kongtrol up`. Si ya tienes uno activo (`kongtrol up` en background o vía dashboard), aplica el cambio sin reiniciarlo con `kongtrol reload --policy` — ver [CLI.md → reload](CLI.md#kongtrol-reload).

---

## 6. Verificar con doctor

Antes de conectar por primera vez, deja que Kongtrol valide todo:

```bash
kongtrol doctor
```

Salida esperada:

```
Kongtrol Doctor
────────────────────────────────────────────────────

  Configuration
  ✓  config file                       /Users/tu/.kongtrol/kongtrol.yaml
  ✓  config valid                      4 profile(s) defined

  VPN Binaries
  ✓  forticlient binary                FortiClient 6.4.10.1821
  ✓  openvpn binary                    OpenVPN 2.6.8
  ✓  protonvpn binary                  protonvpn-cli 3.10.0

  Certificates & Keys
  ✓  office: cert                      /Users/tu/.kongtrol/certs/office.crt
  ✓  office: key                       /Users/tu/.kongtrol/certs/office.key
  ✓  dev-server: config file           /Users/tu/.kongtrol/configs/server.ovpn
  ✓  aws: config file                  /Users/tu/.kongtrol/configs/aws.ovpn

  Keychain Credentials
  ✓  office: password                  found in OS keychain
  ✓  us-content: password              found in OS keychain

  Permissions
  ✓  kill switch                       platform implementation available
  ✓  dns guard                         available

  Registered Adapters
  ✓  registered adapters               ciscoanyconnect, cloudflarewarp, forticlient, ...

All checks passed. You're good to go.
```

<!-- feature:wizard -->
Cada aviso o error trae debajo una línea **qué hacer** (`→ …`). Lo mismo, sin terminal: en el
panel nuevo (`/app/`) la tarjeta **Diagnóstico** → «Revisar ahora» muestra cada comprobación
con su estado y qué hacer, y ofrece **Reparar** cuando falta el ayudante (pide el permiso del
sistema una vez). API: `GET /api/v1/doctor`.

Si hay ✗ en alguna línea, el mensaje te dice exactamente qué falta.

---

<!-- feature:daemon_lifecycle -->
## 7. Primera conexión

### Modo trabajo (oficina + servidores)

```bash
kongtrol up --group work
```

```
[+] office connected
[+] dev-server connected
[+] aws connected
```

Kongtrol se queda corriendo en primer plano. Cuando termines:

```
Ctrl+C   →  desconecta todo limpiamente
```

O en otra terminal:

```bash
kongtrol down --group work
```

### Solo una VPN

```bash
kongtrol up office        # solo FortiClient
kongtrol up us-content    # solo ProtonVPN
```

### Varios perfiles a la vez (sin grupo)

```bash
kongtrol up office dev-server aws
```

Los tres túneles se levantan en paralelo. El routing policy determina qué tráfico va por cuál — configura `policies:` en el YAML antes de usar esto (sección 5b).

### Ver estado

```bash
kongtrol status

PROFILE          STATUS         IP                 UPTIME
-------          ------         --                 ------
office           connected      10.10.0.5          1h 23m
dev-server       connected      185.x.x.x          1h 23m
aws              connected      172.31.4.7         1h 23m
us-content       disconnected   —                  —

Kill Switch: ON
```

Vista en tiempo real (se refresca cada 2 segundos):

```bash
kongtrol status --watch
```

---

<!-- feature:dashboard -->
## 8. Dashboard web

```bash
kongtrol dashboard
# Dashboard running at http://127.0.0.1:9741
```

`kongtrol dashboard` (o "Abrir panel" en el ícono de la bandeja) abre el panel en tu navegador
ya autorizado, con un código de un solo uso. Escribir `http://127.0.0.1:9741` a mano no basta:
la página te pedirá abrirla desde Kongtrol ([ADR-0036](adr/0036-browser-sessions-one-time-nonce.md)).
Es una consola de gestión completa, no solo
monitoreo, con estas páginas:

- **Overview** — túneles en tiempo real, gráficos de tráfico por túnel, rutas activas, resolución de policy, grupos de acceso rápido, conectar/desconectar todo
- **Studio** — pestañas: políticas de routing (CRUD + probador de reglas), perfiles VPN (CRUD, actualiza config + keychain; requiere reiniciar el daemon para conectar con un perfil nuevo/editado), y grupos (CRUD + conectar/desconectar)
- **Security** — activar/desactivar Kill Switch y DNS Guard en vivo, override de kill switch por perfil, estado de leak check y consulta de IP pública opcional (apagada por defecto, botón Revisar)
- **Audit Log** — eventos filtrables por perfil/nivel, con indicador de firma HMAC
- **Settings** — pestañas: general (health check, split DNS), seguridad (kill switch/DNS guard tuning, IP pública y fugas: servicios https opcionales, audit log), scheduler (reglas)
- **Panel nuevo (`/app/`)** — primeros pasos (ayudante, tus VPN, IP pública) y **Diagnóstico**: las mismas comprobaciones que `kongtrol doctor` con qué hacer en cada aviso, **Reparar** si falta el ayudante y **Exportar configuración** (sin contraseñas)

> El dashboard está compilado dentro del binario. No necesitas instalar nada extra. El
> puerto/bind del propio dashboard es CLI-only (`kongtrol config dashboard set-port <puerto>`) —
> cambiarlo desde la página que sirve la petición rompería la conexión.

---

## 9. Uso diario

### Comandos más usados

```bash
kongtrol up --group work          # empezar el día (office + dev-server + aws)
kongtrol down --group work        # terminar el día
kongtrol up us-content            # Netflix, Hulu, Claude AI...
kongtrol down --all               # apagar todo

kongtrol status                   # ver qué está conectado
kongtrol status --watch           # monitoreo en vivo
kongtrol ip                       # consulta tu IP pública y prueba fugas (requiere kongtrol config public-ip on)
kongtrol up --group work --dry-run # simulación completa sin tocar rutas/DNS/firewall
kongtrol policy list              # todas las políticas y su VPN
kongtrol policy show github.com   # qué regla aplica a un destino y por qué (--json: salida estructurada)
kongtrol config killswitch on     # kill switch (también: off, status)
kongtrol config dns-guard on      # DNS guard (también: off, status)
kongtrol dashboard                # abrir UI web
```

### Endpoints útiles del daemon/dashboard

```bash
GET  /api/v1/metrics/history                # histórico persistente por perfil
GET  /api/v1/dns/resolve?domain=...&via=... # split-DNS por túnel específico
GET  /api/v1/resolve?target=...&app=...     # policy flow-aware (app + target)
POST /api/v1/security/killswitch            # {"enabled": true|false} — activa/desactiva en vivo
POST /api/v1/security/dnsguard              # {"enabled": true|false} — activa/desactiva en vivo
GET  /api/v1/vpns                           # perfiles VPN · POST/PUT/DELETE para CRUD
GET  /api/v1/groups                         # grupos · POST/PUT/DELETE + /connect /disconnect
GET  /api/v1/audit?profile=...&level=...    # eventos de auditoría
```

Ver [docs/API.md](API.md) para la lista completa.

Para idioma operativo del CLI (mensajes nuevos), usa `KONGTROL_LANG=es` o `KONGTROL_LANG=en`.

### Si algo falla

```bash
kongtrol doctor                   # diagnóstico completo
kongtrol doctor --json            # salida estructurada para CI/automatización
```

### Actualizar una contraseña en el keychain

```bash
# Si cambias tu contraseña de FortiClient:
kongtrol init
# → elige el perfil "office" → "¿Actualizar credenciales de este perfil?" → ingresa la nueva
```

### Agregar un perfil nuevo sin tocar los existentes

```bash
kongtrol init
# → el wizard muestra los perfiles existentes y ofrece refrescar credenciales
# → luego pregunta "¿Agregar un nuevo perfil VPN?" → s
# → los perfiles anteriores no se modifican
```

---

## 10. Para compañeros de equipo

### Generar una plantilla compartible

```bash
kongtrol export > kongtrol-template.yaml
```

Esto genera el YAML completo (perfiles, políticas, grupos, seguridad, programador) **sin
contraseñas** — Kongtrol las guarda en el llavero del sistema, no en el archivo. El resultado se
vuelve a cargar tal cual. Sin terminal: panel nuevo → tarjeta **Diagnóstico** → «Exportar
configuración» (descarga `kongtrol-export.yaml`; API `GET /api/v1/config/export`). Ejemplo:

```yaml
# Configuración de Kongtrol (exportada). Sirve como plantilla para otro equipo.
# No lleva contraseñas: Kongtrol las guarda en el llavero del sistema, no en este archivo.
# Las rutas de archivos (.ovpn, .conf, certificados) son de este equipo: cópialos y ajústalas.
# En el otro equipo: guárdalo como kongtrol.yaml y vuelve a guardar las contraseñas (panel o kongtrol init).

vpns:
  office:
    type: forticlient
    host: vpn.tuempresa.com
    port: 443
    tunnel_name: Office
    auth:
      method: certificate+credentials
      cert: /Users/tu/.kongtrol/certs/office.crt
      key: /Users/tu/.kongtrol/certs/office.key
      username: tu_usuario
      password_keychain: office.password
policies:
  - name: corp
    match:
      ip_ranges:
        - 10.0.0.0/8
      domains:
        - '*.tuempresa.com'
    via: office
  ...
```

### Pasos para un compañero nuevo

1. Instala los clientes VPN (FortiClient, OpenVPN, etc.)
2. Descarga `kongtrol` para su plataforma
3. Recibe la plantilla (`kongtrol-template.yaml`) y sus propios certs
4. Copia el template a `~/.kongtrol/kongtrol.yaml`
5. Ajusta los paths de certs
6. Ejecuta `kongtrol init` para guardar sus credenciales
7. Ejecuta `kongtrol doctor` para verificar
8. `kongtrol up --group work` ✓

---

## 11. Solución de problemas

### El cliente VPN no aparece en la detección automática

El wizard busca en rutas estándar e instalaciones en subcarpetas. Si aun así no lo detecta:

1. **Ejecuta `kongtrol init`** y elige el tipo de adaptador igual — el wizard preguntará la ruta al binario si no lo encuentra:
   ```
   !  Este cliente VPN no fue detectado automáticamente en tu sistema.
   Ruta al binario (vacío = autodetectar): C:\ruta\personalizada\vpn.exe
   ```
2. O agrega `binary_path` manualmente en `~/.kongtrol/kongtrol.yaml`:
   ```yaml
   office:
     type: forticlient
     binary_path: "C:\ruta\personalizada\FortiClient.exe"
     ...
   ```
3. Verifica que el cliente esté instalado:
   ```bash
   which openvpn        # Linux/macOS
   where openvpn        # Windows
   ```

### "not registered — run 'warp-cli register' first"

Para Cloudflare WARP, debes registrar el dispositivo una vez:

```bash
warp-cli register
```

### DNS no restaurado después de un crash

Si Kongtrol se cierra inesperadamente con DNS guard activo:

```bash
# Windows
netsh interface ip set dns "Ethernet" dhcp

# Linux
sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf

# macOS
networksetup -setdnsservers Wi-Fi empty
```

### "unknown profile" al usar --group

Verifica que el nombre del grupo esté en `~/.kongtrol/kongtrol.yaml` bajo `groups:` y que el config sea válido:

```bash
kongtrol config validate
```

### Kill switch activo y sin internet después de desconectar

```bash
kongtrol down --all    # desactiva kill switch automáticamente
```

Si Kongtrol se cerró abruptamente:

```bash
# Windows (como Administrador)
netsh advfirewall reset

# Linux
sudo iptables -F OUTPUT

# macOS
sudo pfctl -d
```

### Contraseña incorrecta / expirada

```bash
kongtrol init
# Selecciona el perfil → "Refresh credentials" → ingresa la nueva contraseña
```

Kongtrol también emite alerta de **reauth requerida** cuando detecta errores típicos
de sesión/token/credenciales en reconnect/connect, con hint por tipo de adapter.

### Split-DNS transparente del sistema

Si activas `monitor.split_dns.enabled`, Kongtrol inyecta entradas controladas en el
archivo de hosts del sistema para dominios de policy, de modo que apps normales
(sin consumir la API) resuelvan por el túnel correspondiente.

### Historial persistente

Por defecto se guarda en `~/.kongtrol/history.json` y se actualiza en background
(`monitor.history.flush_interval`).

---

<!-- feature:uninstall feature:update_check -->
## 12. Actualizar y desinstalar

**Actualizar:** usa lo mismo con lo que instalaste (vuelve a correr el instalador de Windows o
`install.sh`, `brew upgrade kongtrol`, `npm update -g kongtrol`, `scoop update kongtrol`,
`winget upgrade Kongtrol.Kongtrol` o tu gestor de paquetes). Tu configuración se queda.

**Aviso de versión nueva (opcional, apagado):** `kongtrol config update-check on` o el panel
(Ajustes). Consulta api.github.com una vez al día como máximo; GitHub ve tu IP y
`kongtrol/<versión>`. Nunca descarga ni instala. `kongtrol config update-check status` muestra la
última revisión.

**Desinstalar (tus datos se quedan):**

```bash
kongtrol uninstall --dry-run   # muestra qué haría, sin cambiar nada
kongtrol uninstall             # quita la app; guarda ~/.kongtrol y tus contraseñas
kongtrol uninstall --purge     # también borra tus datos y contraseñas (pregunta [s/N])
kongtrol uninstall --data-only # solo borra tus datos; la app queda instalada
```

- Windows: Configuración › Aplicaciones › Kongtrol › Desinstalar. Pregunta «¿Borrar también tu
  configuración y las contraseñas guardadas? Si piensas reinstalar, elige No.» (por defecto No; en
  modo silencioso nunca borra datos).
- Instalado con `install.sh`/`install.ps1`: `… | sh -s -- --uninstall [--purge]`, o
  `kongtrol uninstall`.
- Homebrew, npm, Scoop, winget, `.deb`/`.rpm`, AUR: Kongtrol no toca esos archivos. Si quieres
  borrar tus datos, corre `kongtrol uninstall --purge` **antes** de quitar el paquete.
- Desde el panel: Ajustes › «Borrar todos mis datos» (la app queda instalada y empieza de cero).
- La bandeja no tiene opción de desinstalar: lo hace el desinstalador de tu sistema.

---

## Estructura de archivos

```
~/.kongtrol/
├── kongtrol.yaml          ← config principal (NO contiene contraseñas)
├── certs/
│   ├── office.crt
│   ├── office.key
│   └── ...
├── configs/
│   ├── server.ovpn
│   ├── aws.ovpn
│   └── ...
└── audit.log              ← log firmado de todos los eventos
```

Las contraseñas viven en:
- **Windows:** Windows Credential Manager
- **macOS:** Keychain
- **Linux:** libsecret / GNOME Keyring

---

*¿Problemas o preguntas? Abre un issue en el repositorio.*
