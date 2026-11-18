const API = '';

const I18N = {
  es: {
    'nav.overview': 'Resumen',
    'nav.studio': 'Estudio',
    'nav.security': 'Seguridad',
    'nav.auditLog': 'Registro de Auditoría',
    'nav.settings': 'Configuración',
    'page.overview.title': 'Resumen',
    'ws.connecting': 'conectando…',
    'ws.live': 'en vivo',
    'ws.reconnecting': 'reconectando…',
    'sections.tunnels': 'Túneles',
    'sections.trafficMap': 'Mapa de Tráfico y Resolución de Políticas',
    'sections.baseNetwork': 'Egreso de Red Base',
    'sections.security': 'Controles de Seguridad',
    'sections.activeRoutes': 'Rutas Activas (En Vivo)',
    'table.profile': 'Perfil',
    'table.status': 'Estado',
    'table.assignedIp': 'IP asignada',
    'table.uptime': 'Tiempo activo',
    'table.sent': '↑ Enviado',
    'table.received': '↓ Recibido',
    'table.latency': 'Latencia',
    'table.trend': 'Tendencia',
    'table.actions': 'Acciones',
    'sections.trafficOverTime': 'Tráfico en el Tiempo',
    'chart.noData': 'Sin túneles conectados — los gráficos se completan al conectar un túnel.',
    'chart.sent': 'Enviado',
    'chart.received': 'Recibido',
    'stat.reconnects': 'Reconexiones',
    'stat.jitter': 'Jitter',
    'stat.totalConnected': 'Tiempo conectado (total)',
    'policies.policy': 'Política',
    'policies.match': 'Match',
    'policies.vpnProfile': 'Perfil VPN',
    'policies.resolvedIps': 'IPs resueltas',
    'base.connectedTunnels': 'Túneles conectados',
    'base.defaultEgress': 'Salida predeterminada',
    'base.gateway': 'Gateway',
    'base.localIps': 'IPs locales',
    'base.publicIp': 'IP pública',
    'base.note': 'El tráfico sin match usa tu ruta predeterminada.',
    'security.killSwitch': 'Kill Switch',
    'security.dnsGuard': 'DNS Guard',
    'security.leakCheck': 'Leak Check',
    'security.publicIp': 'IP pública',
    'security.lastChecked': 'Última revisión',
    'security.manage': 'Gestionar →',
    'bulk.connectAll': 'Conectar todo',
    'bulk.disconnectAll': 'Desconectar todo',
    'sections.groups': 'Grupos',
    'empty.noGroups': 'No hay grupos configurados — créalos en Estudio.',
    'unit.profiles': 'perfiles',
    'toast.groupConnecting': 'Conectando grupo "{name}"…',
    'toast.groupDisconnecting': 'Grupo "{name}" desconectado.',
    'toast.groupActionFailed': 'La acción sobre el grupo falló',
    'toast.daemonLost': 'Se perdió la conexión con el daemon — reintentando…',
    'toast.daemonRestored': 'Conexión con el daemon restablecida',
    'routes.filter': 'Filtro',
    'routes.all': 'Todas las rutas',
    'routes.default': 'Ruta predeterminada',
    'routes.policyPrefix': 'Política',
    'routes.defaultTag': 'predeterminada',
    'routes.profilePrefix': 'Perfil',
    'routes.originPrefix': 'Origen',
    'origin.server': 'servidor',
    'origin.policy': 'política',
    'origin.vpn': 'vpn',
    'tunnelRoutes.toggle': 'Qué sale por este túnel ({n})',
    'tunnelRoutes.toggleAll': 'Qué sale por este túnel (todo)',
    'tunnelRoutes.title': 'Qué sale por este túnel',
    'tunnelRoutes.server': 'Del servidor ({n})',
    'tunnelRoutes.policy': 'Política {name} ({n})',
    'tunnelRoutes.vpn': 'De la VPN ({n})',
    'tunnelRoutes.serverExplain': 'Las rutas las envía el servidor de la VPN · no hace falta agregar IPs · si falta algo, agrégalo con una política',
    'tunnelRoutes.allTraffic': 'Todo tu tráfico sale por este túnel',
    'tunnelRoutes.none': 'Nada visible por este túnel todavía.',
    'tunnelRoutes.rest': 'Todo lo demás → internet normal ({iface})',
    'tunnelRoutes.restUnknown': 'Todo lo demás → internet normal',
    'tunnelRoutes.restTunnel': 'Todo lo demás → {name} (todo tu tráfico)',
    'resolve.placeholder': 'IP/dominio, URL, política/perfil, o app:proceso — ej. claude.ai, https://chatgpt.com, us-content, app:chrome.exe',
    'resolve.check': 'Revisar',
    'resolve.clear': 'Limpiar',
    'resolve.defaultRoute': 'ruta predeterminada',
    'resolve.noMatchingPolicy': 'sin política coincidente',
    'resolve.matchedRule': '{lhs} → {via} ({rule})',
    'resolve.unmatchedRule': '{lhs} → {defaultRoute} ({reason}){publicIp}',
    'empty.noTunnels': 'No hay túneles configurados.',
    'empty.noPolicies': 'No hay políticas configuradas.',
    'empty.noPoliciesForFilter': 'No hay políticas para este filtro.',
    'empty.noRoutesForFilter': 'No hay rutas para este filtro.',
    'actions.connect': 'Conectar',
    'actions.disconnect': 'Desconectar',
    'actions.cancel': 'Cancelar',
    'actions.reload': 'Recargar',
    'toast.tunnelReloading': 'Recargando "{name}" con la config del disco…',
    'toast.tunnelRestartRequired': 'Túnel "{name}" no está registrado en el daemon en ejecución (probablemente añadido a mano) — requiere reiniciar el proceso completo.',
    'alerts.tunnelReloadFailed': 'Error al recargar el túnel',
    'status.connected': 'conectado',
    'status.connecting': 'conectando',
    'status.disconnected': 'desconectado',
    'status.error': 'error',
    'tunnel.lastError': 'Último error ({time})',
    'lastErrorKey.notInstalled': 'Instala el ayudante una vez: pulsa Instalar ayudante arriba.',
    'lastErrorKey.outdated': 'El ayudante está desactualizado. Actualízalo: pulsa Actualizar ayudante en el aviso de arriba.',
    'lastErrorKey.denied': 'El ayudante no acepta a tu usuario. Reinstálalo desde tu cuenta: kongtrol service install',
    'lastErrorKey.wgNotManaged': 'Este túnel WireGuard no coincide con ningún perfil tuyo (el archivo .conf debe llamarse igual que el túnel). Desconéctalo en la app de WireGuard, o corrige el perfil y reintenta.',
    'lastErrorKey.notWireGuard': 'Ese nombre pertenece a algo que no es un túnel WireGuard, así que Kongtrol no lo toca. Revisa el nombre del túnel en el perfil.',
    'lastErrorKey.fileOwner': 'El archivo de configuración de la VPN no es tuyo. Cópialo a tu carpeta de usuario (por ejemplo ~/.kongtrol) y apunta el perfil a la copia.',
    'lastErrorKey.fileWritable': 'Otros usuarios pueden modificar el archivo de configuración de la VPN. Muévelo a tu carpeta de usuario (por ejemplo ~/.kongtrol) y reintenta.',
    'lastErrorKey.sessionOwner': 'Esa conexión OpenVPN la inició otro usuario. Desconéctala desde la sesión de ese usuario.',
    'helper.banner': 'Instala el ayudante una vez: kongtrol service install',
    'helper.install': 'Instalar ayudante',
    'helper.installing': 'Acepta el permiso de administrador en la ventana que aparece…',
    'helper.installed': 'Ayudante listo · ya no hace falta volver a instalarlo',
    'helper.failed': 'No se pudo instalar el ayudante.',
    'helper.failedHint': 'Qué hacer: pulsa Instalar ayudante otra vez y acepta el permiso.',
    'helper.bannerOutdated': 'El ayudante está desactualizado: Kongtrol no puede comprobar si los servidores de tus VPN responden. Actualízalo una vez (un permiso de administrador).',
    'helper.update': 'Actualizar ayudante',
    'health.unverified': 'sin verificar',
    'attach.attached': '{app} ya conectado · kongtrol lo toma',
    'attach.auto': 'Conectado con {app}',
    'attach.auto_location': 'Conectado con {app} ({location})',
    'attach.user': 'Conectado en {app} · kongtrol lo toma',
    'attach.waiting_user': 'Esperando conexión en {app}…',
    'attach.automation_failed': 'Auto-conexión falló ({reason}) · conéctate en {app}',
    'attach.waiting_disconnect': 'Desconéctate en {app} · esperando…',
    'attach.dropped': '{app} se cayó · reconéctate en la app',
    'health.no_response': 'Sin respuesta del servidor · Kongtrol reintenta solo; si sigue, revisa tu internet o avisa a quien administra esa VPN',
    'health.idle': 'Sin tráfico reciente · normal si no la estás usando',
    'health.unknown': 'No se puede comprobar si el servidor responde',
    'health.unknown.fix': 'si algo no carga, desconéctala y vuelve a conectarla',
    'badge.disabled': 'DESHABILITADO',
    'badge.active': 'ACTIVO',
    'badge.armed': 'ARMADO',
    'badge.off': 'INACTIVO',
    'badge.idle': 'INACTIVO',
    'badge.leak': 'FUGA',
    'badge.clean': 'LIMPIO',
    'badge.pending': 'PENDIENTE',
    'badge.error': 'ERROR',
    'badge.unavailable': 'NO DISPONIBLE',
    'filter.showing': 'Mostrando {shown} de {total} políticas para: {query}',
    'filter.showingAll': 'Mostrando {total} políticas',
    'network.metric': 'métrica',
    'network.onLink': 'enlace directo',
    'common.refresh': 'Actualizar',
    'common.publicIpSuffix': ' (IP pública {ip})',
    'unit.ips': 'IPs',
  },
  en: {
    'nav.overview': 'Overview',
    'nav.studio': 'Studio',
    'nav.security': 'Security',
    'nav.auditLog': 'Audit Log',
    'nav.settings': 'Settings',
    'page.overview.title': 'Overview',
    'ws.connecting': 'connecting…',
    'ws.live': 'live',
    'ws.reconnecting': 'reconnecting…',
    'sections.tunnels': 'Tunnels',
    'sections.trafficMap': 'Traffic Map & Policy Resolver',
    'sections.baseNetwork': 'Base Network Egress',
    'sections.security': 'Security Controls',
    'sections.activeRoutes': 'Active Routes (Live)',
    'table.profile': 'Profile',
    'table.status': 'Status',
    'table.assignedIp': 'Assigned IP',
    'table.uptime': 'Uptime',
    'table.sent': '↑ Sent',
    'table.received': '↓ Received',
    'table.latency': 'Latency',
    'table.trend': 'Trend',
    'table.actions': 'Actions',
    'sections.trafficOverTime': 'Traffic Over Time',
    'chart.noData': 'No connected tunnels — charts fill in once a tunnel connects.',
    'chart.sent': 'Sent',
    'chart.received': 'Received',
    'stat.reconnects': 'Reconnects',
    'stat.jitter': 'Jitter',
    'stat.totalConnected': 'Total time connected',
    'policies.policy': 'Policy',
    'policies.match': 'Match',
    'policies.vpnProfile': 'VPN Profile',
    'policies.resolvedIps': 'Resolved IPs',
    'base.connectedTunnels': 'Connected tunnels',
    'base.defaultEgress': 'Default egress',
    'base.gateway': 'Gateway',
    'base.localIps': 'Local IPs',
    'base.publicIp': 'Public IP',
    'base.note': 'Unmatched traffic follows your default route.',
    'security.killSwitch': 'Kill Switch',
    'security.dnsGuard': 'DNS Guard',
    'security.leakCheck': 'Leak Check',
    'security.publicIp': 'Public IP',
    'security.lastChecked': 'Last checked',
    'security.manage': 'Manage →',
    'bulk.connectAll': 'Connect All',
    'bulk.disconnectAll': 'Disconnect All',
    'sections.groups': 'Groups',
    'empty.noGroups': 'No groups configured — create some in Studio.',
    'unit.profiles': 'profiles',
    'toast.groupConnecting': 'Connecting group "{name}"…',
    'toast.groupDisconnecting': 'Group "{name}" disconnected.',
    'toast.groupActionFailed': 'Group action failed',
    'toast.daemonLost': 'Lost connection to the daemon — retrying…',
    'toast.daemonRestored': 'Daemon connection restored',
    'routes.filter': 'Filter',
    'routes.all': 'All routes',
    'routes.default': 'Default route',
    'routes.policyPrefix': 'Policy',
    'routes.defaultTag': 'default',
    'routes.profilePrefix': 'Profile',
    'routes.originPrefix': 'Origin',
    'origin.server': 'server',
    'origin.policy': 'policy',
    'origin.vpn': 'vpn',
    'tunnelRoutes.toggle': 'What goes through this tunnel ({n})',
    'tunnelRoutes.toggleAll': 'What goes through this tunnel (everything)',
    'tunnelRoutes.title': 'What goes through this tunnel',
    'tunnelRoutes.server': 'From the server ({n})',
    'tunnelRoutes.policy': 'Policy {name} ({n})',
    'tunnelRoutes.vpn': 'From the VPN ({n})',
    'tunnelRoutes.serverExplain': 'Routes are sent by the VPN server · no need to add IPs · if something is missing, add it with a policy',
    'tunnelRoutes.allTraffic': 'All your traffic goes through this tunnel',
    'tunnelRoutes.none': 'Nothing visible through this tunnel yet.',
    'tunnelRoutes.rest': 'Everything else → regular internet ({iface})',
    'tunnelRoutes.restUnknown': 'Everything else → regular internet',
    'tunnelRoutes.restTunnel': 'Everything else → {name} (all your traffic)',
    'resolve.placeholder': 'IP/domain, URL, policy/profile, or app:process — e.g. claude.ai, https://chatgpt.com, us-content, app:chrome.exe',
    'resolve.check': 'Check',
    'resolve.clear': 'Clear',
    'resolve.defaultRoute': 'default route',
    'resolve.noMatchingPolicy': 'no matching policy',
    'resolve.matchedRule': '{lhs} → {via} ({rule})',
    'resolve.unmatchedRule': '{lhs} → {defaultRoute} ({reason}){publicIp}',
    'empty.noTunnels': 'No tunnels configured.',
    'empty.noPolicies': 'No policies configured.',
    'empty.noPoliciesForFilter': 'No policies match this filter.',
    'empty.noRoutesForFilter': 'No routes for this filter.',
    'actions.connect': 'Connect',
    'actions.disconnect': 'Disconnect',
    'actions.cancel': 'Cancel',
    'actions.reload': 'Reload',
    'toast.tunnelReloading': 'Reloading "{name}" with config from disk…',
    'toast.tunnelRestartRequired': 'Tunnel "{name}" is not registered with the running daemon (likely added by a hand edit) — needs a full process restart.',
    'alerts.tunnelReloadFailed': 'Failed to reload tunnel',
    'status.connected': 'connected',
    'status.connecting': 'connecting',
    'status.disconnected': 'disconnected',
    'status.error': 'error',
    'tunnel.lastError': 'Last error ({time})',
    'lastErrorKey.notInstalled': 'Install the helper once: click Install helper above.',
    'lastErrorKey.outdated': 'The helper is out of date. Update it: click Update helper in the notice above.',
    'lastErrorKey.denied': 'The helper does not accept your user. Reinstall it from your account: kongtrol service install',
    'lastErrorKey.wgNotManaged': 'This WireGuard tunnel does not match any of your profiles (the .conf file name must equal the tunnel name). Disconnect it in the WireGuard app, or fix the profile and retry.',
    'lastErrorKey.notWireGuard': 'That name belongs to something that is not a WireGuard tunnel, so Kongtrol will not touch it. Check the profile tunnel name.',
    'lastErrorKey.fileOwner': 'The VPN config file is not yours. Copy it into your user folder (e.g. ~/.kongtrol) and point the profile at the copy.',
    'lastErrorKey.fileWritable': 'Other users can change the VPN config file. Move it into your user folder (e.g. ~/.kongtrol) and retry.',
    'lastErrorKey.sessionOwner': 'That OpenVPN connection was started by another user. Disconnect it from that user session.',
    'helper.banner': 'Install the helper once: kongtrol service install',
    'helper.install': 'Install helper',
    'helper.installing': 'Accept the admin permission in the window that appears…',
    'helper.installed': 'Helper ready · no need to install it again',
    'helper.failed': 'Could not install the helper.',
    'helper.failedHint': 'What to do: click Install helper again and accept the permission.',
    'helper.bannerOutdated': "The helper is out of date: Kongtrol can't check whether your VPN servers answer. Update it once (one admin prompt).",
    'helper.update': 'Update helper',
    'health.unverified': 'not verified',
    'attach.attached': '{app} already connected · kongtrol takes over',
    'attach.auto': 'Connected with {app}',
    'attach.auto_location': 'Connected with {app} ({location})',
    'attach.user': 'Connected in {app} · kongtrol takes over',
    'attach.waiting_user': 'Waiting for connection in {app}…',
    'attach.automation_failed': 'Auto-connect failed ({reason}) · connect in {app}',
    'attach.waiting_disconnect': 'Disconnect in {app} · waiting…',
    'attach.dropped': '{app} dropped · reconnect in the app',
    'health.no_response': 'No answer from the server · Kongtrol keeps retrying; if it persists, check your internet or tell whoever runs that VPN',
    'health.idle': 'No recent traffic · normal if you are not using it',
    'health.unknown': "Can't check whether the server answers",
    'health.unknown.fix': "if something doesn't load, disconnect and connect it again",
    'badge.disabled': 'DISABLED',
    'badge.active': 'ACTIVE',
    'badge.armed': 'ARMED',
    'badge.off': 'OFF',
    'badge.idle': 'IDLE',
    'badge.leak': 'LEAK',
    'badge.clean': 'CLEAN',
    'badge.pending': 'PENDING',
    'badge.error': 'ERROR',
    'badge.unavailable': 'UNAVAILABLE',
    'filter.showing': 'Showing {shown} of {total} policies for: {query}',
    'filter.showingAll': 'Showing {total} policies',
    'network.metric': 'metric',
    'network.onLink': 'on-link',
    'common.refresh': 'Refresh',
    'common.publicIpSuffix': ' (public IP {ip})',
    'unit.ips': 'IPs',
  },
};

