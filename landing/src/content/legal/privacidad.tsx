// Privacy policy — version 1.3.0 · 2026-11-18.
// 1.3.0: opt-in new version notice (GitHub releases API, off by default; ADR-0034); uninstall and
//   "delete all my data" keep/erase local data; fixed a repeated sentence in the ES summary.
// 1.2.1: the on-demand public IP lookup command is now `kongtrol ip` (`kongtrol check` still works).
// 1.2.0: the public IP check is off by default (opt-in), runs only on tunnel connect/disconnect and
//   on demand, and is no longer turned on by the kill switch or DNS guard.
// 1.1.0: one-person project, author receives no user data; dashboard control cookie disclosed;
//   the dashboard now self-hosts its fonts and logos (served from the binary at /brand/).
//
// Every statement about the app was checked against the code on this date:
// - public-IP check: internal/security/leaktest.go (DefaultPublicIPEndpoint), opt-in via
//   Config.PublicIPCheckEnabled (internal/config/publicip.go); triggered in cmd/kongtrol/daemon_runtime.go
//   (connect/disconnect) and on demand (cmd_check.go, POST /api/v1/security/publicip/check).
// - pre-connect reachability check: internal/app/profile_service.go (net.Dialer to host:port).
// - DNS lookups of policy domains: internal/app/profile_service.go, internal/monitor/policyresolver.go.
// - new version notice: opt-in `updates.check_enabled` (default false), GET api.github.com/repos/DerotLuna/
//   vpn-kongtrol/releases/latest at most every 24h, headers User-Agent "kongtrol/<version>" + Accept only (ADR-0034).
// - no telemetry / analytics / crash reporting: repo-wide search, no such code.
// - dashboard fonts/logos self-hosted: web/dashboard/*.html -> /brand/ (internal/api/brand_assets.go); dashboard localStorage: web/dashboard/*.js.
// Re-check these when the code changes (CLAUDE.md "Docs stay in sync").
import type { Lang } from '../../i18n'
import type { LegalDocContent } from '../../components/sections/LegalDoc'
import { pathFor } from '../../site'
import { GITHUB_REPO } from '../../links'
import IC from '../../components/ui/InlineCode'

const VERSION = '1.3.0'
const UPDATED = '2026-11-18'
const ext = (href: string, text: string) => <a href={href} target="_blank" rel="noreferrer">{text}</a>
const issues = ext(`${GITHUB_REPO}/issues`, 'github.com/DerotLuna/vpn-kongtrol/issues')

