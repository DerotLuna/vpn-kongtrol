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

function resolveInitialLang() {
  const saved = localStorage.getItem(LANG_KEY);
  if (saved === 'es' || saved === 'en') return saved;
  return navigator.language && navigator.language.toLowerCase().startsWith('es') ? 'es' : 'en';
}

let lang = resolveInitialLang();

function t(key, vars = {}) {
  const dict = I18N[lang] || I18N.en;
  const raw = dict[key] || I18N.en[key] || key;
  return raw.replace(/\{(\w+)\}/g, (_, k) => String(vars[k] ?? ''));
}

function setLang(next) {
  lang = next === 'es' ? 'es' : 'en';
  localStorage.setItem(LANG_KEY, lang);
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

  document.querySelectorAll('[data-i18n-html]').forEach((el) => {
    const key = el.getAttribute('data-i18n-html');
    if (!key) return;
    el.innerHTML = t(key);
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
    setLang(next);
    if (typeof shellSavePreferences === 'function') shellSavePreferences({ language: next });
  });
}

function applyPreferenceLanguage(prefs) {
  if (!prefs || (prefs.language !== 'es' && prefs.language !== 'en')) return;
  localStorage.setItem(LANG_KEY, prefs.language);
  if (prefs.language !== lang) setLang(prefs.language);
}

window.addEventListener('kongtrol:preferences-loaded', (event) => {
  applyPreferenceLanguage(event.detail);
});
if (window.kongtrolPreferences) applyPreferenceLanguage(window.kongtrolPreferences);

function esc(s) {
  const d = document.createElement('div');
  d.textContent = s || '';
  return d.innerHTML;
}
