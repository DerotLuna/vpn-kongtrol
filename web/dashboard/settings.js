const API = '';
const WS_INDICATOR_KEY = 'ws.settings';

const I18N = {
  es: {
    'nav.overview': 'Resumen',
    'nav.studio': 'Estudio',
    'nav.security': 'Seguridad',
    'nav.auditLog': 'Registro de Auditoría',
    'nav.settings': 'Configuración',
    'nav.collapse': 'Contraer',
    'page.settings.title': 'Configuración',
    'ws.settings': 'configuración',
    'settings.tabGeneral': 'General',
    'settings.tabSecurity': 'Seguridad',
    'settings.tabScheduler': 'Programador',
    'settings.dashboard': 'Dashboard',
    'settings.dashboardBind': 'Dirección de bind',
    'settings.dashboardPort': 'Puerto',
    'settings.dashboardNote': 'No editable desde aquí — cambiarlo desde la página que sirve este request rompería la conexión. Usa el CLI:',
    'settings.healthCheck': 'Health Check',
    'settings.interval': 'Intervalo',
    'settings.timeout': 'Timeout',
    'settings.splitDns': 'Split DNS',
    'settings.enabled': 'Activado',
    'settings.killSwitchTuning': 'Ajustes de Kill Switch',
    'settings.mode': 'Modo',
    'settings.allowLan': 'Permitir tráfico LAN',
    'settings.killSwitchNote': 'Activa/desactiva el kill switch en sí desde la página de Seguridad.',
    'settings.dnsGuardTuning': 'Ajustes de DNS Guard',
    'settings.fallbackDns': 'DNS de respaldo',
    'settings.leakDetection': 'Detección de Fugas',
    'settings.action': 'Acción',
    'settings.auditLog': 'Registro de Auditoría',
    'settings.path': 'Ruta del archivo de log',
    'settings.maxSize': 'Tamaño máx. (MB)',
    'settings.sign': 'Firmar entradas con HMAC',
    'settings.save': 'Guardar Configuración',
    'settings.revert': 'Revertir',
    'settings.schedulerRules': 'Reglas de Programación',
    'settings.schedulerEnabled': 'Programador activado',
    'settings.ruleEditor': 'Editor de Regla',
    'settings.ruleName': 'Nombre',
    'settings.ruleNamePlaceholder': 'horario-laboral',
    'settings.weekdays': 'Días (separados por coma, opcional)',
    'settings.start': 'Inicio (HH:MM, opcional)',
    'settings.end': 'Fin (HH:MM, opcional)',
    'settings.existingRules': 'Reglas existentes',
    'settings.window': 'Ventana',
    'groups.profiles': 'Perfiles',
    'table.name': 'Nombre',
    'table.actions': 'Acciones',
    'actions.create': 'Crear',
    'actions.update': 'Actualizar',
    'actions.delete': 'Eliminar',
    'actions.clear': 'Limpiar',
    'actions.edit': 'Editar',
    'empty.loading': 'Cargando…',
    'empty.noRules': 'No hay reglas configuradas.',
    'alerts.saveFailed': 'No se pudo guardar la configuración',
    'alerts.ruleCreateFailed': 'Error al crear la regla',
    'alerts.ruleUpdateFailed': 'Error al actualizar la regla',
    'alerts.ruleDeleteFailed': 'Error al eliminar la regla',
    'alerts.ruleSelectFirst': 'Selecciona primero una regla o escribe un nombre.',
    'alerts.ruleSelectDelete': 'Selecciona una regla para eliminar.',
    'confirm.deleteRule': '¿Eliminar la regla "{name}"?',
    'toast.saved': 'Configuración guardada.',
    'toast.ruleCreated': 'Regla "{name}" creada.',
    'toast.ruleUpdated': 'Regla "{name}" actualizada.',
    'toast.ruleDeleted': 'Regla "{name}" eliminada.',
    'common.refresh': 'Actualizar',
  },
  en: {
    'nav.overview': 'Overview',
    'nav.studio': 'Studio',
    'nav.security': 'Security',
    'nav.auditLog': 'Audit Log',
    'nav.settings': 'Settings',
    'nav.collapse': 'Collapse',
    'page.settings.title': 'Settings',
    'ws.settings': 'settings',
    'settings.tabGeneral': 'General',
    'settings.tabSecurity': 'Security',
    'settings.tabScheduler': 'Scheduler',
    'settings.dashboard': 'Dashboard',
    'settings.dashboardBind': 'Bind address',
    'settings.dashboardPort': 'Port',
    'settings.dashboardNote': 'Not editable from here — changing it from the page serving this request would break the connection. Use the CLI:',
    'settings.healthCheck': 'Health Check',
    'settings.interval': 'Interval',
    'settings.timeout': 'Timeout',
    'settings.splitDns': 'Split DNS',
    'settings.enabled': 'Enabled',
    'settings.killSwitchTuning': 'Kill Switch Tuning',
    'settings.mode': 'Mode',
    'settings.allowLan': 'Allow LAN traffic',
    'settings.killSwitchNote': 'Enable/disable the kill switch itself from the Security page.',
    'settings.dnsGuardTuning': 'DNS Guard Tuning',
    'settings.fallbackDns': 'Fallback DNS',
    'settings.leakDetection': 'Leak Detection',
    'settings.action': 'Action',
    'settings.auditLog': 'Audit Log',
    'settings.path': 'Log file path',
    'settings.maxSize': 'Max size (MB)',
    'settings.sign': 'HMAC sign entries',
    'settings.save': 'Save Settings',
    'settings.revert': 'Revert',
    'settings.schedulerRules': 'Scheduler Rules',
    'settings.schedulerEnabled': 'Scheduler enabled',
    'settings.ruleEditor': 'Rule Editor',
    'settings.ruleName': 'Name',
    'settings.ruleNamePlaceholder': 'business-hours',
    'settings.weekdays': 'Weekdays (comma-separated, optional)',
    'settings.start': 'Start (HH:MM, optional)',
    'settings.end': 'End (HH:MM, optional)',
    'settings.existingRules': 'Existing Rules',
    'settings.window': 'Window',
    'groups.profiles': 'Profiles',
    'table.name': 'Name',
    'table.actions': 'Actions',
    'actions.create': 'Create',
    'actions.update': 'Update',
    'actions.delete': 'Delete',
    'actions.clear': 'Clear',
    'actions.edit': 'Edit',
    'empty.loading': 'Loading…',
    'empty.noRules': 'No rules configured.',
    'alerts.saveFailed': 'Failed to save settings',
    'alerts.ruleCreateFailed': 'Failed to create rule',
    'alerts.ruleUpdateFailed': 'Failed to update rule',
    'alerts.ruleDeleteFailed': 'Failed to delete rule',
    'alerts.ruleSelectFirst': 'Select a rule first or provide a name.',
    'alerts.ruleSelectDelete': 'Select a rule to delete.',
    'confirm.deleteRule': 'Delete rule "{name}"?',
    'toast.saved': 'Settings saved.',
    'toast.ruleCreated': 'Rule "{name}" created.',
    'toast.ruleUpdated': 'Rule "{name}" updated.',
    'toast.ruleDeleted': 'Rule "{name}" deleted.',
    'common.refresh': 'Refresh',
  },
};

