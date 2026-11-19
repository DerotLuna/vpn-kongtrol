// Privacy policy — version 1.6.0 · 2026-11-19.
// 1.6.0: the audit log keeps 30 days by default and at most 500 MB in total, oldest deleted first
//   (ADR-0064).
// 1.5.0: plain-language summary in points and shorter sentences. Fixes: the dashboard keeps a
//   session cookie `kongtrol_session` and never receives the token (ADR-0039; the old text named a
//   `kongtrol_control` cookie carrying the token); dashboard browser storage is language + theme
//   (+ the last notice number for the tab), no sidebar state; the audit log is one file per day,
//   kept today and yesterday by default; imported .conf/.ovpn files in ~/.kongtrol/configs/; the
//   helper's own local log; tunnel health checks and the VPN server lookup for the kill switch.
// 1.4.1: the site no longer stores a theme choice (one designed theme, ADR-0060); only `kglang`.
// 1.4.0: recent destinations list (ADR-0054): memory only, 500 entries, 1 h, on by default, switch
//   `kongtrol config recent-connections off` (internal/monitor/recent.go).
// 1.3.0: opt-in new version notice (GitHub releases API, off by default; ADR-0034); uninstall and
//   "delete all my data" keep/erase local data; fixed a repeated sentence in the ES summary.
// 1.2.1: the on-demand public IP lookup command is now `kongtrol ip` (`kongtrol check` still works).
// 1.2.0: the public IP check is off by default (opt-in), runs only on tunnel connect/disconnect and
//   on demand, and is no longer turned on by the kill switch or DNS guard.
// 1.1.0: one-person project, author receives no user data; dashboard control cookie disclosed;
//   the dashboard now self-hosts its fonts and logos (served from the binary at /brand/).
//
// Every statement about the app was checked against the code on this date (source of truth for
// connections: docs/SECURITY.md "Network connections Kongtrol makes"):
// - public-IP check: internal/security/leaktest.go (DefaultPublicIPEndpoint), opt-in via
//   Config.PublicIPCheckEnabled (internal/config/publicip.go); triggered in cmd/kongtrol/daemon_runtime.go
//   (connect/disconnect) and on demand (cmd_check.go, POST /api/v1/security/publicip/check).
// - pre-connect reachability + health checks: internal/app/profile_service.go, cmd/kongtrol/daemon_runtime.go
//   (TCP to host:port; DNS query for connectivitycheck.gstatic.com sent only to the tunnel's DNS).
// - VPN server lookup for the kill switch allow-list: internal/app/killswitch_service.go.
// - DNS lookups of policy domains: internal/app/profile_service.go, internal/monitor/policyresolver.go.
// - new version notice: opt-in `updates.check_enabled` (default false), GET api.github.com/repos/DerotLuna/
//   vpn-kongtrol/releases/latest at most every 24h, headers User-Agent "kongtrol/<version>" + Accept only (ADR-0034).
// - local files: audit-YYYY-MM-DD.log (internal/security/audit.go, retention_days 0 = today and
//   yesterday), history.json, daemon.log, update-check.json, configs/ (internal/profileimport, 0600),
//   helper audit log (internal/privhelper/platform_*.go).
// - dashboard: session cookie kongtrol_session (internal/api/session.go); localStorage
//   kongtrol-dashboard-lang / kongtrol-dashboard-theme (web/ui/src/i18n/lang.ts, theme/theme.ts),
//   sessionStorage kongtrol.noticeSeq (web/ui/src/data/home.ts).
// - website: only `kglang` (landing/src/hooks/usePreferences.ts); CSP connect-src 'self' (landing/public/_headers).
// - no telemetry / analytics / crash reporting: repo-wide search, no such code.
// Re-check these when the code changes (AGENTS.md "Docs stay in sync").
import type { Lang } from '../../i18n'
import type { LegalDocContent } from '../../components/sections/LegalDoc'
import { pathFor } from '../../site'
import { GITHUB_REPO } from '../../links'
import IC from '../../components/ui/InlineCode'

const VERSION = '1.6.0'
const UPDATED = '2026-11-19'
const ext = (href: string, text: string) => <a href={href} target="_blank" rel="noreferrer">{text}</a>
const issues = ext(`${GITHUB_REPO}/issues`, 'github.com/DerotLuna/vpn-kongtrol/issues')
const GITHUB_PRIVACY = 'https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement'

