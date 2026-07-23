// Shared sidebar/topbar shell used by every dashboard page. Renders into the
// #shell-sidebar / #shell-topbar containers already present in each page's
// HTML, using the same element IDs (lang-toggle, ws-label,
// connection-indicator) the page's own script already wires up — so this
// file only owns navigation, theme, and mobile nav; language state stays
// with each page's existing I18N/setLang plumbing.

const SHELL_THEME_KEY = 'kongtrol-dashboard-theme';
const SHELL_COLLAPSE_KEY = 'kongtrol-dashboard-sidebar-collapsed';

const SHELL_NAV = [
  { page: 'overview', href: '/', i18n: 'nav.overview', icon: 'grid' },
  { page: 'studio', href: '/studio.html', i18n: 'nav.studio', icon: 'layers' },
  { page: 'security', href: '/security.html', i18n: 'nav.security', icon: 'shield' },
  { page: 'audit', href: '/audit.html', i18n: 'nav.auditLog', icon: 'list' },
  { page: 'settings', href: '/settings.html', i18n: 'nav.settings', icon: 'gear' },
];

const SHELL_ICONS = {
  grid: '<path d="M4 4h7v7H4V4Zm9 0h7v7h-7V4ZM4 13h7v7H4v-7Zm9 0h7v7h-7v-7Z"/>',
  layers: '<path d="M12 3 3 8l9 5 9-5-9-5Z"/><path d="M3 13l9 5 9-5"/><path d="M3 18l9 5 9-5"/>',
  shield: '<path d="M12 3l7 3v6c0 4.6-3 7.7-7 9-4-1.3-7-4.4-7-9V6l7-3Z"/>',
  list: '<path d="M8 6h13M8 12h13M8 18h13"/><circle cx="3.5" cy="6" r="1.3" fill="currentColor" stroke="none"/><circle cx="3.5" cy="12" r="1.3" fill="currentColor" stroke="none"/><circle cx="3.5" cy="18" r="1.3" fill="currentColor" stroke="none"/>',
  gear: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.87l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.87-.34 1.7 1.7 0 0 0-1.04 1.56V21a2 2 0 1 1-4 0v-.09A1.7 1.7 0 0 0 9 19.35a1.7 1.7 0 0 0-1.87.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.65 15a1.7 1.7 0 0 0-1.56-1.04H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.65 9a1.7 1.7 0 0 0-.34-1.87l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.7 1.7 0 0 0 9 4.65a1.7 1.7 0 0 0 1.04-1.56V3a2 2 0 1 1 4 0v.09A1.7 1.7 0 0 0 15 4.65a1.7 1.7 0 0 0 1.87-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.7 1.7 0 0 0 19.35 9a1.7 1.7 0 0 0 1.56 1.04H21a2 2 0 1 1 0 4h-.09A1.7 1.7 0 0 0 19.4 15Z"/>',
  collapse: '<path d="M15 5v14M9 5l-5 7 5 7"/>',
};

