const API = '';

const I18N = {
  es: {
    'nav.overview': 'Resumen',
    'nav.studio': 'Estudio',
    'nav.security': 'Seguridad',
    'nav.auditLog': 'Registro de Auditoría',
    'nav.settings': 'Configuración',
    'nav.collapse': 'Contraer',
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
    'stat.samples': 'Muestras',
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
    'resolve.placeholder': 'IP/dominio, URL, política/perfil, o app:proceso — ej. claude.ai, https://chatgpt.com, us-content, app:chrome.exe',
    'resolve.check': 'Revisar',
    'resolve.clear': 'Limpiar',
    'resolve.defaultRoute': 'ruta predeterminada',
    'resolve.noMatchingPolicy': 'sin política coincidente',
    'resolve.matchedRule': '{lhs} → {via} ({rule})',
    'resolve.unmatchedRule': '{lhs} → {defaultRoute} ({reason}){publicIp}',
    'empty.connectingDaemon': 'Conectando con el daemon…',
    'empty.loading': 'Cargando…',
    'empty.noTunnels': 'No hay túneles configurados.',
    'empty.noPolicies': 'No hay políticas configuradas.',
    'empty.noPoliciesForFilter': 'No hay políticas para este filtro.',
    'empty.noRoutesForFilter': 'No hay rutas para este filtro.',
    'actions.connect': 'Conectar',
    'actions.disconnect': 'Desconectar',
    'actions.cancel': 'Cancelar',
    'status.connected': 'conectado',
    'status.connecting': 'conectando',
    'status.disconnected': 'desconectado',
    'status.error': 'error',
    'badge.disabled': 'DESACTIVADO',
    'badge.on': 'ACTIVO',
    'badge.idle': 'INACTIVO',
    'badge.leak': 'FUGA',
    'badge.clean': 'LIMPIO',
    'badge.pending': 'PENDIENTE',
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
    'nav.collapse': 'Collapse',
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
    'stat.samples': 'Samples',
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
    'resolve.placeholder': 'IP/domain, URL, policy/profile, or app:process — e.g. claude.ai, https://chatgpt.com, us-content, app:chrome.exe',
    'resolve.check': 'Check',
    'resolve.clear': 'Clear',
    'resolve.defaultRoute': 'default route',
    'resolve.noMatchingPolicy': 'no matching policy',
    'resolve.matchedRule': '{lhs} → {via} ({rule})',
    'resolve.unmatchedRule': '{lhs} → {defaultRoute} ({reason}){publicIp}',
    'empty.connectingDaemon': 'Connecting to daemon…',
    'empty.loading': 'Loading…',
    'empty.noTunnels': 'No tunnels configured.',
    'empty.noPolicies': 'No policies configured.',
    'empty.noPoliciesForFilter': 'No policies match this filter.',
    'empty.noRoutesForFilter': 'No routes for this filter.',
    'actions.connect': 'Connect',
    'actions.disconnect': 'Disconnect',
    'actions.cancel': 'Cancel',
    'status.connected': 'connected',
    'status.connecting': 'connecting',
    'status.disconnected': 'disconnected',
    'status.error': 'error',
    'badge.disabled': 'DISABLED',
    'badge.on': 'ON',
    'badge.idle': 'IDLE',
    'badge.leak': 'LEAK',
    'badge.clean': 'CLEAN',
    'badge.pending': 'PENDING',
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
let lastPublicIP = '';
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
    if (!tunnelHistory[name]) tunnelHistory[name] = { latency: [], sent: [], recv: [], lastSent: null, lastRecv: null };
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

    h.latency.push(m.LatencyMS || 0);
    h.sent.push(sentDelta);
    h.recv.push(recvDelta);
    if (h.latency.length > HISTORY_MAX_SAMPLES) h.latency.shift();
    if (h.sent.length > HISTORY_MAX_SAMPLES) h.sent.shift();
    if (h.recv.length > HISTORY_MAX_SAMPLES) h.recv.shift();
  });
}

