// Shared sidebar/topbar shell used by every dashboard page. Renders into the
// #shell-sidebar / #shell-topbar containers already present in each page's
// HTML, using the same element IDs (lang-toggle, ws-label,
// connection-indicator) the page's own script already wires up — so this
// file only owns navigation, theme, and mobile nav; language state stays
// with each page's existing I18N/setLang plumbing.

const SHELL_THEME_KEY = 'kongtrol-dashboard-theme';

const SHELL_NAV = [
  { page: 'overview', href: '/', i18n: 'nav.overview', icon: 'grid' },
  { page: 'policies', href: '/policies.html', i18n: 'nav.policyStudio', icon: 'route' },
  { page: 'security', href: '/security.html', i18n: 'nav.security', icon: 'shield' },
  { page: 'profiles', href: '/profiles.html', i18n: 'nav.vpnProfiles', icon: 'server' },
  { page: 'audit', href: '/audit.html', i18n: 'nav.auditLog', icon: 'list' },
  { page: 'settings', href: '/settings.html', i18n: 'nav.settings', icon: 'gear' },
];

const SHELL_ICONS = {
  grid: '<path d="M4 4h7v7H4V4Zm9 0h7v7h-7V4ZM4 13h7v7H4v-7Zm9 0h7v7h-7v-7Z"/>',
  route: '<circle cx="6" cy="6" r="2.5"/><circle cx="18" cy="18" r="2.5"/><path d="M8.2 7.4C11 10 9.6 13 12 15.5c1 1 1.6 1.6 2.6 2"/>',
  shield: '<path d="M12 3l7 3v6c0 4.6-3 7.7-7 9-4-1.3-7-4.4-7-9V6l7-3Z"/>',
  server: '<rect x="4" y="4" width="16" height="6" rx="1.5"/><rect x="4" y="14" width="16" height="6" rx="1.5"/><circle cx="7.5" cy="7" r="0.8" fill="currentColor" stroke="none"/><circle cx="7.5" cy="17" r="0.8" fill="currentColor" stroke="none"/>',
  list: '<path d="M8 6h13M8 12h13M8 18h13"/><circle cx="3.5" cy="6" r="1.3" fill="currentColor" stroke="none"/><circle cx="3.5" cy="12" r="1.3" fill="currentColor" stroke="none"/><circle cx="3.5" cy="18" r="1.3" fill="currentColor" stroke="none"/>',
  gear: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.87l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.87-.34 1.7 1.7 0 0 0-1.04 1.56V21a2 2 0 1 1-4 0v-.09A1.7 1.7 0 0 0 9 19.35a1.7 1.7 0 0 0-1.87.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.65 15a1.7 1.7 0 0 0-1.56-1.04H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.65 9a1.7 1.7 0 0 0-.34-1.87l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.7 1.7 0 0 0 9 4.65a1.7 1.7 0 0 0 1.04-1.56V3a2 2 0 1 1 4 0v.09A1.7 1.7 0 0 0 15 4.65a1.7 1.7 0 0 0 1.87-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.7 1.7 0 0 0 19.35 9a1.7 1.7 0 0 0 1.56 1.04H21a2 2 0 1 1 0 4h-.09A1.7 1.7 0 0 0 19.4 15Z"/>',
};

function shellIconSvg(name) {
  return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${SHELL_ICONS[name] || ''}</svg>`;
}

const SHELL_SUN_SVG = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M12 18a6 6 0 1 1 0-12 6 6 0 0 1 0 12Zm0-16a1 1 0 0 1 1 1v2h-2V3a1 1 0 0 1 1-1Zm0 17a1 1 0 0 1 1 1v2h-2v-2a1 1 0 0 1 1-1Zm10-8v2h-2v-2h2ZM4 11v2H2v-2h2Zm14.95-6.54 1.41 1.41-1.42 1.42-1.41-1.42 1.42-1.41ZM6.47 16.95l1.41 1.41-1.41 1.42-1.42-1.42 1.42-1.41Zm12.48 2.83-1.41-1.42 1.41-1.41 1.42 1.41-1.42 1.42ZM6.47 7.05 5.05 5.63l1.42-1.41 1.41 1.41-1.41 1.42Z"/></svg>';
const SHELL_MOON_SVG = '<svg viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M21 14.2A9 9 0 1 1 9.8 3a7 7 0 1 0 11.2 11.2Z"/></svg>';

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
        <button id="lang-toggle" class="lang-toggle" type="button" aria-label="Toggle language">EN</button>
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
    shellApplyTheme(next);
    localStorage.setItem(SHELL_THEME_KEY, next);
  });
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

renderShell();
initShellTheme();
initShellMobileNav();
