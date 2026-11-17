// Shared i18n runtime for every dashboard page. Each page still owns its own
// `I18N = { es: {...}, en: {...} }` dictionary (translations differ per
// page) and its own HTML markup (data-i18n / data-i18n-html /
// data-i18n-placeholder attributes) — this file only owns the mechanics
// (t/setLang/applyStaticTranslations/initLangToggle/esc) so a change to how
// translation works happens in one place instead of five.
//
// Load order: shell.js, i18n.js, then the page's own script (index.html
// etc. already load shell.js/toast.js/select.js/charts.js before the page
// script; i18n.js joins that same shared-first group). `I18N` itself is
// only read from inside function bodies below, which run after the page
// script has finished defining its own `const I18N` — so declaration order
// between this file and the page dictionary doesn't matter, only the
// *load* order relative to when these functions get called (bottom of the
// page script) does.
//
// Optional per-page hooks — declare these as top-level `const`/`function`
// in the page script, before its own bottom-of-file init calls run:
//   - WS_INDICATOR_KEY: i18n key applied to #connection-indicator's title
//     automatically by applyStaticTranslations(). Skip it on pages (like
//     app.js) that drive the indicator's title dynamically from live
//     WebSocket state instead.
//   - onLangChange(): called at the end of setLang(), after
//     applyStaticTranslations(), for any page-specific re-render of
//     already-fetched data in the new language (e.g. re-rendering a cached
//     policy/route table without re-fetching).

const LANG_KEY = 'kongtrol-dashboard-lang';

// Language resolution mirrors the theme: an explicit choice (the toggle, or
// `kongtrol config lang`, both stored in ~/.kongtrol/preferences.json) wins;
// otherwise the browser's preferred languages decide (es* -> es, else en).
function storedLang() {
  try {
    const saved = localStorage.getItem(LANG_KEY);
    return saved === 'es' || saved === 'en' ? saved : '';
  } catch (_) {
    return '';
  }
}

function storeLang(value) {
  try {
    if (value) localStorage.setItem(LANG_KEY, value);
    else localStorage.removeItem(LANG_KEY);
  } catch (_) {}
}

function browserLang() {
  const list = navigator.languages && navigator.languages.length ? navigator.languages : [navigator.language || ''];
  return String(list[0] || '').toLowerCase().startsWith('es') ? 'es' : 'en';
}

function resolveInitialLang() {
  return storedLang() || browserLang();
}

let lang = resolveInitialLang();

function t(key, vars = {}) {
  const dict = I18N[lang] || I18N.en;
  const raw = dict[key] || I18N.en[key] || key;
  return raw.replace(/\{(\w+)\}/g, (_, k) => String(vars[k] ?? ''));
}

function setLang(next) {
  lang = next === 'es' ? 'es' : 'en';
  document.documentElement.lang = lang;
  applyStaticTranslations();
  if (typeof onLangChange === 'function') onLangChange();
}

function applyStaticTranslations() {
  document.querySelectorAll('[data-i18n]').forEach((el) => {
    const key = el.getAttribute('data-i18n');
    if (!key) return;
    el.textContent = t(key);
  });

  // Rich strings support only <code>…</code>; everything is inserted as text
  // nodes (never innerHTML), so no markup from a key or translation can run.
  document.querySelectorAll('[data-i18n-html]').forEach((el) => {
    const key = el.getAttribute('data-i18n-html');
    if (!key) return;
    const parts = t(key).split(/<code>([\s\S]*?)<\/code>/);
    el.replaceChildren(...parts.map((part, i) => {
      if (i % 2 === 0) return document.createTextNode(part);
      const code = document.createElement('code');
      code.textContent = part;
      return code;
    }));
  });

  document.querySelectorAll('[data-i18n-placeholder]').forEach((el) => {
    const key = el.getAttribute('data-i18n-placeholder');
    if (!key) return;
    el.setAttribute('placeholder', t(key));
  });

  document.querySelectorAll('.refresh-btn').forEach((btn) => {
    btn.title = t('common.refresh');
  });

  const toggle = document.getElementById('lang-toggle');
  if (toggle) {
    toggle.setAttribute('aria-label', lang === 'es' ? 'Cambiar idioma' : 'Toggle language');
    toggle.title = lang === 'es' ? 'Cambiar a English' : 'Switch to Español';
  }

  if (typeof WS_INDICATOR_KEY !== 'undefined') {
    const indicator = document.getElementById('connection-indicator');
    if (indicator) {
      indicator.className = 'indicator connected';
      indicator.title = t(WS_INDICATOR_KEY);
    }
  }
}

function initLangToggle() {
  const toggle = document.getElementById('lang-toggle');
  if (!toggle) return;
  toggle.addEventListener('click', () => {
    const next = lang === 'es' ? 'en' : 'es';
    shellPulseLangButton(next);
    storeLang(next);
    setLang(next);
    if (typeof shellSavePreferences === 'function') shellSavePreferences({ language: next });
  });
}

function applyPreferenceLanguage(prefs) {
  if (!prefs) return;
  // The server copy is authoritative: empty means "no explicit choice", so
  // drop any stale browser copy and follow the browser languages again.
  const chosen = prefs.language === 'es' || prefs.language === 'en' ? prefs.language : '';
  storeLang(chosen);
  const next = chosen || browserLang();
  if (next !== lang) setLang(next);
}

window.addEventListener('languagechange', () => {
  if (!storedLang() && browserLang() !== lang) setLang(browserLang());
});

window.addEventListener('kongtrol:preferences-loaded', (event) => {
  applyPreferenceLanguage(event.detail);
});
if (window.kongtrolPreferences) applyPreferenceLanguage(window.kongtrolPreferences);

function esc(s) {
  const d = document.createElement('div');
  d.textContent = s || '';
  return d.innerHTML;
}
