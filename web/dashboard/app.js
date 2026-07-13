/* VPN Kongtrol Dashboard — vanilla JS, WebSocket live feed, no build step */

const API = '';

// ── WebSocket live metrics ──────────────────────────────────────────────────

let ws = null;
let wsReconnectTimer = null;
let currentPolicies = [];
let currentRoutes = [];
let lastPublicIP = '';
let currentOverview = null;

function connectWS() {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  ws = new WebSocket(`${proto}://${location.host}/api/v1/ws/metrics`);

  ws.onopen = () => {
    setWSStatus(true);
    clearTimeout(wsReconnectTimer);
  };

  ws.onmessage = (e) => {
    try { renderTunnels(JSON.parse(e.data)); } catch (_) {}
  };

  ws.onclose = () => {
    setWSStatus(false);
    wsReconnectTimer = setTimeout(connectWS, 3000);
  };

  ws.onerror = () => ws.close();
}

function setWSStatus(connected) {
  const dot = document.getElementById('connection-indicator');
  const label = document.getElementById('ws-label');
  dot.className = 'indicator ' + (connected ? 'connected' : 'disconnected');
  label.textContent = connected ? 'live' : 'reconnecting…';
}

// ── Tunnel table ────────────────────────────────────────────────────────────

function renderTunnels(tunnels) {
  const tbody = document.getElementById('tunnels-body');

  if (!tunnels || Object.keys(tunnels).length === 0) {
    tbody.innerHTML = '<tr><td colspan="8" class="empty">No tunnels configured.</td></tr>';
    return;
  }

  const rows = Object.entries(tunnels)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, m]) => {
      const status = (m.Status || 'disconnected').toLowerCase();
      const isConnected = status === 'connected';
      const isConnecting = status === 'connecting';

      const dot   = `<span class="status-dot ${status}"></span>`;
      const ip    = m.AssignedIP || '<span class="muted">—</span>';
      const up    = isConnected && m.UptimeSeconds > 0 ? formatDuration(m.UptimeSeconds) : '<span class="muted">—</span>';
      const sent  = m.BytesSent    > 0 ? formatBytes(m.BytesSent)    : '<span class="muted">—</span>';
      const recv  = m.BytesReceived > 0 ? formatBytes(m.BytesReceived) : '<span class="muted">—</span>';
      const lat   = m.LatencyMS    > 0 ? `${m.LatencyMS}ms`          : '<span class="muted">—</span>';

      let btn = `<button class="action" onclick="connectVPN('${name}')">Connect</button>`;
      if (isConnected) {
        btn = `<button class="action disconnect" onclick="disconnectVPN('${name}')">Disconnect</button>`;
      } else if (isConnecting) {
        btn = `<button class="action disconnect" onclick="cancelConnectVPN('${name}')">Cancel</button>`;
      }

      return `<tr>
        <td>${name}</td>
        <td>${dot}${status}</td>
        <td>${ip}</td>
        <td>${up}</td>
        <td>${sent}</td>
        <td>${recv}</td>
        <td>${lat}</td>
        <td>${btn}</td>
      </tr>`;
    });

  tbody.innerHTML = rows.join('');
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
      setBadge('leak-status', 'DISABLED', 'disabled');
      document.getElementById('public-ip').textContent = knownPublicIP() || '—';
      document.getElementById('base-public-ip').textContent = knownPublicIP() || '—';
      document.getElementById('leak-time').textContent = '—';
      return;
    }

    if (data.leak_check) {
      const lk = data.leak_check;
      setBadge('leak-status', lk.has_leak ? 'LEAK' : 'CLEAN', lk.has_leak ? 'leak' : 'ok');
      lastPublicIP = lk.public_ip || '';
      document.getElementById('public-ip').textContent = lastPublicIP || '—';
      document.getElementById('base-public-ip').textContent = lastPublicIP || '—';
      if (lk.checked_at) {
        const d = new Date(lk.checked_at);
        document.getElementById('leak-time').textContent = d.toLocaleTimeString();
      } else {
        document.getElementById('leak-time').textContent = '—';
      }
      return;
    }
    setBadge('leak-status', 'PENDING', 'pending');
    document.getElementById('public-ip').textContent = knownPublicIP() || '—';
    document.getElementById('base-public-ip').textContent = knownPublicIP() || '—';
    document.getElementById('leak-time').textContent = '—';
  } catch (_) {}
}

function setFeatureBadge(id, enabled, active) {
  if (!enabled) {
    setBadge(id, 'DISABLED', 'disabled');
    return;
  }
  setBadge(id, active ? 'ON' : 'IDLE', active ? 'on' : 'off');
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
    ul.innerHTML = '<li><span class="empty">No routes for this filter.</span></li>';
    return;
  }

  ul.innerHTML = filtered.map(r => {
    const tag = r.is_default
      ? '<span class="route-tag default">default</span>'
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
  document.getElementById('base-egress').textContent = topDefault ? `${topDefault.interface || '—'} (metric ${topDefault.metric ?? '—'})` : '—';
  document.getElementById('base-gateway').textContent = topDefault ? (topDefault.gateway || 'on-link') : '—';

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
    '<option value="all">All routes</option>',
    '<option value="default">Default route</option>',
    ...names.map((name) => `<option value="policy:${esc(name)}">Policy: ${esc(name)}</option>`),
  ];
  sel.innerHTML = options.join('');
  const stillExists = Array.from(sel.options).some((o) => o.value === prev);
  sel.value = stillExists ? prev : 'all';
}