const es: LegalDocContent = {
  label: '$ cat PRIVACY.md',
  title: 'Política de privacidad',
  sub: 'Qué se queda en tu equipo, qué sale de él y hacia quién.',
  version: VERSION,
  updated: UPDATED,
  session: 'kongtrol:privacidad',
  summaryBadge: 'En resumen',
  summary: <>Kongtrol es una herramienta gratuita y de código abierto, hecha por una persona (no hay empresa detrás),
    que funciona en tu equipo. El autor no recibe ni procesa ningún dato tuyo: cada usuario controla los suyos. No hay
    cuentas, servidores propios, telemetría ni analíticas. Tu configuración y tus contraseñas se quedan en tu equipo. Las únicas conexiones que hace por su cuenta son: comprobar que tu servidor VPN responde antes de conectar
    y resolver los dominios de tus reglas. Consultar tu IP pública y el aviso de versión nueva son opcionales y
    vienen apagados. Este sitio no usa cookies ni analíticas.</>,
  toc: 'Contenido',
  goto: 'Ir a',
  versionLabel: 'Versión',
  contactTitle: 'Contacto',
  contact: <>para preguntas sobre privacidad, abre un issue en {issues}. Para reportar una vulnerabilidad, usa la{' '}
    <a href={pathFor('seguridad', 'es')}>política de seguridad</a>.</>,
  sections: [
    {
      id: 'local', num: '1', label: 'Datos en tu equipo', tmux: 'local', title: 'Lo que se queda en tu equipo',
      body: [<>Kongtrol guarda todo localmente, en tu carpeta de usuario. Nada de esto se envía a nadie:</>],
      list: [
        <><IC c="~/.kongtrol/kongtrol.yaml" />: tus perfiles VPN, grupos, reglas y ajustes. Nunca contiene contraseñas.</>,
        <><IC c="~/.kongtrol/preferences.json" />: preferencias de esta máquina (idioma, favoritos, puerto del panel).</>,
        <><IC c="~/.kongtrol/audit.log" />: registro de eventos (conexiones, cambios de seguridad), firmado con HMAC.</>,
        <>Historial de métricas, registros del servicio en segundo plano y el token del panel local (<IC c="~/.kongtrol/api-token" />).</>,
        <>Contraseñas y la clave de firma del registro: en el almacén de credenciales del sistema (Windows Credential
          Manager, Llavero de macOS, Secret Service/GNOME Keyring en Linux).</>,
      ],
      after: [<>Al desinstalar, estos datos se quedan, salvo que pidas borrarlos. Para borrar todo: en el panel,
        Ajustes › «Borrar todos mis datos», o <IC c="kongtrol uninstall --data-only" />. Para desinstalar y
        borrarlos a la vez: <IC c="kongtrol uninstall --purge" />. Ver la{' '}
        <a href={`${pathFor('guia', 'es')}#uninstall`}>guía</a>.</>],
    },
    {
      id: 'conexiones', num: '2', label: 'Conexiones propias', tmux: 'conexiones', title: 'Conexiones que Kongtrol hace por su cuenta',
      body: ['Además del tráfico de tus propias VPN, Kongtrol hace estas conexiones:'],
      list: [
        <><strong>Comprobación de IP pública (opcional, apagada por defecto)</strong> — solo si la activas, una petición
          HTTPS a <IC c="api.ipify.org" /> (servicio de terceros) u otro servicio que tú elijas, para mostrar tu IP
          pública y detectar fugas. Cuando la activas, tu IP es visible para ese servicio, como en cualquier sitio que
          visitas. Solo se hace al conectar o desconectar un túnel (como mucho una vez cada 30 segundos) y cuando lo
          pides (<IC c="kongtrol ip" />, «Revisar» en el panel o en la bandeja); nunca de forma periódica. El kill
          switch y el DNS guard no la necesitan. La petición sale por la ruta activa (tu VPN, si la hay).</>,
        <><strong>Comprobación previa al conectar</strong> — una conexión TCP breve al servidor y puerto de tu
          perfil VPN, para avisarte si no responde. Solo va al servidor que tú configuraste.</>,
        <><strong>Resolución DNS de tus reglas</strong> — Kongtrol resuelve los dominios de tus políticas con el DNS
          del sistema o el del túnel, para saber por qué túnel enviarlos.</>,
        <><strong>Aviso de versión nueva (opcional, apagado por defecto)</strong> — solo si lo activas, como mucho
          una vez cada 24 horas, una petición HTTPS a GitHub:{' '}
          <IC c="GET https://api.github.com/repos/DerotLuna/vpn-kongtrol/releases/latest" />. Solo lleva las
          cabeceras <IC c="User-Agent: kongtrol/<versión>" /> y <IC c="Accept" />. No envía cuenta, token, cookies,
          perfiles, las IP de tus VPN ni otros datos. GitHub ve tu IP pública, la hora y ese User-Agent (aplica la{' '}
          {ext('https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement', 'declaración de privacidad de GitHub')}).
          Espera como mucho 5 segundos e ignora las versiones de prueba. Si hay versión nueva, te avisa una vez;
          nunca descarga ni instala nada. Apagado, no hace ninguna petición.</>,
      ],
      after: [<>Kongtrol <strong>no</strong> envía telemetría, analíticas ni informes de errores (revisado en el
        código fuente el {UPDATED}). Tus clientes VPN (NordVPN, FortiClient, etc.)
        pueden hacer sus propias conexiones; eso depende de cada proveedor y de su política.</>],
    },
    {
      id: 'panel', num: '3', label: 'Panel web', tmux: 'panel', title: 'Panel web local',
      body: [<>El panel funciona en tu equipo (<IC c="127.0.0.1" />) y pide un token local para cada petición. Ese
        token viaja en una única cookie local (<IC c="kongtrol_control" />, HttpOnly) que nunca sale de tu equipo.
        Guarda en el almacenamiento local del navegador solo tu idioma, tema y el estado de la barra lateral.</>,
      <>El panel sirve sus tipografías y logotipos desde el propio binario: abrirlo no hace ninguna petición a
        Google ni a otro tercero.</>],
    },
    {
      id: 'sitio', num: '4', label: 'Este sitio web', tmux: 'sitio', title: 'Este sitio web (kongtrol.app)',
      body: [<>Este sitio no usa cookies, analíticas ni píxeles de seguimiento, y sirve sus propias tipografías (no
        carga nada de Google). Solo guarda en tu navegador el idioma o tema que elijas explícitamente
        (<IC c="kglang" />, <IC c="kgtheme" />).</>,
      <>El sitio está alojado en Cloudflare (Workers): Cloudflare procesa los datos técnicos de cada visita (como la IP y
        el navegador) para servir y proteger el sitio, según su{' '}
        {ext('https://www.cloudflare.com/privacypolicy/', 'política de privacidad')}.</>,
      <>Las descargas y el código están en GitHub; al usarlos aplica la{' '}
        {ext('https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement', 'declaración de privacidad de GitHub')}.</>],
    },
    {
      id: 'control', num: '5', label: 'Tu control', tmux: 'control', title: 'Cómo controlarlo',
      body: ['Qué puedes ajustar:'],
      list: [
        <>La comprobación de IP pública viene apagada. Actívala o apágala en el panel (Seguridad), en el asistente
          o con <IC c="kongtrol config public-ip on|off" /> (<IC c="security.public_ip_check.enabled" />). Elige
          otro servicio con <IC c="security.public_ip_check.endpoints" /> (solo https).</>,
        <>El aviso de versión nueva viene apagado. Actívalo o apágalo en el panel (Ajustes › «Aviso de versión
          nueva»), en <IC c="kongtrol init" /> o con <IC c="kongtrol config update-check on|off" />{' '}
          (<IC c="updates.check_enabled" />).</>,
        <>La comprobación previa al conectar solo se hace en perfiles que tienen un host configurado.</>,
      ],
    },
    {
      id: 'cambios', num: '6', label: 'Cambios', tmux: 'cambios', title: 'Cambios a esta política',
      body: [<>Esta política cambia cuando cambia lo que hace el código. Cada versión lleva su número y fecha arriba
        y su historial queda en el repositorio de GitHub.</>],
    },
  ],
}