let ws = null;
let wsReconnectTimer = null;
let currentPolicies = [];
let currentRoutes = [];
let tunnelRoutes = { tunnels: [] }; // GET /api/v1/tunnels/routes
const openRouteDrawers = new Set();
let lastPolicyList = [];
let lastPublicIP = '';
let publicIPEnabled = false;
let currentOverview = null;
let currentPolicyFilter = '';
let lastTunnels = {};
let metricsHistory = {};
const HISTORY_MAX_SAMPLES = 120;
const tunnelHistory = {};

// t/setLang/resolveInitialLang/applyStaticTranslations/initLangToggle/esc
// live in the shared i18n.js — this page only supplies the I18N dict above
// and the re-render hook below (called at the end of every setLang()).
function onLangChange() {
  renderPoliciesFromCache();
  renderRoutes();
  renderTrafficCharts();
  if (lastHelperNeedsInstall && !helperInstalling) renderHelperBanner(true);
}

// ── Privileged helper (one-click install) ───────────────────────────────────

// Tunnel last_error_key → this page's translated "what to do" line.
const LAST_ERROR_KEYS = {
  'service.error.not_installed': 'lastErrorKey.notInstalled',
  'service.error.outdated': 'lastErrorKey.outdated',
  'service.error.denied': 'lastErrorKey.denied',
  'privops.denied.wg_not_managed': 'lastErrorKey.wgNotManaged',
  'privops.denied.not_wireguard': 'lastErrorKey.notWireGuard',
  'privops.denied.file_owner': 'lastErrorKey.fileOwner',
  'privops.denied.file_writable': 'lastErrorKey.fileWritable',
  'privops.denied.session_owner': 'lastErrorKey.sessionOwner',
};