function renderTunnels(tunnels) {
  const tbody = document.getElementById('tunnels-body');
  lastTunnels = tunnels || {};
  recordTunnelSamples(lastTunnels);

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
        btn = `<button class="action disconnect" onclick="disconnectVPN('${name}')">${t('actions.disconnect')}</button>`;
      } else if (isConnecting) {
        btn = `<button class="action disconnect" onclick="cancelConnectVPN('${name}')">${t('actions.cancel')}</button>`;
      }

      return `<tr>
        <td>${name}</td>
        <td>${dot}${statusLabel}</td>
        <td>${ip}</td>
        <td>${up}</td>
        <td>${sent}</td>
        <td>${recv}</td>
        <td>${lat}</td>
        <td>${trend}</td>
        <td>${btn}</td>
      </tr>`;
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
          <div class="stat-tile"><span class="stat-label">${t('stat.samples')}</span><span class="stat-value">${hist.samples ?? 0}</span></div>
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

    setFeatureBadge('ks-status', data.kill_switch_enabled, data.kill_switch);
    setFeatureBadge('dns-status', data.dns_guard_enabled, data.dns_guard);

    if (!data.leak_detection_enabled) {
      setBadge('leak-status', t('badge.disabled'), 'disabled');
      document.getElementById('public-ip').textContent = knownPublicIP() || '—';
      document.getElementById('base-public-ip').textContent = knownPublicIP() || '—';
      document.getElementById('leak-time').textContent = '—';
      return;
    }

    if (data.leak_check) {
      const lk = data.leak_check;
      setBadge('leak-status', lk.has_leak ? t('badge.leak') : t('badge.clean'), lk.has_leak ? 'leak' : 'ok');
      lastPublicIP = lk.public_ip || '';
      document.getElementById('public-ip').textContent = lastPublicIP || '—';
      document.getElementById('base-public-ip').textContent = lastPublicIP || '—';
      if (lk.checked_at) {
        const d = new Date(lk.checked_at);
        document.getElementById('leak-time').textContent = d.toLocaleTimeString(lang);
      } else {
        document.getElementById('leak-time').textContent = '—';
      }
      return;
    }

    setBadge('leak-status', t('badge.pending'), 'pending');
    document.getElementById('public-ip').textContent = knownPublicIP() || '—';
    document.getElementById('base-public-ip').textContent = knownPublicIP() || '—';
    document.getElementById('leak-time').textContent = '—';
  } catch (_) {}
}

function setFeatureBadge(id, enabled, active) {
  if (!enabled) {
    setBadge(id, t('badge.disabled'), 'disabled');
    return;
  }
  setBadge(id, active ? t('badge.on') : t('badge.idle'), active ? 'on' : 'off');
}

function setBadge(id, label, stateClass) {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = label;
  el.className = 'badge ' + stateClass;
}

function knownPublicIP() {
  if (lastPublicIP) return lastPublicIP;
  if (currentOverview && currentOverview.public_ip) return currentOverview.public_ip;
  return '';
}

// ── Route list ──────────────────────────────────────────────────────────────

async function refreshRoutes() {
  try {
    const res = await fetch(`${API}/api/v1/routes`);
    if (!res.ok) return;
    currentRoutes = await res.json();
    renderRoutes();
  } catch (_) {}
}

function renderRoutes() {
  const ul = document.getElementById('routes-list');
  const filterSel = document.getElementById('routes-filter');
  const filter = filterSel ? filterSel.value : 'all';
  const routes = Array.isArray(currentRoutes) ? currentRoutes : [];

  const filtered = routes.filter((r) => {
    if (filter === 'all') return true;
    if (filter === 'default') return !!r.is_default;
    if (!filter.startsWith('policy:')) return true;
    return r.policy_name === filter.slice('policy:'.length);
  });

  if (filtered.length === 0) {
    ul.innerHTML = `<li><span class="empty">${t('empty.noRoutesForFilter')}</span></li>`;
    return;
  }

  ul.innerHTML = filtered.map((r) => {
    const tag = r.is_default
      ? `<span class="route-tag default">${t('routes.defaultTag')}</span>`
      : (r.policy_name ? `<span class="route-tag policy">${esc(r.policy_name)}</span>` : '');
    return `<li>
      <span class="dest">${r.destination}</span>
      <span class="route-meta">
        ${tag}
        <span class="via">→ ${esc(r.interface || r.gateway || '—')}</span>
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
  if (o.public_ip) lastPublicIP = o.public_ip;
  const pub = knownPublicIP();
  document.getElementById('base-public-ip').textContent = pub || '—';
  document.getElementById('public-ip').textContent = pub || '—';
}

function syncRouteFilter(policies) {
  const sel = document.getElementById('routes-filter');
  if (!sel) return;
  const prev = sel.value || 'all';
  const names = [...new Set((policies || []).map((p) => p.name).filter(Boolean))].sort((a, b) => a.localeCompare(b));
  const options = [
    `<option value="all">${t('routes.all')}</option>`,
    `<option value="default">${t('routes.default')}</option>`,
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
      const publicIP = lastPublicIP ? t('common.publicIpSuffix', { ip: esc(lastPublicIP) }) : '';
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

document.getElementById('theme-toggle')?.addEventListener('click', () => {
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