// t/setLang/resolveInitialLang/applyStaticTranslations/initLangToggle/esc
// live in the shared i18n.js — this page only supplies the I18N dict and
// WS_INDICATOR_KEY above, plus the re-render hook below.
function onLangChange() {
  renderRules();
}

function lines(id) {
  return document.getElementById(id).value.split(/\r?\n/).map((v) => v.trim()).filter(Boolean);
}

// ── Settings form ───────────────────────────────────────────────────────────

function fillSettingsForm(s) {
  document.getElementById('s-dashboard-bind').textContent = s.dashboard_bind || '—';
  document.getElementById('s-dashboard-port').textContent = s.dashboard_port || '—';
  document.getElementById('s-hc-interval').value = s.health_check_interval || '';
  document.getElementById('s-hc-timeout').value = s.health_check_timeout || '';
  document.getElementById('s-sched-enabled').checked = !!s.scheduler_enabled;
  document.getElementById('s-sched-interval').value = s.scheduler_interval || '';
  document.getElementById('s-splitdns-enabled').checked = !!s.split_dns_enabled;
  document.getElementById('s-splitdns-interval').value = s.split_dns_interval || '';
  document.getElementById('s-ks-mode').value = s.kill_switch_mode || 'strict';
  document.getElementById('s-ks-allowlan').checked = !!s.kill_switch_allow_lan;
  document.getElementById('s-dns-fallback').value = s.dns_guard_fallback_dns || '';
  document.getElementById('s-leak-enabled').checked = !!s.leak_detection_enabled;
  document.getElementById('s-leak-interval').value = s.leak_detection_interval || '';
  document.getElementById('s-leak-action').value = s.leak_detection_action || 'notify';
  document.getElementById('s-audit-path').value = s.audit_log_path || '';
  document.getElementById('s-audit-maxsize').value = s.audit_log_max_size_mb || '';
  document.getElementById('s-audit-sign').checked = !!s.audit_log_sign;
  enhanceSelect(document.getElementById('s-ks-mode'));
  enhanceSelect(document.getElementById('s-leak-action'));
}

