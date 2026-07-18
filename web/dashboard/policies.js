const API = '';
const LANG_KEY = 'kongtrol-dashboard-lang';

const I18N = {
  es: {
    'nav.overview': 'Resumen',
    'nav.policyStudio': 'Policy Studio',
    'ws.configEditor': 'editor de config',
    'policyStudio.title': 'Policy Studio (CRUD + Prueba)',
    'editor.title': 'Editor',
    'editor.name': 'Nombre',
    'editor.namePlaceholder': 'Contenido US',
    'editor.viaProfile': 'Vía perfil VPN',
    'editor.domains': 'Dominios (uno por línea)',
    'editor.ipRanges': 'Rangos IP (uno por línea)',
    'editor.apps': 'Apps (una por línea)',
    'editor.tip': 'Tip: elige una fila de la tabla para editarla.',
    'editor.template': 'Plantilla: usa dominios como <code>*.java.com</code>, rangos IP como <code>10.2.0.2/32</code>, y apps como <code>chrome.exe</code>.',
    'actions.create': 'Crear',
    'actions.update': 'Actualizar',
    'actions.delete': 'Eliminar',
    'actions.clear': 'Limpiar',
    'actions.edit': 'Editar',
    'test.title': 'Probar política antes de guardar',
    'test.targetPlaceholder': 'Objetivo: dominio/IP/URL',
    'test.appPlaceholder': 'o app: chrome.exe',
    'test.button': 'Probar',
    'existing.title': 'Políticas existentes',
    'table.name': 'Nombre',
    'table.match': 'Match',
    'table.via': 'Vía',
    'table.actions': 'Acciones',
    'empty.loadingPolicies': 'Cargando políticas…',
    'empty.noPolicies': 'No hay políticas configuradas.',
    'alerts.createFailed': 'Error al crear política',
    'alerts.updateFailed': 'Error al actualizar política',
    'alerts.deleteFailed': 'Error al eliminar política',
    'alerts.testFailed': 'Falló la prueba',
    'alerts.selectFirst': 'Selecciona primero una política o escribe un nombre.',
    'alerts.selectDelete': 'Selecciona una política para eliminar.',
    'confirm.delete': '¿Eliminar política "{name}"?',
    'result.matched': 'Match → <strong>{via}</strong> ({rule})',
    'result.noMatch': 'Sin match ({reason})',
    'result.noMatchReason': 'la regla no coincide con esta entrada',
    'common.refresh': 'Actualizar',
  },
  en: {
    'nav.overview': 'Overview',
    'nav.policyStudio': 'Policy Studio',
    'ws.configEditor': 'config editor',
    'policyStudio.title': 'Policy Studio (CRUD + Test)',
    'editor.title': 'Editor',
    'editor.name': 'Name',
    'editor.namePlaceholder': 'US Content',
    'editor.viaProfile': 'Via VPN profile',
    'editor.domains': 'Domains (one per line)',
    'editor.ipRanges': 'IP ranges (one per line)',
    'editor.apps': 'Apps (one per line)',
    'editor.tip': 'Tip: pick a row from the table to edit it.',
    'editor.template': 'Template: use domains like <code>*.java.com</code>, IP ranges like <code>10.2.0.2/32</code>, and apps like <code>chrome.exe</code>.',
    'actions.create': 'Create',
    'actions.update': 'Update',
    'actions.delete': 'Delete',
    'actions.clear': 'Clear',
    'actions.edit': 'Edit',
    'test.title': 'Test Policy Before Saving',
    'test.targetPlaceholder': 'Target: domain/IP/URL',
    'test.appPlaceholder': 'or app: chrome.exe',
    'test.button': 'Test',
    'existing.title': 'Existing Policies',
    'table.name': 'Name',
    'table.match': 'Match',
    'table.via': 'Via',
    'table.actions': 'Actions',
    'empty.loadingPolicies': 'Loading policies…',
    'empty.noPolicies': 'No policies configured.',
    'alerts.createFailed': 'Failed to create policy',
    'alerts.updateFailed': 'Failed to update policy',
    'alerts.deleteFailed': 'Failed to delete policy',
    'alerts.testFailed': 'Test failed',
    'alerts.selectFirst': 'Select a policy first or provide a name.',
    'alerts.selectDelete': 'Select a policy to delete.',
    'confirm.delete': 'Delete policy "{name}"?',
    'result.matched': 'Matched → <strong>{via}</strong> ({rule})',
    'result.noMatch': 'No match ({reason})',
    'result.noMatchReason': 'rule does not match this input',
    'common.refresh': 'Refresh',
  },
};

let policies = [];
let selectedName = '';
let lang = resolveInitialLang();

function resolveInitialLang() {
  const saved = localStorage.getItem(LANG_KEY);
  if (saved === 'es' || saved === 'en') return saved;
  return navigator.language && navigator.language.toLowerCase().startsWith('es') ? 'es' : 'en';
}

function t(key, vars = {}) {
  const dict = I18N[lang] || I18N.en;
  const raw = dict[key] || I18N.en[key] || key;
  return raw.replace(/\{(\w+)\}/g, (_, k) => String(vars[k] ?? ''));
}

function esc(s) {
  const d = document.createElement('div');
  d.textContent = s || '';
  return d.innerHTML;
}

function setLang(next) {
  lang = next === 'es' ? 'es' : 'en';
  localStorage.setItem(LANG_KEY, lang);
  document.documentElement.lang = lang;
  applyStaticTranslations();
  renderPolicies();
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
    toggle.textContent = lang.toUpperCase();
    toggle.setAttribute('aria-label', lang === 'es' ? 'Cambiar idioma' : 'Toggle language');
    toggle.title = lang === 'es' ? 'Cambiar a English' : 'Switch to Español';
  }
}

