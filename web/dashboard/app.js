/* VPN Kongtrol Dashboard — vanilla JS, WebSocket live feed, no build step */

const API = '';

// ── WebSocket live metrics ──────────────────────────────────────────────────

let ws = null;
let wsReconnectTimer = null;

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

      const dot   = `<span class="status-dot ${status}"></span>`;
      const ip    = m.AssignedIP || '<span class="muted">—</span>';
      const up    = isConnected && m.UptimeSeconds > 0 ? formatDuration(m.UptimeSeconds) : '<span class="muted">—</span>';
      const sent  = m.BytesSent    > 0 ? formatBytes(m.BytesSent)    : '<span class="muted">—</span>';
      const recv  = m.BytesReceived > 0 ? formatBytes(m.BytesReceived) : '<span class="muted">—</span>';
      const lat   = m.LatencyMS    > 0 ? `${m.LatencyMS}ms`          : '<span class="muted">—</span>';

      const btn = isConnected
        ? `<button class="action disconnect" onclick="disconnectVPN('${name}')">Disconnect</button>`
        : `<button class="action" onclick="connectVPN('${name}')">Connect</button>`;

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

    setBadge('ks-status',  data.kill_switch, 'ON', 'OFF');
    setBadge('dns-status', data.dns_guard, 'ON', 'OFF');

    if (data.leak_check) {
      const lk = data.leak_check;
      setBadge('leak-status', !lk.has_leak, 'CLEAN', 'LEAK');
      document.getElementById('public-ip').textContent = lk.public_ip || '—';
      if (lk.checked_at) {
        const d = new Date(lk.checked_at);
        document.getElementById('leak-time').textContent = d.toLocaleTimeString();
      }
    }
  } catch (_) {}
}

function setBadge(id, positive, trueLabel, falseLabel) {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = positive ? trueLabel : falseLabel;
  el.className = 'badge ' + (positive ? 'on' : 'off');
}

// ── Route list ──────────────────────────────────────────────────────────────

async function refreshRoutes() {
  try {
    const res = await fetch(`${API}/api/v1/routes`);
    if (!res.ok) return;
    const routes = await res.json();
    const ul = document.getElementById('routes-list');

    if (!routes || routes.length === 0) {
      ul.innerHTML = '<li><span class="empty">No routes managed.</span></li>';
      return;
    }

    ul.innerHTML = routes.map(r =>
      `<li>
        <span class="dest">${r.destination}</span>
        <span class="via">→ ${r.interface || r.gateway || '—'}</span>
      </li>`
    ).join('');
  } catch (_) {}
}

// ── Traffic Map (policies + resolve) ─────────────────────────────────────────

async function refreshPolicies() {
  try {
    const res = await fetch(`${API}/api/v1/policies`);
    if (!res.ok) return;
    const policies = await res.json();
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
    const res = await fetch(`${API}/api/v1/resolve?target=${encodeURIComponent(target)}`);
    if (!res.ok) return;
    const data = await res.json();

    resultDiv.style.display = 'block';
    if (data.matched) {
      resultDiv.className = 'resolve-result matched';
      resultDiv.innerHTML = `<strong>${esc(data.target)}</strong> → <strong>${esc(data.via)}</strong> (${esc(data.rule)})`;
    } else {
      resultDiv.className = 'resolve-result no-match';
      resultDiv.innerHTML = `<strong>${esc(data.target)}</strong> → default route (no matching policy)`;
    }
  } catch (_) {
    resultDiv.style.display = 'none';
  }
}

// Enter key in resolve input triggers search.
document.getElementById('resolve-input').addEventListener('keydown', (e) => {
  if (e.key === 'Enter') resolveTarget();
});

// Simple HTML escape.
function esc(s) {
  const d = document.createElement('div');
  d.textContent = s || '';
  return d.innerHTML;
}

// ── VPN actions ──────────────────────────────────────────────────────────────

async function connectVPN(name) {
  const btn = event.target;
  btn.disabled = true;
  btn.textContent = 'Connecting…';
  try {
    await fetch(`${API}/api/v1/tunnels/${name}/connect`, { method: 'POST' });
  } finally {
    setTimeout(() => { btn.disabled = false; }, 2000);
  }
}

async function disconnectVPN(name) {
  const btn = event.target;
  btn.disabled = true;
  btn.textContent = 'Disconnecting…';
  try {
    await fetch(`${API}/api/v1/tunnels/${name}/disconnect`, { method: 'POST' });
  } finally {
    setTimeout(() => { btn.disabled = false; }, 2000);
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
refreshSecurity();
refreshRoutes();
refreshPolicies();

setInterval(refreshSecurity, 30_000);
setInterval(refreshRoutes, 30_000);
setInterval(refreshPolicies, 30_000);