function draftSettings() {
  return {
    health_check_interval: document.getElementById('s-hc-interval').value.trim(),
    health_check_timeout: document.getElementById('s-hc-timeout').value.trim(),
    scheduler_enabled: document.getElementById('s-sched-enabled').checked,
    scheduler_interval: document.getElementById('s-sched-interval').value.trim(),
    split_dns_enabled: document.getElementById('s-splitdns-enabled').checked,
    split_dns_interval: document.getElementById('s-splitdns-interval').value.trim(),
    kill_switch_mode: document.getElementById('s-ks-mode').value,
    kill_switch_allow_lan: document.getElementById('s-ks-allowlan').checked,
    dns_guard_fallback_dns: document.getElementById('s-dns-fallback').value.trim(),
    leak_detection_enabled: document.getElementById('s-leak-enabled').checked,
    leak_detection_interval: document.getElementById('s-leak-interval').value.trim(),
    leak_detection_action: document.getElementById('s-leak-action').value,
    audit_log_path: document.getElementById('s-audit-path').value.trim(),
    audit_log_max_size_mb: parseInt(document.getElementById('s-audit-maxsize').value, 10) || 0,
    audit_log_sign: document.getElementById('s-audit-sign').checked,
  };
}

async function loadSettings() {
  const res = await fetch(`${API}/api/v1/settings`);
  if (!res.ok) return;
  fillSettingsForm(await res.json());
}

