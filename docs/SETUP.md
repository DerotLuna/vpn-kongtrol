# Guía de configuración — Kongtrol

Versión 1.22 · 2026-11-25

Esta guía te lleva desde cero hasta gestionar tus perfiles VPN con Kongtrol.

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
   para tu usuario, sin permisos de administrador. `kongtrol-windows-amd64.exe` es solo el
   comando de terminal: si lo abres con doble clic, solo te indica que descargues el instalador.
2. Deja marcadas las opciones recomendadas: abrir Kongtrol al iniciar sesión, instalar el
   ayudante de red (Windows pide permiso **una sola vez**) y agregar el comando `kongtrol` a la
   terminal (PATH de tu usuario; se quita al desinstalar).
3. Al terminar, con "Abrir Kongtrol y crear mi primera VPN" marcado, se abre el panel en la
   página Perfiles: sigue en [Primer uso sin terminal](#primer-uso-sin-terminal).

Para actualizar, ejecuta el instalador nuevo encima: si ya lo tienes, el instalador lo actualiza
sin perder nada. Muestra "Actualizar Kongtrol <actual> → <nueva>" y no vuelve a preguntar la
licencia, la carpeta ni las opciones (usa las de antes). Cierra Kongtrol (las VPN se desconectan
un momento; tu internet normal sigue funcionando), actualiza, vuelve a conectar solas las VPN
que tenías conectadas (también en una actualización silenciosa) y ofrece
"Abrir Kongtrol". Si la
opción de instalar el ayudante quedó seleccionada, Setup comprueba si falta o necesita
reparación o actualización. Windows pide permiso una vez si hace falta. Con la misma versión dice "Reinstalar (reparar)"; con una versión anterior te
pide confirmar (por defecto No).

<!-- feature:windows_installer_shutdown -->
Si Windows avisa que no pudo cerrar las aplicaciones, cancela el intento, abre Kongtrol desde
Inicio y elige **Salir y desconectar todo** en la bandeja. Vuelve a abrir el instalador.

Si antes copiaste `kongtrol.exe` a mano en otra carpeta del PATH (por ejemplo `C:\tools`), la
terminal puede seguir usando esa copia vieja: `kongtrol doctor` lo avisa en «kongtrol en la
terminal». Bórrala o quita esa carpeta del PATH y abre una terminal nueva.

Sin el instalador, en una línea de PowerShell (sin administrador):

```powershell
iwr https://kongtrol.app/install.ps1 -useb | iex
```

Instala lo mismo que el instalador: la app con su ícono de bandeja (`kongtrol-tray.exe`) y el
comando `kongtrol` en `%LOCALAPPDATA%\kongtrol\bin`, el acceso "Kongtrol" en el menú Inicio, el
inicio automático al entrar en Windows (`$env:KONGTROL_NO_AUTOSTART=1` para no ponerlo) y abre
Kongtrol. Solo el comando, sin la bandeja: `$env:KONGTROL_CLI_ONLY=1` antes de la línea (o
`-CliOnly` con `& ([scriptblock]::Create((iwr https://kongtrol.app/install.ps1 -useb))) -CliOnly`).
En Windows Server Core (sin escritorio) instala solo el comando y lo dice. Al actualizar respeta
lo que elegiste; `$env:KONGTROL_CLI_ONLY=0` agrega la bandeja después. Al terminar dice qué
instaló y cómo agregar la otra parte.

Si ya instalaste Kongtrol con `Kongtrol-Setup`, el script no instala una segunda copia: te dice
que actualices con el `Kongtrol-Setup` nuevo (o, para usar el script, que primero desinstales
Kongtrol en Configuración > Aplicaciones; tus datos se quedan).

winget y Scoop: pendientes (todavía no publicados).

### macOS

```bash
curl -fsSL https://kongtrol.app/install.sh | sh
```

El script comprueba la descarga con `checksums.txt` y, si tienes
[cosign](https://github.com/sigstore/cosign) 2.4 o más nuevo, también la
firma de ese archivo: si no coincide, se detiene sin instalar nada. Sin cosign lo dice en una línea
y muestra el comando para verificar a mano (lo mismo hace `install.ps1` en Windows). Detalles en
[SECURITY.md](SECURITY.md#verifying-releases).

Instala `~/Applications/Kongtrol.app` y el comando `kongtrol`, y abre Kongtrol. Para actualizar,
vuelve a ejecutarlo: dice "Updating Kongtrol <actual> → <nueva>", usa las mismas carpetas,
reinicia Kongtrol en la versión nueva y reconecta tus VPN. Homebrew: pendiente (todavía no
publicado).

La app no está notarizada por Apple (cuesta 99 USD al año); el script no marca
la descarga como "en cuarentena", así que abre sin aviso. Si descargas el `.zip` con el
navegador, la primera vez haz clic derecho → **Abrir**, o ve a Ajustes del Sistema → Privacidad
y seguridad → **Abrir igualmente**.

### Linux

Descarga el `.deb` (Ubuntu, Debian) o el `.rpm` (Fedora) de
[Releases](https://github.com/DerotLuna/vpn-kongtrol/releases/latest) y ábrelo con doble clic,
o usa el mismo `install.sh` de macOS (instala `kongtrol` y `kongtrol-tray`, la entrada del menú
y el inicio automático). Sin sesión de escritorio (sin `DISPLAY`, `WAYLAND_DISPLAY` ni
`XDG_CURRENT_DESKTOP`: un servidor, SSH) instala solo el comando `kongtrol` y lo dice en una
línea. Solo el comando a propósito: `curl -fsSL https://kongtrol.app/install.sh | sh -s -- --cli-only`
(o `KONGTROL_CLI_ONLY=1`); una actualización respeta esa elección y `--with-tray` agrega la
bandeja después. Para actualizar el `.deb`/`.rpm`, instala el paquete nuevo igual y luego
`kongtrol restart` aplica la versión nueva. AUR (Arch): pendiente (todavía no publicado).

En GNOME el ícono de bandeja necesita la extensión AppIndicator (Ubuntu ya la trae; en otras
distribuciones instala `gnome-shell-extension-appindicator`).

### npm

Pendiente (todavía no publicado).

<!-- feature:first_run -->
### Primer uso sin terminal

1. Abre Kongtrol (menú de inicio, Launchpad o el menú de aplicaciones). Aparece el ícono en la
   bandeja y se abre el panel en tu navegador.
2. En **Perfiles VPN**, elige el tipo, completa los datos y pulsa **Guardar y activar** (la guarda y
   la conecta) o **Guardar** (solo la guarda; la conectas cuando quieras) ([ADR-0115](adr/0115-save-without-connecting.md)).
3. Al guardar el primer perfil, el panel te deja crear una regla o volver a Inicio (y conectarlo,
   si elegiste **Guardar**).
   Puedes seguir en la misma pestaña y el menú de la bandeja se actualiza en unos segundos.

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

<!-- feature:profile_import -->
### Importar un `.conf` o `.ovpn`

Si te enviaron el archivo (o lo bajaste de tu proveedor), Kongtrol puede guardarlo por ti en
`~/.kongtrol/configs/`, solo para tu usuario:

- **Panel:** **＋ Agregar → Importar un archivo de VPN → Elegir archivo** (abre la ventana de archivos
  de tu sistema), o arrastra el archivo al panel. En Perfiles → Editar → WireGuard u OpenVPN →
  **Importar archivo** el campo de ruta se llena solo; también puedes escribir una ruta a mano.
- **Terminal:** `kongtrol profile import oficina.conf` (opcional `--name casa`, `--overwrite`). Con
  `--create` también crea el perfil que lo usa, con el nombre del archivo (o `--name`).

En cuanto eliges, arrastras o agregas un archivo, un certificado o una carpeta, el panel lo revisa
(«Revisando el archivo…») sin guardar nada y te dice al momento si está listo, qué certificados irán
dentro, dónde los encontró, si ya hay un archivo con ese nombre o qué falta y qué hacer. «Guardar» y
«Guardar y activar» se activan cuando la revisión pasa. En la terminal, lo mismo sin guardar nada:
`kongtrol profile import oficina.ovpn --check` (también con `--json`) (ADR-0118).

Si el `.ovpn` nombra otros archivos (`ca ca.crt`, `tls-auth ta.key 1`…), Kongtrol los guarda dentro
del perfil: con «Elegir archivo» o en la terminal los busca en la misma carpeta del `.ovpn`, y una
ruta completa que escriba el archivo (`ca C:\\certs\\ca.crt`) también se lee si es un certificado o
una clave en texto (PEM). Si arrastras el `.ovpn` solo, Kongtrol busca su original idéntico en las
carpetas habituales (OpenVPN, Descargas, Escritorio, Documentos) y toma los certificados de allí; si
aun así falta alguno, el panel lo nombra con «Elegir la carpeta donde están» y un botón
«Agregar <archivo>»; en la terminal pásalo después del archivo
(`kongtrol profile import oficina.ovpn ~/certs/ca.crt`). Solo se leen archivos normales del disco del
equipo (no accesos directos, carpetas de red ni unidades de red), de hasta 256 KB (ADR-0105, ADR-0112).

Sin preguntas, para scripts: `kongtrol profile detect` lista las apps de VPN instaladas y
`kongtrol profile add <tipo> --name <nombre> [--host …] [--country …]` crea el perfil con las mismas
reglas que el panel. La contraseña nunca va en la línea de comandos: se pide sin mostrarla en una
terminal o se agrega en el panel. Si Kongtrol está en marcha, te ofrece reiniciarlo (`--yes` para
scripts) para usar el perfil nuevo.

Se revisa con las mismas reglas que usa el servicio de Kongtrol antes de conectar: se rechazan los
archivos que ejecutan comandos (`PostUp`/`PreUp` en WireGuard; `up`, `down`, `plugin`,
`script-security 2` en OpenVPN) y la copia guardada de un `.ovpn` lleva sus certificados dentro
(`<ca>…</ca>`). Nunca reemplaza un archivo con el mismo nombre sin confirmarlo. Si el archivo es de tu cuenta de
Proton, al importarlo te avisa que no lo conectes mientras la app de Proton esté conectada (ver
«Archivo de Proton junto a la app de Proton», ADR-0103).

**Archivos que solo puede leer el administrador (Linux, p. ej. `/etc/wireguard/wg0.conf`).**
Kongtrol corre con tu usuario y no puede leerlos ahí. No hace falta moverlos a mano:

- **Terminal:** `kongtrol profile import /etc/wireguard/wg0.conf` (o `kongtrol init`) pregunta
  «¿Copiarlo a Kongtrol con tu permiso?» y pide tu contraseña una vez (`sudo`).
- **Panel:** escribe la ruta en el campo del archivo y guarda; Kongtrol avisa que solo el
  administrador puede leerlo y ofrece **Copiar con permiso de administrador** (la ventana de
  contraseña del sistema, `pkexec`).

Solo se copia si es un `.conf`/`.ovpn` válido (mismas reglas de arriba; un `.ovpn` necesita
`remote` y `client` o `tls-client`), un archivo normal y no un enlace (ni simbólico ni duro); queda
en `~/.kongtrol/configs/` con permiso solo para ti. Si no se acepta, el aviso es siempre el mismo y
no dice qué línea falla (corre como administrador y no muestra nada del archivo): guarda una copia
sin las líneas dudosas en tu carpeta e impórtala para ver el detalle. (ADR-0063) Sin terminal en macOS:
`sudo install -m 600 -o $USER /ruta/al/archivo ~/` y luego impórtalo. (ADR-0058)

---

<!-- feature:wizard -->
Al crear tu primer perfil VPN con `kongtrol init`, puedes guardarlo y salir o
continuar con las reglas y la seguridad. Tu perfil queda guardado antes del
paso de reglas y puedes conectar cuando estés listo.

## 4. Registrar tus perfiles VPN: `kongtrol init`

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
  >  FortiClient · conexión: CLI de FortiClient
     (en Windows: «FortiClient 6.4.10.1821 detectado · conexión: …»)

  (Si el cliente no fue detectado automáticamente):
  !  Este cliente VPN no fue detectado automáticamente en tu sistema.
  Ruta al binario (vacío = autodetectar): ← Enter para omitir

  Host VPN (ej. vpn.empresa.com): vpn.tuempresa.com
  Puerto [443]:
  Nombre del túnel (como aparece en la UI de FortiClient) [Office]:
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
### Opciones de seguridad y reglas

```
¿Activar kill switch? (si una VPN se cae sola, bloquea lo que iba por ella) [S/n]:    ← Enter = sí
¿Activar DNS guard? (previene fugas de DNS) [S/n]:
¿Activar log de auditoría firmado? [S/n]:
¿Activar panel web? (http://127.0.0.1:9741) [S/n]:

── Reglas de salida ───────────────────────────────────────────────────────

¿Agregar una regla de salida? [S/n]: s

  Nombre de la regla (ej: trabajo, streaming): Claude AI
  Perfil VPN para esta regla:
    1) office
    2) us-content
  Elige [1]: 2

    Dominio o sufijo — deja en blanco para terminar
  Dominio: claude.ai
  Dominio: *.anthropic.com
  Dominio:

    Rango IP o IP — deja en blanco para terminar
  IP / rango:

¿Agregar una regla de salida? [S/n]:

    También puedes editar reglas directamente en kongtrol.yaml → sección 'policies'.

¿Escribir configuración en ~/.kongtrol/kongtrol.yaml? [S/n]: s

[✓] Configuración escrita en ~/.kongtrol/kongtrol.yaml
[✓] Configuración válida.

!  Falta el ayudante de Kongtrol.
>  ¿Instalarlo ahora? (el sistema te pide permiso una vez) [S/n]:      ← Enter = sí
✓  Ayudante instalado y en marcha.
>  ¿Conectar ahora office, us-content? [S/n]:
```

Al terminar, `init` revisa el **ayudante** (el servicio que hace los cambios de red, así nunca necesitas una terminal de administrador) y te ofrece instalarlo; luego conecta lo que acabas de crear, igual que `kongtrol up`. En scripts: `kongtrol init --yes`. Kill switch y DNS guard se cambian después con `kongtrol config killswitch on|off|status` y `kongtrol config dns-guard on|off|status`, o en el panel → Seguridad.

> **¿Quieres agregar otro perfil o regla después?** Vuelve a ejecutar `kongtrol init`. El wizard detecta el config existente, muestra los perfiles y reglas actuales, y no sobreescribe nada sin confirmación. Para más detalle sobre reglas ve a la [sección 5b](#5b-reglas-de-salida).

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

Si Kongtrol ya está funcionando en segundo plano (el daemon que arranca `kongtrol up`), no hace falta reiniciarlo para que tome el grupo nuevo — `kongtrol reload --group work` recarga el `kongtrol.yaml` recién editado y reinicia solo los túneles activos de ese grupo. Para un solo túnel, `kongtrol reload <perfil>`; para reglas, `kongtrol reload --policy`; sin argumentos, `kongtrol reload` aplica todo y, si agregaste perfiles, ofrece reiniciar Kongtrol (`--yes` en scripts). Ver [CLI.md → reload](CLI.md#kongtrol-reload).

---

<!-- feature:policies -->
## 5b. Reglas de salida

### ¿Para qué sirven?

Un **perfil VPN** guarda una configuración de conexión; puedes tener varios perfiles del mismo proveedor. Una **regla** indica qué destinos usan un perfil y qué hacer si está desconectado. Una **ruta** es un camino de red que usa el equipo, incluso si la trae la VPN o el sistema.

**Sin reglas:** cada VPN puede instalar sus propias rutas al conectarse. El sistema decide cuál usa según esas rutas y sus métricas.

**Con reglas:**
- `claude.ai` y `*.anthropic.com` → van por `us-content` (ProtonVPN)
- `10.10.0.0/16` (red interna) → van por `office` (FortiClient)
- `*.tuempresa.com` → van por `office`
- Todo lo demás → conexión directa, sin VPN

### La forma rápida: panel o una línea en la terminal

En el panel, usa **＋ Agregar** y escribe sitios, enlaces, IP o rangos separados por comas
o saltos de línea. Por ejemplo, `claude.com, api.anthropic.com, *.anthropic.com, 192.0.2.42`.
Elige el perfil y guarda la regla pausada. Puedes activarla al terminar o después en **Reglas**.
Después de guardar un perfil puedes elegir **Agregar regla de salida**. Al crear o editar
una regla, los botones Claude, ChatGPT/OpenAI y Oracle/Java agregan varios destinos a la lista
editable. Son aproximaciones, no garantizan acceso: los bloqueos varían por red, las apps pueden
usar otras direcciones y los proveedores tienen sus propias condiciones. Un comodín alcanza un
nivel y Kongtrol descubre solo algunos subdominios; agrega los nombres concretos que necesites.
La terminal admite varios destinos como argumentos:

```bash
kongtrol policy add claude.ai --via us-content            # claude.ai y *.claude.ai
kongtrol policy add 10.10.0.0/16 intranet.tuempresa.com --via office --block-when-down
kongtrol policy add 10.20.0.0/16 --via office --name red-oficina
kongtrol policy add claude.com api.anthropic.com '*.anthropic.com' --via us-content --name claude --paused
kongtrol policy state claude on                          # usa off para pausarla
kongtrol policy remove red-oficina                        # pregunta antes; --yes para scripts
```

Un enlace (`https://…/x`) se guarda como su sitio; `--no-subdomains` deja solo el sitio exacto. Si
no pones `--name`, el nombre sale solo ("claude.ai por us-content"). Con Kongtrol en marcha se aplica
al momento, salvo que uses `--paused`. Sin Kongtrol en marcha se guarda en `kongtrol.yaml`.

Las apps (`app:teams.exe`) se guardan y `kongtrol policy show` las reconoce, pero **aún no se aplican**:
ningún programa sale por la VPN, se bloquea ni conecta una VPN por su nombre. El panel, la terminal
y la documentación lo dicen igual ([ADR-0113](adr/0113-label-app-rules-honestly.md)). Para dirigir un
programa, agrega los sitios o IPs que usa.

<!-- feature:policies -->
Al crear el perfil o en **Perfiles**, elige qué hacen sus reglas cuando la VPN está apagada. Cada regla puede heredar
esa opción o elegir otra. También puedes cambiarla desde la terminal.

Si una regla activa intenta conectar su VPN al detectar tráfico, Kongtrol hace hasta tres
intentos por perfil. Tras un fallo, espera al menos 3 minutos para el segundo y 6 para el tercero.
Si no conecta, conserva el bloqueo de los destinos conocidos y avisa para que conectes manualmente.
En Windows, si detecta Proton VPN activa, aplaza el arranque automático de WireGuard directo.
El aviso identifica la otra VPN y el aplazamiento no consume un intento.

```bash
kongtrol profile on-tunnel-down office connect
kongtrol policy on-tunnel-down teams allow
```

`block` protege las direcciones conocidas. `connect` también las bloquea e intenta conectar si
se detecta tráfico TCP. Un intento rechazado puede pasar inadvertido. En macOS, conecta desde Inicio.

### Opción A: desde el wizard

Al final de `kongtrol init` (después de seguridad), el wizard pregunta:

```
── Reglas de salida ───────────────────────────────────────

¿Agregar una regla de salida? [S/n]: s

  Nombre de la regla (ej: trabajo, streaming): Claude AI
  Perfil VPN para esta regla:
    1) office
    2) us-content
  Elige [1]: 2

    Dominio o sufijo — deja en blanco para terminar
  Dominio: claude.ai
  Dominio: *.anthropic.com
  Dominio:

    Rango IP o IP — deja en blanco para terminar
  IP / rango:

¿Agregar una regla de salida? [S/n]:

    También puedes editar reglas directamente en kongtrol.yaml → sección 'policies'.
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
| `claude.com` | Dominio raíz `claude.com` | `www.claude.com` |
| `api.anthropic.com` | Subdominio exacto `api.anthropic.com` | `docs.anthropic.com` |
| `*.anthropic.com` | Un nivel, como `api.anthropic.com` o `docs.anthropic.com` | `anthropic.com`, `a.b.anthropic.com` |

Kongtrol consulta algunos nombres frecuentes para instalar rutas a partir del comodín.
Escribe explícitamente los subdominios que necesites. Puedes pegar nombres con
mayúsculas, un punto final o caracteres internacionales como `café.example`.
Para destinos por dirección, `192.0.2.42` identifica una IP y `192.0.2.0/24` una red.

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

Con la misma longitud de prefijo (o con dominios y apps, donde gana la primera regla que coincide), desempata la `priority` del perfil: **gana el número mayor**. Si el perfil no tiene `priority`, cuenta la posición de la regla (las primeras van antes). Va de 0 a 1000 (0 = sin prioridad). Ojo: sin `priority`, la posición cuenta en la misma escala; si solo uno de dos perfiles tiene número, el resultado depende también de la posición, así que pon prioridad a los dos. Cámbiala en el panel (**Perfiles › Orden de sus reglas**, o «Cambiar prioridad» en **Reglas**, que marca las reglas que se cruzan) o con `kongtrol profile priority <perfil> <N>` (`none` la quita); `kongtrol profile list` y `kongtrol policy list` la muestran ([ADR-0108](adr/0108-show-and-change-profile-priority.md)). `priority` no conecta ni cambia perfiles.

<!-- feature:watchdog -->
#### Si una VPN se cae y no vuelve (failover)

Kongtrol reintenta solo **esa misma** VPN, **3 veces** (a los 2 s, 30 s y 2 min). Si no vuelve, deja de intentarlo y te avisa **una sola vez** en la bandeja, el dashboard y la vista en vivo: «No se pudo reconectar <perfil> tras 3 intentos», con **Reintentar** (o `kongtrol up <perfil>`). Vuelve a probar solo cuando cambia tu red (otra Wi‑Fi, un cable, vuelve internet) o el equipo sale de suspensión, sin avisarte otra vez. Si el kill switch protegía esa VPN, **sigue bloqueando** (sus destinos, o todo si llevaba todo tu tráfico): el aviso te dice que la desconectes para volver a tener internet ([ADR-0098](adr/0098-bounded-reconnect-and-restore-at-start.md)). Nunca conecta otra VPN por su cuenta: cada perfil llega a redes distintas, aunque los nombres se parezcan. Si la VPN es de una app de proveedor que Kongtrol no puede reconectar (sin CLI, o con `auto_connect: false`), te avisa igual en esos tres lugares para que la conectes en su app; Kongtrol la sigue en cuanto vuelve.

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

Si no hay un daemon corriendo todavía, las reglas se leen al ejecutar `kongtrol up`. Si ya tienes uno activo (`kongtrol up` en background o vía dashboard), aplica el cambio sin reiniciarlo con `kongtrol reload --policy` — ver [CLI.md → reload](CLI.md#kongtrol-reload).

---

## 6. Verificar con doctor

<!-- feature:direct_diagnostics -->
Desde un error en Inicio o en la bandeja, usa **Abrir diagnóstico**. También
puedes abrir esa pantalla con `kongtrol dashboard --diagnostics` o leer las
comprobaciones en la terminal con `kongtrol doctor`.

<!-- feature:wizard_doctor -->
**Convivencia de VPN.** Diagnóstico avisa cuando detecta dos VPN con rutas que
cubren la misma familia de IP completa, o AnyConnect activo junto con otro
túnel. Para aislar el problema, desconecta temporalmente las otras VPN desde sus
aplicaciones y vuelve a intentar. Los filtros del proveedor, DNS y errores de
controlador pueden requerir revisar el diagnóstico de la aplicación.
[Investigación y pruebas de convivencia](research/2026-11-21-vpn-coexistence.md)

En Windows, Diagnóstico también comprueba si aparece un adaptador de TunnelBear
y muestra el código si Windows informa un problema con el dispositivo. Desde el
panel puedes abrir la guía del fabricante. Algunos protocolos de TunnelBear usan
adaptadores de Windows, por lo que la ausencia del dispositivo con su nombre es
solo una pista.

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
panel, **Ajustes → Diagnóstico** → «Revisar ahora» muestra cada comprobación
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
ya autorizado, con un código de un solo uso. Escribir `http://127.0.0.1:9741` a mano, o abrir el
panel en otra pestaña nueva, no basta: la página te pedirá abrirla desde Kongtrol; recargar la misma
pestaña sí mantiene la sesión ([ADR-0039](adr/0039-browser-sessions-one-time-nonce.md),
[ADR-0062](adr/0062-origin-bound-sessions-token-rotation-listener-owner.md)). Si otro usuario de
este equipo ocupa el puerto de Kongtrol, la bandeja y la terminal no le envían tu clave y te dicen
qué hacer (cambiar el puerto con `kongtrol config dashboard set-port <puerto>`).
Usa **Buscar** o **Ctrl/Cmd+K** para encontrar secciones, acciones, perfiles, reglas y grupos;
**Esc** cierra la búsqueda. Se realiza solo en este equipo.
En Inicio y en el panel de la bandeja, las listas largas permiten buscar perfiles por nombre
o tipo y mostrar más. Se ordenan al abrirlas: conectados primero y después los conectados
más recientemente, según el historial local; los perfiles conectados no se ocultan por el límite.
Durante una restauración o reinicio, los perfiles pendientes dicen «Reconectando» y puedes
cancelarlos. En el menú nativo de la bandeja, los perfiles mantienen su lugar y las listas
largas ofrecen un acceso a la búsqueda del panel.
Es una consola de gestión completa, no solo
monitoreo. Arriba, «＋ Agregar» abre una sola hoja para agregar un perfil con una VPN que ya usas, importar un
`.conf`/`.ovpn` (también soltándolo sobre el panel) o agregar una regla de salida;
también se abre con un enlace (`/?add=menu|vpn|file|rule&via=<perfil>&target=<sitio>`,
[ADR-0055](adr/0055-add-sheet-one-entry-point.md)). Las páginas:

- **Inicio** — tus perfiles VPN en vivo (estado, IP, tiempo, tráfico, latencia con tendencia, qué sale por cada túnel) con conectar / cancelar / desconectar / recargar y conectar o desconectar todos; avisos (con «Conectar <perfil>» cuando una regla necesita un perfil desconectado), resumen de protección, salida a internet y grupos
- **Perfiles** — crear, editar, renombrar y borrar perfiles VPN (la contraseña va al llavero del sistema), respaldo (`failover`) y la guía del primer uso; pestaña **Grupos** (crear, editar, conectar o desconectar varios perfiles a la vez)
- **Reglas** — qué sitios y redes salen por cada perfil (las apps se muestran con «aún no se aplica») (crear, editar, probar antes de guardar, recargar del archivo) y qué pasa si su perfil está desconectado; pestaña **Horarios** (conectar o desconectar perfiles a ciertas horas)
- **Tráfico** — tráfico por perfil en el tiempo, por dónde sale un sitio o IP, rutas activas por túnel, la tabla de rutas del equipo y **Recientes** (a dónde se conectó el equipo en la última hora y por dónde salió; se guarda solo en memoria y se apaga en Ajustes › Privacidad) con «Agregar como regla»
- **Seguridad** — Kill Switch y DNS Guard en vivo (con sus ajustes), el interruptor por perfil, la IP pública opcional (apagada por defecto) y DNS dividido
- **Actividad** — registro de auditoría (filtros por fecha, perfil y nivel; firma HMAC comprobada; 10 entradas y «Ver 20 más», que pide las más antiguas al daemon cuando hace falta; en vivo, las nuevas esperan tras «N nuevas · Mostrar») y los avisos del daemon
- **Ajustes** — idioma, tema, **Diagnóstico** (las mismas comprobaciones que `kongtrol doctor`, **Reparar** si falta el ayudante, **Exportar configuración** sin contraseñas), comprobación de salud, avisos de versión nueva, ayudante, reiniciar / detener Kongtrol y borrar datos

> El dashboard está compilado dentro del binario. No necesitas instalar nada extra. El
> puerto/bind del propio dashboard es CLI-only (`kongtrol config dashboard set-port <puerto>`) —
> cambiarlo desde la página que sirve la petición rompería la conexión.

---

## 9. Uso diario

<!-- feature:default_egress -->
En Windows o Linux, abre la edición de un perfil WireGuard directo y elige **Salida por defecto de WireGuard**.
También puedes revisar el archivo con `kongtrol routes check-default <perfil>` y cambiar la opción
con `kongtrol routes default-egress <perfil> on|off`. Desconecta el perfil antes de cambiarla y
conéctalo para aplicarla. Las rutas más específicas de la red local y de otras VPN conservan
su prioridad. Consulta [ADR-0090](adr/0090-wireguard-default-egress-windows.md) para los requisitos.

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
kongtrol policy list              # todas las reglas y su perfil
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

Esto genera el YAML completo (perfiles, reglas, grupos, seguridad, programador) **sin
contraseñas** — Kongtrol las guarda en el llavero del sistema, no en el archivo. El resultado se
vuelve a cargar tal cual. Sin terminal: panel → **Ajustes → Diagnóstico** → «Exportar
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

### "not registered — run 'warp-cli registration new' once"

Para Cloudflare WARP, debes registrar el dispositivo una vez:

```bash
warp-cli registration new
```

Los clientes WARP anteriores a 2024 usaban `warp-cli register`; los actuales ya no lo aceptan.

### DNS no restaurado después de un crash

Si Kongtrol se cierra inesperadamente con DNS guard activo:

```bash
# Windows
netsh interface ip set dns "Ethernet" dhcp

# Linux con systemd-resolved (Ubuntu, Fedora): Kongtrol fija el DNS solo en la interfaz del
# túnel; se borra solo cuando la interfaz desaparece. Si quedó colgado:
resolvectl revert tun0        # el nombre de la interfaz del túnel

# Linux sin systemd-resolved
sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf

# macOS
networksetup -setdnsservers Wi-Fi empty
```

<!-- feature:killswitch feature:dns_guard -->
### Linux: se cae la VPN y no abre ningún sitio

Desde esta versión, cuando una VPN se cae sola Kongtrol devuelve el DNS del sistema (sus
servidores DNS solo se alcanzan por el túnel caído) y avisa qué bloquea el kill switch:

- **«…bloquea solo sus destinos»**: el resto de internet sigue normal.
- **«…llevaba todo tu tráfico»**: el perfil es de túnel completo (en WireGuard,
  `AllowedIPs = 0.0.0.0/0`). Con el kill switch activo, internet queda bloqueado hasta que vuelva;
  eso es lo que pide un túnel completo. Para navegar ya: pulsa «Recuperar internet» en el aviso (lo
  desconecta; o
  `kongtrol down <perfil>`). Si solo necesitas algunos sitios por esa VPN, agrega reglas
  (Agregar → «Agregar regla de salida»): con reglas, Kongtrol lleva por el túnel solo
  esos destinos y lo demás no depende de él.
- **«…bloquea internet y tu red local (impresoras, NAS)»**: igual que el anterior, con
  `allow_lan` apagado. Mientras el túnel completo está caído también se cortan los equipos de tu
  red (impresora, NAS, el router), porque el router suele ser el DNS al que el sistema vuelve y
  por ahí saldrían tus búsquedas fuera de la VPN. Para seguir usándolos: panel → Seguridad →
  «Permitir la red local», o `allow_lan: true` en `security.kill_switch` (entonces las búsquedas
  pueden salir por el DNS de tu red mientras la VPN no vuelve).
- **«…se cayó mientras <otra VPN> lleva todo tu tráfico · el kill switch no bloquea»**: otra VPN
  de túnel completo tomó la salida a internet (otro perfil de Kongtrol, o Proton VPN, NordVPN o
  Surfshark conectada en su propia app). Bloquear todo cortaría también esa VPN, así que Kongtrol
  no bloquea y tu tráfico sigue por ella. Para volver al perfil caído: desconecta la otra VPN y
  conéctalo.

Dos VPN que llevan todo tu tráfico a la vez se cortan entre sí (WireGuard para Windows, con
`0.0.0.0/0`, bloquea todo lo que no pase por su túnel). Por eso Kongtrol no conecta un perfil de
túnel completo (Proton VPN, NordVPN, Surfshark, WireGuard con `default_egress`, OpenVPN con
`redirect-gateway`) mientras otro está conectado: el panel, la bandeja y `kongtrol up` dicen cuál
desconectar primero. Un perfil con reglas (túnel parcial) sí puede convivir con uno completo.

**Archivo de Proton junto a la app de Proton** (ADR-0103). Un perfil `wireguard` u `openvpn` hecho
con el archivo que descargas de tu cuenta de Proton usa tu misma cuenta y, en WireGuard, la misma
dirección interna (`10.2.0.2`) que la app: a la vez chocan, aunque el perfil tenga reglas. Kongtrol
lo reconoce por el archivo (no hay que marcar nada) y:

- al importarlo avisa: «Este archivo es de tu cuenta de Proton. No lo conectes mientras la app de
  Proton esté conectada: chocan.»;
- no lo conecta mientras Proton está conectado en su app (dice que desconectes Proton en su app);
- si conectas la app con el perfil arriba, Kongtrol desconecta el perfil y te avisa una vez; pulsa
  Reintentar cuando desconectes Proton en su app. Sin un perfil de Proton VPN en Kongtrol, el aviso
  ofrece agregarlo para ver la app aquí.

Dos perfiles directos con la misma dirección interna (dos archivos de Proton con `10.2.0.2`) tampoco
se conectan a la vez. Para usar dos, cambia la de uno en su archivo: `Address = 10.3.0.2/32` y
`DNS = 10.3.0.1`, como indica Proton. Si un perfil que no es de Proton deja de responder mientras
Proton está conectado en su app, revisa el «kill switch avanzado» de Proton: puede bloquear otras
VPN (desactívalo o agrega ese perfil en «Split tunneling»; no verificado).

WireGuard con línea `DNS =`: `wg-quick` manda **todas** las búsquedas al DNS del túnel mientras la
interfaz exista (en systemd-resolved, `resolvconf -x` equivale a `~.`). Si el servidor deja de
responder, nada resuelve hasta que Kongtrol lo reconecta. Si no necesitas nombres internos de esa
red, quita la línea `DNS =` del archivo.

### "unknown profile" al usar --group

Verifica que el nombre del grupo esté en `~/.kongtrol/kongtrol.yaml` bajo `groups:` y que el config sea válido:

```bash
kongtrol config validate
```

### Kill switch activo y sin internet después de desconectar

```bash
kongtrol down --all    # desactiva kill switch automáticamente
```

Si Kongtrol se cerró abruptamente: con el ayudante instalado, sus bloqueos se quitan solos en unos
20 segundos (la red queda abierta, nunca cortada), y el ayudante también los borra cada vez que
arranca (por ejemplo, al reiniciar el equipo). Si siguen ahí, quita **solo** las reglas de Kongtrol.
No uses `netsh advfirewall reset`, `pfctl -d` ni `iptables -F` sin cadena: borran o apagan las
reglas de todo el equipo, no solo las de Kongtrol.

```bash
# Windows (PowerShell como Administrador): solo las reglas con nombre Kongtrol*
netsh advfirewall firewall delete rule name=KongtrolBlock
netsh advfirewall firewall delete rule name=KongtrolKillSwitchBlock
netsh advfirewall firewall delete rule name=KongtrolKillSwitchAllow
netsh advfirewall firewall delete rule name=KongtrolKillSwitchEndpoint
netsh advfirewall firewall delete rule name=KongtrolKillSwitchLAN
netsh advfirewall firewall delete rule name=KongtrolKillSwitchLoopback

# Linux: solo las cadenas de Kongtrol (si -X falla, repite el -D: queda otro salto)
sudo iptables -D OUTPUT -j KONGTROL_BLOCK; sudo iptables -F KONGTROL_BLOCK; sudo iptables -X KONGTROL_BLOCK
sudo ip6tables -D OUTPUT -j KONGTROL_BLOCK; sudo ip6tables -F KONGTROL_BLOCK; sudo ip6tables -X KONGTROL_BLOCK
sudo iptables -D OUTPUT -j KONGTROL; sudo iptables -F KONGTROL; sudo iptables -X KONGTROL

# macOS: vacía solo el anchor de Kongtrol; pf sigue activo para lo demás
sudo pfctl -a com.kongtrol -F all
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
`install.sh`/`install.ps1`, o instala el `.deb`/`.rpm` nuevo y corre `kongtrol restart`). Tu
configuración se queda. El
instalador de Windows y los scripts detectan la versión que tienes y dicen "Actualizar" (o
"Reinstalar", o avisan si es una versión anterior) sin volver a preguntar lo que ya elegiste.
`kongtrol service install` con el ayudante ya al día dice "nada que hacer" y no pide permiso.

**Aviso de versión nueva (opcional, apagado):** `kongtrol config update-check on` o el panel
(Ajustes). Consulta api.github.com una vez al día como máximo; GitHub ve tu IP y
`kongtrol/<versión>`. Nunca descarga ni instala. `kongtrol config update-check status` muestra la
última revisión.

**Desinstalar (tus datos siguen en este equipo; `--purge` los borra):**

```bash
kongtrol uninstall --dry-run   # muestra qué haría, sin cambiar nada
kongtrol uninstall             # quita la app; guarda ~/.kongtrol y tus contraseñas
kongtrol uninstall --purge     # también borra tus datos y contraseñas (pregunta [s/N])
kongtrol uninstall --data-only # solo borra tus datos; la app queda instalada
```

- Windows: Configuración › Aplicaciones › Kongtrol › Desinstalar. Elige **Sí** para borrar la
  configuración y las contraseñas guardadas, o **No** para conservarlas. Si la limpieza falla,
  el desinstalador lo indica por separado. Desde el panel, **Ajustes › Borrar todos mis datos**
  permite empezar de cero con la aplicación instalada.
- Instalado con `install.sh`/`install.ps1`: `… | sh -s -- --uninstall [--purge]`, o
  `kongtrol uninstall`.
- `.deb`/`.rpm`: Kongtrol no toca esos archivos. Si quieres
  borrar tus datos, corre `kongtrol uninstall --purge` **antes** de quitar el paquete.
- Desde el panel: Ajustes › **Borrar todos mis datos**. Se eliminan los perfiles, ajustes y
  credenciales guardadas. Quedan los archivos que usan el servicio y la bandeja mientras están abiertos.
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