function initLangToggle() {
  const toggle = document.getElementById('lang-toggle');
  if (!toggle) return;
  toggle.addEventListener('click', () => {
    setLang(lang === 'es' ? 'en' : 'es');
  });
}

function lines(id) {
  return document.getElementById(id).value.split(/\r?\n/).map((v) => v.trim()).filter(Boolean);
}

function draftRule() {
  return {
    name: document.getElementById('p-name').value.trim(),
    via: document.getElementById('p-via').value.trim(),
    match: {
      domains: lines('p-domains'),
      ip_ranges: lines('p-ip-ranges'),
      apps: lines('p-apps'),
    },
  };
}

function fillForm(p) {
  selectedName = p.name;
  document.getElementById('p-name').value = p.name || '';
  document.getElementById('p-via').value = p.via || '';
  document.getElementById('p-domains').value = (p.domains || []).join('\n');
  document.getElementById('p-ip-ranges').value = (p.ip_ranges || []).join('\n');
  document.getElementById('p-apps').value = (p.apps || []).join('\n');
}

function clearForm() {
  selectedName = '';
  document.getElementById('p-name').value = '';
  document.getElementById('p-domains').value = '';
  document.getElementById('p-ip-ranges').value = '';
  document.getElementById('p-apps').value = '';
  document.getElementById('test-target').value = '';
  document.getElementById('test-app').value = '';
  const result = document.getElementById('test-result');
  result.style.display = 'none';
}

function renderPolicies() {
  const tbody = document.getElementById('policy-list-body');
  if (!Array.isArray(policies) || policies.length === 0) {
    tbody.innerHTML = `<tr><td colspan="4" class="empty">${t('empty.noPolicies')}</td></tr>`;
    return;
  }

  tbody.innerHTML = policies.map((p, idx) => {
    const match = [
      ...(p.domains || []),
      ...(p.ip_ranges || []),
      ...(p.apps || []).map((a) => `app:${a}`),
    ];
    return `<tr>
      <td>${esc(p.name)}</td>
      <td>${match.length ? esc(match.join(', ')) : '—'}</td>
      <td>${esc(p.via)}</td>
      <td>
        <div class="policy-row-actions">
          <button class="action" onclick='selectPolicyByIndex(${idx})'>${t('actions.edit')}</button>
          <button class="action disconnect" onclick='deletePolicyByIndex(${idx})'>${t('actions.delete')}</button>
        </div>
      </td>
    </tr>`;
  }).join('');
}

function selectPolicyByIndex(index) {
  const p = policies[index];
  if (!p) return;
  fillForm(p);
}

async function loadMeta() {
  const res = await fetch(`${API}/api/v1/policies/meta`);
  if (!res.ok) return;
  const meta = await res.json();
  const sel = document.getElementById('p-via');
  const opts = (meta.profiles || []).map((p) => `<option value="${esc(p)}">${esc(p)}</option>`);
  sel.innerHTML = opts.join('');
}

async function loadPolicies() {
  await loadMeta();
  const res = await fetch(`${API}/api/v1/policies`);
  if (!res.ok) return;
  policies = await res.json();
  policies.sort((a, b) => a.name.localeCompare(b.name));
  renderPolicies();
}

async function createPolicy() {
  const rule = draftRule();
  const res = await fetch(`${API}/api/v1/policies`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rule),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.createFailed') }));
    alert(e.error || t('alerts.createFailed'));
    return;
  }
  await loadPolicies();
  selectedName = rule.name;
}

async function updatePolicy() {
  const rule = draftRule();
  const key = selectedName || rule.name;
  if (!key) {
    alert(t('alerts.selectFirst'));
    return;
  }
  const res = await fetch(`${API}/api/v1/policies/${encodeURIComponent(key)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rule),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.updateFailed') }));
    alert(e.error || t('alerts.updateFailed'));
    return;
  }
  await loadPolicies();
  selectedName = rule.name || key;
}

async function deletePolicy() {
  const name = selectedName || document.getElementById('p-name').value.trim();
  if (!name) {
    alert(t('alerts.selectDelete'));
    return;
  }
  await deletePolicyByName(name);
}

async function deletePolicyByName(name) {
  if (!confirm(t('confirm.delete', { name }))) return;
  const res = await fetch(`${API}/api/v1/policies/${encodeURIComponent(name)}`, { method: 'DELETE' });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.deleteFailed') }));
    alert(e.error || t('alerts.deleteFailed'));
    return;
  }
  if (selectedName === name) clearForm();
  await loadPolicies();
}

async function deletePolicyByIndex(index) {
  const p = policies[index];
  if (!p) return;
  await deletePolicyByName(p.name);
}

async function testDraftPolicy() {
  const body = {
    rule: draftRule(),
    target: document.getElementById('test-target').value.trim(),
    app: document.getElementById('test-app').value.trim(),
  };
  const res = await fetch(`${API}/api/v1/policies/test`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const box = document.getElementById('test-result');
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.testFailed') }));
    box.className = 'resolve-result no-match';
    box.innerHTML = esc(e.error || t('alerts.testFailed'));
    box.style.display = 'block';
    return;
  }
  const data = await res.json();
  box.style.display = 'block';
  if (data.matched) {
    box.className = 'resolve-result matched';
    box.innerHTML = t('result.matched', { via: esc(data.via), rule: esc(data.rule) });
  } else {
    box.className = 'resolve-result no-match';
    box.innerHTML = t('result.noMatch', { reason: esc(data.reason || t('result.noMatchReason')) });
  }
}

setLang(lang);
initLangToggle();
loadPolicies();