// ── Traffic Map (policies + resolve) ─────────────────────────────────────────

async function refreshPolicies() {
  try {
    const res = await fetch(`${API}/api/v1/policies`);
    if (!res.ok) return;
    const policies = await res.json();
    currentPolicies = policies || [];
    syncRouteFilter(currentPolicies);
    const tbody = document.getElementById('policies-body');

    if (!policies || policies.length === 0) {
      tbody.innerHTML = '<tr><td colspan="4" class="empty">No policies configured.</td></tr>';
      return;
    }

    tbody.innerHTML = policies.map(p => {
      // Build match tags
      const tags = [];
      (p.domains || []).forEach(d => tags.push(`<span class="match-tag domain">${esc(d)}</span>`));
      (p.ip_ranges || []).forEach(r => tags.push(`<span class="match-tag ip">${esc(r)}</span>`));

      const resolvedN = (p.resolved_cidrs || []).length;
      const resolvedClass = resolvedN > 0 ? 'resolved-count' : 'resolved-count zero';
      const resolvedText = resolvedN > 0 ? `${resolvedN} IPs` : '—';

      return `<tr>
        <td>${esc(p.name)}</td>
        <td><div class="match-tags">${tags.join('')}</div></td>
        <td><span class="via-profile">${esc(p.via)}</span></td>
        <td><span class="${resolvedClass}">${resolvedText}</span></td>
      </tr>`;
    }).join('');
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

    resultDiv.style.display = 'block';
    if (data.matched) {
      resultDiv.className = 'resolve-result matched';
      const lhs = data.app ? `app:${esc(data.app)}` : esc(data.target);
      resultDiv.innerHTML = `<strong>${lhs}</strong> → <strong>${esc(data.via)}</strong> (${esc(data.rule)})`;
    } else {
      resultDiv.className = 'resolve-result no-match';
      const lhs = data.app ? `app:${esc(data.app)}` : esc(data.target);
      const publicIP = lastPublicIP ? ` (public IP ${esc(lastPublicIP)})` : '';
      resultDiv.innerHTML = `<strong>${lhs}</strong> → default route (no matching policy)${publicIP}`;
    }
  } catch (_) {
    resultDiv.style.display = 'none';
  }
}

async function resolveBatchTargets() {
  const input = document.getElementById('resolve-batch-input');
  const out = document.getElementById('resolve-batch-result');
  const lines = input.value.split(/\r?\n/).map((v) => v.trim()).filter(Boolean);
  if (lines.length === 0) {
    out.style.display = 'none';
    return;
  }
  const rows = [];
  for (const item of lines) {
    const data = await resolveOneTarget(item);
    if (!data) continue;
    const target = data.app ? `app:${data.app}` : data.target;
    const route = data.matched ? data.via : 'default route';
    const rule = data.matched ? (data.rule || 'matched') : 'no matching policy';
    rows.push({ target, route, rule, matched: data.matched });
  }

  if (rows.length === 0) {
    out.style.display = 'none';
    return;
  }
  out.style.display = 'block';
  out.innerHTML = `<table>
    <thead><tr><th>Target</th><th>Route</th><th>Rule</th></tr></thead>
    <tbody>${rows.map((r) => `<tr>
      <td>${esc(r.target)}</td>
      <td>${r.matched ? `<span class="route-tag policy">${esc(r.route)}</span>` : `<span class="route-tag default">default</span>`}</td>
      <td>${esc(r.rule)}</td>
    </tr>`).join('')}</tbody>
  </table>`;
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

// Enter key in resolve input triggers search.
document.getElementById('resolve-input').addEventListener('keydown', (e) => {
  if (e.key === 'Enter') resolveTarget();
});
document.getElementById('routes-filter').addEventListener('change', renderRoutes);

// Simple HTML escape.
function esc(s) {
  const d = document.createElement('div');
  d.textContent = s || '';
  return d.innerHTML;
}

// ── VPN actions ──────────────────────────────────────────────────────────────

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

// ── Formatters ───────────────────────────────────────────────────────────────

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

// ── Boot ─────────────────────────────────────────────────────────────────────

connectWS();
refreshRoutes();
refreshPolicies();
refreshNetworkOverview();
refreshSecurity();

setInterval(refreshSecurity, 30_000);
setInterval(refreshRoutes, 30_000);
setInterval(refreshPolicies, 30_000);
setInterval(refreshNetworkOverview, 30_000);