function lastErrorText(m) {
  const key = LAST_ERROR_KEYS[m.last_error_key];
  return key ? t(key) : (m.last_error || '');
}

let helperInstalling = false;
let lastHelperNeedsInstall = false;
// lastHelperOutdated: the helper answers with an older protocol. The banner
// then says "update" once for the whole page, and tunnel rows only add a
// short "sin verificar" (unverifiedTag) instead of the long sentence.
let lastHelperOutdated = false;

async function refreshHelperStatus() {
  if (helperInstalling) return;
  try {
    const res = await fetch(`${API}/api/v1/helper/status`);
    if (!res.ok) return;
    const st = await res.json();
    lastHelperNeedsInstall = !!st.needs_install;
    lastHelperOutdated = !!st.outdated;
    renderHelperBanner(lastHelperNeedsInstall);
  } catch (_) {}
}

function helperBannerEl() {
  let el = document.getElementById('helper-banner');
  if (!el) {
    el = document.createElement('div');
    el.id = 'helper-banner';
    el.className = 'helper-banner';
    el.setAttribute('role', 'status');
    el.hidden = true;
    document.querySelector('main')?.prepend(el);
  }
  return el;
}

// renderHelperBanner shows the "install the helper" banner; state may carry
// a busy flag or a translated error message + hint from the daemon.
function renderHelperBanner(show, state = {}) {
  const el = helperBannerEl();
  el.hidden = !show;
  if (!show) return;
  const busy = !!state.busy;
  el.classList.toggle('busy', busy);
  el.classList.toggle('failed', !!state.hint);
  const outdated = lastHelperOutdated;
  el.innerHTML = `
    <div class="helper-banner-text">
      <strong>${esc(state.message || t(outdated ? 'helper.bannerOutdated' : 'helper.banner'))}</strong>
      ${state.hint ? `<span>${esc(state.hint)}</span>` : ''}
    </div>
    <button type="button" class="action helper-install-btn"${busy ? ' disabled' : ''}>${esc(t(outdated ? 'helper.update' : 'helper.install'))}</button>`;
  el.querySelector('button').addEventListener('click', installHelper);
}

