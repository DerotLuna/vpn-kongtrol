// Setup guide content — Spanish.
// Version 3.10.0 · 2026-11-25.
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
import { DEFENDER_FALSE_POSITIVE_URL, INSTALL_CMD, UNINSTALL_SCRIPT_BASE, WINDOWS_INSTALLER_FILE } from '../links'

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
          t: <>Crea tu primer perfil VPN</>,
          d: <>El panel se abre en tu navegador, en <strong>Perfiles</strong>. Elige tu VPN, completa los datos y pulsa <strong>Guardar</strong>.</>,
          see: <>Tu perfil en la lista, listo para conectar.</>,
        },
        {
          t: <>Conecta</>,
          d: <>Desde el ícono o desde <strong>Inicio</strong> en el panel. Si usas la app de tu VPN, conecta ahí como siempre: Kongtrol lo detecta.</>,
          see: <>El perfil aparece conectado y el ícono cambia de estado.</>,
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
    // feature:providers feature:rule_move
    prereqs: {
      title: <>Antes de empezar</>,
      lead: <>Ten tu VPN funcionando como siempre. Kongtrol la usa; no la configura por ti.</>,
      col1: 'Tu VPN',
      col2: 'Qué necesitas',
      rows: [
        { vpn: <>NordVPN, Proton VPN, Surfshark, FortiClient, Mullvad, PIA, ExpressVPN, Windscribe, CyberGhost</>, need: <>Su app oficial instalada y tu sesión iniciada. Si Kongtrol no detecta el túnel, indica su interfaz en las opciones avanzadas. ExpressVPN en Windows y CyberGhost se conectan desde el cliente del proveedor.</> },
        { vpn: <>WireGuard, OpenVPN</>, need: <>El archivo <IC c=".conf" /> o <IC c=".ovpn" /> que te dio tu proveedor o tu equipo de IT, y el programa{' '}
          {os === 'windows' ? 'WireGuard u OpenVPN para Windows' : os === 'macos' ? 'wireguard-tools u OpenVPN' : 'wireguard-tools u openvpn'}.</> },
        { vpn: <>Tailscale, Cloudflare WARP, Cisco Secure Client, GlobalProtect</>, need: <>Su programa oficial instalado y con sesión iniciada.</> },
      ],
      note: <>Lista completa y qué hace Kongtrol con cada una: <a href="/#vpns">VPN compatibles</a>. Proton VPN también funciona sin su app,
        como perfil <IC c="wireguard" /> con el archivo que descargas de tu cuenta. Usa uno u otro, no los dos a la vez:
        comparten tu cuenta y chocan. Kongtrol no conecta ese archivo mientras la app de Proton esté conectada, y si conectas
        la app, desconecta el archivo y te avisa. Si agregas la app de Proton cuando ya tienes reglas con
        ese archivo, Kongtrol te ofrece pasarlas a la app (o después: <IC c="kongtrol policy move" />).</>,
    },
    // feature:helper feature:windows_installer_shutdown
    install: {
      title: <>Instalar</>,
      steps: (os === 'windows'
        ? [
          { t: <>Descarga el instalador</>, d: <>En <a href="/#install">descargas</a>, el botón de Windows baja <IC c={WINDOWS_INSTALLER_FILE} />.</>, see: <>El archivo en tu carpeta Descargas.</> },
          { t: <>Ábrelo</>, d: <>Si sale un aviso azul de Windows (SmartScreen), pulsa <strong>Más información</strong> y luego <strong>Ejecutar de todos modos</strong>. Sale porque el archivo aún no está firmado.
            {' '}Si Microsoft Defender lo marca como amenaza (por ejemplo <IC c="Trojan:Win32/Bearfoos.A!ml" />) y lo pone en cuarentena, es un falso positivo de los programas nuevos sin firma: compruébalo en <a href="/seguridad">Seguridad</a>, permítelo en Seguridad de Windows → Historial de protección solo si confías en él y repórtalo en{' '}
            <a href={DEFENDER_FALSE_POSITIVE_URL}>microsoft.com/wdsi/filesubmission</a>.</>, see: <>El asistente de instalación de Kongtrol.</> },
          { t: <>Deja las opciones recomendadas</>, d: <>Abrir al iniciar sesión, instalar el ayudante de red y agregar el comando <IC c="kongtrol" />. No pide ser administrador, salvo el ayudante (una vez).</>, see: <>Un aviso de Windows para el ayudante: pulsa <strong>Sí</strong>.</> },
          { t: <>Termina</>, d: <>Con «Abrir Kongtrol y crear mi primera VPN» marcado, sigue en <a href="#wizard">Tu primer perfil VPN</a>.</>, see: <>El ícono en la bandeja y el panel en tu navegador.</> },
        ]
        : [
          { t: <>Abre la Terminal</>, d: os === 'macos' ? <>Búscala con <strong>Cmd + Espacio</strong> → «Terminal».</> : <>Desde el menú de aplicaciones de tu escritorio.</>, see: <>Una ventana con una línea para escribir.</> },
          { t: <>Pega esta línea y pulsa Enter</>, code: `$ ${INSTALL_CMD.unix}`, d: <>Descarga la última versión, comprueba que no se haya alterado y la instala.</>, see: os === 'macos' ? <><IC c="~/Applications/Kongtrol.app" /> instalado, y Kongtrol se abre.</> : <>El comando <IC c="kongtrol" />, el ícono y una entrada en el menú; Kongtrol se abre.</> },
          { t: <>Instala el ayudante</>, d: <>En el menú del ícono, <strong>Instalar ayudante (una vez)</strong>.</>, see: <>Te pide tu contraseña una vez; luego «Ayudante listo».</> },
        ]) as GuideStep[],
      scriptNote: os === 'macos'
        ? <>Instalado así, macOS no muestra el aviso de app sin verificar. Si descargas desde el navegador, la primera vez:
          clic derecho → <strong>Abrir</strong>, o Ajustes del Sistema → Privacidad y seguridad → <strong>Abrir igualmente</strong>.
          {' '}¿Solo el comando <IC c="kongtrol" />, sin la app? Termina la línea en <IC c="| sh -s -- --cli-only" />.</>
        : os === 'linux'
          ? <>Otra opción: el paquete <IC c=".deb" /> (Ubuntu, Debian) o <IC c=".rpm" /> (Fedora) de{' '}
            <a href="/#install">descargas</a>, con doble clic. En GNOME el ícono necesita la extensión AppIndicator (Ubuntu ya la trae).
            {' '}¿Solo el comando <IC c="kongtrol" />? Termina la línea en <IC c="| sh -s -- --cli-only" />; sin escritorio (un servidor, SSH) lo hace solo y te lo dice.</>
          : <>¿Sin el instalador? Esta línea en PowerShell (sin administrador) instala lo mismo: la app con su ícono y el comando <IC c="kongtrol" />.
            {' '}Solo el comando: antes escribe <IC c="$env:KONGTROL_CLI_ONLY=1" />. Si ya usaste el instalador, actualiza con el instalador nuevo.
            {' '}Si Windows avisa que no pudo cerrar las aplicaciones, cancela la instalación, elige <strong>Salir y desconectar todo</strong> en la bandeja y vuelve a ejecutar el instalador.</>,
      winScript: INSTALL_CMD.windows,
      helper: <><strong>Un permiso, una sola vez.</strong> Después ningún paso necesita una terminal de administrador ni{' '}
        <IC c="sudo" />. Si falta el ayudante, el panel lo avisa con un botón <strong>Reparar</strong>; en la terminal:{' '}
        <IC c="kongtrol service install" />.</>,
    },
    // feature:wizard feature:profile_import feature:add_sheet
    wizard: {
      title: <>Tu primer perfil VPN</>,
      lead: <>En el panel, <strong>Perfiles</strong> (o el botón <strong>＋ Agregar</strong> arriba). Elige una de estas:</>,
      ways: [
        <><strong>Una VPN que ya usas</strong> (NordVPN, Proton, FortiClient…): elige la app y ponle un nombre, por ejemplo <IC c="oficina" />.</>,
        <><strong>Un archivo <IC c=".conf" /> u <IC c=".ovpn" /></strong>: impórtalo con <strong>Elegir archivo</strong>, o suéltalo sobre el panel. Si trae certificados aparte (<IC c="ca.crt" />…), Kongtrol los toma de la misma carpeta o de la ruta que escribe el archivo; si lo soltaste, busca el original en tus carpetas habituales (OpenVPN, Descargas, Escritorio, Documentos). Lo revisa en cuanto lo eliges y te dice al momento si está listo o qué falta, antes de guardar. Si faltan certificados, te los nombra: elige la carpeta donde están o agrégalos uno por uno. Al guardar, guarda una copia protegida.</>,
        <><strong>Usuario y contraseña</strong>: van al llavero del sistema, nunca a un archivo de texto.</>,
        <><strong>Guardar y activar</strong> guarda y conecta el perfil; <strong>Guardar</strong> solo lo guarda, para conectarlo cuando quieras.</>,
      ],
      attach: <><strong>Si usas la app de tu VPN:</strong> al conectar, si el túnel ya está arriba, Kongtrol se engancha.
        Si no, abre la app y conecta por ti cuando puede. Si no puede, te dice qué hacer y espera a que conectes tú.</>,
      cliTitle: <>Desde la terminal</>,
      cli: <>Para crear tu primer perfil, ejecuta <IC c="kongtrol init" />.</>,
      cliSteps: [
        <>Tus VPN: detecta las apps instaladas (✓) y te pide nombre y datos de cada perfil.</>,
        <>Guarda el perfil y sal, o sigue con las reglas y la seguridad.</>,
        <>Si sigues, elige qué sitios o redes salen por la VPN y configura opciones como kill switch y protección DNS.</>,
        <>Kongtrol guarda los cambios en <IC c="~/.kongtrol/kongtrol.yaml" /> y te ofrece instalar el ayudante si hace falta.</>,
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
      lead: <>Desde el ícono: elige el perfil. Desde el panel: <strong>Inicio</strong> → <strong>Conectar</strong>.</>,
      bg: <>Kongtrol sigue trabajando en segundo plano. Cerrar el panel, la terminal o el ícono no desconecta nada.
        Después de tu primera VPN, Kongtrol te pregunta si quieres que al encender el equipo vuelva a conectar las que dejaste conectadas; espera a tener internet y no abre apps de VPN por su cuenta. Puedes cambiarlo en <strong>Ajustes</strong> → <strong>Al iniciar</strong>.</>,
      stop: <>Para desconectar todo: <strong>Salir y desconectar todo</strong> en el ícono, o <IC c="kongtrol down --all" />.
        El ícono se queda con «Kongtrol detenido»; <strong>Iniciar Kongtrol</strong> lo vuelve a arrancar.</>,
      code: `$ kongtrol up oficina          # conecta un perfil
$ kongtrol up oficina aws      # varios a la vez
$ kongtrol status --watch      # estado en vivo · q cierra la vista, siguen conectados
$ kongtrol tray                # abre el icono si instalaste la app de bandeja
$ kongtrol down oficina        # desconecta un perfil
$ kongtrol down --all          # desconecta todos`,
      waiting: <><strong>«Esperando conexión en …»</strong> significa que Kongtrol abrió la app de tu VPN y espera que conectes ahí.
        Conecta en la app; Kongtrol lo detecta solo.</>,
      drop: <><strong>Si una VPN se cae,</strong> Kongtrol la reconecta sola (hasta 3 intentos); si no puede, te avisa una vez con <strong>Reintentar</strong> y vuelve a probar cuando cambia tu red.
        Solo prueba otra VPN si tú la declaraste como respaldo (<IC c="failover" />).</>,
    },
    // feature:policies feature:routes
    policies: {
      title: <>Qué sale por cada perfil</>,
      lead: <>Un <strong>perfil VPN</strong> guarda cómo se conecta Kongtrol: puedes tener varios perfiles del mismo proveedor. Una <strong>regla</strong> elige qué destinos van por un perfil. Una <strong>ruta</strong> es el camino que usa tu equipo para llegar a ese destino. El resto del tráfico sigue las rutas del equipo.</>,
      diagram: `  destino              regla                        sale por
  ─────────────────────────────────────────────────────────────────
  claude.ai        ──▶  domains: claude.ai      ──▶  us-content
  10.10.4.12       ──▶  ip_ranges: 10.10.0.0/16 ──▶  oficina
  ejemplo.com      ──▶  sin regla               ──▶  internet normal`,
      panel: <>En el panel, abre <strong>Reglas</strong> → <strong>＋ Agregar</strong>. Escribe un sitio o una red, elige el perfil y prueba antes de guardar.
        Puedes agregar varias sugerencias editables (Claude, ChatGPT/OpenAI y otros sitios) al crear o editar una regla.
        No garantizan acceso: las restricciones varían por red, las apps usan otros destinos y cada servicio tiene sus condiciones.
        Al guardar un perfil, elige «Agregar regla de salida» para configurar sus destinos.
        También puedes crear la regla desde la terminal.</>,
      code: `$ kongtrol policy add github.com --via oficina
$ kongtrol policy show github.com     # por cuál perfil sale y por qué
$ kongtrol policy list
$ kongtrol policy remove <nombre>`,
      down: <><strong>Si el perfil está desconectado,</strong> al crearlo o editarlo puedes permitir la conexión normal, bloquear los destinos conocidos o bloquearlos e intentar conectar al detectar actividad TCP. Cada regla puede elegir su propia acción.
        Puedes cambiarlo con <IC c="kongtrol profile on-tunnel-down oficina block" />. La detección puede omitir intentos cortos o rechazados. En macOS, conecta manualmente.</>,
      // feature:default_egress
      defaultEgress: <>En Windows o Linux, elige <strong>Salida por defecto de WireGuard</strong> en un perfil directo compatible y desconectado. También puedes usar <IC c="kongtrol routes check-default oficina" /> y <IC c="kongtrol routes default-egress oficina on" />. Al conectarlo, el perfil se encarga del tráfico sin una ruta más específica.</>,
      // feature:vpn_routes
      vpnRoutes: <><strong>Rutas propias de cada perfil.</strong> Debajo de tus reglas, el panel identifica cada perfil por su nombre y, cuando se conoce, su proveedor y tecnología. Muestra las rutas que vienen del archivo de la VPN o de su servidor. Kongtrol no las cambia: para eso, edita el archivo o pídeselo a tu proveedor.
        Gana la dirección más específica, así que una regla para <IC c="10.1.2.3" /> por otro perfil gana sobre <IC c="10.0.0.0/8" /> o sobre el «Todo el tráfico» de un archivo .ovpn o .conf. En la terminal: <IC c="kongtrol routes vpn" />.</>,
      // feature:profile_priority
      order: <><strong>Cuando dos reglas se cruzan.</strong> Si dos reglas con perfiles distintos cubren el mismo sitio o el mismo rango de IP, gana el perfil con más <strong>prioridad</strong> (el número mayor). Si ninguno tiene prioridad, va antes la regla que está más arriba.
        El panel lo marca en esas reglas («Prioridad 10» o «por posición») y explica quién decidió, con un enlace «Cambiar prioridad». En la terminal: <IC c="kongtrol profile priority oficina 20" /> y <IC c="kongtrol policy list" />.</>,
      yamlLead: <>También puedes escribirlas en <IC c="kongtrol.yaml" />, bajo <IC c="policies:" />:</>,
      yaml: `policies:
  - name: Claude AI
    match:
      domains: [claude.com, "*.claude.com", api.anthropic.com, "*.anthropic.com"]
    via: us-content

  - name: Red de la oficina
    match:
      ip_ranges: [10.10.0.0/16]
      domains: ["*.tuempresa.com"]
    via: oficina
    on_tunnel_down: block   # opcional: si oficina está apagada, bloquear`,
      match: <><strong>Varios destinos.</strong> Separa por comas o saltos de línea, por ejemplo <IC c="claude.com, api.anthropic.com, *.anthropic.com, 192.0.2.42" />. El comodín cubre un nivel de subdominios. Guarda la regla pausada y actívala cuando quieras aplicarla.
        Puedes usar <IC c="café.example" />, y el nombre coincide aunque lo pegues con mayúsculas o un punto final.
        Usa <IC c="192.0.2.42" /> para una IP, <IC c="2001:db8::42" /> para una IPv6 o <IC c="192.0.2.0/24" /> para una red. Entre redes superpuestas gana la más específica.
        Las reglas por app (<IC c="apps" />) por ahora solo explican, no enrutan.</>,
      apply: <>Después de editar el archivo: <IC c="kongtrol config validate" /> y <IC c="kongtrol reload --policy" /> (aplica sin tocar los túneles).
        Lo que cambies en el panel se aplica solo.</>,
    },
    groups: {
      title: <>Grupos</>,
      lead: <>Un grupo conecta varios perfiles VPN con un clic. Créalo en el panel: <strong>Perfiles</strong> → pestaña <strong>Grupos</strong>. O en el archivo:</>,
      yaml: `groups:
  trabajo:
    profiles: [oficina, dev-server, aws]`,
      code: `$ kongtrol up --group trabajo
$ kongtrol down --group trabajo`,
      note: <>Un perfil puede estar en varios grupos. Si editas el archivo con Kongtrol en marcha:{' '}
        <IC c="kongtrol reload" /> aplica los cambios sin reiniciar.</>,
    },
    faq: {
      title: 'Preguntas frecuentes',
      items: [
        { question: '¿Necesito instalar WireGuard o la app de mi VPN?', answer: <>Un perfil directo de WireGuard requiere las herramientas de WireGuard de tu sistema. Si eliges un proveedor que funciona con su app, instala la app de ese proveedor y conecta desde ella cuando Kongtrol te lo pida.</> },
        { question: '¿Puedo tener varias VPN a la vez?', answer: <>Puedes separar destinos entre perfiles compatibles. Algunas apps controlan toda la conexión, el DNS o el firewall. Si una VPN falla al activar otra, abre <a href="#doctor">Diagnóstico</a> para revisar qué muestra el equipo.</> },
        { question: '¿Qué pasa si cierro el panel?', answer: <>Tus conexiones siguen activas. Abre el panel o la bandeja para verlas. Usa <IC c="kongtrol down --all" /> si quieres desconectarlas todas.</> },
      ],
    },
    // feature:wizard_doctor feature:direct_diagnostics
    doctor: {
      title: <>Revisar que todo esté bien</>,
      lead: <>En el panel: <strong>Ajustes</strong> → <strong>Diagnóstico</strong>. Revisa tus VPN, archivos, contraseñas guardadas y el ayudante.
        Cada problema trae qué hacer. Diagnóstico también avisa si dos VPN detectadas anuncian una salida completa para IPv4 o IPv6.</>,
      code: `$ kongtrol dashboard --diagnostics  # abre Diagnóstico directamente
$ kongtrol doctor          # lo mismo en la terminal
$ kongtrol doctor --fix    # y repara lo que se puede`,
      export: <><strong>Exportar configuración</strong> (en Diagnóstico, o <IC c="kongtrol export" />) te da un archivo sin contraseñas,
        útil para pedir ayuda o preparar el equipo de un compañero.</>,
    },
    // feature:dashboard
    dashboard: {
      title: <>El panel</>,
      lead: <>Ábrelo desde el ícono (<strong>Abrir panel</strong>) o con <IC c="kongtrol dashboard" />. Se abre ya autorizado:
        escribir la dirección a mano no basta, por seguridad. Funciona dentro de tu equipo; nada sale de él.
        Busca secciones, acciones, perfiles, reglas y grupos con <strong>Buscar</strong> o Ctrl/Cmd+K; Esc cierra la búsqueda.
        En Inicio y en la bandeja puedes buscar perfiles cuando la lista es larga; se muestran primero los conectados
        y después los conectados más recientemente. Mientras se restaura una conexión, su estado
        indica «Reconectando» y puedes cancelarla.</>,
      rows: [
        { name: <>Inicio</>, desc: <>Tus perfiles VPN en vivo (estado, IP, tiempo, tráfico), conectar y desconectar, avisos y grupos de acceso rápido.</> },
        { name: <>Perfiles</>, desc: <>Crear, editar, renombrar y borrar perfiles; respaldo (failover); pestaña Grupos.</> },
        { name: <>Reglas</>, desc: <>Qué sitios y redes salen por cada perfil, con prueba antes de guardar; pestaña Horarios.</> },
        { name: <>Tráfico</>, desc: <>Qué pasa por cada perfil, por dónde sale un sitio, conexiones recientes (solo en memoria) y las rutas del sistema.</> },
        { name: <>Seguridad</>, desc: <>Kill switch y protección de DNS, ajustes por perfil y la consulta opcional de IP pública.</> },
        { name: <>Actividad</>, desc: <>Registro firmado con filtros (10 entradas y <strong>Ver 20 más</strong>; las nuevas esperan tras <strong>Mostrar</strong>) y los avisos de Kongtrol.</> },
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
        { cmd: 'kongtrol policy show github.com', desc: <>Por cuál perfil sale un destino y por qué</> },
        { cmd: 'kongtrol ip', desc: <>Tu IP pública ahora (si activaste la consulta)</> },
        { cmd: 'kongtrol logs', desc: <>Registro de conexiones y avisos</> },
        { cmd: 'kongtrol restart', desc: <>Reiniciar Kongtrol; reconecta las VPN que estaban arriba</> },
      ],
      ks: <><strong>Kill switch:</strong> si una VPN se cae sola, bloquea solo sus destinos; lo demás sigue funcionando.
        Si esa VPN llevaba todo tu tráfico, internet queda bloqueado hasta que vuelva, y Kongtrol te lo avisa.
        Con <strong>Permitir la red local</strong> apagado (<IC c="allow_lan: false" />) también se cortan tu impresora,
        tu NAS y los demás equipos de tu red; enciéndelo para mantenerlos.
        Si otra VPN pasa a llevar todo tu tráfico (por ejemplo, conectas Proton VPN en su app), no bloquea:
        tu tráfico sigue por esa VPN. Y Kongtrol no conecta dos VPN que llevan todo tu tráfico a la vez:
        te dice cuál desconectar primero.
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
          fix: <>Esa VPN llevaba todo tu tráfico y el kill switch la protege. Para navegar ya, pulsa «Recuperar internet» en el aviso
            (la desconecta), o crea reglas
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
      keep: <><strong>Tus datos siguen en este equipo:</strong> la carpeta <IC c="~/.kongtrol" /> (configuración, registros) y las
        contraseñas en el llavero del sistema. Kongtrol no guarda nada fuera de tu equipo. Si reinstalas, todo sigue ahí.
        Solo se borran si lo pides, y eso no se puede deshacer.</>,
      how: os === 'windows'
        ? <>Configuración → Aplicaciones → Kongtrol → <strong>Desinstalar</strong>. Te pregunta si borrar también tus datos (por defecto No).
          Desconecta tus VPN, cierra Kongtrol y quita el ayudante.</>
        : <>En la terminal. Desconecta tus VPN, cierra Kongtrol y quita el ayudante. Con un paquete <IC c=".deb" />/<IC c=".rpm" />,
          te dice cómo terminar (<IC c="sudo apt remove kongtrol" /> o <IC c="sudo dnf remove kongtrol" />).</>,
      code: `$ kongtrol uninstall                    # quita la app; tus datos siguen en este equipo
$ kongtrol uninstall --purge            # también borra de este equipo datos y contraseñas (pregunta)
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
