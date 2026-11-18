// Browser sign-in for the dashboard (ADR-0036). Loaded first on every page.
//
// Kongtrol (tray "Abrir panel", `kongtrol dashboard`) opens the page with a
// one-time nonce in the fragment: /#k=<nonce>. This script removes it from
// the address bar, trades it for an HttpOnly session cookie (POST
// /api/v1/session) and holds every fetch() until that finished. Pages
// never receive the control token. A 401 from the API shows one banner
// telling the user how to open the panel from Kongtrol.

(function () {
  const STRINGS = {
    es: {
      title: 'Esta página no tiene permiso para hablar con Kongtrol.',
      fix: 'Qué hacer: ábrela desde Kongtrol, con el ícono de la bandeja («Abrir panel») o con kongtrol dashboard.',
    },
    en: {
      title: 'This page is not allowed to talk to Kongtrol.',
      fix: 'What to do: open it from Kongtrol, with the tray icon ("Open dashboard") or with kongtrol dashboard.',
    },
  };

  function pageLang() {
    try {
      const saved = localStorage.getItem('kongtrol-dashboard-lang');
      if (saved === 'es' || saved === 'en') return saved;
    } catch (_) {}
    const list = navigator.languages && navigator.languages.length ? navigator.languages : [navigator.language || ''];
    return String(list[0] || '').toLowerCase().startsWith('es') ? 'es' : 'en';
  }

  function takeNonce() {
    const params = new URLSearchParams(location.hash.slice(1));
    const nonce = params.get('k');
    if (!nonce) return '';
    params.delete('k');
    const rest = params.toString();
    try {
      history.replaceState(null, '', location.pathname + location.search + (rest ? '#' + rest : ''));
    } catch (_) {}
    return nonce;
  }

  const realFetch = window.fetch.bind(window);
  const nonce = takeNonce();
  const ready = nonce
    ? realFetch('/api/v1/session', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ nonce }),
      }).catch(() => null)
    : Promise.resolve(null);

  let shown = false;
  function showSignInHelp() {
    if (shown) return;
    shown = true;
    const s = STRINGS[pageLang()];
    const box = document.createElement('div');
    box.setAttribute('role', 'alert');
    box.className = 'auth-banner';
    box.style.cssText = 'position:fixed;left:16px;right:16px;bottom:16px;z-index:1000;padding:14px 16px;' +
      'border-radius:10px;background:#7a1f1f;color:#fff;font:14px/1.4 system-ui,sans-serif;box-shadow:0 6px 24px rgba(0,0,0,.3)';
    const strong = document.createElement('strong');
    strong.textContent = s.title;
    const fix = document.createElement('div');
    fix.textContent = s.fix;
    box.append(strong, fix);
    const mount = () => document.body.append(box);
    if (document.body) mount();
    else document.addEventListener('DOMContentLoaded', mount);
  }

  window.fetch = async function (input, init) {
    await ready;
    const res = await realFetch(input, init);
    const url = typeof input === 'string' ? input : (input && input.url) || '';
    if (res.status === 401 && url.indexOf('/api/') !== -1) showSignInHelp();
    return res;
  };
  window.kongtrolSessionReady = ready;
})();