const es: LegalDocContent = {
  label: '$ cat PRIVACY.md',
  title: 'Política de privacidad',
  sub: 'Qué se queda en tu equipo, qué sale de él y hacia quién.',
  version: VERSION,
  updated: UPDATED,
  session: 'kongtrol:privacidad',
  summaryBadge: 'En resumen',
  summary: [
    <>Kongtrol es gratis y de código abierto. Lo hace una persona; no hay empresa detrás.</>,
    <>Funciona en tu equipo. No hay cuentas ni servidores de Kongtrol. El autor no recibe ningún dato tuyo.</>,
    <>Tu configuración y tus contraseñas se quedan en tu equipo.</>,
    <>No hay telemetría, analíticas ni informes de errores.</>,
    <>Por su cuenta, Kongtrol solo habla con tu servidor VPN y con el DNS. Consultar tu IP pública y el aviso de
      versión nueva son opcionales y vienen apagados.</>,
    <>Este sitio no usa cookies ni analíticas.</>,
  ],
  toc: 'Contenido',
  goto: 'Ir a',
  versionLabel: 'Versión',
  updatedLabel: 'actualizada el',
  contactTitle: 'Contacto',
  contact: <>para preguntas sobre privacidad, abre un issue en {issues}. Para reportar una vulnerabilidad, usa la{' '}
    <a href={pathFor('seguridad', 'es')}>política de seguridad</a>.</>,
  sections: [
    {
      id: 'local', num: '1', label: 'Datos en tu equipo', tmux: 'local', title: 'Lo que se queda en tu equipo',
      body: [<>Kongtrol guarda sus datos en tu carpeta de usuario (<IC c="~/.kongtrol/" />). Nada de esto se envía a
        nadie:</>],
      list: [
        <><IC c="kongtrol.yaml" />: tus perfiles VPN, grupos, reglas y ajustes. Nunca guarda contraseñas.</>,
        <><IC c="configs/" />: los archivos <IC c=".conf" /> y <IC c=".ovpn" /> que importas. Pueden contener
          claves de tu VPN.</>,
        <><IC c="preferences.json" />: ajustes de este equipo (idioma, favoritos, puerto del panel).</>,
        <>Registro de auditoría (<IC c="audit-AAAA-MM-DD.log" />): qué se conectó y qué cambió en la seguridad. Un
          archivo por día, firmado para notar si alguien lo cambia. Por defecto se guardan 30 días y como máximo 500 MB; lo más viejo se borra solo.</>,
        <>Historial de métricas, registros del programa, el estado del aviso de versión y el token del panel
          (<IC c="api-token" />).</>,
        <>Contraseñas y la clave que firma el registro: en el almacén de credenciales del sistema (Administrador de
          credenciales de Windows, Llavero de macOS, Secret Service o GNOME Keyring en Linux).</>,
        <>Si instalas el ayudante del sistema, este guarda su propio registro local de lo que se le pide.</>,
        // feature:recent_connections
        <>Destinos recientes (Windows y Linux): a qué direcciones se conectó este equipo en la última hora, por qué
          VPN salió cada una y qué programa la abrió (<IC c="kongtrol traffic recent" />). Solo vive en la memoria de
          Kongtrol: nunca se escribe en disco ni en los registros y no sale del equipo. Guarda como mucho 500
          destinos y olvida cada uno una hora después de verlo. Se borra al apagarlo o al cerrar Kongtrol. Viene
          encendido porque muestra lo mismo que <IC c="netstat" /> en tu equipo.</>,
      ],
      after: [<>Al desinstalar, estos datos se quedan, salvo que pidas borrarlos. Para borrar todo: en el panel,
        Ajustes › «Borrar todos mis datos», o <IC c="kongtrol uninstall --data-only" />. Para desinstalar y
        borrarlos a la vez: <IC c="kongtrol uninstall --purge" />. Más en la{' '}
        <a href={`${pathFor('guia', 'es')}#uninstall`}>guía</a>.</>],
    },
    {
      id: 'conexiones', num: '2', label: 'Conexiones propias', tmux: 'conexiones', title: 'Conexiones que Kongtrol hace por su cuenta',
      body: ['Además del tráfico de tus propias VPN, Kongtrol hace solo estas conexiones. Ninguna lleva tu configuración, tus contraseñas ni datos de uso.'],
      list: [
        <><strong>Comprobar tu servidor VPN.</strong> Antes de conectar, y mientras el túnel está activo, abre y
          cierra una conexión breve con el servidor y el puerto de tu perfil. Así te avisa si no responde. Solo va al
          servidor que tú configuraste.</>,
        <><strong>DNS.</strong> Busca los dominios de tus reglas, para saber por qué túnel enviarlos, y el nombre de
          tu servidor VPN cuando aplica el kill switch. Para comprobar el DNS dentro del túnel, pregunta por{' '}
          <IC c="connectivitycheck.gstatic.com" /> solo al DNS de tu VPN; nada se envía a Google.</>,
        <><strong>Tu IP pública (opcional, apagada por defecto).</strong> Solo si la activas: una petición HTTPS a{' '}
          <IC c="api.ipify.org" /> (un servicio de terceros) u otro servicio que elijas. Sirve para mostrar tu IP
          pública y detectar fugas. Ese servicio ve tu IP, como cualquier sitio que visitas. Se hace al conectar o
          desconectar un túnel (como mucho una vez cada 30 segundos) y cuando lo pides (<IC c="kongtrol ip" />,
          «Revisar» en el panel o la bandeja). Nunca de forma periódica. El kill switch y el DNS guard no la
          necesitan.</>,
        <><strong>Aviso de versión nueva (opcional, apagado por defecto).</strong> Solo si lo activas: como mucho una
          vez cada 24 horas, una petición HTTPS a GitHub{' '}
          (<IC c="api.github.com/repos/DerotLuna/vpn-kongtrol/releases/latest" />). Solo lleva las cabeceras{' '}
          <IC c="User-Agent: kongtrol/<versión>" /> y <IC c="Accept" />: sin cuenta, token, cookies, perfiles ni
          las IP de tus VPN. GitHub ve tu IP pública, la hora y esa versión (aplica su{' '}
          {ext(GITHUB_PRIVACY, 'declaración de privacidad')}). Te avisa una vez por versión; nunca descarga ni
          instala nada.</>,
      ],
      after: [<>Kongtrol <strong>no</strong> envía telemetría, analíticas ni informes de errores (revisado en el
        código fuente el {UPDATED}). Tus clientes VPN (NordVPN, FortiClient, etc.) hacen sus propias conexiones;
        eso depende de cada proveedor y de su política.</>],
    },
    {
      id: 'panel', num: '3', label: 'Panel web', tmux: 'panel', title: 'Panel web local',
      body: [<>El panel funciona solo en tu equipo (<IC c="127.0.0.1" />). Kongtrol lo abre con un código de un solo
        uso y el navegador recibe una cookie de sesión local (<IC c="kongtrol_session" />, HttpOnly). La cookie no
        contiene tu token ni sale de tu equipo; la sesión caduca sola.</>,
      <>En el navegador, el panel solo guarda tu idioma y tu tema, y en la pestaña abierta el último aviso que
        mostró. Sus tipografías y logotipos vienen dentro del programa: abrirlo no hace ninguna petición a Google
        ni a otro tercero.</>],
    },
    {
      id: 'sitio', num: '4', label: 'Este sitio web', tmux: 'sitio', title: 'Este sitio web (kongtrol.app)',
      body: [<>Este sitio no usa cookies, analíticas ni píxeles de seguimiento. Sirve sus propias tipografías y tu
        navegador no pide nada a terceros mientras lo visitas. Solo guarda el idioma que elijas
        (<IC c="kglang" />).</>,
      <>El sitio está alojado en Cloudflare. Cloudflare procesa los datos técnicos de cada visita (como tu IP y tu
        navegador) para servir y proteger el sitio, según su{' '}
        {ext('https://www.cloudflare.com/privacypolicy/', 'política de privacidad')}.</>,
      <>Las descargas y el código están en GitHub. Al usarlos aplica la{' '}
        {ext(GITHUB_PRIVACY, 'declaración de privacidad de GitHub')}.</>],
    },
    {
      id: 'control', num: '5', label: 'Tu control', tmux: 'control', title: 'Cómo controlarlo',
      body: ['Qué puedes cambiar:'],
      list: [
        <><strong>IP pública:</strong> apagada por defecto. Actívala o apágala en el panel (Seguridad), en el
          asistente o con <IC c="kongtrol config public-ip on|off" />. Elige otro servicio con{' '}
          <IC c="security.public_ip_check.endpoints" /> (solo https).</>,
        <><strong>Aviso de versión nueva:</strong> apagado por defecto. Actívalo o apágalo en el panel (Ajustes),
          en <IC c="kongtrol init" /> o con <IC c="kongtrol config update-check on|off" />.</>,
        <><strong>Destinos recientes:</strong> encendido por defecto, solo en memoria. Apágalo con{' '}
          <IC c="kongtrol config recent-connections off" />: la lista se borra al momento.</>,
        <><strong>Comprobar tu servidor VPN:</strong> solo se hace en perfiles con un servidor configurado.</>,
        <><strong>Tus datos:</strong> bórralos cuando quieras (sección 1).</>,
      ],
    },
    {
      id: 'cambios', num: '6', label: 'Cambios', tmux: 'cambios', title: 'Cambios a esta política',
      body: [<>Esta política cambia cuando cambia lo que hace el programa. Cada versión lleva su número y fecha
        arriba, y su historial queda en el repositorio de GitHub.</>],
    },
  ],
}