function shellIconSvg(name) {
  return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${SHELL_ICONS[name] || ''}</svg>`;
}

const SHELL_SUN_SVG = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 18a6 6 0 1 1 0-12 6 6 0 0 1 0 12Zm0-16a1 1 0 0 1 1 1v2h-2V3a1 1 0 0 1 1-1Zm0 17a1 1 0 0 1 1 1v2h-2v-2a1 1 0 0 1 1-1Zm10-8v2h-2v-2h2ZM4 11v2H2v-2h2Zm14.95-6.54 1.41 1.41-1.42 1.42-1.41-1.42 1.42-1.41ZM6.47 16.95l1.41 1.41-1.41 1.42-1.42-1.42 1.42-1.41Zm12.48 2.83-1.41-1.42 1.41-1.41 1.42 1.41-1.42 1.42ZM6.47 7.05 5.05 5.63l1.42-1.41 1.41 1.41-1.41 1.42Z"/></svg>';
const SHELL_MOON_SVG = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M21 14.2A9 9 0 1 1 9.8 3a7 7 0 1 0 11.2 11.2Z"/></svg>';
const SHELL_GLOBE_SVG = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2Zm7.9 9h-3.2a15.9 15.9 0 0 0-1.1-5.1A8.04 8.04 0 0 1 19.9 11ZM12 4.1A14 14 0 0 1 14.6 11H9.4A14 14 0 0 1 12 4.1ZM8.4 5.9A15.9 15.9 0 0 0 7.3 11H4.1a8.04 8.04 0 0 1 4.3-5.1ZM4.1 13h3.2a15.9 15.9 0 0 0 1.1 5.1A8.04 8.04 0 0 1 4.1 13Zm7.9 6.9A14 14 0 0 1 9.4 13h5.2A14 14 0 0 1 12 19.9Zm3.6-1.8a15.9 15.9 0 0 0 1.1-5.1h3.2a8.04 8.04 0 0 1-4.3 5.1Z"/></svg>';

function renderShell() {
  const page = document.body.dataset.page || '';
  const titleKey = document.body.dataset.i18nTitle || '';

  const sidebar = document.getElementById('shell-sidebar');
  if (sidebar) {
    sidebar.innerHTML = `
      <a href="/" class="brand">
        <img src="/logo-kong.svg" alt="VPN Kongtrol" class="brand-logo" />
        <span class="brand-name"><strong>K O N G T R O L</strong><span class="brand-badges"><small>CLI</small><small class="beta">BETA</small></span></span>
      </a>
      <nav class="side-nav">
        ${SHELL_NAV.map((item) => `
          <a class="side-nav-link${item.page === page ? ' active' : ''}" href="${item.href}">
            ${shellIconSvg(item.icon)}<span data-i18n="${item.i18n}"></span>
          </a>`).join('')}
      </nav>
    `;
  }

  // Rendered as a sibling of the sidebar (not nested inside it) so it can
  // straddle the sidebar/content boundary without being clipped by the
  // sidebar's own overflow-x:hidden.
  if (!document.getElementById('sidebar-collapse-toggle')) {
    const btn = document.createElement('button');
    btn.id = 'sidebar-collapse-toggle';
    btn.className = 'sidebar-collapse-toggle';
    btn.type = 'button';
    btn.setAttribute('aria-label', 'Collapse sidebar');
    btn.innerHTML = shellIconSvg('collapse');
    document.body.appendChild(btn);
  }

  const topbar = document.getElementById('shell-topbar');
  if (topbar) {
    topbar.innerHTML = `
      <div class="topbar-left">
        <button id="sidebar-toggle" class="sidebar-toggle" type="button" aria-label="Menu">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 6h18M3 12h18M3 18h18"/></svg>
        </button>
        <h1 class="topbar-title" data-i18n="${titleKey}"></h1>
      </div>
      <div class="topbar-right">
        <button id="theme-toggle" class="icon-toggle" type="button" aria-label="Toggle theme"></button>
        <button id="lang-toggle" class="lang-toggle" type="button" aria-label="Toggle language">
          <span class="lang-icon">${SHELL_GLOBE_SVG}</span>
          <span class="lang-flash"></span>
        </button>
        <span id="ws-label" class="ws-label" data-i18n="ws.connecting"></span>
        <div id="connection-indicator" class="indicator disconnected" title="Daemon connection"></div>
      </div>
    `;
  }

  if (!document.getElementById('sidebar-scrim')) {
    const scrim = document.createElement('div');
    scrim.id = 'sidebar-scrim';
    scrim.className = 'sidebar-scrim';
    document.body.appendChild(scrim);
  }
}

function shellResolveInitialTheme() {
  const saved = localStorage.getItem(SHELL_THEME_KEY);
  if (saved === 'light' || saved === 'dark') return saved;
  return window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function shellApplyTheme(theme) {
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
  const colorScheme = document.querySelector('meta[name="color-scheme"]');
  if (colorScheme) colorScheme.content = theme;
  const btn = document.getElementById('theme-toggle');
  if (btn) {
    btn.innerHTML = theme === 'dark' ? SHELL_MOON_SVG : SHELL_SUN_SVG;
    btn.title = theme === 'dark' ? 'Dark theme' : 'Light theme';
  }
}

function initShellTheme() {
  shellApplyTheme(shellResolveInitialTheme());
  const btn = document.getElementById('theme-toggle');
  if (!btn) return;
  btn.addEventListener('click', () => {
    const next = document.documentElement.dataset.theme === 'light' ? 'dark' : 'light';
    const apply = () => {
      shellApplyTheme(next);
      localStorage.setItem(SHELL_THEME_KEY, next);
      shellSavePreferences({ theme: next });
    };
    // Circular reveal growing from the toggle button — see the
    // ::view-transition-* rules in style.css. Falls back to an instant
    // switch on browsers without the API. The marker class keeps this
    // reveal from also applying to plain cross-document page navigation,
    // which uses its own (simpler) crossfade — see the same CSS block.
    if (document.startViewTransition) {
      document.documentElement.classList.add('kongtrol-theme-transition');
      const transition = document.startViewTransition(apply);
      transition.finished.finally(() => {
        document.documentElement.classList.remove('kongtrol-theme-transition');
      });
    } else {
      apply();
    }
  });
}

async function shellSavePreferences(update) {
  try {
    const res = await fetch('/api/v1/preferences', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(update),
    });
    if (!res.ok) throw new Error(`preferences: ${res.status}`);
    window.kongtrolPreferences = await res.json();
    return window.kongtrolPreferences;
  } catch (_) {
    return null;
  }
}

async function shellLoadPreferences() {
  try {
    const res = await fetch('/api/v1/preferences');
    if (!res.ok) return;
    const prefs = await res.json();
    window.kongtrolPreferences = prefs;
    if (prefs.theme === 'light' || prefs.theme === 'dark') {
      localStorage.setItem(SHELL_THEME_KEY, prefs.theme);
      shellApplyTheme(prefs.theme);
    }
    window.dispatchEvent(new CustomEvent('kongtrol:preferences-loaded', { detail: prefs }));
  } catch (_) {}
}

// Briefly hides the globe icon and flashes the target language code,
// mirroring landing's Nav.tsx handleLangToggle. Each page's own
// initLangToggle() calls this with the language it's about to switch to,
// right alongside its own setLang(...) — language state itself stays
// page-local, this only drives the button's animation.
function shellPulseLangButton(nextLang) {
  const btn = document.getElementById('lang-toggle');
  if (!btn) return;
  const icon = btn.querySelector('.lang-icon');
  const flash = btn.querySelector('.lang-flash');
  if (flash) {
    flash.textContent = (nextLang || '').toUpperCase();
    flash.classList.remove('show');
    void flash.offsetWidth; // restart the animation if clicked again mid-pulse
    flash.classList.add('show');
  }
  if (icon) icon.classList.add('hide');
  clearTimeout(btn._langPulseTimer);
  btn._langPulseTimer = setTimeout(() => {
    if (icon) icon.classList.remove('hide');
  }, 700);
}

function initShellMobileNav() {
  const toggle = document.getElementById('sidebar-toggle');
  const sidebar = document.getElementById('shell-sidebar');
  const scrim = document.getElementById('sidebar-scrim');
  if (!toggle || !sidebar || !scrim) return;

  const close = () => {
    sidebar.classList.remove('open');
    scrim.classList.remove('show');
  };
  const open = () => {
    sidebar.classList.add('open');
    scrim.classList.add('show');
  };

  toggle.addEventListener('click', () => {
    sidebar.classList.contains('open') ? close() : open();
  });
  scrim.addEventListener('click', close);
  sidebar.querySelectorAll('a').forEach((a) => a.addEventListener('click', close));
}

// Persistent desktop collapse — independent from the mobile drawer above.
// Icon-only rail; nav labels are hidden via CSS and exposed as native
// tooltips instead (see initShellNavTooltips).
function shellSetCollapsed(collapsed) {
  document.body.classList.toggle('sidebar-collapsed', collapsed);
  const btn = document.getElementById('sidebar-collapse-toggle');
  if (btn) btn.setAttribute('aria-label', collapsed ? 'Expand sidebar' : 'Collapse sidebar');
}

function initShellCollapse() {
  const btn = document.getElementById('sidebar-collapse-toggle');
  if (!btn) return;
  shellSetCollapsed(localStorage.getItem(SHELL_COLLAPSE_KEY) === '1');
  btn.addEventListener('click', () => {
    const collapsed = !document.body.classList.contains('sidebar-collapsed');
    shellSetCollapsed(collapsed);
    localStorage.setItem(SHELL_COLLAPSE_KEY, collapsed ? '1' : '0');
  });
}

// Nav labels are translated by each page's own i18n pass (which runs after
// this file, at the bottom of app.js/security.js/etc). Mirror the label text
// into a native `title` attribute reactively so icon-only (collapsed) mode
// still has an accessible/hover name, in whichever language is active.
function initShellNavTooltips() {
  const nav = document.querySelector('.side-nav');
  if (!nav) return;
  const sync = () => {
    nav.querySelectorAll('.side-nav-link').forEach((a) => {
      const span = a.querySelector('span');
      if (span) a.title = span.textContent;
    });
  };
  sync();
  new MutationObserver(sync).observe(nav, { subtree: true, characterData: true, childList: true });
}

// ── Tabs ─────────────────────────────────────────────────────────────────
// Generic tab component: <div data-tabs data-tabs-id="studio"> containing
// buttons [data-tab="x"] and panels [data-tab-panel="x"]. Remembers the last
// active tab per data-tabs-id in localStorage.
function initTabs(root) {
  root.querySelectorAll('[data-tabs]').forEach((group) => {
    const id = group.dataset.tabsId || 'default';
    const key = `kongtrol-dashboard-tab-${id}`;
    const buttons = Array.from(group.querySelectorAll('[data-tab]'));
    const panels = Array.from(group.querySelectorAll('[data-tab-panel]'));

    const activate = (name) => {
      buttons.forEach((b) => b.classList.toggle('active', b.dataset.tab === name));
      panels.forEach((p) => { p.hidden = p.dataset.tabPanel !== name; });
      localStorage.setItem(key, name);
    };

    buttons.forEach((b) => b.addEventListener('click', () => activate(b.dataset.tab)));

    const saved = localStorage.getItem(key);
    const initial = (saved && buttons.some((b) => b.dataset.tab === saved)) ? saved : (buttons[0] && buttons[0].dataset.tab);
    if (initial) activate(initial);
  });
}

// Returns `rows` <tr> of skeleton placeholders for a `cols`-column table,
// used as the initial-load state instead of a static "Loading…" text row.
// widthsPct optionally gives each column a different bar width (e.g. a
// short status column vs. a long name column) so the skeleton reads more
// like the real row shape; defaults to varied widths that work for most
// tables.
function shellSkeletonRows(cols, rows = 3, widthsPct) {
  const widths = widthsPct || Array.from({ length: cols }, (_, i) => (i === 0 ? 70 : 45 + ((i * 17) % 40)));
  let out = '';
  for (let r = 0; r < rows; r++) {
    out += '<tr class="skeleton-row">';
    for (let c = 0; c < cols; c++) {
      out += `<td><span class="skeleton" style="width:${widths[c % widths.length]}%"></span></td>`;
    }
    out += '</tr>';
  }
  return out;
}

// Same idea for non-table lists/paragraphs (e.g. groups quicklaunch).
function shellSkeletonLines(n = 2, widthsPct) {
  const widths = widthsPct || Array.from({ length: n }, () => 55 + Math.random() * 30);
  return Array.from({ length: n }, (_, i) =>
    `<div style="margin:${i === 0 ? 0 : 10}px 0 0;"><span class="skeleton" style="width:${widths[i % widths.length]}%"></span></div>`
  ).join('');
}

renderShell();
initShellTheme();
initShellMobileNav();
initShellCollapse();
initShellNavTooltips();
initTabs(document);
shellLoadPreferences();
