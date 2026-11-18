// Daemon notices (GET /api/v1/events): a tunnel the watchdog cannot
// reconnect (or that a declared failover backup replaced), and traffic of a
// policy whose tunnel is off — "claude.ai sale por internet normal · brainbox está apagado". The
// daemon sends an i18n key + args; this file renders them in the page's
// language and shows a toast with a "Conectar brainbox" button. Loaded on
// every dashboard page after toast.js.

const NOTICE_I18N = {
  es: {
    'policy.event.allow': '{0} sale por internet normal · {1} está apagado',
    'policy.event.block': '{0} bloqueado · {1} está apagado',
    'policy.event.connect': 'Conectando {1}: hay tráfico hacia {0}',
    'policy.event.connect_failed': 'No se pudo conectar {1} para {0}',
    'policy.event.action_connect': 'Conectar {0}',
    'notice.connecting': 'Conectando {0}…',
    'notice.connectFailed': 'No se pudo conectar {0} · revisa el panel de túneles',
    'watchdog.event.reconnect_failing': '{0} no se pudo reconectar · Kongtrol sigue intentando solo; revisa tu internet o el servidor de esa VPN',
    'watchdog.event.failover': '{0} no se pudo reconectar · se conectó su respaldo {1}. Para volver, conecta {0}',
    'watchdog.event.reconnected': '{0} se reconectó',
    'watchdog.event.no_response': '{0} dejó de responder: su servidor no contesta · Kongtrol la reconecta solo; si sigue, revisa tu internet o avisa a quien administra esa VPN',
    'watchdog.event.responding': '{0} vuelve a responder',
  },
  en: {
    'policy.event.allow': '{0} is going over the normal internet · {1} is off',
    'policy.event.block': '{0} blocked · {1} is off',
    'policy.event.connect': 'Connecting {1}: traffic to {0}',
    'policy.event.connect_failed': 'Could not connect {1} for {0}',
    'policy.event.action_connect': 'Connect {0}',
    'notice.connecting': 'Connecting {0}…',
    'notice.connectFailed': 'Could not connect {0} · check the tunnels panel',
    'watchdog.event.reconnect_failing': "{0} could not reconnect · Kongtrol keeps retrying on its own; check your internet or that VPN's server",
    'watchdog.event.failover': '{0} could not reconnect · its backup {1} is now connected. To go back, connect {0}',
    'watchdog.event.reconnected': '{0} reconnected',
    'watchdog.event.no_response': '{0} stopped responding: its server does not answer · Kongtrol reconnects it on its own; if it persists, check your internet or tell whoever runs that VPN',
    'watchdog.event.responding': '{0} responds again',
  },
};

function noticeText(key, args) {
  const l = typeof lang === 'string' && lang === 'es' ? 'es' : 'en';
  const raw = NOTICE_I18N[l][key] || NOTICE_I18N.en[key] || key;
  return raw.replace(/\{(\d+)\}/g, (_, i) => String((args || [])[Number(i)] ?? ''));
}

// showActionToast is a toast with one button; it stays until dismissed.
function showActionToast(message, variant, actionLabel, onAction) {
  let stack = document.getElementById('toast-stack');
  if (!stack) {
    stack = document.createElement('div');
    stack.id = 'toast-stack';
    stack.className = 'toast-stack';
    document.body.appendChild(stack);
  }
  const el = document.createElement('div');
  el.className = `toast toast-${variant}`;
  el.textContent = message;
  const dismiss = () => {
    el.classList.remove('show');
    setTimeout(() => el.remove(), 220);
  };
  if (actionLabel) {
    const row = document.createElement('div');
    row.className = 'toast-detail';
    const btn = document.createElement('button');
    btn.className = 'action';
    btn.textContent = actionLabel;
    btn.addEventListener('click', (ev) => {
      ev.stopPropagation();
      dismiss();
      onAction();
    });
    row.appendChild(btn);
    el.appendChild(row);
  }
  el.addEventListener('click', dismiss);
  stack.appendChild(el);
  requestAnimationFrame(() => el.classList.add('show'));
  setTimeout(dismiss, 60000);
}

async function noticeConnect(profile) {
  showToast(noticeText('notice.connecting', [profile]), 'info');
  try {
    const res = await fetch(`/api/v1/tunnels/${encodeURIComponent(profile)}/connect`, {
      method: 'POST',
      headers: { 'X-Kongtrol-Lang': typeof lang === 'string' ? lang : 'en' },
    });
    if (!res.ok) throw new Error(String(res.status));
  } catch (_) {
    showToast(noticeText('notice.connectFailed', [profile]), 'error', 8000);
  }
}

// The last event seen survives page changes (sessionStorage); a fresh tab
// starts at -1: only notices from now on, no backlog.
function noticeSeq() {
  try {
    const v = sessionStorage.getItem('kongtrol.noticeSeq');
    return v === null ? -1 : Number(v);
  } catch (_) {
    return -1;
  }
}

function setNoticeSeq(v) {
  try {
    sessionStorage.setItem('kongtrol.noticeSeq', String(v));
  } catch (_) {
    /* private mode: notices may repeat after a page change */
  }
}

async function pollNotices() {
  try {
    const res = await fetch(`/api/v1/events?after=${noticeSeq()}`);
    if (!res.ok) return;
    const body = await res.json();
    if (typeof body.seq === 'number') setNoticeSeq(body.seq);
    (body.events || []).forEach((e) => {
      if (e.kind !== 'policy_tunnel_down' && e.kind !== 'tunnel_watchdog') return;
      const msg = noticeText(e.key, e.args);
      const variant = e.level === 'WARN' ? 'error' : 'info';
      if (e.can_connect && e.profile) {
        showActionToast(msg, variant, noticeText('policy.event.action_connect', [e.profile]), () => noticeConnect(e.profile));
      } else {
        showToast(msg, variant, 8000);
      }
    });
  } catch (_) {
    /* daemon unreachable: the pages already say so */
  }
}

pollNotices();
setInterval(pollNotices, 5000);
