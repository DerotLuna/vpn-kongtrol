const API = '';
const WS_INDICATOR_KEY = 'ws.auditLog';

const I18N = {
  es: {
    'nav.overview': 'Resumen',
    'nav.studio': 'Estudio',
    'nav.security': 'Seguridad',
    'nav.auditLog': 'Registro de Auditoría',
    'nav.settings': 'Configuración',
    'nav.collapse': 'Contraer',
    'page.audit.title': 'Registro de Auditoría',
    'ws.auditLog': 'registro de auditoría',
    'audit.title': 'Registro de Auditoría',
    'audit.filterProfile': 'Filtrar por perfil…',
    'audit.allLevels': 'Todos los niveles',
    'audit.apply': 'Aplicar',
    'audit.time': 'Hora',
    'audit.level': 'Nivel',
    'audit.action': 'Acción',
    'audit.profile': 'Perfil',
    'audit.message': 'Mensaje',
    'audit.signed': 'Firmado',
    'audit.signedYes': 'firmado',
    'audit.signedNo': 'sin firmar',
    'audit.showing': 'Mostrando {count} eventos',
    'audit.notConfigured': 'El registro de auditoría no está configurado (security.audit_log.path).',
    'empty.loading': 'Cargando…',
    'empty.noEvents': 'No hay eventos de auditoría para este filtro.',
    'common.refresh': 'Actualizar',
  },
  en: {
    'nav.overview': 'Overview',
    'nav.studio': 'Studio',
    'nav.security': 'Security',
    'nav.auditLog': 'Audit Log',
    'nav.settings': 'Settings',
    'nav.collapse': 'Collapse',
    'page.audit.title': 'Audit Log',
    'ws.auditLog': 'audit log',
    'audit.title': 'Audit Log',
    'audit.filterProfile': 'Filter by profile…',
    'audit.allLevels': 'All levels',
    'audit.apply': 'Apply',
    'audit.time': 'Time',
    'audit.level': 'Level',
    'audit.action': 'Action',
    'audit.profile': 'Profile',
    'audit.message': 'Message',
    'audit.signed': 'Signed',
    'audit.signedYes': 'signed',
    'audit.signedNo': 'unsigned',
    'audit.showing': 'Showing {count} events',
    'audit.notConfigured': 'Audit logging is not configured (security.audit_log.path).',
    'empty.loading': 'Loading…',
    'empty.noEvents': 'No audit events for this filter.',
    'common.refresh': 'Refresh',
  },
};

// t/setLang/resolveInitialLang/applyStaticTranslations/initLangToggle/esc
// live in the shared i18n.js — this page only supplies the I18N dict and
// WS_INDICATOR_KEY above.

const LEVEL_BADGE = {
  INFO: 'ok',
  WARN: 'pending',
  ERROR: 'off',
  SECURITY: 'on',
};

function renderAudit(events) {
  const tbody = document.getElementById('audit-body');
  const note = document.getElementById('audit-note');

  if (!Array.isArray(events) || events.length === 0) {
    tbody.innerHTML = `<tr><td colspan="6" class="empty">${t('empty.noEvents')}</td></tr>`;
    note.textContent = '';
    return;
  }

  note.textContent = t('audit.showing', { count: events.length });

  tbody.innerHTML = events.map((ev) => {
    const level = (ev.level || 'INFO').toUpperCase();
    const badgeClass = LEVEL_BADGE[level] || 'disabled';
    const time = ev.timestamp ? new Date(ev.timestamp).toLocaleString(lang) : '—';
    const signed = ev.hmac
      ? `<span class="badge ok">${t('audit.signedYes')}</span>`
      : `<span class="badge disabled">${t('audit.signedNo')}</span>`;
    return `<tr>
      <td class="mono">${esc(time)}</td>
      <td><span class="badge ${badgeClass}">${esc(level)}</span></td>
      <td>${esc(ev.action || '—')}</td>
      <td class="mono">${esc(ev.profile || '—')}</td>
      <td class="muted">${esc(ev.message || '')}</td>
      <td>${signed}</td>
    </tr>`;
  }).join('');
}

async function loadAudit() {
  const profile = document.getElementById('audit-profile').value.trim();
  const level = document.getElementById('audit-level').value;
  const params = new URLSearchParams();
  if (profile) params.set('profile', profile);
  if (level) params.set('level', level);
  params.set('limit', '300');

  try {
    const res = await fetch(`${API}/api/v1/audit?${params.toString()}`);
    if (!res.ok) {
      showToast(t('audit.notConfigured'), 'error');
      return;
    }
    const events = await res.json();
    renderAudit(events);
  } catch (_) {}
}

document.getElementById('audit-profile').addEventListener('keydown', (e) => {
  if (e.key === 'Enter') loadAudit();
});

setLang(lang);
initLangToggle();
enhanceAllSelects();
loadAudit();