// installHelper asks the daemon to install the helper: the OS shows one
// permission prompt, so this can take minutes.
async function installHelper() {
  if (helperInstalling) return;
  helperInstalling = true;
  renderHelperBanner(true, { busy: true, message: t('helper.installing') });
  let res = null;
  let body = {};
  try {
    res = await fetch(`${API}/api/v1/helper/install`, {
      method: 'POST',
      headers: { 'X-Kongtrol-Lang': lang },
    });
    body = await res.json().catch(() => ({}));
  } catch (_) {}
  helperInstalling = false;
  if (res && res.ok && !body.needs_install) {
    lastHelperNeedsInstall = false;
    lastHelperOutdated = false;
    renderHelperBanner(false);
    showToast(t('helper.installed'), 'success');
    refreshTunnels();
    refreshHelperStatus(); // confirm at once, not on the next poll
    return;
  }
  const message = body.error || t('helper.failed');
  const hint = body.hint || t('helper.failedHint');
  renderHelperBanner(true, { message, hint });
  showToast(message, 'error', 8000, hint, body.detail || '');
}

// ── WebSocket live metrics ──────────────────────────────────────────────────

function connectWS() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  ws = new WebSocket(`${proto}://${location.host}/api/v1/ws/metrics`);

  ws.onopen = () => {
    setWSStatus(true);
    clearTimeout(wsReconnectTimer);
  };

  ws.onmessage = (e) => {
    try {
      renderTunnels(JSON.parse(e.data));
    } catch (_) {}
  };

  ws.onclose = () => {
    setWSStatus(false);
    wsReconnectTimer = setTimeout(connectWS, 3000);
  };

  ws.onerror = () => ws.close();
}

let wsWasConnected = null; // null = not yet known (initial load), avoids a spurious toast on first connect

function setWSStatus(connected) {
  const dot = document.getElementById('connection-indicator');
  dot.className = 'indicator ' + (connected ? 'connected' : 'disconnected');
  dot.title = connected ? t('ws.live') : t('ws.reconnecting');

  if (wsWasConnected === true && !connected) {
    showToast(t('toast.daemonLost'), 'error', 8000);
  } else if (wsWasConnected === false && connected) {
    showToast(t('toast.daemonRestored'), 'success');
  }
  wsWasConnected = connected;
}

// ── Tunnel table ────────────────────────────────────────────────────────────

async function refreshTunnels() {
  try {
    const res = await fetch(`${API}/api/v1/tunnels`);
    if (!res.ok) return;
    const list = await res.json();
    const map = {};
    (list || []).forEach((m) => { map[m.Name] = m; });
    renderTunnels(map);
  } catch (_) {}
}

function recordTunnelSamples(tunnels) {
  Object.entries(tunnels || {}).forEach(([name, m]) => {
    const status = (m.Status || 'disconnected').toLowerCase();
    if (status !== 'connected') return;
    if (!tunnelHistory[name]) tunnelHistory[name] = { latency: [], sent: [], recv: [], lastSent: null, lastRecv: null, lastLatency: null };
    const h = tunnelHistory[name];

    // BytesSent/BytesReceived are cumulative counters, so plotting them
    // directly only ever rises (or sits flat) — never a meaningful shape.
    // Chart per-tick deltas (throughput) instead, clamped to >=0 in case a
    // tunnel reconnected and the counter reset.
    const sentTotal = m.BytesSent || 0;
    const recvTotal = m.BytesReceived || 0;
    const sentDelta = h.lastSent === null ? 0 : Math.max(0, sentTotal - h.lastSent);
    const recvDelta = h.lastRecv === null ? 0 : Math.max(0, recvTotal - h.lastRecv);
    h.lastSent = sentTotal;
    h.lastRecv = recvTotal;

    // LatencyMS only gets a fresh reading once per health-check interval
    // (default 30s, see cmd/kongtrol/main.go healthCheckProfile), while
    // this runs on every ~5s WS push. Pushing the same stale value 5-6
    // times in a row before each real change would flood the sparkline's
    // fixed-size window with flat duplicate runs even when latency has
    // genuinely moved recently — so only record a point when the reading
    // actually changes (plus the very first one), keeping the visible
    // window representative of real variation instead of repeats.
    const latency = m.LatencyMS || 0;
    if (h.lastLatency === null || latency !== h.lastLatency) {
      h.latency.push(latency);
      if (h.latency.length > HISTORY_MAX_SAMPLES) h.latency.shift();
    }
    h.lastLatency = latency;

    h.sent.push(sentDelta);
    h.recv.push(recvDelta);
    if (h.sent.length > HISTORY_MAX_SAMPLES) h.sent.shift();
    if (h.recv.length > HISTORY_MAX_SAMPLES) h.recv.shift();
  });
}