async function saveSettings() {
  const res = await fetch(`${API}/api/v1/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(draftSettings()),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.saveFailed') }));
    showToast(e.error || t('alerts.saveFailed'), 'error');
    return;
  }
  fillSettingsForm(await res.json());
  showToast(t('toast.saved'), 'success');
}

// ── Scheduler rules ─────────────────────────────────────────────────────────

let rules = [];
let selectedRuleName = '';

function draftRule() {
  return {
    name: document.getElementById('r-name').value.trim(),
    profiles: lines('r-profiles'),
    weekdays: document.getElementById('r-weekdays').value.split(',').map((v) => v.trim()).filter(Boolean),
    start: document.getElementById('r-start').value.trim(),
    end: document.getElementById('r-end').value.trim(),
  };
}

function fillRuleForm(rule) {
  selectedRuleName = rule.name;
  document.getElementById('r-name').value = rule.name || '';
  document.getElementById('r-profiles').value = (rule.profiles || []).join('\n');
  document.getElementById('r-weekdays').value = (rule.weekdays || []).join(',');
  document.getElementById('r-start').value = rule.start || '';
  document.getElementById('r-end').value = rule.end || '';
}

function clearRuleForm() {
  selectedRuleName = '';
  ['r-name', 'r-profiles', 'r-weekdays', 'r-start', 'r-end'].forEach((id) => {
    document.getElementById(id).value = '';
  });
}

function renderRules() {
  const tbody = document.getElementById('rule-list-body');
  if (!Array.isArray(rules) || rules.length === 0) {
    tbody.innerHTML = `<tr><td colspan="4" class="empty">${t('empty.noRules')}</td></tr>`;
    return;
  }

  tbody.innerHTML = rules.map((rule, idx) => {
    const window = [rule.weekdays && rule.weekdays.length ? rule.weekdays.join(',') : '', rule.start && rule.end ? `${rule.start}-${rule.end}` : ''].filter(Boolean).join(' · ') || '—';
    return `<tr>
      <td>${esc(rule.name)}</td>
      <td class="muted">${esc((rule.profiles || []).join(', '))}</td>
      <td class="mono">${esc(window)}</td>
      <td>
        <div class="policy-row-actions">
          <button class="action" onclick='selectRuleByIndex(${idx})'>${t('actions.edit')}</button>
          <button class="action disconnect" onclick='deleteRuleByIndex(${idx})'>${t('actions.delete')}</button>
        </div>
      </td>
    </tr>`;
  }).join('');
}

function selectRuleByIndex(index) {
  const rule = rules[index];
  if (!rule) return;
  fillRuleForm(rule);
}

async function loadScheduleRules() {
  const res = await fetch(`${API}/api/v1/scheduler/rules`);
  if (!res.ok) return;
  rules = await res.json();
  renderRules();
}

async function createRule() {
  const draft = draftRule();
  const res = await fetch(`${API}/api/v1/scheduler/rules`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(draft),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.ruleCreateFailed') }));
    showToast(e.error || t('alerts.ruleCreateFailed'), 'error');
    return;
  }
  showToast(t('toast.ruleCreated', { name: draft.name }), 'success');
  await loadScheduleRules();
  selectedRuleName = draft.name;
}

async function updateRule() {
  const draft = draftRule();
  const key = selectedRuleName || draft.name;
  if (!key) {
    showToast(t('alerts.ruleSelectFirst'), 'error');
    return;
  }
  const res = await fetch(`${API}/api/v1/scheduler/rules/${encodeURIComponent(key)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(draft),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.ruleUpdateFailed') }));
    showToast(e.error || t('alerts.ruleUpdateFailed'), 'error');
    return;
  }
  showToast(t('toast.ruleUpdated', { name: draft.name || key }), 'success');
  await loadScheduleRules();
  selectedRuleName = draft.name || key;
}

async function deleteRule() {
  const name = selectedRuleName || document.getElementById('r-name').value.trim();
  if (!name) {
    showToast(t('alerts.ruleSelectDelete'), 'error');
    return;
  }
  await deleteRuleByName(name);
}

async function deleteRuleByName(name) {
  if (!confirm(t('confirm.deleteRule', { name }))) return;
  const res = await fetch(`${API}/api/v1/scheduler/rules/${encodeURIComponent(name)}`, { method: 'DELETE' });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.ruleDeleteFailed') }));
    showToast(e.error || t('alerts.ruleDeleteFailed'), 'error');
    return;
  }
  showToast(t('toast.ruleDeleted', { name }), 'info');
  if (selectedRuleName === name) clearRuleForm();
  await loadScheduleRules();
}

async function deleteRuleByIndex(index) {
  const rule = rules[index];
  if (!rule) return;
  await deleteRuleByName(rule.name);
}

setLang(lang);
initLangToggle();
enhanceAllSelects();
loadSettings();
loadScheduleRules();
