// Opt-in public IP lookup (security.public_ip_check). Off by default: while
// off the dashboard shows "—" and a short hint, and never asks for the IP.
// When on, the daemon looks it up only on tunnel connect/disconnect and when
// the user presses "Revisar" here (POST /api/v1/security/publicip/check).
// Loaded on the overview and security pages after toast.js.

const PUBLICIP_I18N = {
  es: {
    'publicip.check': 'Revisar',
    'publicip.checking': 'Revisando…',
    'publicip.hint': 'Actívalo para ver tu IP pública',
    'publicip.notice': 'Usa un servicio externo ({host}) que ve tu IP pública',
    'publicip.on': 'Consulta de IP pública activada',
    'publicip.off': 'Consulta de IP pública apagada · Kongtrol ya no consulta ningún servicio externo',
    'publicip.failed': 'No se pudo consultar tu IP pública',
    'publicip.failedHint': 'Revisa tu conexión a internet y vuelve a probar.',
    'publicip.toggleFailed': 'No se pudo cambiar la consulta de IP pública',
  },
  en: {
    'publicip.check': 'Check',
    'publicip.checking': 'Checking…',
    'publicip.hint': 'Turn it on to see your public IP',
    'publicip.notice': 'Uses an external service ({host}) that sees your public IP',
    'publicip.on': 'Public IP lookup turned on',
    'publicip.off': 'Public IP lookup turned off · Kongtrol no longer contacts any external service',
    'publicip.failed': 'Could not look up your public IP',
    'publicip.failedHint': 'Check your internet connection and try again.',
    'publicip.toggleFailed': 'Could not change the public IP lookup',
  },
};

function publicIpT(key, vars) {
  const l = typeof lang === 'string' && lang === 'es' ? 'es' : 'en';
  const raw = PUBLICIP_I18N[l][key] || PUBLICIP_I18N.en[key] || key;
  return raw.replace(/\{(\w+)\}/g, (_, k) => String((vars || {})[k] ?? ''));
}

// renderPublicIPField fills a value element: the IP, or "—" (and the hint
// while the lookup is off). The "Revisar" buttons ([data-publicip-check])
// only show while the lookup is on.
function renderPublicIPField(id, data) {
  const el = document.getElementById(id);
  if (!el) return;
  const d = data || {};
  const on = !!d.public_ip_check_enabled;
  const ip = (d.leak_check && d.leak_check.public_ip) || d.public_ip || '';
  el.textContent = on && ip ? ip : '—';
  let hint = el.parentElement && el.parentElement.querySelector('.publicip-hint');
  if (!on) {
    if (!hint && el.parentElement) {
      hint = document.createElement('span');
      hint.className = 'publicip-hint muted';
      el.parentElement.appendChild(hint);
    }
    if (hint) hint.textContent = publicIpT('publicip.hint');
  } else if (hint) {
    hint.remove();
  }
  document.querySelectorAll('[data-publicip-check]').forEach((b) => {
    b.hidden = !on;
    b.textContent = publicIpT('publicip.check');
  });
}

// checkPublicIPNow is the "Revisar" button: one lookup now, then refresh.
async function checkPublicIPNow(btn) {
  if (btn) {
    btn.disabled = true;
    btn.textContent = publicIpT('publicip.checking');
  }
  try {
    const res = await fetch('/api/v1/security/publicip/check', { method: 'POST' });
    const body = await res.json().catch(() => null);
    if (!res.ok) {
      showToast((body && body.error) || publicIpT('publicip.failed'), 'error', 6000, (body && body.hint) || '');
    } else if (body && body.leak_check && body.leak_check.state === 'error') {
      showToast(publicIpT('publicip.failed'), 'error', 6000, publicIpT('publicip.failedHint'));
    }
  } catch (_) {
    showToast(publicIpT('publicip.failed'), 'error', 6000, publicIpT('publicip.failedHint'));
  } finally {
    if (btn) {
      btn.disabled = false;
      btn.textContent = publicIpT('publicip.check');
    }
    if (typeof refreshSecurity === 'function') refreshSecurity();
    if (typeof refreshNetworkOverview === 'function') refreshNetworkOverview();
  }
}

// togglePublicIP is the security page switch. Turning it on shows which
// external service will see the public IP.
async function togglePublicIP(input) {
  const enabled = input.checked;
  input.disabled = true;
  try {
    const res = await fetch('/api/v1/security/publicip', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    });
    const body = await res.json().catch(() => null);
    if (!res.ok) throw Object.assign(new Error('toggle'), { body });
    if (enabled) {
      showToast(publicIpT('publicip.on'), 'success', 6000, publicIpT('publicip.notice', { host: body.public_ip_check_host || 'api.ipify.org' }));
    } else {
      showToast(publicIpT('publicip.off'), 'info');
    }
  } catch (err) {
    input.checked = !enabled;
    showApiError(publicIpT('publicip.toggleFailed'), err.body || null);
  } finally {
    input.disabled = false;
    if (typeof refreshSecurity === 'function') refreshSecurity();
  }
}