function renderTunnels(tunnels, opts = {}) {
  const tbody = document.getElementById('tunnels-body');
  lastTunnels = tunnels || {};
  // Re-renders (route drawer toggle, routes refresh) must not add samples.
  if (opts.record !== false) recordTunnelSamples(lastTunnels);

  if (!tunnels || Object.keys(tunnels).length === 0) {
    tbody.innerHTML = `<tr><td colspan="9" class="empty">${t('empty.noTunnels')}</td></tr>`;
    renderTrafficCharts();
    return;
  }

  const rows = Object.entries(tunnels)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, m]) => {
      const status = (m.Status || 'disconnected').toLowerCase();
      const statusLabel = t(`status.${status}`);
      const isConnected = status === 'connected';
      const isConnecting = status === 'connecting';

      const dot = `<span class="status-dot ${status}"></span>`;
      // Why the last accepted (async) connect failed; the daemon clears it
      // when a new connect starts or the tunnel comes up. A known
      // last_error_key wins with its translated "what to do" line; the raw
      // error stays in the tooltip.
      const lastErr = !isConnected && (m.last_error || m.last_error_key)
        ? `<div class="tunnel-error" title="${esc(m.last_error || '').replace(/"/g, '&quot;')}">${esc(t('tunnel.lastError', { time: m.last_error_at ? new Date(m.last_error_at).toLocaleTimeString() : '' }))}: ${esc(lastErrorText(m))}</div>`
        : '';
      // Provider adapters (attach mode): how the tunnel came up, or what
      // the user must do in the provider app (see attachMessage).
      const attachMsg = attachMessage(m.attach);
      const attachLine = attachMsg
        ? `<div class="tunnel-attach${attachWaiting(m.attach) ? ' waiting' : ''}">${esc(attachMsg)}</div>`
        : '';
      // Does the server answer (WireGuard handshake, ADR-0030)? See
      // healthMessage; "no answer" uses the caution colour, other problems
      // are highlighted like a waiting attach.
      // "unknown" (can't verify, e.g. outdated helper) stays short on the
      // row — "conectado · sin verificar", the reason in the tooltip — and
      // the page-level helper banner explains the fix once.
      const healthMsg = healthMessage(m.health);
      const unverified = m.health?.state === 'unknown';
      const healthCls = m.health?.state === 'no_response' ? ' caution' : healthProblem(m.health) ? ' waiting' : '';
      const healthLine = healthMsg && !unverified
        ? `<div class="tunnel-attach${healthCls}">${esc(healthMsg)}</div>`
        : '';
      const unverifiedTag = unverified
        ? ` <span class="tunnel-unverified" title="${esc(healthMsg).replace(/"/g, '&quot;')}">· ${esc(t('health.unverified'))}</span>`
        : '';
      const tr = isConnected ? tunnelRoutesFor(name) : null;
      const routesToggle = tr
        ? `<div><button class="link-btn tunnel-routes-toggle" aria-expanded="${openRouteDrawers.has(name)}" onclick="toggleTunnelRoutes('${esc(name)}')">${esc(tr.all_traffic ? t('tunnelRoutes.toggleAll') : t('tunnelRoutes.toggle', { n: tr.total }))}</button></div>`
        : '';
      const drawer = tr && openRouteDrawers.has(name)
        ? `<tr class="tunnel-routes-row"><td colspan="9">${renderTunnelRoutesDrawer(tr)}</td></tr>`
        : '';
      const ip = m.AssignedIP || '<span class="muted">—</span>';
      const up = isConnected && m.UptimeSeconds > 0 ? formatDuration(m.UptimeSeconds) : '<span class="muted">—</span>';
      const sent = m.BytesSent > 0 ? formatBytes(m.BytesSent) : '<span class="muted">—</span>';
      const recv = m.BytesReceived > 0 ? formatBytes(m.BytesReceived) : '<span class="muted">—</span>';
      const lat = m.LatencyMS > 0 ? `${m.LatencyMS}ms` : '<span class="muted">—</span>';
      const trend = isConnected
        ? `<canvas class="latency-spark" data-profile="${esc(name)}" width="64" height="20"></canvas>`
        : '<span class="muted">—</span>';

      let btn = `<button class="action" onclick="connectVPN('${name}')">${t('actions.connect')}</button>`;
      if (isConnected) {
        btn = `<button class="action disconnect" onclick="disconnectVPN('${name}')">${t('actions.disconnect')}</button>
          <button class="action" onclick="reloadVPN('${name}')">${t('actions.reload')}</button>`;
      } else if (isConnecting) {
        btn = `<button class="action disconnect" onclick="cancelConnectVPN('${name}')">${t('actions.cancel')}</button>`;
      }

      return `<tr>
        <td>${name}</td>
        <td>${dot}${statusLabel}${unverifiedTag}${attachLine}${healthLine}${lastErr}${routesToggle}</td>
        <td>${ip}</td>
        <td>${up}</td>
        <td>${sent}</td>
        <td>${recv}</td>
        <td>${lat}</td>
        <td>${trend}</td>
        <td>${btn}</td>
      </tr>${drawer}`;
    });

  tbody.innerHTML = rows.join('');

  Object.entries(tunnels).forEach(([name, m]) => {
    if ((m.Status || '').toLowerCase() !== 'connected') return;
    const canvas = tbody.querySelector(`canvas.latency-spark[data-profile="${cssEsc(name)}"]`);
    const h = tunnelHistory[name];
    if (canvas && h) drawSparkline(canvas, h.latency, { color: themeColor('--signal') });
  });

  renderTrafficCharts();
}

function themeColor(varName) {
  return getComputedStyle(document.documentElement).getPropertyValue(varName).trim() || undefined;
}