const en: LegalDocContent = {
  label: '$ cat PRIVACY.md',
  title: 'Privacy policy',
  sub: 'What stays on your machine, what leaves it, and to whom.',
  version: VERSION,
  updated: UPDATED,
  session: 'kongtrol:privacy',
  summaryBadge: 'TL;DR',
  summary: <>Kongtrol is a free, open-source tool made by one person (there is no company behind it) that runs on
    your machine. The author receives and processes none of your data: each user controls their own. There are no
    accounts, no servers of its own, no telemetry and no analytics. Your configuration and passwords stay on your machine. The only connections it
    makes on its own are checking that your VPN server answers before connecting and resolving the domains in your
    rules. Looking up your public IP and the new version notice are optional and off by default. This website uses no cookies and no analytics.</>,
  toc: 'Contents',
  goto: 'Jump to',
  versionLabel: 'Version',
  contactTitle: 'Contact',
  contact: <>for privacy questions, open an issue at {issues}. To report a vulnerability, use the{' '}
    <a href={pathFor('seguridad', 'en')}>security policy</a>.</>,
  sections: [
    {
      id: 'local', num: '1', label: 'Data on your machine', tmux: 'local', title: 'What stays on your machine',
      body: [<>Kongtrol keeps everything locally, in your user folder. None of it is sent to anyone:</>],
      list: [
        <><IC c="~/.kongtrol/kongtrol.yaml" />: your VPN profiles, groups, rules and settings. It never contains passwords.</>,
        <><IC c="~/.kongtrol/preferences.json" />: this machine's preferences (language, favorites, dashboard port).</>,
        <><IC c="~/.kongtrol/audit.log" />: event log (connections, security changes), HMAC-signed.</>,
        <>Metrics history, background-service logs and the local dashboard token (<IC c="~/.kongtrol/api-token" />).</>,
        <>Passwords and the log signing key: in the operating system's credential store (Windows Credential Manager,
          macOS Keychain, Secret Service/GNOME Keyring on Linux).</>,
      ],
      after: [<>When you uninstall, this data stays unless you ask to delete it. To erase everything: in the
        dashboard, Settings › "Delete all my data", or <IC c="kongtrol uninstall --data-only" />. To uninstall and
        delete it at once: <IC c="kongtrol uninstall --purge" />. See the{' '}
        <a href={`${pathFor('guia', 'en')}#uninstall`}>guide</a>.</>],
    },
    {
      id: 'conexiones', num: '2', label: 'Its own connections', tmux: 'connections', title: 'Connections Kongtrol makes on its own',
      body: ['Besides the traffic of your own VPNs, Kongtrol makes these connections:'],
      list: [
        <><strong>Public IP check (optional, off by default)</strong> — only if you turn it on, an HTTPS request to
          <IC c="api.ipify.org" /> (a third-party service) or another service you choose, to show your public IP and
          detect leaks. When you enable it, your IP is visible to that service, like any site you visit. It runs only
          when a tunnel connects or disconnects (at most once every 30 seconds) and when you ask
          (<IC c="kongtrol ip" />, "Check" in the dashboard or the tray); never periodically. The kill switch and
          DNS guard do not need it. The request leaves through the active route (your VPN, if there is one).</>,
        <><strong>Pre-connect check</strong> — a short TCP connection to your VPN profile's server and port, to warn
          you if it doesn't answer. It only goes to the server you configured.</>,
        <><strong>DNS resolution for your rules</strong> — Kongtrol resolves your policy domains with the system's
          or the tunnel's DNS, to know which tunnel to send them through.</>,
        <><strong>New version notice (optional, off by default)</strong> — only if you turn it on, at most once every
          24 hours, one HTTPS request to GitHub:{' '}
          <IC c="GET https://api.github.com/repos/DerotLuna/vpn-kongtrol/releases/latest" />. It only carries the
          headers <IC c="User-Agent: kongtrol/<version>" /> and <IC c="Accept" />. No account, token, cookies,
          profiles, the IPs of your VPNs or any other data. GitHub sees your public IP, the time and that User-Agent
          (the {ext('https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement', 'GitHub General Privacy Statement')} applies).
          It waits at most 5 seconds and ignores pre-releases. If there is a new version, it tells you once; it never
          downloads or installs anything. When off, it makes no request at all.</>,
      ],
      after: [<>Kongtrol does <strong>not</strong> send telemetry, analytics or crash reports (checked in the
        source code on {UPDATED}). Your VPN clients (NordVPN, FortiClient, etc.) may make their own
        connections; that depends on each provider and its policy.</>],
    },
    {
      id: 'panel', num: '3', label: 'Web dashboard', tmux: 'dashboard', title: 'Local web dashboard',
      body: [<>The dashboard runs on your machine (<IC c="127.0.0.1" />) and requires a local token on every request.
        That token travels in a single local cookie (<IC c="kongtrol_control" />, HttpOnly) that never leaves your
        machine. It keeps only your language, theme and sidebar state in the browser's local storage.</>,
      <>The dashboard serves its fonts and logos from the binary itself: opening it makes no request to Google or
        any other third party.</>],
    },
    {
      id: 'sitio', num: '4', label: 'This website', tmux: 'website', title: 'This website (kongtrol.app)',
      body: [<>This site uses no cookies, analytics or tracking pixels, and serves its own fonts (nothing is loaded
        from Google). It only stores in your browser the language or theme you explicitly pick (<IC c="kglang" />,{' '}
        <IC c="kgtheme" />).</>,
      <>The site is hosted on Cloudflare (Workers): Cloudflare processes the technical data of each visit (such as IP
        address and browser) to serve and protect the site, under its{' '}
        {ext('https://www.cloudflare.com/privacypolicy/', 'privacy policy')}.</>,
      <>Downloads and source code live on GitHub; using them is covered by the{' '}
        {ext('https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement', 'GitHub General Privacy Statement')}.</>],
    },
    {
      id: 'control', num: '5', label: 'Your control', tmux: 'control', title: 'How to control it',
      body: ['What you can adjust:'],
      list: [
        <>The public IP check is off by default. Turn it on or off in the dashboard (Security), in the setup wizard or
          with <IC c="kongtrol config public-ip on|off" /> (<IC c="security.public_ip_check.enabled" />). Pick another
          service with <IC c="security.public_ip_check.endpoints" /> (https only).</>,
        <>The new version notice is off by default. Turn it on or off in the dashboard (Settings › "New version
          notice"), in <IC c="kongtrol init" /> or with <IC c="kongtrol config update-check on|off" />{' '}
          (<IC c="updates.check_enabled" />).</>,
        <>The pre-connect check only runs for profiles that have a host set.</>,
      ],
    },
    {
      id: 'cambios', num: '6', label: 'Changes', tmux: 'changes', title: 'Changes to this policy',
      body: [<>This policy changes when what the code does changes. Each version carries its number and date at the
        top, and its history stays in the GitHub repository.</>],
    },
  ],
}

const doc: Record<Lang, LegalDocContent> = { es, en }
export default doc
