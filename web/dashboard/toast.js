// Lightweight toast notifications, used in place of alert()/confirm() result
// messages across the dashboard pages.

// Stable API error codes (internal/api writeError "code") mapped to a short
// "what to do" hint per language. Shared by every page.
const API_ERROR_HINTS = {
  es: {
    invalid_json: 'Datos no válidos · recarga la página e inténtalo de nuevo',
    bad_request: 'Revisa los campos e inténtalo de nuevo',
    not_found: 'Ya no existe · recarga la página',
    conflict: 'Ya existe o está en uso · cambia el nombre o espera',
    unavailable: 'Kongtrol no responde · espera unos segundos y reintenta',
    not_supported: 'No disponible en esta versión · usa la terminal',
    forbidden: 'Sesión vencida · recarga la página',
    internal: 'Error interno · ejecuta kongtrol doctor',
  },
  en: {
    invalid_json: 'Invalid data · reload the page and try again',
    bad_request: 'Check the fields and try again',
    not_found: 'It no longer exists · reload the page',
    conflict: 'Already exists or in use · rename it or wait',
    unavailable: 'Kongtrol is not responding · wait a few seconds and retry',
    not_supported: 'Not available in this version · use the terminal',
    forbidden: 'Session expired · reload the page',
    internal: 'Internal error · run kongtrol doctor',
  },
};

// showApiError shows the translated context message first (fallback,
// already passed through t()), a translated hint for the API's error code
// as a second line, and the raw English API detail only as a tooltip.
// body is the parsed JSON error response ({error, code}) or null.
function showApiError(fallback, body) {
  const curLang = typeof lang === 'string' && lang === 'es' ? 'es' : 'en';
  const code = body && body.code;
  const hint = code ? (API_ERROR_HINTS[curLang][code] || '') : '';
  const raw = body && body.error ? String(body.error) : '';
  showToast(fallback, 'error', 6000, hint, raw);
}

function showToast(message, variant = 'info', timeoutMs = 4500, detail = '', tooltip = '') {
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
  if (detail) {
    const sub = document.createElement('div');
    sub.className = 'toast-detail';
    sub.textContent = detail;
    el.appendChild(sub);
  }
  if (tooltip) el.title = tooltip;
  stack.appendChild(el);

  requestAnimationFrame(() => el.classList.add('show'));

  let dismissed = false;
  const dismiss = () => {
    if (dismissed) return;
    dismissed = true;
    el.classList.remove('show');
    setTimeout(() => el.remove(), 220);
  };

  el.addEventListener('click', dismiss);
  setTimeout(dismiss, timeoutMs);
}