function cssEsc(s) {
  return window.CSS && CSS.escape ? CSS.escape(s) : String(s).replace(/["\\]/g, '\\$&');
}

function renderTrafficCharts() {
  const container = document.getElementById('traffic-chart-list');
  if (!container) return;

  const connected = Object.entries(lastTunnels).filter(([, m]) => (m.Status || '').toLowerCase() === 'connected');
  if (connected.length === 0) {
    container.innerHTML = `<p class="empty">${t('chart.noData')}</p>`;
    return;
  }

  container.innerHTML = connected.map(([name]) => {
    const hist = metricsHistory[name];
    const tiles = hist
      ? `<div class="stat-tiles">
          <div class="stat-tile"><span class="stat-label">${t('stat.reconnects')}</span><span class="stat-value">${hist.reconnects ?? 0}</span></div>
          <div class="stat-tile"><span class="stat-label">${t('stat.jitter')}</span><span class="stat-value">${hist.jitter_ms ?? 0}ms</span></div>
          <div class="stat-tile"><span class="stat-label">${t('stat.totalConnected')}</span><span class="stat-value">${hist.uptime_seconds > 0 ? formatDuration(hist.uptime_seconds) : '—'}</span></div>
        </div>`
      : '';
    return `<div class="traffic-chart-item">
      <div class="chart-head">
        <span class="chart-name">${esc(name)}</span>
        <span class="chart-legend">
          <span><span class="legend-dot" style="background:${themeColor('--steel')}"></span>${t('chart.sent')}</span>
          <span><span class="legend-dot" style="background:${themeColor('--signal')}"></span>${t('chart.received')}</span>
        </span>
      </div>
      <canvas data-traffic-profile="${esc(name)}" width="600" height="64"></canvas>
      ${tiles}
    </div>`;
  }).join('');

  connected.forEach(([name]) => {
    const canvas = container.querySelector(`canvas[data-traffic-profile="${cssEsc(name)}"]`);
    const h = tunnelHistory[name];
    if (!canvas || !h) return;
    drawTimeSeries(canvas, [
      { data: h.sent, color: themeColor('--steel') },
      { data: h.recv, color: themeColor('--signal') },
    ]);
  });
}

async function refreshMetricsHistory() {
  try {
    const res = await fetch(`${API}/api/v1/metrics/history`);
    if (!res.ok) return;
    metricsHistory = await res.json();
    renderTrafficCharts();
  } catch (_) {}
}

// ── Security status ─────────────────────────────────────────────────────────

async function refreshSecurity() {
  try {
    const res = await fetch(`${API}/api/v1/security/status`);
    if (!res.ok) return;
    const data = await res.json();

    // Same states and labels as `kongtrol status` (see shellSecurityBadges).
    const badges = shellSecurityBadges(data);
    setBadge('ks-status', badges.ks.text || t(badges.ks.key), badges.ks.cls);
    setBadge('dns-status', t(badges.dns.key), badges.dns.cls);
    setBadge('leak-status', t(badges.leak.key), badges.leak.cls);

    // Opt-in public IP lookup: the IP, or "—" + a short hint while it is
    // off. This page never triggers a lookup by itself (only "Revisar").
    lastPublicIP = (data.public_ip_check_enabled && data.leak_check && data.leak_check.public_ip) || '';
    publicIPEnabled = !!data.public_ip_check_enabled;
    renderPublicIPField('public-ip', data);
    renderPublicIPField('base-public-ip', data);

    if (!data.public_ip_check_enabled) {
      document.getElementById('leak-time').textContent = '—';
      return;
    }

    if (data.leak_check) {
      const lk = data.leak_check;
      if (lk.checked_at) {
        const d = new Date(lk.checked_at);
        document.getElementById('leak-time').textContent = d.toLocaleTimeString(lang);
      } else {
        document.getElementById('leak-time').textContent = '—';
      }
      return;
    }

    document.getElementById('leak-time').textContent = '—';
  } catch (_) {}
}

function setBadge(id, label, stateClass) {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = label;
  el.className = 'badge ' + stateClass;
}

function knownPublicIP() {
  if (!publicIPEnabled) return '';
  if (lastPublicIP) return lastPublicIP;
  if (currentOverview && currentOverview.public_ip) return currentOverview.public_ip;
  return '';
}

// ── Route list ──────────────────────────────────────────────────────────────

async function refreshRoutes() {
  refreshTunnelRoutes();
  try {
    const res = await fetch(`${API}/api/v1/routes`);
    if (!res.ok) return;
    currentRoutes = await res.json();
    renderRoutes();
  } catch (_) {}
}

// What goes through each connected tunnel, grouped by origin (server /
// policy / vpn) — feeds the per-tunnel drawer and the routes card's
// profile/origin labels.
async function refreshTunnelRoutes() {
  try {
    const res = await fetch(`${API}/api/v1/tunnels/routes`);
    if (!res.ok) return;
    const data = await res.json();
    tunnelRoutes = data && Array.isArray(data.tunnels) ? data : { tunnels: [] };
    renderTunnels(lastTunnels, { record: false });
    syncRouteFilter(lastPolicyList);
    renderRoutes();
  } catch (_) {}
}

function tunnelRoutesFor(name) {
  return (tunnelRoutes.tunnels || []).find((tr) => tr.name === name && tr.connected) || null;
}

function toggleTunnelRoutes(name) {
  if (openRouteDrawers.has(name)) openRouteDrawers.delete(name);
  else openRouteDrawers.add(name);
  renderTunnels(lastTunnels, { record: false });
}

// routeOriginIndex maps "iface|destination" (and bare destination) to the
// tunnel that carries it, so the live routes card can label each row.
function routeOriginIndex() {
  const idx = {};
  (tunnelRoutes.tunnels || []).forEach((tr) => {
    (tr.routes || []).forEach((r) => {
      const info = { profile: tr.name, origin: r.origin, policy: r.policy || '' };
      idx[`${tr.interface || ''}|${r.destination}`] = info;
      if (!idx[r.destination]) idx[r.destination] = info;
    });
  });
  return idx;
}

function originTag(origin, policy) {
  const label = t(`origin.${origin}`);
  const title = origin === 'policy' && policy ? `${label}: ${policy}` : label;
  return `<span class="route-tag origin-${esc(origin)}" title="${esc(title)}">${esc(label)}</span>`;
}

function restLineText() {
  if (tunnelRoutes.rest_via_tunnel) return t('tunnelRoutes.restTunnel', { name: tunnelRoutes.rest_via_tunnel });
  if (tunnelRoutes.rest_via) return t('tunnelRoutes.rest', { iface: tunnelRoutes.rest_via });
  return t('tunnelRoutes.restUnknown');
}

function renderTunnelRoutesDrawer(tr) {
  const groups = [];
  (tr.routes || []).forEach((r) => {
    const last = groups[groups.length - 1];
    if (last && last.origin === r.origin && last.policy === (r.policy || '')) last.dests.push(r.destination);
    else groups.push({ origin: r.origin, policy: r.policy || '', dests: [r.destination] });
  });
  const parts = [`<div class="tunnel-routes-title">${esc(t('tunnelRoutes.title'))}${tr.interface ? ` <span class="muted">· ${esc(tr.interface)}</span>` : ''}</div>`];
  if (tr.all_traffic) parts.push(`<div class="tunnel-routes-all">${esc(t('tunnelRoutes.allTraffic'))}</div>`);
  if (!groups.length && !tr.all_traffic) parts.push(`<div class="muted">${esc(t('tunnelRoutes.none'))}</div>`);
  groups.forEach((g) => {
    let title = t('tunnelRoutes.vpn', { n: g.dests.length });
    if (g.origin === 'server') title = t('tunnelRoutes.server', { n: g.dests.length });
    if (g.origin === 'policy') title = t('tunnelRoutes.policy', { name: g.policy, n: g.dests.length });
    parts.push(`<div class="tunnel-routes-group">
      <div class="tunnel-routes-group-title">${originTag(g.origin, g.policy)} ${esc(title)}</div>
      ${g.origin === 'server' ? `<div class="tunnel-routes-explain">${esc(t('tunnelRoutes.serverExplain'))}</div>` : ''}
      <ul class="tunnel-routes-chips">${g.dests.map((d) => `<li>${esc(d)}</li>`).join('')}</ul>
    </div>`);
  });
  parts.push(`<div class="tunnel-routes-rest muted">${esc(restLineText())}</div>`);
  return `<div class="tunnel-routes">${parts.join('')}</div>`;
}

function renderRoutes() {
  const ul = document.getElementById('routes-list');
  const filterSel = document.getElementById('routes-filter');
  const filter = filterSel ? filterSel.value : 'all';
  const routes = Array.isArray(currentRoutes) ? currentRoutes : [];
  const idx = routeOriginIndex();
  const infoOf = (r) => idx[`${r.interface || ''}|${r.destination}`] || idx[r.destination] || null;

  const filtered = routes.filter((r) => {
    if (filter === 'all') return true;
    if (filter === 'default') return !!r.is_default;
    const info = infoOf(r);
    if (filter.startsWith('profile:')) return !!info && info.profile === filter.slice('profile:'.length);
    if (filter.startsWith('origin:')) return !!info && info.origin === filter.slice('origin:'.length);
    if (!filter.startsWith('policy:')) return true;
    return r.policy_name === filter.slice('policy:'.length);
  });

  if (filtered.length === 0) {
    ul.innerHTML = `<li><span class="empty">${t('empty.noRoutesForFilter')}</span></li>`;
    return;
  }

  ul.innerHTML = filtered.map((r) => {
    const info = infoOf(r);
    let tag = '';
    if (r.is_default) tag = `<span class="route-tag default">${t('routes.defaultTag')}</span>`;
    else if (info) tag = originTag(info.origin, info.policy) + (info.policy ? `<span class="route-tag policy">${esc(info.policy)}</span>` : '');
    else if (r.policy_name) tag = `<span class="route-tag policy">${esc(r.policy_name)}</span>`;
    const via = info ? `${info.profile} · ${r.interface || r.gateway || '—'}` : (r.interface || r.gateway || '—');
    return `<li>
      <span class="dest">${esc(r.destination)}</span>
      <span class="route-meta">
        ${tag}
        <span class="via">→ ${esc(via)}</span>
      </span>
    </li>`;
  }).join('');
}

async function refreshNetworkOverview() {
  try {
    const res = await fetch(`${API}/api/v1/network/overview`);
    if (!res.ok) return;
    currentOverview = await res.json();
    renderNetworkOverview();
  } catch (_) {}
}

function renderNetworkOverview() {
  const o = currentOverview || {};
  document.getElementById('base-connected').textContent = `${o.connected_tunnels || 0}`;

  const defaults = Array.isArray(o.default_routes) ? o.default_routes : [];
  const topDefault = defaults[0] || null;
  document.getElementById('base-egress').textContent = topDefault
    ? `${topDefault.interface || '—'} (${t('network.metric')} ${topDefault.metric ?? '—'})`
    : '—';
  document.getElementById('base-gateway').textContent = topDefault ? (topDefault.gateway || t('network.onLink')) : '—';

  const localIPs = Array.isArray(o.local_ips) ? o.local_ips : [];
  document.getElementById('base-local-ips').textContent = localIPs.length ? localIPs.join(', ') : '—';
  publicIPEnabled = !!o.public_ip_check_enabled;
  if (o.public_ip) lastPublicIP = o.public_ip;
  const ipData = { public_ip_check_enabled: publicIPEnabled, public_ip: knownPublicIP() };
  renderPublicIPField('base-public-ip', ipData);
  renderPublicIPField('public-ip', ipData);
}

function syncRouteFilter(policies) {
  lastPolicyList = policies || [];
  const sel = document.getElementById('routes-filter');
  if (!sel) return;
  const prev = sel.value || 'all';
  const names = [...new Set((policies || []).map((p) => p.name).filter(Boolean))].sort((a, b) => a.localeCompare(b));
  const profiles = (tunnelRoutes.tunnels || []).filter((tr) => tr.connected).map((tr) => tr.name).sort((a, b) => a.localeCompare(b));
  const options = [
    `<option value="all">${t('routes.all')}</option>`,
    `<option value="default">${t('routes.default')}</option>`,
    ...profiles.map((name) => `<option value="profile:${esc(name)}">${t('routes.profilePrefix')}: ${esc(name)}</option>`),
    ...['server', 'policy', 'vpn'].map((o) => `<option value="origin:${o}">${t('routes.originPrefix')}: ${t(`origin.${o}`)}</option>`),
    ...names.map((name) => `<option value="policy:${esc(name)}">${t('routes.policyPrefix')}: ${esc(name)}</option>`),
  ];
  sel.innerHTML = options.join('');
  const stillExists = Array.from(sel.options).some((o) => o.value === prev);
  sel.value = stillExists ? prev : 'all';
  enhanceSelect(sel);
}

function policyMatchesFilter(p, query) {
  if (!query) return true;
  const q = query.toLowerCase();
  const fields = [
    p.name || '',
    p.via || '',
    ...(p.domains || []),
    ...(p.ip_ranges || []),
    ...(p.apps || []),
  ].map((v) => String(v).toLowerCase());
  return fields.some((v) => v.includes(q));
}

// ── Traffic Map (policies + resolve) ───────────────────────────────────────

async function refreshPolicies() {
  try {
    const res = await fetch(`${API}/api/v1/policies`);
    if (!res.ok) return;
    const policies = await res.json();
    currentPolicies = policies || [];
    syncRouteFilter(currentPolicies);
    renderPoliciesFromCache();
  } catch (_) {}
}

async function resolveTarget() {
  const input = document.getElementById('resolve-input');
  const resultDiv = document.getElementById('resolve-result');
  const target = input.value.trim();

  if (!target) {
    resultDiv.style.display = 'none';
    return;
  }

  try {
    const data = await resolveOneTarget(target);
    if (!data) return;
    currentPolicyFilter = target;
    renderPoliciesFromCache();

    resultDiv.style.display = 'block';
    const lhs = data.app ? `app:${esc(data.app)}` : esc(data.target);

    if (data.matched) {
      resultDiv.className = 'resolve-result matched';
      resultDiv.innerHTML = `<strong>${t('resolve.matchedRule', { lhs, via: esc(data.via), rule: esc(data.rule) })}</strong>`;
    } else {
      resultDiv.className = 'resolve-result no-match';
      const publicIp = knownPublicIP() ? t('common.publicIpSuffix', { ip: esc(knownPublicIP()) }) : '';
      resultDiv.innerHTML = `<strong>${t('resolve.unmatchedRule', {
        lhs,
        defaultRoute: t('resolve.defaultRoute'),
        reason: t('resolve.noMatchingPolicy'),
        publicIp,
      })}</strong>`;
    }
  } catch (_) {
    resultDiv.style.display = 'none';
  }
}

function renderPoliciesFromCache() {
  const tbody = document.getElementById('policies-body');
  const note = document.getElementById('policy-filter-note');
  if (!Array.isArray(currentPolicies) || currentPolicies.length === 0) {
    tbody.innerHTML = `<tr><td colspan="4" class="empty">${t('empty.noPolicies')}</td></tr>`;
    note.textContent = '';
    return;
  }

  const filtered = currentPolicies.filter((p) => policyMatchesFilter(p, currentPolicyFilter));
  note.textContent = currentPolicyFilter
    ? t('filter.showing', { shown: filtered.length, total: currentPolicies.length, query: currentPolicyFilter })
    : t('filter.showingAll', { total: currentPolicies.length });

  if (filtered.length === 0) {
    tbody.innerHTML = `<tr><td colspan="4" class="empty">${t('empty.noPoliciesForFilter')}</td></tr>`;
    return;
  }

  tbody.innerHTML = filtered.map((p) => {
    const tags = [];
    (p.domains || []).forEach((d) => tags.push(`<span class="match-tag domain">${esc(d)}</span>`));
    (p.ip_ranges || []).forEach((r) => tags.push(`<span class="match-tag ip">${esc(r)}</span>`));
    const resolvedN = (p.resolved_cidrs || []).length;
    const resolvedClass = resolvedN > 0 ? 'resolved-count' : 'resolved-count zero';
    const resolvedText = resolvedN > 0 ? `${resolvedN} ${t('unit.ips')}` : '—';

    return `<tr>
      <td>${esc(p.name)}</td>
      <td><div class="match-tags">${tags.join('')}</div></td>
      <td><span class="via-profile">${esc(p.via)}</span></td>
      <td><span class="${resolvedClass}">${resolvedText}</span></td>
    </tr>`;
  }).join('');
}

function clearResolve() {
  currentPolicyFilter = '';
  const input = document.getElementById('resolve-input');
  const resultDiv = document.getElementById('resolve-result');
  input.value = '';
  resultDiv.style.display = 'none';
  renderPoliciesFromCache();
}

async function resolveOneTarget(raw) {
  const value = (raw || '').trim();
  if (!value) return null;
  const isApp = value.toLowerCase().startsWith('app:');
  const query = isApp
    ? `app=${encodeURIComponent(value.slice(4).trim())}`
    : `target=${encodeURIComponent(value)}`;
  const res = await fetch(`${API}/api/v1/resolve?${query}`);
  if (!res.ok) return null;
  return res.json();
}

document.getElementById('resolve-input').addEventListener('keydown', (e) => {
  if (e.key === 'Enter') resolveTarget();
});

document.getElementById('routes-filter').addEventListener('change', renderRoutes);

// ── VPN actions ─────────────────────────────────────────────────────────────

async function connectVPN(name) {
  try {
    await fetch(`${API}/api/v1/tunnels/${name}/connect`, { method: 'POST' });
    refreshRoutes();
  } catch (_) {
  } finally {
    setTimeout(() => {
      refreshRoutes();
      refreshNetworkOverview();
    }, 600);
  }
}

async function cancelConnectVPN(name) {
  try {
    await fetch(`${API}/api/v1/tunnels/${name}/cancel_connect`, { method: 'POST' });
  } catch (_) {
  } finally {
    setTimeout(() => {
      refreshRoutes();
      refreshNetworkOverview();
    }, 600);
  }
}

// attachMessage mirrors vpn.AttachState.MessageKey (internal/vpn/adapter.go):
// a provider adapter's attach phase as one translated line, or ''.
function attachMessage(a) {
  if (!a || !a.phase) return '';
  const vars = { app: a.app || '', location: a.location || '', reason: a.reason || '' };
  switch (a.phase) {
    case 'attached': return t('attach.attached', vars);
    case 'auto': return t(a.location ? 'attach.auto_location' : 'attach.auto', vars);
    case 'user': return t('attach.user', vars);
    case 'waiting_user': return t(a.reason ? 'attach.automation_failed' : 'attach.waiting_user', vars);
    case 'waiting_disconnect': return t('attach.waiting_disconnect', vars);
    case 'dropped': return t('attach.dropped', vars);
    default: return '';
  }
}

// healthMessage mirrors healthLine (cmd/kongtrol/attach_render.go) on
// vpn.TunnelHealth: whether a connected tunnel's server answers, or ''.
// "unknown" adds the helper's "what to do" line when the daemon sent one.
function healthMessage(h) {
  if (!h || !h.state) return '';
  switch (h.state) {
    case 'idle': return t('health.idle');
    case 'no_response': return t('health.no_response');
    case 'unknown': {
      const fix = LAST_ERROR_KEYS[h.reason_key];
      return `${t('health.unknown')} · ${fix ? t(fix) : t('health.unknown.fix')}`;
    }
    default: return '';
  }
}

// healthProblem mirrors vpn.TunnelHealth.Problem.
function healthProblem(h) {
  return !!h && (h.state === 'no_response' || h.state === 'unknown');
}

// attachWaiting mirrors vpn.AttachState.Waiting: the user has to act.
function attachWaiting(a) {
  return !!a && ['waiting_user', 'waiting_disconnect', 'dropped'].includes(a.phase);
}

async function disconnectVPN(name) {
  try {
    await fetch(`${API}/api/v1/tunnels/${name}/disconnect`, { method: 'POST' });
  } catch (_) {
  } finally {
    setTimeout(() => {
      refreshRoutes();
      refreshNetworkOverview();
    }, 600);
  }
}

// reloadVPN re-reads kongtrol.yaml from disk (hot-swapping the policy
// engine along the way) and, if this one tunnel is currently connected,
// restarts it in place — disconnect + reconnect through the daemon's real
// lifecycle — so routes/DNS/kill-switch edits made by hand for just this
// profile take effect without cycling its whole group. A 409 means the
// profile isn't registered with the running daemon (added by the hand
// edit) — that needs a full daemon restart, surfaced as an error toast
// rather than silently doing nothing.
async function reloadVPN(name) {
  try {
    const res = await fetch(`${API}/api/v1/tunnels/${encodeURIComponent(name)}/reload`, { method: 'POST' });
    if (res.status === 409) {
      showToast(t('toast.tunnelRestartRequired', { name }), 'error');
      return;
    }
    if (!res.ok) {
      const e = await res.json().catch(() => null);
      showApiError(t('alerts.tunnelReloadFailed'), e);
      return;
    }
    showToast(t('toast.tunnelReloading', { name }), 'info');
  } catch (_) {
    showToast(t('alerts.tunnelReloadFailed'), 'error');
  } finally {
    setTimeout(() => {
      refreshRoutes();
      refreshNetworkOverview();
    }, 600);
  }
}

// ── Groups quick-launch ─────────────────────────────────────────────────────

async function refreshGroupsQuicklaunch() {
  try {
    const res = await fetch(`${API}/api/v1/groups`);
    if (!res.ok) return;
    renderGroupsQuicklaunch(await res.json());
  } catch (_) {}
}

function renderGroupsQuicklaunch(groups) {
  const el = document.getElementById('groups-quicklaunch-list');
  if (!Array.isArray(groups) || groups.length === 0) {
    el.innerHTML = `<p class="empty">${t('empty.noGroups')}</p>`;
    return;
  }
  el.innerHTML = groups.map((g) => `
    <div class="group-chip">
      <span class="group-chip-name">${esc(g.name)}</span>
      <span class="group-chip-count">${(g.profiles || []).length} ${t('unit.profiles')}</span>
      <button class="action" onclick='connectGroupQuick(${JSON.stringify(g.name)})'>${t('actions.connect')}</button>
      <button class="action disconnect" onclick='disconnectGroupQuick(${JSON.stringify(g.name)})'>${t('actions.disconnect')}</button>
    </div>
  `).join('');
}

async function connectGroupQuick(name) {
  try {
    await fetch(`${API}/api/v1/groups/${encodeURIComponent(name)}/connect`, { method: 'POST' });
    showToast(t('toast.groupConnecting', { name }), 'info');
  } catch (_) {
    showToast(t('toast.groupActionFailed'), 'error');
  } finally {
    setTimeout(() => { refreshRoutes(); refreshNetworkOverview(); }, 600);
  }
}

async function disconnectGroupQuick(name) {
  const res = await fetch(`${API}/api/v1/groups/${encodeURIComponent(name)}/disconnect`, { method: 'POST' }).catch(() => null);
  if (!res || !res.ok) {
    showToast(t('toast.groupActionFailed'), 'error');
  } else {
    showToast(t('toast.groupDisconnecting', { name }), 'success');
  }
  setTimeout(() => { refreshRoutes(); refreshNetworkOverview(); }, 600);
}

async function connectAll() {
  const targets = Object.entries(lastTunnels).filter(([, m]) => (m.Status || '').toLowerCase() !== 'connected');
  await Promise.all(targets.map(([name]) => connectVPN(name)));
}

async function disconnectAll() {
  const targets = Object.entries(lastTunnels).filter(([, m]) => (m.Status || '').toLowerCase() === 'connected');
  await Promise.all(targets.map(([name]) => disconnectVPN(name)));
}

// ── Formatters ──────────────────────────────────────────────────────────────

function formatBytes(b) {
  if (!b || b === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(b) / Math.log(1024));
  return (b / Math.pow(1024, i)).toFixed(1) + '\u202f' + units[i];
}

function formatDuration(seconds) {
  if (!seconds || seconds < 1) return '< 1s';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) return `${h}h\u202f${m}m`;
  if (m > 0) return `${m}m\u202f${s}s`;
  return `${s}s`;
}

// ── Boot ────────────────────────────────────────────────────────────────────

setLang(lang);
initLangToggle();
enhanceAllSelects();
connectWS();
refreshRoutes();
refreshPolicies();
refreshNetworkOverview();
refreshSecurity();
refreshMetricsHistory();
refreshGroupsQuicklaunch();

// Fired by shell.js on every effective theme change: the toggle, a server
// preference, or the OS switching light/dark while following the system.
window.addEventListener('kongtrol:theme-changed', () => {
  setTimeout(() => {
    document.querySelectorAll('canvas.latency-spark').forEach((c) => {
      const h = tunnelHistory[c.dataset.profile];
      if (h) drawSparkline(c, h.latency, { color: themeColor('--signal') });
    });
    renderTrafficCharts();
  }, 30);
});

setInterval(refreshSecurity, 30_000);
setInterval(refreshRoutes, 30_000);
setInterval(refreshPolicies, 30_000);
setInterval(refreshNetworkOverview, 30_000);
setInterval(refreshMetricsHistory, 30_000);
setInterval(refreshGroupsQuicklaunch, 30_000);
refreshHelperStatus();
setInterval(refreshHelperStatus, 30_000);

// First run (daemon started with no kongtrol.yaml, see studio.js): send the
// user straight to the profile editor, where the setup guide is.
fetch('/api/v1/setup')
  .then((res) => (res.ok ? res.json() : null))
  .then((st) => {
    if (st && st.needs_setup) window.location.replace('/studio.html');
  })
  .catch(() => {});