const en: LegalDocContent = {
  label: '$ cat PRIVACY.md',
  title: 'Privacy policy',
  sub: 'What stays on your computer, what leaves it, and to whom.',
  version: VERSION,
  updated: UPDATED,
  session: 'kongtrol:privacy',
  summaryBadge: 'In short',
  summary: [
    <>Kongtrol is free and open source. One person makes it; there is no company behind it.</>,
    <>It runs on your computer. There are no accounts and no Kongtrol servers. The author receives none of your data.</>,
    <>Your configuration and passwords stay on your computer.</>,
    <>There is no telemetry, no analytics and no crash reporting.</>,
    <>On its own, Kongtrol only talks to your VPN server and to DNS. Looking up your public IP and the new version
      notice are optional and off by default.</>,
    <>This website uses no cookies and no analytics.</>,
  ],
  toc: 'Contents',
  goto: 'Jump to',
  versionLabel: 'Version',
  updatedLabel: 'updated',
  contactTitle: 'Contact',
  contact: <>for privacy questions, open an issue at {issues}. To report a vulnerability, use the{' '}
    <a href={pathFor('seguridad', 'en')}>security policy</a>.</>,
  sections: [
    {
      id: 'local', num: '1', label: 'Data on your computer', tmux: 'local', title: 'What stays on your computer',
      body: [<>Kongtrol keeps its data in your user folder (<IC c="~/.kongtrol/" />). None of it is sent to
        anyone:</>],
      list: [
        <><IC c="kongtrol.yaml" />: your VPN profiles, groups, rules and settings. It never holds passwords.</>,
        <><IC c="configs/" />: the <IC c=".conf" /> and <IC c=".ovpn" /> files you import. They can contain
          your VPN keys.</>,
        <><IC c="preferences.json" />: this computer's settings (language, favorites, dashboard port).</>,
        <>Audit log (<IC c="audit-YYYY-MM-DD.log" />): what connected and what changed in security. One file per
          day, signed so tampering shows. By default 30 days are kept, at most 500 MB; the oldest is deleted on its own.</>,
        <>Metrics history, the program's logs, the new version notice state and the dashboard token
          (<IC c="api-token" />).</>,
        <>Passwords and the key that signs the log: in the operating system's credential store (Windows Credential
          Manager, macOS Keychain, Secret Service or GNOME Keyring on Linux).</>,
        <>If you install the system helper, it keeps its own local log of what it is asked to do.</>,
        // feature:recent_connections
        <>Recent destinations (Windows and Linux): which addresses this computer connected to in the last hour,
          which VPN each one went through and which program opened it (<IC c="kongtrol traffic recent" />). It lives
          only in Kongtrol's memory: never written to disk or logs, and it never leaves the computer. It holds at most
          500 destinations and forgets each one an hour after it was last seen. It is erased when you turn it off or
          close Kongtrol. On by default because it shows the same thing as <IC c="netstat" /> on your computer.</>,
      ],
      after: [<>When you uninstall, this data stays unless you ask to delete it. To erase everything: in the
        dashboard, Settings › "Delete all my data", or <IC c="kongtrol uninstall --data-only" />. To uninstall and
        delete it at once: <IC c="kongtrol uninstall --purge" />. More in the{' '}
        <a href={`${pathFor('guia', 'en')}#uninstall`}>guide</a>.</>],
    },
    {
      id: 'conexiones', num: '2', label: 'Its own connections', tmux: 'connections', title: 'Connections Kongtrol makes on its own',
      body: ['Besides the traffic of your own VPNs, Kongtrol makes only these connections. None of them carries your configuration, passwords or usage data.'],
      list: [
        <><strong>Checking your VPN server.</strong> Before connecting, and while the tunnel is up, it opens and
          closes a short connection to your profile's server and port. That way it warns you if the server does
          not answer. It only goes to the server you configured.</>,
        <><strong>DNS.</strong> It looks up the domains in your rules, to know which tunnel to send them through, and
          your VPN server's name when the kill switch is applied. To check DNS inside the tunnel, it asks for{' '}
          <IC c="connectivitycheck.gstatic.com" /> only from your VPN's DNS server; nothing is sent to Google.</>,
        <><strong>Your public IP (optional, off by default).</strong> Only if you turn it on: one HTTPS request to{' '}
          <IC c="api.ipify.org" /> (a third-party service) or another service you choose. It shows your public IP and
          detects leaks. That service sees your IP, like any site you visit. It runs when a tunnel connects or
          disconnects (at most once every 30 seconds) and when you ask (<IC c="kongtrol ip" />, "Check" in the
          dashboard or the tray). Never periodically. The kill switch and DNS guard do not need it.</>,
        <><strong>New version notice (optional, off by default).</strong> Only if you turn it on: at most once every
          24 hours, one HTTPS request to GitHub{' '}
          (<IC c="api.github.com/repos/DerotLuna/vpn-kongtrol/releases/latest" />). It only carries the headers{' '}
          <IC c="User-Agent: kongtrol/<version>" /> and <IC c="Accept" />: no account, token, cookies, profiles or
          the IPs of your VPNs. GitHub sees your public IP, the time and that version (its{' '}
          {ext(GITHUB_PRIVACY, 'privacy statement')} applies). It tells you once per version; it never downloads or
          installs anything.</>,
      ],
      after: [<>Kongtrol does <strong>not</strong> send telemetry, analytics or crash reports (checked in the
        source code on {UPDATED}). Your VPN clients (NordVPN, FortiClient, etc.) make their own connections; that
        depends on each provider and its policy.</>],
    },
    {
      id: 'panel', num: '3', label: 'Web dashboard', tmux: 'dashboard', title: 'Local web dashboard',
      body: [<>The dashboard runs only on your computer (<IC c="127.0.0.1" />). Kongtrol opens it with a one-time
        code and the browser gets a local session cookie (<IC c="kongtrol_session" />, HttpOnly). The cookie does
        not hold your token and never leaves your computer; the session expires on its own.</>,
      <>In the browser, the dashboard only stores your language and theme, and, in the open tab, the last notice it
        showed. Its fonts and logos are built into the program: opening it makes no request to Google or any other
        third party.</>],
    },
    {
      id: 'sitio', num: '4', label: 'This website', tmux: 'website', title: 'This website (kongtrol.app)',
      body: [<>This site uses no cookies, analytics or tracking pixels. It serves its own fonts, and your browser
        asks no third party for anything while you visit. It only stores the language you pick
        (<IC c="kglang" />).</>,
      <>The site is hosted on Cloudflare. Cloudflare processes the technical data of each visit (such as your IP
        address and browser) to serve and protect the site, under its{' '}
        {ext('https://www.cloudflare.com/privacypolicy/', 'privacy policy')}.</>,
      <>Downloads and source code live on GitHub. Using them is covered by the{' '}
        {ext(GITHUB_PRIVACY, 'GitHub General Privacy Statement')}.</>],
    },
    {
      id: 'control', num: '5', label: 'Your control', tmux: 'control', title: 'How to control it',
      body: ['What you can change:'],
      list: [
        <><strong>Public IP:</strong> off by default. Turn it on or off in the dashboard (Security), in the setup
          wizard or with <IC c="kongtrol config public-ip on|off" />. Pick another service with{' '}
          <IC c="security.public_ip_check.endpoints" /> (https only).</>,
        <><strong>New version notice:</strong> off by default. Turn it on or off in the dashboard (Settings), in{' '}
          <IC c="kongtrol init" /> or with <IC c="kongtrol config update-check on|off" />.</>,
        <><strong>Recent destinations:</strong> on by default, memory only. Turn it off with{' '}
          <IC c="kongtrol config recent-connections off" />: the list is erased at once.</>,
        <><strong>Checking your VPN server:</strong> only runs for profiles with a server set.</>,
        <><strong>Your data:</strong> delete it whenever you want (section 1).</>,
      ],
    },
    {
      id: 'cambios', num: '6', label: 'Changes', tmux: 'changes', title: 'Changes to this policy',
      body: [<>This policy changes when what the program does changes. Each version carries its number and date at
        the top, and its history stays in the GitHub repository.</>],
    },
  ],
}

const doc: Record<Lang, LegalDocContent> = { es, en }
export default doc
