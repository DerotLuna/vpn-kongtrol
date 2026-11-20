// Setup guide content — Spanish.
// Version 3.1.0 · 2026-11-19.
//
// Rendered by components/sections/Guide.tsx, which owns the markup; this file
// holds only the words. Keys and shapes must match guide.en.tsx exactly
// (enforced by the GuideContent type and scripts/check-i18n.mjs), so both
// languages always carry the same content. Every fact here is checked against
// the code and docs/SETUP.md (research note
// docs/research/2026-11-19-landing-guide-accuracy.md). File names come only
// from links.ts.
import type { ReactNode } from 'react'
import type { OS } from '../hooks/useOS'
import IC from '../components/ui/InlineCode'
import { pathFor } from '../site'
import { INSTALL_CMD, UNINSTALL_SCRIPT_BASE, WINDOWS_INSTALLER_FILE } from '../links'

/** One step: what to do, and what the user sees when it worked. */
export interface GuideStep { t: ReactNode; d: ReactNode; see: ReactNode; code?: string }
/** One problem: the symptom, the one "what to do" line, an optional command. */
export interface GuideFix { q: ReactNode; fix: ReactNode; code?: string }

const tray = (os: OS) => os === 'windows'
  ? 'en la bandeja, junto al reloj (si no lo ves, abre la flecha ^)'
  : os === 'macos'
    ? 'en la barra de menús, arriba a la derecha'
    : 'en la bandeja del sistema de tu escritorio'

