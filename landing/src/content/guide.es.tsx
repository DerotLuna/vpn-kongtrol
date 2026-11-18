// Setup guide content — Spanish.
// Version 2.5.0 · 2026-11-18.
//
// Rendered by components/sections/Guide.tsx, which owns the markup; this file
// holds only the words. Keys and shapes must match guide.en.tsx exactly
// (enforced by the GuideContent type), so both languages always carry the
// same content.
import type { OS } from '../hooks/useOS'
import IC from '../components/ui/InlineCode'
import { pathFor } from '../site'

export default function guideES(os: OS) {
  return {
    intro1: <><span className="guide-intro-badge">Cómo funciona</span></>,
    intro2: <>Kongtrol no es un cliente VPN: es la capa que se sienta entre tus apps y los
            clientes VPN que ya tienes instalados (FortiClient, OpenVPN, ProtonVPN, etc.).
            El <strong>policy engine</strong> mira cada conexión saliente (dominio, IP o app) y
            decide por cuál túnel debe salir; el resto de comandos (<IC c="up" />, <IC c="doctor" />,
            <IC c="status" />) solo controlan ese enrutamiento y el ciclo de vida de los túneles.</>,
    intro3: <>tus apps</>,
    intro4: <>Chrome, Slack, terminal, código…</>,
    intro5: <>policy engine: dominios · IPs · apps</>,
    intro6: <>tus VPNs</>,
    intro7: <>kill switch · dns guard · audit log firmado</>,
    intro8: <>Contenido</>,
    intro9: "Ir a",
    quickstart1: <><span className="guide-quickstart-badge">Inicio rápido</span></>,
    // feature:first_run
    quickstart2: <>Kongtrol trabaja con las VPN que ya tienes. Todo se hace desde la app (bandeja y panel); la terminal es opcional y hace lo mismo. Las secciones de abajo son referencia: vuelve a ellas solo si algo no calza.</>,
    quickstart3: <><strong>Instala la app y abre el panel</strong></>,
    quickstart4: <>Usa el instalador de <a href="/#install">descargas</a>. Al abrirla aparece el ícono de Kongtrol en la bandeja y el panel se abre en tu navegador para crear tu primera VPN. La primera vez el sistema te pide permiso una sola vez para instalar el ayudante (un servicio que cambia rutas y DNS por ti).</>,
    quickstart5: <><strong>Conecta desde la bandeja o el panel</strong></>,
    quickstart6: <>El «daemon» que verás en la terminal es el proceso de Kongtrol que sigue funcionando en segundo plano: puedes cerrar el panel o la terminal y tus VPN siguen conectadas. Solo «Salir y desconectar todo» en la bandeja, o <IC c="kongtrol down" />, las desconecta.</>,
    quickstart7: <><strong>Desde la terminal (opcional)</strong></>,
    quickstart8: <><IC c="kongtrol init" /> te guía, revisa el ayudante (te ofrece instalarlo) y al final te pregunta si conectas ya. Después: estado en vivo y el panel.</>,
    // feature:providers
    prereqs1: <>Prerequisitos</>,
    prereqs2: <>Kongtrol <strong>orquesta</strong> los clientes VPN existentes, no los reemplaza.
                Deben estar instalados antes de correr <IC c="kongtrol init" />.</>,
    prereqs3: 'Cliente requerido',
    prereqs4: 'Verificar',
    prereqs5: <>Abrir FortiClient: debe aparecer al menos una conexión guardada en la lista.
                      Si puedes conectarte manualmente desde la GUI, está listo.</>,
    prereqs6: <>{os === 'windows'
                        ? <>CLI o GUI según instalador. Community: <IC c="openvpn --version" /> en terminal (agrega al PATH). OpenVPN Connect: solo GUI. Kongtrol lo detecta automáticamente.</>
                        : <IC c="openvpn --version" />}</>,
    prereqs7: 'App oficial de ProtonVPN (o WireGuard con su .conf)',
    prereqs8: <>{os === 'linux'
                        ? <>Inicia sesión en la app. En Linux Kongtrol puede conectar por ti; si no, conectas tú y lo detecta.</>
                        : <>Inicia sesión en la app. Conectas tú y Kongtrol detecta el túnel solo.</>}
                      {' '}Sin app: perfil <IC c="wireguard" /> con el <IC c=".conf" /> de Proton.</>,
    prereqs9: <><strong>Nota:</strong> si el wizard muestra ✓ para un cliente, está detectado y listo: no necesitas verificar manualmente en terminal.</>,
    prereqs10: <><strong>No cierres la app de tu VPN:</strong> Kongtrol trabaja con ella.
                Si el túnel ya está arriba, se engancha a él. Si no, abre la app y conecta por ti
                donde se puede; si no puede, te dice qué hacer y espera a que conectes.</>,
    prereqs11: <>Comando</>,
    prereqs12: <>Estado requerido de los clientes</>,
    prereqs13: <>Cualquiera: solo lee archivos y keychain</>,
    prereqs14: <>Cualquiera: solo valida, no conecta</>,
    prereqs15: <>kongtrol up &lt;perfil&gt;</>,
    prereqs16: <>App instalada; abierta o cerrada da igual (Kongtrol la abre si hace falta)</>,
    // feature:helper
    install1: <>Instalación</>,
    install2: <>1. Descarga <IC c="kongtrol_windows_amd64.zip" /> desde{' '}
                    <a href="/#install">la sección de descargas</a>.</>,
    install3: <>2. Extrae el ZIP. Verás un archivo <IC c="kongtrol.exe" />.
                    Muévelo a una carpeta fija; por ejemplo, crea{' '}
                    <IC c="C:\tools\" /> si no tienes una:</>,
    install4: `# Crea la carpeta (si no existe) y mueve el binario
New-Item -ItemType Directory -Force "C:\\tools"
Move-Item kongtrol.exe C:\\tools\\kongtrol.exe`,
    install5: <>3. Agrega <IC c="C:\tools" /> a tu PATH de usuario (no hace falta ser administrador) para
                    correr <IC c="kongtrol" /> desde cualquier terminal:</>,
    install6: `# En PowerShell normal, una sola vez:
[Environment]::SetEnvironmentVariable(
  "Path",
  [Environment]::GetEnvironmentVariable("Path", "User") + ";C:\\tools",
  "User"
)`,
    install7: <>4. Abre una terminal <strong>nueva</strong> y verifica:</>,
    install8: <><strong>¿Por qué una terminal nueva?</strong> Los cambios al PATH solo
                    aplican en terminales abiertas después del cambio; las que ya estaban
                    abiertas no lo ven.</>,
    install9: `# Apple Silicon (M1+)
$ curl -L https://github.com/DerotLuna/vpn-kongtrol/releases/latest/download/kongtrol_darwin_arm64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Intel
$ curl -L https://github.com/DerotLuna/vpn-kongtrol/releases/latest/download/kongtrol_darwin_amd64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Verificar
$ kongtrol --help`,
    install10: <>macOS puede pedir confirmación la primera vez que corres el binario.
                    Ve a <strong>Ajustes del sistema → Privacidad y seguridad</strong> y haz clic en "Permitir".</>,
    install11: `$ curl -L https://github.com/DerotLuna/vpn-kongtrol/releases/latest/download/kongtrol_linux_amd64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Verificar
$ kongtrol --help`,
    install12: <><strong>Un solo permiso, una sola vez:</strong> el instalador (o <IC c="kongtrol init" />, o{' '}
                    <IC c="kongtrol service install" />) instala un ayudante que hace los cambios de red (rutas, DNS, kill switch)
                    y el sistema te pide permiso una vez. Después ningún comando necesita una terminal de administrador ni <IC c="sudo" />.</>,
    // feature:wizard
    wizard1: <>Registrar tus VPNs: kongtrol init</>,
    wizard2: <><IC c="kongtrol init" /> <strong>no configura tus VPNs</strong>: eso ya lo
                tienes hecho. Lo que hace es decirle a Kongtrol dónde están y cómo hablarles:
                qué cliente usar, qué tunnel name tiene, dónde está el <IC c=".ovpn" />,
                y guarda las credenciales en el <strong>keychain del OS</strong> (nunca en el YAML).</>,
    wizard3: <>Si ya tienes FortiClient con un túnel funcionando y OpenVPN con su{' '}
                <IC c=".ovpn" />, este paso solo recopila esa información.</>,
    wizard4: <>Archivos de conexión</>,
    wizard5: <><strong>Si ya tienes tus VPNs configuradas, no necesitas hacer nada especial aquí.</strong>{' '}
                Kongtrol apunta a los archivos donde ya están. En el wizard simplemente
                escribes la ruta actual de cada archivo. Solo hace falta actuar si recibes
                archivos nuevos de un compañero o de IT y no tienes dónde guardarlos, o si
                quieres reorganizar todo en un directorio único.</>,
    wizard6: 'Adaptador',
    wizard7: 'Qué necesita el wizard',
    wizard8: '¿Tienes que hacer algo?',
    wizard9: os === 'windows' ? 'Nombre del túnel + usuario/contraseña' : 'Ruta al cert, ruta a la key, credenciales',
    wizard10: <>{os === 'windows'
                        ? <>No. El túnel ya está en la GUI de FortiClient. Kongtrol lo activa por nombre. Sin certs.</>
                        : <>Tener el <IC c=".crt" /> y <IC c=".key" /> en alguna ruta accesible y anotarla.</>}</>,
    wizard11: <>Ruta al <IC c=".ovpn" /></>,
    wizard12: <>{os === 'windows'
                        ? <>No. Usa el <IC c=".ovpn" /> donde ya está (ej. <IC c="C:\Users\TU\OpenVPN\config\" />).</>
                        : <>No, si ya tienes el <IC c=".ovpn" />. Solo anotar su ruta actual.</>}</>,
    wizard13: <>Modo GUI: credenciales Proton. Modo automático: archivo <IC c=".conf" /> de WireGuard.</>,
    wizard14: <>GUI: no archivos. WireGuard: exportar config desde Proton y guardar la ruta.</>,
    wizard15: <>WireGuard / otros</>,
    wizard16: <>Ruta al archivo de config</>,
    wizard17: <>No, si ya tienes el archivo. Solo anotar su ruta.</>,
    wizard18: <><strong>Organización opcional:</strong> si quieres centralizar, puedes crear{' '}
                <IC c={os === 'windows' ? '%USERPROFILE%\\.kongtrol\\certs\\' : '~/.kongtrol/certs/'} />{' '}
                y copiar los archivos ahí. Útil al incorporar un compañero nuevo o al recibir
                certs frescos. No es requerido si ya tienes todo en su lugar.</>,
    wizard19: <>Ejecutar el wizard</>,
    wizard20: <>Primero seleccionas el idioma, luego el wizard muestra los clientes detectados
                y te pregunta si quieres agregar un perfil:</>,
    wizard21: `▸  Clientes VPN detectados en este sistema:
    ✓  FortiClient             6.4.10.1821
    ✓  OpenVPN                 2.6.8
    ✓  ProtonVPN               4.3.14

¿Agregar un nuevo perfil VPN? [s/N]: s`,
    wizard22: <>El wizard usa <strong>menús numerados</strong> donde aplica, así no tienes que
                recordar los valores válidos:</>,
    wizard23: `  Nombre del perfil: office

  Tipo de adaptador:
    1) forticlient         ✓ detectado  FortiClient SSL VPN
    2) openvpn             ✓ detectado  OpenVPN (archivo .ovpn)
    3) protonvpn           ✓ detectado  ProtonVPN (cuenta Proton)
    ───────────────────────────────
    4) ciscoanyconnect                  Cisco AnyConnect
    5) wireguard                        WireGuard (archivo .conf)
    6) globalprotect                    Palo Alto GlobalProtect
    7) tailscale                        Tailscale mesh / exit node
    8) cloudflarewarp                   Cloudflare WARP

  Elige [1]: 1`,
    wizard24: <>Cada campo tiene una pista de dónde encontrar el valor:</>,
    wizard25: <>Ver ejemplo del flujo por campos (FortiClient)</>,
    wizard26: os === 'windows'
? `    Encuéntralo en FortiClient > Ajustes o pídelo a IT. Ej: vpn.empresa.com
  Host VPN: vpn.tuempresa.com

    443 para SSL VPN (default). Cambia solo si IT indica otro puerto.
  Puerto [443]:

    El nombre exacto de la conexión tal como aparece en la lista de FortiClient GUI.
  Nombre del túnel [Office]: Office

  Versión de FortiClient:
    1) 6  6.4.x — más común
    2) 7  7.x
    3) 5  5.x — legacy
  Elige [1]:

    En Windows FortiClient conecta por nombre de túnel — sin certificado. Se usará 'credentials'.
  Usuario: tu_usuario
  Contraseña para office (guardada en llavero): ****`
: `    Encuéntralo en FortiClient > Ajustes o pídelo a IT. Ej: vpn.empresa.com
  Host VPN: vpn.tuempresa.com

    443 para SSL VPN (default). Cambia solo si IT indica otro puerto.
  Puerto [443]:

    El nombre exacto de la conexión tal como aparece en la lista de FortiClient GUI.
  Nombre del túnel [Office]: Office

  Versión de FortiClient:
    1) 6  6.4.x — más común
    2) 7  7.x
    3) 5  5.x — legacy
  Elige [1]:

  Método de autenticación:
    1) credentials              usuario y contraseña
    2) certificate              solo certificado cliente
    3) certificate+credentials  certificado + usuario y contraseña (más seguro)
  Elige [3]:

  Ruta al certificado cliente: ~/.kongtrol/certs/office.crt
  Ruta a la clave privada: ~/.kongtrol/certs/office.key
  Usuario: tu_usuario
  Contraseña para office (guardada en llavero): ****`,
    wizard27: <>Para OpenVPN, el wizard apunta a tu archivo <IC c=".ovpn" /> existente:</>,
    wizard28: <>Ver ejemplo del flujo por campos (OpenVPN)</>,
    wizard29: os === 'windows'
? `  Nombre del perfil: dev-server

  Tipo de adaptador:
    ...
  Elige [2]: 2   ← openvpn

    Ruta completa al .ovpn. Usa la ubicación donde ya está, no hace falta copiar.
  Ruta al archivo .ovpn: C:\\Users\\TU\\OpenVPN\\config\\server.ovpn

    Si el .ovpn tiene <cert> y <key> embebidas, elige 'certificate' y deja los paths en blanco.
  Método de autenticación:
    1) credentials              usuario y contraseña
    2) certificate              solo certificado cliente
    3) certificate+credentials  certificado + usuario y contraseña (más seguro)
  Elige [2]:

  Ruta al certificado cliente (vacío si está dentro del .ovpn):
  Ruta a la clave privada (vacío si está dentro del .ovpn):`
: `  Nombre del perfil: dev-server

  Tipo de adaptador:
    ...
  Elige [2]: 2   ← openvpn

    Ruta completa al .ovpn. Usa la ubicación donde ya está, no hace falta copiar.
  Ruta al archivo .ovpn: ~/.config/openvpn/server.ovpn

    Si el .ovpn tiene <cert> y <key> embebidas, elige 'certificate' y deja los paths en blanco.
  Método de autenticación:
    1) credentials              usuario y contraseña
    2) certificate              solo certificado cliente
    3) certificate+credentials  certificado + usuario y contraseña (más seguro)
  Elige [2]:

  Ruta al certificado cliente (vacío si está dentro del .ovpn):
  Ruta a la clave privada (vacío si está dentro del .ovpn):`,
    wizard30: <>ProtonVPN: GUI manual vs WireGuard automático</>,
    wizard31: <>Tienes dos caminos válidos con ProtonVPN según cuánto quieras automatizar.</>,
    wizard32: <><strong>GUI manual (rápido):</strong> el usuario opera Proton desde su app y Kongtrol
                  enruta políticas sobre ese perfil.</>,
    wizard33: <><strong>WireGuard automático (recomendado):</strong> exporta un archivo <IC c=".conf" /> desde Proton,
                  crea un perfil <IC c="type: wireguard" /> y Kongtrol puede levantar/bajar la conexión por comando.</>,
    wizard34: <><strong>ProtonVPN junto a otros adapters (Forti/OpenVPN/etc.):</strong> úsalo en un
                perfil separado y enruta solo dominios/IPs objetivo (ej. streaming). Mantén
                tráfico corporativo en sus perfiles dedicados para evitar conflictos de ruta.</>,
    wizard35: <><strong>¿Dónde encuentro el Host y el Tunnel Name de FortiClient?</strong><br />
                Abre FortiClient → pestaña <strong>Remote Access</strong>. Verás una lista de conexiones guardadas.
                El <strong>nombre de la conexión</strong> (ej. "Office VPN") es el <em>tunnel name</em>.
                Haz clic en el ícono de lápiz (editar) junto a esa conexión; ahí verás el campo{' '}
                <strong>Remote Gateway</strong> o <strong>Server</strong>: ese es el <em>host</em>.</>,
    wizard36: <>Si un cliente no fue detectado automáticamente, el wizard te avisa y pide
                la ruta al binario. También puedes ponerla después en el YAML con <IC c="binary_path" />.</>,
    wizard37: <>Al final, opciones de seguridad, políticas de routing y confirmación:</>,
    wizard38: <>Ver flujo de seguridad y políticas</>,
    wizard39: `¿Activar kill switch? [S/n]:
¿Activar DNS guard? [S/n]:
¿Activar log de auditoría firmado? [S/n]:
¿Activar panel web? (http://127.0.0.1:9741) [S/n]:
¿Mostrar tu IP pública? (consulta un servicio externo) [s/N]:
  Usa un servicio externo (api.ipify.org) que ve tu IP pública

── Políticas de enrutamiento ──────────────────────────────────────────────

¿Agregar una política de enrutamiento? [S/n]: s

  Nombre de la política (ej: trabajo, streaming): Claude AI
  Perfil VPN para esta política:
    1) office
    2) us-content
  Elige [1]: 2

    Dominio o sufijo (ej: empresa.com, .internal) — deja en blanco para terminar
  Dominio: claude.ai
  Dominio: *.anthropic.com
  Dominio:

    Rango IP o IP (ej: 10.0.0.0/8) — deja en blanco para terminar
  IP / rango:

¿Agregar una política de enrutamiento? [S/n]: s

  Nombre de la política (ej: trabajo, streaming): Red interna
  Perfil VPN para esta política:
    1) office
    2) us-content
  Elige [1]: 1

  Dominio:

  IP / rango: 10.10.0.0/16
  IP / rango: 192.168.50.0/24
  IP / rango:

¿Agregar una política de enrutamiento? [S/n]:

    También puedes editar políticas directamente en kongtrol.yaml → sección 'policies'.

¿Escribir configuración en ~/.kongtrol/kongtrol.yaml? [S/n]: s

[✓] Configuración escrita en ~/.kongtrol/kongtrol.yaml
[✓] Configuración válida.`,
    wizard40: <><strong>Puedes escribir varios dominios separados por coma</strong> en un solo prompt:
                el wizard los divide automáticamente. <IC c="claude.ai, *.anthropic.com" /> → dos entradas.</>,
    wizard41: <>Al terminar, el wizard revisa el ayudante (si falta, te ofrece instalarlo) y te pregunta{' '}
                «¿Conectar ahora?»: con Enter conecta lo que acabas de crear. Para agregar otro perfil o política después:
                vuelve a correr <IC c="kongtrol init" />. El wizard detecta el config existente y no sobreescribe nada sin confirmación.</>,
    groups1: <>Grupos</>,
    groups2: <>Agrupan perfiles para levantarlos juntos con un solo comando.
                Edita <IC c={os === 'windows' ? 'C:\\Users\\TU\\.kongtrol\\kongtrol.yaml' : '~/.kongtrol/kongtrol.yaml'} /> y añade al final:</>,
    groups3: `kongtrol up --group work     # levanta office + dev-server + aws en paralelo
kongtrol down --group work`,
    groups4: <>Un perfil puede pertenecer a varios grupos a la vez: <IC c="office" /> puede
                estar en <IC c="work" /> y también en <IC c="full" /> sin conflicto.</>,
    groups5: <>Si Kongtrol ya está funcionando en segundo plano y editas el grupo a mano, no hace falta
                reiniciarlo: <IC c="kongtrol reload --group work" /> recarga el <IC c="kongtrol.yaml" /> desde disco y
                reconecta solo los túneles activos de ese grupo. Para un solo túnel, <IC c="kongtrol reload office" />;
                sin argumentos, <IC c="kongtrol reload" /> aplica todo y, si agregaste perfiles, te ofrece reiniciar Kongtrol.</>,
    // feature:policies feature:routes
    policies1: <>Políticas de routing</>,
    policies2: <>Cuando tienes varios túneles activos al mismo tiempo, ¿cómo sabe tu computadora
                qué tráfico va por cuál VPN? Eso lo definen las <strong>políticas</strong>.</>,
    policies3: `  petición                    política                          túnel
  ──────────────────────────────────────────────────────────────────────────
  claude.ai            ──▶   domains: claude.ai            ──▶  us-content (ProtonVPN)
  10.10.4.12           ──▶   ip_ranges: 10.10.0.0/16        ──▶  office     (FortiClient)
  api.tuempresa.com    ──▶   sin coincidencia               ──▶  conexión directa (sin VPN)`,
    policies4: <>Sin políticas, cada VPN instala sus propias rutas al conectarse, y el último
                en conectar suele ganar el tráfico general, y no puedes kongtrolar qué va por dónde.
                Con políticas, Kongtrol enruta de forma precisa:</>,
    policies5: <><IC c="claude.ai" /> y <IC c="*.anthropic.com" /> → van por <IC c="us-content" /> (ProtonVPN)</>,
    policies6: <><IC c="10.10.0.0/16" /> (red de oficina) → van por <IC c="office" /> (FortiClient)</>,
    policies7: <><IC c="*.tuempresa.com" /> → van por <IC c="office" /></>,
    policies8: <>Todo lo demás → conexión directa sin VPN</>,
    policies9: <>El wizard (<IC c="kongtrol init" />) te permite crear políticas al final del flujo.
                También puedes agregarlas o editarlas directamente en el YAML bajo la sección <IC c="policies:" />:</>,
    policies10: `policies:

  # ── Contenido US / Claude AI ───────────────────────────────────────────
  - name: Claude AI
    match:
      domains:
        - claude.ai
        - "*.anthropic.com"
    via: us-content

  - name: Contenido geo-restringido
    match:
      domains:
        - "*.netflix.com"
        - "*.hulu.com"
        - "*.disneyplus.com"
    via: us-content

  # ── Oficina ────────────────────────────────────────────────────────────
  - name: Red interna de oficina
    match:
      ip_ranges:
        - 10.10.0.0/16          # red interna — ajusta a la tuya
        - 192.168.50.0/24
    via: office

  - name: Dominios internos
    match:
      domains:
        - "*.tuempresa.com"     # cambia al dominio de tu empresa
        - intranet.local
    via: office

  # ── Servidor de desarrollo ─────────────────────────────────────────────
  - name: Servidor dev
    match:
      ip_ranges:
        - 185.0.0.0/32          # reemplaza con la IP real de tu servidor
    via: dev-server

  # ── AWS ────────────────────────────────────────────────────────────────
  - name: AWS workloads
    match:
      ip_ranges:
        - 172.31.0.0/16         # VPC default de AWS — ajusta a tus CIDRs
      domains:
        - "*.amazonaws.com"
    via: aws

  # El tráfico que no coincide con ninguna regla va por tu
  # conexión normal (sin VPN). Para forzar TODO por una VPN:
  # - name: Default
  #   match:
  #     ip_ranges: [0.0.0.0/0]
  #   via: office`,
    policies11: <><strong>¿Cómo funciona el matching?</strong><br />
                Dominios: glob exacto o con prefijo <IC c="*." />. <IC c="*.empresa.com" /> cubre todos los subdominios pero no <IC c="empresa.com" /> solo (agrega ambos si los necesitas).<br />
                IPs: longest-prefix match: gana la regla más específica. <IC c="10.10.1.0/24" /> tiene prioridad sobre <IC c="10.0.0.0/8" />.</>,
    policies12: <><strong>Reglas por app (experimental):</strong> puedes escribir <IC c="apps" /> en una regla,
                pero hoy no cambia el ruteo: solo cuentan <IC c="domains" /> e <IC c="ip_ranges" />.
                Por ahora <IC c="apps" /> solo se ve en <IC c="kongtrol policy show app:<exe>" />.</>,
    policies13: <><strong>Split-DNS transparente:</strong> con <IC c="monitor.split_dns.enabled: true" />,
                Kongtrol inyecta dominios de policy en resolución del sistema (hosts) para que apps
                normales resuelvan por el túnel correcto sin llamar la API.</>,
    policies14: `# Validar después de editar
$ kongtrol config validate
# [OK] Config is valid.`,
    policies15: <>Si Kongtrol todavía no está funcionando, las políticas se leen al ejecutar <IC c="kongtrol up" />.
                Si ya está activo, aplica el cambio sin reiniciarlo con <IC c="kongtrol reload --policy" />:
                recarga el motor de políticas desde el <IC c="kongtrol.yaml" /> editado, sin tocar túneles.
                Para ver qué VPN usa un destino: <IC c="kongtrol policy show github.com" />.</>,
    doctor1: <>Verificar con doctor</>,
    doctor2: <>Antes de conectar por primera vez, deja que Kongtrol valide que todo esté en orden:</>,
    doctor3: os === 'windows'
? `Kongtrol Doctor
────────────────────────────────────────────────────

  Configuración
  ✓  archivo de config   C:\\Users\\tu\\.kongtrol\\kongtrol.yaml
  ✓  config válida       4 perfil(es) definido(s)

  Binarios VPN
  ✓  forticlient         FortiClient 6.4.10.1821
  ✓  openvpn             OpenVPN 2.6.8
  ✓  protonvpn           ProtonVPN 4.3.14

  Certificados y claves
  ✓  office: certificado C:\\Users\\tu\\.kongtrol\\certs\\office.crt
  ✓  office: clave       C:\\Users\\tu\\.kongtrol\\certs\\office.key
  ✓  dev-server: archivo config  C:\\Users\\tu\\.kongtrol\\configs\\server.ovpn

  Credenciales en keychain
  ✓  office: password    encontrada en keychain del sistema (Credential Manager)
  ✓  us-content: password  encontrada en keychain del sistema (Credential Manager)

  Permisos
  ✓  kill switch         implementación de plataforma disponible
  ✓  DNS guard           implementación de plataforma disponible

Todos los checks pasaron. Listo para usar.`
: `Kongtrol Doctor
────────────────────────────────────────────────────

  Configuración
  ✓  archivo de config   ~/.kongtrol/kongtrol.yaml
  ✓  config válida       4 perfil(es) definido(s)

  Binarios VPN
  ✓  forticlient         FortiClient 6.4.10.1821
  ✓  openvpn             OpenVPN 2.6.8
  ✓  protonvpn           protonvpn-cli 4.3.14

  Certificados y claves
  ✓  office: certificado ~/.kongtrol/certs/office.crt
  ✓  office: clave       ~/.kongtrol/certs/office.key
  ✓  dev-server: archivo config  ~/.kongtrol/configs/server.ovpn

  Credenciales en keychain
  ✓  office: password    encontrada en keychain del sistema
  ✓  us-content: password  encontrada en keychain del sistema

  Permisos
  ✓  kill switch         implementación de plataforma disponible
  ✓  DNS guard           implementación de plataforma disponible

Todos los checks pasaron. Listo para usar.`,
    doctor4: <>Si hay ✗ o ! en alguna línea, debajo aparece qué hacer. Sin terminal: en el panel, la tarjeta <strong>Diagnóstico</strong> muestra lo mismo, ofrece <strong>Reparar</strong> si falta el ayudante y <strong>Exportar configuración</strong> (sin contraseñas).</>,
    // feature:daemon_lifecycle
    connect1: <>Primera conexión</>,
    connect2: <><strong>Un grupo completo:</strong></>,
    connect3: `$ kongtrol up --group work
[+] office      conectado
[+] dev-server  conectado
[+] aws         conectado`,
    connect4: <><strong>Varios perfiles sin grupo:</strong></>,
    connect5: <>Los tres túneles se levantan en paralelo. El routing policy determina qué tráfico va por cuál.</>,
    connect6: <><strong>Un solo perfil:</strong></>,
    connect7: <>Kongtrol sigue corriendo en segundo plano: cerrar la vista no desconecta nada. Cuando termines:</>,
    connect8: `# q o Ctrl+C  →  cierra la vista; las VPN siguen conectadas

# Para desconectar (desde cualquier terminal):
$ kongtrol down --group work
$ kongtrol down --all`,
    connect9: <><strong>Ver estado:</strong></>,
    connect10: `$ kongtrol status

PROFILE      STATUS        IP           UPTIME
office       connected     10.10.0.5    1h 23m
dev-server   connected     185.x.x.x    1h 23m
aws          connected     172.31.4.7   1h 23m
us-content   disconnected  —            —

Kill switch: ARMADO (protegiendo 3 túnel(es))

# Vista en tiempo real (refresca cada 2s):
$ kongtrol status --watch`,
    // feature:dashboard
    dashboard1: <>Dashboard web</>,
    dashboard2: <>Abre <a href="http://localhost:9741" target="_blank" rel="noreferrer">http://localhost:9741</a> en
                tu navegador. Es una consola de gestión completa, con navegación lateral:</>,
    dashboard3: <>Página</>,
    dashboard4: <>Qué hace</>,
    dashboard5: <>Túneles en vivo, gráficos de tráfico por túnel, rutas activas, resolución de policy, grupos de acceso rápido, conectar/desconectar todo</>,
    dashboard6: <>Pestañas: políticas de routing (CRUD + probador de reglas), perfiles VPN (CRUD, config + keychain), y grupos (CRUD + conectar/desconectar)</>,
    dashboard7: <>Activar/desactivar Kill Switch y DNS Guard en vivo, override de kill switch por perfil, estado de leak check, consulta de IP pública opcional (apagada por defecto) con botón Revisar</>,
    dashboard8: <>Eventos de auditoría filtrables por perfil/nivel, con indicador de firma HMAC</>,
    dashboard9: <>Health check, scheduler + reglas, split DNS, ajustes de kill switch/DNS guard, servicios de IP pública, audit log</>,
    dashboard10: `# Endpoints útiles
GET  /api/v1/metrics/history
GET  /api/v1/dns/resolve?domain=claude.ai&via=us-content
GET  /api/v1/resolve?target=portal.empresa.com&app=chrome.exe
POST /api/v1/security/killswitch   {"enabled": true}
POST /api/v1/security/dnsguard     {"enabled": true}
POST /api/v1/security/publicip     {"enabled": true}   # opcional: consulta tu IP pública
POST /api/v1/security/publicip/check                    # Revisar ahora
GET  /api/v1/vpns                  # perfiles · POST/PUT/DELETE para CRUD
GET  /api/v1/groups                # grupos · POST/PUT/DELETE + /connect /disconnect
GET  /api/v1/settings              # PUT para guardar
GET  /api/v1/audit                 # ?profile=&level=&limit=`,
    dashboard11: <>El dashboard está compilado dentro del binario: no necesitas instalar nada
                extra. Es gestión completa (crear/editar perfiles, políticas, grupos, activar
                kill switch en vivo), no solo lectura. El puerto/bind del propio dashboard
                sigue siendo CLI-only (<code>kongtrol config dashboard set-port &lt;puerto&gt;</code>)
                porque cambiarlo desde la página que sirve la petición rompería la conexión.</>,
    // feature:i18n
    daily1: <>Uso diario</>,
    daily2: <>Comando</>,
    daily3: <>Qué hace</>,
    daily4: <>Empezar el día (office + dev-server + aws)</>,
    daily5: <>Terminar el día</>,
    daily6: <>Apagar todo</>,
    daily7: <>Ver qué está conectado</>,
    daily8: <>Monitoreo en vivo (2s)</>,
    daily9: <>Ver qué VPN usa un destino y por qué</>,
    daily10: <>Consultar tu IP pública y probar fugas ahora (si activaste la consulta)</>,
    daily11: <>Abrir UI web</>,
    daily12: <>Diagnóstico completo</>,
    daily13: <>Exportar config sin contraseñas (para compartir)</>,
    daily14: <><strong>Scheduler opcional por horario:</strong></>,
    daily15: <><strong>Kill switch por perfil (override):</strong></>,
    daily16: <><strong>Actualizar una contraseña:</strong></>,
    daily17: `$ kongtrol init
# → elige el perfil → "¿Actualizar credenciales?" → ingresa la nueva`,
    // Referencia de comandos: misma lista y orden que `kongtrol --help`
    // Las filas salen de docs/cli.json (generado desde la CLI: go generate ./cmd/kongtrol).
    cmdref1: <>Referencia de comandos</>,
    cmdref2: <>Los mismos grupos que <IC c="kongtrol --help" />. Cada comando trae su ayuda con ejemplos: <IC c="kongtrol <comando> --help" />.</>,
    // feature:killswitch feature:dns_guard
    trouble1: <>Solución de problemas</>,
    trouble2: <><strong>DNS no restaurado después de un crash:</strong></>,
    trouble3: <><strong>Kill switch activo sin internet después de desconectar:</strong></>,
    trouble4: `$ kongtrol down --all    # desactiva kill switch automáticamente`,
    trouble5: <>Si Kongtrol se cerró de golpe: con el ayudante instalado, sus bloqueos se quitan solos en unos
                20 segundos (la red queda abierta, nunca cortada). Si siguen ahí, quita <strong>solo</strong> las reglas de Kongtrol.
                No uses <IC c="netsh advfirewall reset" />: borra todas tus reglas de firewall.</>,
    trouble6: os === 'windows'
? `# PowerShell como Administrador:
> netsh advfirewall firewall delete rule name=KongtrolBlock
> netsh advfirewall firewall delete rule name=KongtrolKillSwitchBlock`
: os === 'macos'
? `$ sudo pfctl -a com.kongtrol -F all`
: `$ sudo iptables -F KONGTROL_BLOCK
$ sudo ip6tables -F KONGTROL_BLOCK
$ sudo iptables -F KONGTROL`,
    trouble7: <><strong>El cliente VPN no aparece en la detección automática:</strong></>,
    trouble8: os === 'windows'
? `# El wizard pregunta la ruta al binario si no lo detecta:
> kongtrol init
#   !  Este cliente VPN no fue detectado automáticamente.
#   Ruta al binario (vacío = autodetectar): C:\\ruta\\personalizada\\FortiClient.exe

# O agrégalo manualmente en %USERPROFILE%\\.kongtrol\\kongtrol.yaml:
# office:
#   type: forticlient
#   app_path: "C:\\\\ruta\\\\personalizada\\\\FortiClient.exe"`
: `# El wizard pregunta la ruta al binario si no lo detecta:
$ kongtrol init
#   !  Este cliente VPN no fue detectado automáticamente.
#   Ruta al binario (vacío = autodetectar): /ruta/al/binario

# O agrégalo manualmente en ~/.kongtrol/kongtrol.yaml:
# office:
#   type: forticlient
#   app_path: "/ruta/personalizada/FortiClient"`,
    trouble9: <><strong>Contraseña incorrecta / expirada:</strong></>,
    trouble10: `$ kongtrol init
# Selecciona el perfil → "Refresh credentials" → ingresa la nueva`,
    trouble11: <>Si detecta fallo de sesión/token/credenciales durante connect/reconnect,
                Kongtrol emite alerta de <strong>reauth requerida</strong> con hint específico por adapter.</>,
    trouble12: <><strong>Estructura de archivos:</strong></>,
    trouble13: `~/.kongtrol/
├── kongtrol.yaml     ← config principal (sin contraseñas)
├── certs/
│   ├── office.crt
│   └── office.key
├── configs/
│   ├── server.ovpn
│   └── aws.ovpn
└── audit.log         ← log firmado de todos los eventos

Contraseñas guardadas en:
  Windows → Windows Credential Manager
  macOS   → Keychain
  Linux   → libsecret / GNOME Keyring`,
    // feature:update_check
    update1: <>Actualizar</>,
    update2: <>Actualiza por el mismo camino por el que instalaste. Tu configuración y tus contraseñas se quedan.</>,
    update3: <><strong>Instalador de Windows:</strong> descarga el nuevo desde{' '}
                <a href="/#install">la sección de descargas</a> y ábrelo.</>,
    update4: `# Script (Linux / macOS): vuelve a correrlo
$ curl -fsSL https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.sh | sh

# Script (Windows, PowerShell): vuelve a correrlo
> iwr https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.ps1 -useb | iex

# Homebrew
$ brew upgrade kongtrol

# npm
$ npm update -g kongtrol`,
    update5: <><strong>Aviso de versión nueva (opcional, apagado por defecto):</strong> si lo activas, Kongtrol
                mira como mucho una vez al día si hay versión nueva. Te avisa una vez por versión: en la bandeja,
                el panel, <IC c="kongtrol up" />, <IC c="kongtrol version" /> y <IC c="kongtrol status" />.
                Nunca descarga ni instala nada.</>,
    update6: `$ kongtrol config update-check on      # activar
$ kongtrol config update-check off     # apagar (no se hace ninguna petición)
$ kongtrol config update-check status  # ver cómo está`,
    update7: <>También en el panel: Ajustes › «Aviso de versión nueva». <IC c="kongtrol init" /> te lo
                pregunta una vez (por defecto No). Qué se envía y a quién: en la{' '}
                <a href={`${pathFor('privacidad', 'es')}#conexiones`}>política de privacidad</a>.</>,
    // feature:uninstall
    uninstall1: <>Desinstalar</>,
    uninstall2: <>Al desinstalar, <strong>tus datos se quedan</strong>: la configuración (<IC c="~/.kongtrol" />)
                y las contraseñas guardadas en el sistema. Si reinstalas, todo sigue ahí. Solo se borran si lo pides.</>,
    uninstall3: <><strong>Instalador de Windows:</strong> Configuración › Aplicaciones › Kongtrol › Desinstalar.
                Te pregunta una vez si borrar también tu configuración y tus contraseñas (por defecto No).
                Siempre desconecta tus VPN, cierra Kongtrol y quita el ayudante de red.</>,
    uninstall4: <><strong>Desde la terminal:</strong> desconecta tus VPN, cierra Kongtrol y quita el ayudante de red,
                el inicio automático y la entrada del PATH. Si instalaste con Homebrew, npm o el instalador de
                Windows, te dice cómo terminar.</>,
    uninstall5: `$ kongtrol uninstall                    # quita la app; tus datos se quedan
$ kongtrol uninstall --purge            # también borra datos y contraseñas (pregunta [s/N])
$ kongtrol uninstall --data-only        # solo borra tus datos; la app se queda
$ kongtrol uninstall --purge --dry-run  # muestra qué haría, sin tocar nada`,
    uninstall6: <><strong>Con el script de instalación:</strong></>,
    uninstall7: `# Linux / macOS (agrega --purge para borrar también tus datos)
$ curl -fsSL https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.sh | sh -s -- --uninstall

# Windows, PowerShell (agrega -Purge para borrar también tus datos)
> & ([scriptblock]::Create((iwr https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.ps1 -useb))) -Uninstall`,
    uninstall8: <><strong>Homebrew o npm:</strong> no borran tus datos. Si quieres borrarlos, corre primero{' '}
                <IC c="kongtrol uninstall --purge" /> y después desinstala:</>,
    uninstall9: `$ kongtrol uninstall --purge
$ brew uninstall kongtrol     # o: npm uninstall -g kongtrol`,
    uninstall10: <><strong>Solo borrar tus datos, sin desinstalar:</strong> en el panel, Ajustes › «Borrar todos mis
                datos». Te muestra qué se borra y pide confirmación. La app se queda y vuelve a la configuración inicial.</>,
    uninstall11: <>La bandeja no tiene opción de desinstalar, a propósito: usa el desinstalador del sistema
                o <IC c="kongtrol uninstall" />.</>,
  }
}

export type GuideContent = ReturnType<typeof guideES>