export default function guideES(os: OS) {
  return {
    labels: {
      toc: 'Contenido',
      jump: 'Ir a',
      see: 'Verás',
      fix: 'Qué hacer',
      cmd: 'Comando',
      does: 'Qué hace',
    },
    // feature:first_run
    qs: {
      badge: 'Inicio rápido',
      lead: <>Cuatro pasos, sin terminal. Kongtrol trabaja con las VPN que ya usas: no las reemplaza.</>,
      steps: [
        {
          t: <>Instala Kongtrol</>,
          d: os === 'windows'
            ? <>Descarga el instalador desde <a href="/#install">descargas</a> y ábrelo. Se instala solo para tu usuario.</>
            : <>Copia la línea de <a href="#install">Instalar</a> en la Terminal, o descarga desde <a href="/#install">descargas</a>.</>,
          see: <>El ícono de Kongtrol {tray(os)}.</>,
        },
        {
          t: <>Da permiso una sola vez</>,
          d: <>Kongtrol instala un <strong>ayudante</strong>: un pequeño servicio que cambia rutas y DNS por ti.
            {os === 'windows'
              ? <> El instalador lo pregunta: pulsa <strong>Sí</strong> en el aviso de Windows.</>
              : <> En el menú del ícono elige <strong>Instalar ayudante (una vez)</strong> y escribe tu contraseña.</>}</>,
          see: <>«Ayudante listo». No vuelve a pedirlo.</>,
        },
        {
          t: <>Crea tu primera VPN</>,
          d: <>El panel se abre en tu navegador, en <strong>Perfiles</strong>. Elige tu VPN, completa los datos y pulsa <strong>Guardar</strong>.</>,
          see: <>Tu VPN en la lista. Kongtrol se reinicia solo para usarla.</>,
        },
        {
          t: <>Conecta</>,
          d: <>Desde el ícono o desde <strong>Inicio</strong> en el panel. Si usas la app de tu VPN, conecta ahí como siempre: Kongtrol lo detecta.</>,
          see: <>«tu-vpn · conectada» con un punto verde, y el ícono en verde.</>,
        },
      ] as GuideStep[],
      cli: <>¿Prefieres la terminal? <IC c="kongtrol init" /> hace lo mismo con preguntas paso a paso.</>,
    },
    how: {
      title: <>Cómo funciona</>,
      lead: <>Kongtrol no es una VPN. Se pone entre tus apps y las VPN que ya tienes, y decide por cuál sale cada destino.
        Lo que no tiene regla sale por tu internet normal.</>,
      apps: <>tus apps</>,
      appsSub: <>navegador, Slack, terminal…</>,
      engine: <>reglas: sitios · redes</>,
      vpns: <>tus VPN</>,
      foot: <>kill switch · protección de DNS · registro firmado · todo local</>,
    },
    // feature:providers
    prereqs: {
      title: <>Antes de empezar</>,
      lead: <>Ten tu VPN funcionando como siempre. Kongtrol la usa; no la configura por ti.</>,
      col1: 'Tu VPN',
      col2: 'Qué necesitas',
      rows: [
        { vpn: <>NordVPN, Proton VPN, Surfshark, FortiClient</>, need: <>Su app oficial instalada y tu sesión iniciada. No la cierres: Kongtrol trabaja con ella.</> },
        { vpn: <>WireGuard, OpenVPN</>, need: <>El archivo <IC c=".conf" /> o <IC c=".ovpn" /> que te dio tu proveedor o tu equipo de IT, y el programa{' '}
          {os === 'windows' ? 'WireGuard u OpenVPN para Windows' : os === 'macos' ? 'wireguard-tools u OpenVPN' : 'wireguard-tools u openvpn'}.</> },
        { vpn: <>Tailscale, Cloudflare WARP, Cisco Secure Client, GlobalProtect</>, need: <>Su programa oficial instalado y con sesión iniciada.</> },
      ],
      note: <>Lista completa y qué hace Kongtrol con cada una: <a href="/#vpns">VPN compatibles</a>. Proton VPN también funciona sin su app,
        como perfil <IC c="wireguard" /> con el archivo que descargas de tu cuenta.</>,
    },
    // feature:helper
    install: {
      title: <>Instalar</>,
      steps: (os === 'windows'
        ? [
          { t: <>Descarga el instalador</>, d: <>En <a href="/#install">descargas</a>, el botón de Windows baja <IC c={WINDOWS_INSTALLER_FILE} />.</>, see: <>El archivo en tu carpeta Descargas.</> },
          { t: <>Ábrelo</>, d: <>Si sale un aviso azul de Windows (SmartScreen), pulsa <strong>Más información</strong> y luego <strong>Ejecutar de todos modos</strong>. Sale porque el archivo aún no está firmado.</>, see: <>El asistente de instalación de Kongtrol.</> },
          { t: <>Deja las opciones recomendadas</>, d: <>Abrir al iniciar sesión, instalar el ayudante de red y agregar el comando <IC c="kongtrol" />. No pide ser administrador, salvo el ayudante (una vez).</>, see: <>Un aviso de Windows para el ayudante: pulsa <strong>Sí</strong>.</> },
          { t: <>Termina</>, d: <>Con «Abrir Kongtrol y crear mi primera VPN» marcado, sigue en <a href="#wizard">Tu primera VPN</a>.</>, see: <>El ícono en la bandeja y el panel en tu navegador.</> },
        ]
        : [
          { t: <>Abre la Terminal</>, d: os === 'macos' ? <>Búscala con <strong>Cmd + Espacio</strong> → «Terminal».</> : <>Desde el menú de aplicaciones de tu escritorio.</>, see: <>Una ventana con una línea para escribir.</> },
          { t: <>Pega esta línea y pulsa Enter</>, code: `$ ${INSTALL_CMD.unix}`, d: <>Descarga la última versión, comprueba que no se haya alterado y la instala.</>, see: os === 'macos' ? <><IC c="~/Applications/Kongtrol.app" /> instalado, y Kongtrol se abre.</> : <>El comando <IC c="kongtrol" />, el ícono y una entrada en el menú; Kongtrol se abre.</> },
          { t: <>Instala el ayudante</>, d: <>En el menú del ícono, <strong>Instalar ayudante (una vez)</strong>.</>, see: <>Te pide tu contraseña una vez; luego «Ayudante listo».</> },
        ]) as GuideStep[],
      scriptNote: os === 'macos'
        ? <>Instalado así, macOS no muestra el aviso de app sin verificar. Si descargas desde el navegador, la primera vez:
          clic derecho → <strong>Abrir</strong>, o Ajustes del Sistema → Privacidad y seguridad → <strong>Abrir igualmente</strong>.</>
        : os === 'linux'
          ? <>Otra opción: el paquete <IC c=".deb" /> (Ubuntu, Debian) o <IC c=".rpm" /> (Fedora) de{' '}
            <a href="/#install">descargas</a>, con doble clic. En GNOME el ícono necesita la extensión AppIndicator (Ubuntu ya la trae).</>
          : <>¿Solo el comando <IC c="kongtrol" />, sin la app? Esta línea en PowerShell (sin administrador):</>,
      winScript: INSTALL_CMD.windows,
      helper: <><strong>Un permiso, una sola vez.</strong> Después ningún paso necesita una terminal de administrador ni{' '}
        <IC c="sudo" />. Si falta el ayudante, el panel lo avisa con un botón <strong>Reparar</strong>; en la terminal:{' '}
        <IC c="kongtrol service install" />.</>,
    },
    // feature:wizard
    wizard: {
      title: <>Tu primera VPN</>,
      lead: <>En el panel, <strong>Perfiles</strong> (o el botón <strong>＋ Agregar</strong> arriba). Elige una de estas:</>,
      ways: [
        <><strong>Una VPN que ya usas</strong> (NordVPN, Proton, FortiClient…): elige la app y ponle un nombre, por ejemplo <IC c="oficina" />.</>,
        <><strong>Un archivo <IC c=".conf" /> u <IC c=".ovpn" /></strong>: impórtalo, o suéltalo sobre el panel. Kongtrol lo revisa y guarda una copia protegida.</>,
        <><strong>Usuario y contraseña</strong>: van al llavero del sistema, nunca a un archivo de texto.</>,
      ],
      attach: <><strong>Si usas la app de tu VPN:</strong> al conectar, si el túnel ya está arriba, Kongtrol se engancha.
        Si no, abre la app y conecta por ti cuando puede. Si no puede, te dice qué hacer y espera a que conectes tú.</>,
      cliTitle: <>Desde la terminal</>,
      cli: <><IC c="kongtrol init" /> te pregunta, en este orden:</>,
      cliSteps: [
        <>Tus VPN: detecta las apps instaladas (✓) y te pide nombre y datos de cada perfil.</>,
        <>Reglas: qué sitios o redes salen por cada VPN (puedes dejarlo para después).</>,
        <>Seguridad: kill switch, protección de DNS, registro firmado, panel y la consulta opcional de IP pública.</>,
        <>Revisar y guardar en <IC c="~/.kongtrol/kongtrol.yaml" />. Luego ofrece el ayudante y «¿Conectar ahora?».</>,
      ],
      cliMore: <>Si ya tienes perfiles, <IC c="kongtrol init" /> muestra un menú: agregar un perfil, actualizar credenciales,
        reglas, seguridad o guardar. Para scripts: <IC c="kongtrol profile add" /> y <IC c="kongtrol profile import" />.
        El idioma sale de tu sistema; cámbialo con <IC c="kongtrol config lang en" />.</>,
      code: `$ kongtrol init
$ kongtrol profile import ~/Descargas/proton-us.conf`,
      forti: <><strong>FortiClient: ¿host y nombre del túnel?</strong> En FortiClient, pestaña <strong>Remote Access</strong>:
        el nombre de la conexión es el túnel; con el lápiz (editar) ves <strong>Remote Gateway</strong>, que es el host.</>,
    },
    // feature:daemon_lifecycle
    connect: {
      title: <>Conectar y desconectar</>,
      lead: <>Desde el ícono: haz clic en la VPN. Desde el panel: <strong>Inicio</strong> → <strong>Conectar</strong>.</>,
      bg: <>Kongtrol sigue trabajando en segundo plano. Cerrar el panel, la terminal o el ícono no desconecta nada.</>,
      stop: <>Para desconectar todo: <strong>Salir y desconectar todo</strong> en el ícono, o <IC c="kongtrol down --all" />.
        El ícono se queda con «Kongtrol detenido»; <strong>Iniciar Kongtrol</strong> lo vuelve a arrancar.</>,
      code: `$ kongtrol up oficina          # conecta una VPN
$ kongtrol up oficina aws      # varias a la vez
$ kongtrol status --watch      # estado en vivo · q cierra la vista, las VPN siguen
$ kongtrol down oficina        # desconecta una
$ kongtrol down --all          # desconecta todas`,
      waiting: <><strong>«Esperando conexión en …»</strong> significa que Kongtrol abrió la app de tu VPN y espera que conectes ahí.
        Conecta en la app; Kongtrol lo detecta solo.</>,
      drop: <><strong>Si una VPN se cae,</strong> Kongtrol la reconecta sola con esperas crecientes, y te avisa si no puede.
        Solo prueba otra VPN si tú la declaraste como respaldo (<IC c="failover" />).</>,
    },
    // feature:policies feature:routes
    policies: {
      title: <>Qué sale por cada VPN</>,
      lead: <>Con varias VPN a la vez, las <strong>reglas</strong> deciden qué destino va por cuál. Sin regla, sale por tu internet normal.</>,
      diagram: `  destino              regla                        sale por
  ─────────────────────────────────────────────────────────────────
  claude.ai        ──▶  domains: claude.ai      ──▶  us-content
  10.10.4.12       ──▶  ip_ranges: 10.10.0.0/16 ──▶  oficina
  ejemplo.com      ──▶  sin regla               ──▶  internet normal`,
      panel: <>En el panel: <strong>Reglas</strong> → <strong>＋ Agregar</strong>. Escribe un sitio o una red, elige la VPN y prueba antes de guardar.
        En la terminal:</>,
      code: `$ kongtrol policy add github.com --via oficina
$ kongtrol policy show github.com     # por cuál VPN sale y por qué
$ kongtrol policy list
$ kongtrol policy remove <nombre>`,
      down: <><strong>Si la VPN de una regla está apagada,</strong> ese destino sale por tu internet normal. Para bloquearlo en ese caso:{' '}
        <IC c="--block-when-down" /> (en el archivo, <IC c="on_tunnel_down: block" />).</>,
      yamlLead: <>También puedes escribirlas en <IC c="kongtrol.yaml" />, bajo <IC c="policies:" />:</>,
      yaml: `policies:
  - name: Claude AI
    match:
      domains: [claude.ai, "*.anthropic.com"]
    via: us-content

  - name: Red de la oficina
    match:
      ip_ranges: [10.10.0.0/16]
      domains: ["*.tuempresa.com"]
    via: oficina
    on_tunnel_down: block   # opcional: si oficina está apagada, bloquear`,
      match: <><strong>Cómo se elige:</strong> <IC c="*.empresa.com" /> cubre los subdominios, no <IC c="empresa.com" /> (agrega los dos).
        En redes gana la más específica: <IC c="10.10.1.0/24" /> antes que <IC c="10.0.0.0/8" />.
        Las reglas por app (<IC c="apps" />) por ahora solo explican, no enrutan.</>,
      apply: <>Después de editar el archivo: <IC c="kongtrol config validate" /> y <IC c="kongtrol reload --policy" /> (aplica sin tocar los túneles).
        Lo que cambies en el panel se aplica solo.</>,
    },
    groups: {
      title: <>Grupos</>,
      lead: <>Un grupo conecta varias VPN con un clic. Créalo en el panel: <strong>Perfiles</strong> → pestaña <strong>Grupos</strong>. O en el archivo:</>,
      yaml: `groups:
  trabajo:
    profiles: [oficina, dev-server, aws]`,
      code: `$ kongtrol up --group trabajo
$ kongtrol down --group trabajo`,
      note: <>Una VPN puede estar en varios grupos. Si editas el archivo con Kongtrol en marcha:{' '}
        <IC c="kongtrol reload" /> aplica los cambios sin reiniciar.</>,
    },
    doctor: {
      title: <>Revisar que todo esté bien</>,
      lead: <>En el panel: <strong>Ajustes</strong> → <strong>Diagnóstico</strong>. Revisa tus VPN, archivos, contraseñas guardadas y el ayudante.
        Cada problema trae qué hacer; si falta el ayudante, el botón <strong>Reparar</strong> lo instala.</>,
      code: `$ kongtrol doctor          # lo mismo en la terminal
$ kongtrol doctor --fix    # y repara lo que se puede`,
      export: <><strong>Exportar configuración</strong> (en Diagnóstico, o <IC c="kongtrol export" />) te da un archivo sin contraseñas,
        útil para pedir ayuda o preparar el equipo de un compañero.</>,
    },
    // feature:dashboard
    dashboard: {
      title: <>El panel</>,
      lead: <>Ábrelo desde el ícono (<strong>Abrir panel</strong>) o con <IC c="kongtrol dashboard" />. Se abre ya autorizado:
        escribir la dirección a mano no basta, por seguridad. Funciona dentro de tu equipo; nada sale de él.</>,
      rows: [
        { name: <>Inicio</>, desc: <>Tus VPN en vivo (estado, IP, tiempo, tráfico), conectar y desconectar, avisos y grupos de acceso rápido.</> },
        { name: <>Perfiles</>, desc: <>Crear, editar, renombrar y borrar VPN; respaldo (failover); pestaña Grupos.</> },
        { name: <>Reglas</>, desc: <>Qué sitios y redes salen por cada VPN, con prueba antes de guardar; pestaña Horarios.</> },
        { name: <>Tráfico</>, desc: <>Qué pasa por cada VPN, por dónde sale un sitio, conexiones recientes (solo en memoria) y las rutas del sistema.</> },
        { name: <>Seguridad</>, desc: <>Kill switch y protección de DNS, por VPN, y la consulta opcional de IP pública.</> },
        { name: <>Actividad</>, desc: <>Registro firmado con filtros y los avisos de Kongtrol.</> },
        { name: <>Ajustes</>, desc: <>Idioma, tema, diagnóstico, aviso de versión nueva, reiniciar o detener, y borrar tus datos.</> },
      ],
      port: <>El puerto del panel se cambia solo desde la terminal (<IC c="kongtrol config dashboard set-port <puerto>" />):
        cambiarlo desde el propio panel cortaría la conexión.</>,
    },
    // feature:i18n feature:killswitch feature:dns_guard
    daily: {
      title: <>Día a día</>,
      rows: [
        { cmd: 'kongtrol up --group trabajo', desc: <>Empezar el día</> },
        { cmd: 'kongtrol down --all', desc: <>Desconectar todo</> },
        { cmd: 'kongtrol status', desc: <>Qué está conectado, kill switch y DNS</> },
        { cmd: 'kongtrol policy show github.com', desc: <>Por cuál VPN sale un destino y por qué</> },
        { cmd: 'kongtrol ip', desc: <>Tu IP pública ahora (si activaste la consulta)</> },
        { cmd: 'kongtrol logs', desc: <>Registro de conexiones y avisos</> },
        { cmd: 'kongtrol restart', desc: <>Reiniciar Kongtrol; reconecta las VPN que estaban arriba</> },
      ],
      ks: <><strong>Kill switch:</strong> si una VPN se cae sola, bloquea solo sus destinos; lo demás sigue funcionando.
        Si esa VPN llevaba todo tu tráfico, internet queda bloqueado hasta que vuelva, y Kongtrol te lo avisa.
        Con <strong>Permitir la red local</strong> apagado (<IC c="allow_lan: false" />) también se cortan tu impresora,
        tu NAS y los demás equipos de tu red; enciéndelo para mantenerlos.
        Actívalo en <strong>Seguridad</strong> o con <IC c="kongtrol config killswitch on" />; por VPN, en el archivo:</>,
      ksYaml: `vpns:
  oficina:
    kill_switch: true
  us-content:
    kill_switch: false`,
      sched: <><strong>Horarios</strong> (panel: Reglas → Horarios) conectan y desconectan solos:</>,
      schedYaml: `monitor:
  scheduler:
    enabled: true
    rules:
      - name: horario-oficina
        profiles: [oficina]
        weekdays: [mon, tue, wed, thu, fri]
        start: "09:00"
        end: "18:00"`,
      pass: <><strong>¿Cambiaste tu contraseña?</strong> En el panel, edita el perfil. En la terminal:{' '}
        <IC c="kongtrol init" /> → «Actualizar credenciales de un perfil».</>,
      refTitle: <>Todos los comandos</>,
      ref: <>Los mismos grupos que <IC c="kongtrol --help" />. Cada comando trae ejemplos: <IC c="kongtrol <comando> --help" />.</>,
    },
    // feature:killswitch feature:dns_guard
    trouble: {
      title: <>Si algo falla</>,
      lead: <>Primero, <strong>Diagnóstico</strong> en el panel o <IC c="kongtrol doctor" />: casi siempre dice qué hacer.</>,
      items: [
        {
          q: <>No abre ningún sitio después de que Kongtrol se cerró de golpe</>,
          fix: <>Espera unos 20 segundos: el ayudante quita solo los bloqueos de Kongtrol. Si siguen, corre{' '}
            <IC c="kongtrol down --all" />. Último recurso, quita solo las reglas de Kongtrol
            {os === 'windows' ? <> (nunca <IC c="netsh advfirewall reset" />: borra todas tus reglas)</> : null}:</>,
          code: os === 'windows'
            ? `# PowerShell como administrador
> netsh advfirewall firewall delete rule name=KongtrolBlock
> netsh advfirewall firewall delete rule name=KongtrolKillSwitchBlock`
            : os === 'macos'
              ? `$ sudo pfctl -a com.kongtrol -F all`
              : `$ sudo iptables -F KONGTROL_BLOCK
$ sudo ip6tables -F KONGTROL_BLOCK
$ sudo iptables -F KONGTROL`,
        },
        {
          q: <>Se cayó una VPN y no carga nada</>,
          fix: <>Esa VPN llevaba todo tu tráfico y el kill switch la protege. Desconéctala para navegar ya, o crea reglas
            para que solo algunos sitios dependan de ella. Si tampoco llegas a la impresora o al NAS, enciende
            «Permitir la red local» en Seguridad.</>,
          code: `$ kongtrol down <perfil>`,
        },
        {
          q: <>Los nombres de sitios no resuelven tras un cierre inesperado</>,
          fix: <>Devuelve el DNS de tu sistema a como estaba:</>,
          code: os === 'windows'
            ? `> netsh interface ip set dns "Ethernet" dhcp     # o "Wi-Fi"`
            : os === 'macos'
              ? `$ networksetup -setdnsservers Wi-Fi empty`
              : `# con systemd-resolved (Ubuntu, Fedora): el nombre del túnel
$ resolvectl revert tun0
# sin systemd-resolved
$ sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf`,
        },
        {
          q: <>Kongtrol no encuentra la app de mi VPN</>,
          fix: <>Indica dónde está: en el panel, edita el perfil → opciones avanzadas (ruta de la app). En el archivo:</>,
          code: os === 'windows'
            ? `vpns:
  oficina:
    type: forticlient
    app_path: "C:\\\\Ruta\\\\FortiClient\\\\FortiClient.exe"`
            : `vpns:
  oficina:
    type: forticlient
    app_path: "/ruta/a/FortiClient"`,
        },
        {
          q: <>Contraseña incorrecta o vencida</>,
          fix: <>Guarda la nueva: edita el perfil en el panel, o <IC c="kongtrol init" /> → «Actualizar credenciales de un perfil».</>,
        },
        {
          q: <>«Falta el ayudante» o «ayudante desactualizado»</>,
          fix: <>Botón <strong>Reparar</strong> en Diagnóstico, o <IC c="kongtrol service install" />. Pide permiso una vez.</>,
        },
        {
          q: <>Cloudflare WARP: «not registered»</>,
          fix: <>Registra el equipo una vez en la app de WARP, o en la terminal:</>,
          code: `$ warp-cli registration new`,
        },
        {
          q: <><IC c="kongtrol" /> no se reconoce en la terminal</>,
          fix: <>Abre una terminal nueva. Si sigue, <IC c="kongtrol doctor" /> dice qué copia usa la terminal y si hay otra vieja en el PATH.
            El ícono y el panel no lo necesitan.</>,
        },
      ] as GuideFix[],
      files: <><strong>Dónde guarda tus cosas:</strong> la configuración en{' '}
        <IC c={os === 'windows' ? '%USERPROFILE%\\.kongtrol\\' : '~/.kongtrol/'} /> (sin contraseñas). Las contraseñas, en{' '}
        {os === 'windows' ? 'el Administrador de credenciales de Windows' : os === 'macos' ? 'el Llavero de macOS' : 'el llavero del sistema (libsecret / GNOME Keyring)'}.</>,
    },
    // feature:update_check
    update: {
      title: <>Actualizar</>,
      lead: <>Actualiza por el mismo camino por el que instalaste. Tu configuración, tus contraseñas y el ayudante se quedan,
        y las VPN que tenías conectadas vuelven a conectarse.</>,
      how: os === 'windows'
        ? <>Descarga el instalador nuevo y ábrelo: dice «Actualizar Kongtrol» y usa tus opciones de antes.</>
        : os === 'macos'
          ? <>Vuelve a pegar la línea de instalación: dice «Updating Kongtrol» y reinicia en la versión nueva.</>
          : <>Vuelve a pegar la línea de instalación. Con <IC c=".deb" /> o <IC c=".rpm" />: instala el paquete nuevo y corre{' '}
            <IC c="kongtrol restart" />.</>,
      notice: <><strong>Aviso de versión nueva (opcional, apagado):</strong> si lo activas, Kongtrol mira como mucho una vez al día
        si hay versión nueva y te avisa una vez por versión. Nunca descarga ni instala nada. Actívalo en <strong>Ajustes</strong> o:</>,
      code: `$ kongtrol config update-check on       # activar
$ kongtrol config update-check off      # apagar: ninguna petición
$ kongtrol config update-check status`,
      privacy: <>Qué se consulta y a quién: <a href={`${pathFor('privacidad', 'es')}#conexiones`}>política de privacidad</a>.</>,
    },
    // feature:uninstall
    uninstall: {
      title: <>Desinstalar</>,
      keep: <><strong>Tus datos se quedan:</strong> la configuración y las contraseñas guardadas. Si reinstalas, todo sigue ahí.
        Solo se borran si lo pides.</>,
      how: os === 'windows'
        ? <>Configuración → Aplicaciones → Kongtrol → <strong>Desinstalar</strong>. Te pregunta si borrar también tus datos (por defecto No).
          Desconecta tus VPN, cierra Kongtrol y quita el ayudante.</>
        : <>En la terminal. Desconecta tus VPN, cierra Kongtrol y quita el ayudante. Con un paquete <IC c=".deb" />/<IC c=".rpm" />,
          te dice cómo terminar (<IC c="sudo apt remove kongtrol" /> o <IC c="sudo dnf remove kongtrol" />).</>,
      code: `$ kongtrol uninstall                    # quita la app; tus datos se quedan
$ kongtrol uninstall --purge            # también borra datos y contraseñas (pregunta)
$ kongtrol uninstall --purge --dry-run  # muestra qué haría, sin tocar nada`,
      scriptLead: <>Con el script de instalación (agrega <IC c={os === 'windows' ? '-Purge' : '--purge'} /> para borrar también tus datos):</>,
      script: os === 'windows'
        ? `> & ([scriptblock]::Create((iwr ${UNINSTALL_SCRIPT_BASE}install.ps1 -useb))) -Uninstall`
        : `$ curl -fsSL ${UNINSTALL_SCRIPT_BASE}install.sh | sh -s -- --uninstall`,
      data: <><strong>Solo borrar tus datos:</strong> panel → Ajustes → «Borrar todos mis datos», o{' '}
        <IC c="kongtrol uninstall --data-only" />. La app se queda y vuelve al primer uso. El ícono no tiene opción de desinstalar, a propósito.</>,
    },
  }
}

export type GuideContent = ReturnType<typeof guideES>
