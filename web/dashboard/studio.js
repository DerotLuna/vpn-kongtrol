const API = '';
const WS_INDICATOR_KEY = 'ws.studio';

const I18N = {
  es: {
    'nav.overview': 'Resumen',
    'nav.studio': 'Estudio',
    'nav.security': 'Seguridad',
    'nav.auditLog': 'Registro de Auditoría',
    'nav.settings': 'Configuración',
    'nav.collapse': 'Contraer',
    'page.studio.title': 'Estudio',
    'ws.studio': 'editor',
    'studio.tabPolicies': 'Políticas',
    'studio.tabProfiles': 'Perfiles VPN',
    'studio.tabGroups': 'Grupos',
    'studio.configPath': 'Archivo de config activo: {path} (solo lectura — cámbialo con --config al iniciar el daemon)',

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
    'actions.connect': 'Conectar',
    'actions.disconnect': 'Desconectar',
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

    'profileStudio.restartNote': 'Los cambios aquí actualizan el archivo de config y el keychain del sistema — reinicia el daemon para conectar con un perfil nuevo o editado.',
    'pEditor.name': 'Nombre',
    'pEditor.namePlaceholder': 'oficina',
    'pEditor.type': 'Tipo',
    'pEditor.priority': 'Prioridad',
    'pEditor.host': 'Host',
    'pEditor.hostPlaceholder': 'vpn.empresa.com',
    'pEditor.configFile': 'Ruta del archivo de config',
    'pEditor.configPlaceholder': '~/.kongtrol/configs/server.ovpn',
    'pEditor.server': 'Server (protonvpn)',
    'pEditor.protocol': 'Protocolo (protonvpn)',
    'pEditor.authMethod': 'Método de autenticación',
    'pEditor.cert': 'Ruta del certificado',
    'pEditor.key': 'Ruta de la clave',
    'pEditor.username': 'Usuario',
    'pEditor.password': 'Contraseña',
    'pEditor.passwordPlaceholder': 'Dejar en blanco para mantener la actual',
    'pEditor.tip': 'Tip: elige una fila de la tabla para editarla. No todos los campos aplican a todos los tipos de adaptador.',
    'pEditor.search': 'Filtrar por nombre, tipo o host…',
    'existingProfiles.title': 'Perfiles existentes',
    'pTable.type': 'Tipo',
    'pTable.target': 'Host / Server',
    'pTable.credentials': 'Credenciales',
    'empty.loadingProfiles': 'Cargando perfiles…',
    'empty.noProfiles': 'No hay perfiles configurados.',
    'cred.userPass': 'usuario + contraseña',
    'cred.userOnly': 'solo usuario',
    'cred.passOnly': 'solo contraseña',
    'cred.none': '—',
    'alerts.profileCreateFailed': 'Error al crear el perfil',
    'alerts.profileUpdateFailed': 'Error al actualizar el perfil',
    'alerts.profileDeleteFailed': 'Error al eliminar el perfil',
    'alerts.profileSelectFirst': 'Selecciona primero un perfil o escribe un nombre.',
    'alerts.profileSelectDelete': 'Selecciona un perfil para eliminar.',
    'confirm.deleteProfile': '¿Eliminar el perfil "{name}"? Esto no afecta túneles ya conectados hasta el próximo reinicio.',
    'toast.profileCreated': 'Perfil "{name}" creado. Reinicia el daemon para activarlo.',
    'toast.profileUpdated': 'Perfil "{name}" actualizado. Reinicia el daemon para aplicar los cambios.',
    'toast.profileDeleted': 'Perfil "{name}" eliminado.',

    'groups.title': 'Grupos',
    'groups.editorTitle': 'Editor de Grupo',
    'groups.name': 'Nombre',
    'groups.namePlaceholder': 'trabajo',
    'groups.profiles': 'Perfiles',
    'groups.tip': 'Tip: elige una fila de la tabla para editarla.',
    'groups.existingTitle': 'Grupos existentes',
    'empty.loadingGroups': 'Cargando grupos…',
    'empty.noGroups': 'No hay grupos configurados.',
    'alerts.groupCreateFailed': 'Error al crear el grupo',
    'alerts.groupUpdateFailed': 'Error al actualizar el grupo',
    'alerts.groupDeleteFailed': 'Error al eliminar el grupo',
    'alerts.groupSelectFirst': 'Selecciona primero un grupo o escribe un nombre.',
    'alerts.groupSelectDelete': 'Selecciona un grupo para eliminar.',
    'confirm.deleteGroup': '¿Eliminar el grupo "{name}"?',
    'toast.groupCreated': 'Grupo "{name}" creado.',
    'toast.groupUpdated': 'Grupo "{name}" actualizado.',
    'toast.groupDeleted': 'Grupo "{name}" eliminado.',
    'toast.groupConnecting': 'Conectando grupo "{name}"…',
    'toast.groupDisconnecting': 'Grupo "{name}" desconectado.',
    'toast.groupActionFailed': 'La acción sobre el grupo falló',
  },
  en: {
    'nav.overview': 'Overview',
    'nav.studio': 'Studio',
    'nav.security': 'Security',
    'nav.auditLog': 'Audit Log',
    'nav.settings': 'Settings',
    'nav.collapse': 'Collapse',
    'page.studio.title': 'Studio',
    'ws.studio': 'editor',
    'studio.tabPolicies': 'Policies',
    'studio.tabProfiles': 'VPN Profiles',
    'studio.tabGroups': 'Groups',
    'studio.configPath': 'Active config file: {path} (read-only — change it with --config when starting the daemon)',

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
    'actions.connect': 'Connect',
    'actions.disconnect': 'Disconnect',
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

    'profileStudio.restartNote': 'Changes here update the config file and OS keychain only — restart the daemon to connect through a new or edited profile.',
    'pEditor.name': 'Name',
    'pEditor.namePlaceholder': 'office',
    'pEditor.type': 'Type',
    'pEditor.priority': 'Priority',
    'pEditor.host': 'Host',
    'pEditor.hostPlaceholder': 'vpn.corp.com',
    'pEditor.configFile': 'Config file path',
    'pEditor.configPlaceholder': '~/.kongtrol/configs/server.ovpn',
    'pEditor.server': 'Server (protonvpn)',
    'pEditor.protocol': 'Protocol (protonvpn)',
    'pEditor.authMethod': 'Auth method',
    'pEditor.cert': 'Certificate path',
    'pEditor.key': 'Key path',
    'pEditor.username': 'Username',
    'pEditor.password': 'Password',
    'pEditor.passwordPlaceholder': 'Leave blank to keep existing',
    'pEditor.tip': 'Tip: pick a row from the table to edit it. Not every field applies to every adapter type.',
    'pEditor.search': 'Filter by name, type, or host…',
    'existingProfiles.title': 'Existing Profiles',
    'pTable.type': 'Type',
    'pTable.target': 'Host / Server',
    'pTable.credentials': 'Credentials',
    'empty.loadingProfiles': 'Loading profiles…',
    'empty.noProfiles': 'No profiles configured.',
    'cred.userPass': 'username + password',
    'cred.userOnly': 'username only',
    'cred.passOnly': 'password only',
    'cred.none': '—',
    'alerts.profileCreateFailed': 'Failed to create profile',
    'alerts.profileUpdateFailed': 'Failed to update profile',
    'alerts.profileDeleteFailed': 'Failed to delete profile',
    'alerts.profileSelectFirst': 'Select a profile first or provide a name.',
    'alerts.profileSelectDelete': 'Select a profile to delete.',
    'confirm.deleteProfile': 'Delete profile "{name}"? This does not affect already-connected tunnels until the next restart.',
    'toast.profileCreated': 'Profile "{name}" created. Restart the daemon to activate it.',
    'toast.profileUpdated': 'Profile "{name}" updated. Restart the daemon to apply the change.',
    'toast.profileDeleted': 'Profile "{name}" deleted.',

    'groups.title': 'Groups',
    'groups.editorTitle': 'Group Editor',
    'groups.name': 'Name',
    'groups.namePlaceholder': 'work',
    'groups.profiles': 'Profiles',
    'groups.tip': 'Tip: pick a row from the table to edit it.',
    'groups.existingTitle': 'Existing Groups',
    'empty.loadingGroups': 'Loading groups…',
    'empty.noGroups': 'No groups configured.',
    'alerts.groupCreateFailed': 'Failed to create group',
    'alerts.groupUpdateFailed': 'Failed to update group',
    'alerts.groupDeleteFailed': 'Failed to delete group',
    'alerts.groupSelectFirst': 'Select a group first or provide a name.',
    'alerts.groupSelectDelete': 'Select a group to delete.',
    'confirm.deleteGroup': 'Delete group "{name}"?',
    'toast.groupCreated': 'Group "{name}" created.',
    'toast.groupUpdated': 'Group "{name}" updated.',
    'toast.groupDeleted': 'Group "{name}" deleted.',
    'toast.groupConnecting': 'Connecting group "{name}"…',
    'toast.groupDisconnecting': 'Group "{name}" disconnected.',
    'toast.groupActionFailed': 'Group action failed',
  },
};

// t/setLang/resolveInitialLang/applyStaticTranslations/initLangToggle/esc
// live in the shared i18n.js — this page only supplies the I18N dict and
// WS_INDICATOR_KEY above, plus the re-render hook below.
function onLangChange() {
  renderPolicies();
  renderProfiles();
  renderGroups();
}

function lines(id) {
  return document.getElementById(id).value.split(/\r?\n/).map((v) => v.trim()).filter(Boolean);
}

// ── Policies ────────────────────────────────────────────────────────────────

let policies = [];
let selectedPolicyName = '';

function draftPolicyRule() {
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

function fillPolicyForm(p) {
  selectedPolicyName = p.name;
  document.getElementById('p-name').value = p.name || '';
  document.getElementById('p-via').value = p.via || '';
  document.getElementById('p-domains').value = (p.domains || []).join('\n');
  document.getElementById('p-ip-ranges').value = (p.ip_ranges || []).join('\n');
  document.getElementById('p-apps').value = (p.apps || []).join('\n');
  enhanceSelect(document.getElementById('p-via'));
}

function clearPolicyForm() {
  selectedPolicyName = '';
  document.getElementById('p-name').value = '';
  document.getElementById('p-domains').value = '';
  document.getElementById('p-ip-ranges').value = '';
  document.getElementById('p-apps').value = '';
  document.getElementById('test-target').value = '';
  document.getElementById('test-app').value = '';
  document.getElementById('test-result').style.display = 'none';
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
  fillPolicyForm(p);
}

async function loadPolicyMeta() {
  const res = await fetch(`${API}/api/v1/policies/meta`);
  if (!res.ok) return;
  const meta = await res.json();
  const sel = document.getElementById('p-via');
  const opts = (meta.profiles || []).map((p) => `<option value="${esc(p)}">${esc(p)}</option>`);
  sel.innerHTML = opts.join('');
  enhanceSelect(sel);

  const pathEl = document.getElementById('studio-config-path');
  if (pathEl && meta.config_path) {
    pathEl.textContent = t('studio.configPath', { path: meta.config_path });
  }
}

async function loadPolicies() {
  await loadPolicyMeta();
  const res = await fetch(`${API}/api/v1/policies`);
  if (!res.ok) return;
  policies = await res.json();
  policies.sort((a, b) => a.name.localeCompare(b.name));
  renderPolicies();
}

async function createPolicy() {
  const rule = draftPolicyRule();
  const res = await fetch(`${API}/api/v1/policies`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rule),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.createFailed') }));
    showToast(e.error || t('alerts.createFailed'), 'error');
    return;
  }
  await loadPolicies();
  selectedPolicyName = rule.name;
}

async function updatePolicy() {
  const rule = draftPolicyRule();
  const key = selectedPolicyName || rule.name;
  if (!key) {
    showToast(t('alerts.selectFirst'), 'error');
    return;
  }
  const res = await fetch(`${API}/api/v1/policies/${encodeURIComponent(key)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rule),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.updateFailed') }));
    showToast(e.error || t('alerts.updateFailed'), 'error');
    return;
  }
  await loadPolicies();
  selectedPolicyName = rule.name || key;
}

async function deletePolicy() {
  const name = selectedPolicyName || document.getElementById('p-name').value.trim();
  if (!name) {
    showToast(t('alerts.selectDelete'), 'error');
    return;
  }
  await deletePolicyByName(name);
}

async function deletePolicyByName(name) {
  if (!confirm(t('confirm.delete', { name }))) return;
  const res = await fetch(`${API}/api/v1/policies/${encodeURIComponent(name)}`, { method: 'DELETE' });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.deleteFailed') }));
    showToast(e.error || t('alerts.deleteFailed'), 'error');
    return;
  }
  if (selectedPolicyName === name) clearPolicyForm();
  await loadPolicies();
}

async function deletePolicyByIndex(index) {
  const p = policies[index];
  if (!p) return;
  await deletePolicyByName(p.name);
}

async function testDraftPolicy() {
  const body = {
    rule: draftPolicyRule(),
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

// ── VPN Profiles ────────────────────────────────────────────────────────────

let profiles = [];
let selectedProfileName = '';

function draftProfile() {
  return {
    name: document.getElementById('v-name').value.trim(),
    type: document.getElementById('v-type').value,
    priority: parseInt(document.getElementById('v-priority').value, 10) || 0,
    host: document.getElementById('v-host').value.trim(),
    config: document.getElementById('v-config').value.trim(),
    server: document.getElementById('v-server').value.trim(),
    protocol: document.getElementById('v-protocol').value.trim(),
    auth_method: document.getElementById('v-auth-method').value,
    cert: document.getElementById('v-cert').value.trim(),
    key: document.getElementById('v-key').value.trim(),
    username: document.getElementById('v-username').value.trim(),
    password: document.getElementById('v-password').value,
  };
}

function fillProfileForm(p) {
  selectedProfileName = p.name;
  document.getElementById('v-name').value = p.name || '';
  document.getElementById('v-type').value = p.type || 'openvpn';
  document.getElementById('v-priority').value = p.priority || 0;
  document.getElementById('v-host').value = p.host || '';
  document.getElementById('v-config').value = p.config || '';
  document.getElementById('v-server').value = p.server || '';
  document.getElementById('v-protocol').value = p.protocol || '';
  document.getElementById('v-auth-method').value = p.auth_method || 'certificate';
  document.getElementById('v-cert').value = '';
  document.getElementById('v-key').value = '';
  document.getElementById('v-username').value = p.username || '';
  document.getElementById('v-password').value = '';
  enhanceSelect(document.getElementById('v-type'));
  enhanceSelect(document.getElementById('v-auth-method'));
}

function clearProfileForm() {
  selectedProfileName = '';
  ['v-name', 'v-host', 'v-config', 'v-server', 'v-protocol', 'v-cert', 'v-key', 'v-username', 'v-password'].forEach((id) => {
    document.getElementById(id).value = '';
  });
  document.getElementById('v-type').value = 'openvpn';
  document.getElementById('v-priority').value = 10;
  document.getElementById('v-auth-method').value = 'certificate';
  enhanceSelect(document.getElementById('v-type'));
  enhanceSelect(document.getElementById('v-auth-method'));
}

function credentialsLabel(p) {
  if (p.has_username_credential && p.has_password_credential) return t('cred.userPass');
  if (p.has_username_credential) return t('cred.userOnly');
  if (p.has_password_credential) return t('cred.passOnly');
  return t('cred.none');
}

function renderProfiles() {
  const tbody = document.getElementById('profile-list-body');
  if (!Array.isArray(profiles) || profiles.length === 0) {
    tbody.innerHTML = `<tr><td colspan="5" class="empty">${t('empty.noProfiles')}</td></tr>`;
    return;
  }

  const searchEl = document.getElementById('profile-search');
  const query = (searchEl ? searchEl.value : '').trim().toLowerCase();
  const indexed = profiles.map((p, idx) => ({ p, idx }));
  const filtered = query
    ? indexed.filter(({ p }) => [p.name, p.type, p.host, p.server].some((v) => (v || '').toLowerCase().includes(query)))
    : indexed;

  if (filtered.length === 0) {
    tbody.innerHTML = `<tr><td colspan="5" class="empty">${t('empty.noProfiles')}</td></tr>`;
    return;
  }

  tbody.innerHTML = filtered.map(({ p, idx }) => {
    const target = p.host || p.server || '—';
    return `<tr>
      <td>${esc(p.name)}</td>
      <td>${esc(p.type)}</td>
      <td>${esc(target)}</td>
      <td class="muted">${esc(credentialsLabel(p))}</td>
      <td>
        <div class="policy-row-actions">
          <button class="action" onclick='selectProfileByIndex(${idx})'>${t('actions.edit')}</button>
          <button class="action disconnect" onclick='deleteProfileByIndex(${idx})'>${t('actions.delete')}</button>
        </div>
      </td>
    </tr>`;
  }).join('');
}

function selectProfileByIndex(index) {
  const p = profiles[index];
  if (!p) return;
  fillProfileForm(p);
}

async function loadProfiles() {
  const res = await fetch(`${API}/api/v1/vpns`);
  if (!res.ok) return;
  profiles = await res.json();
  profiles.sort((a, b) => a.name.localeCompare(b.name));
  renderProfiles();
}

async function createProfile() {
  const draft = draftProfile();
  const res = await fetch(`${API}/api/v1/vpns`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(draft),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.profileCreateFailed') }));
    showToast(e.error || t('alerts.profileCreateFailed'), 'error');
    return;
  }
  showToast(t('toast.profileCreated', { name: draft.name }), 'success', 7000);
  await loadProfiles();
  selectedProfileName = draft.name;
}

async function updateProfile() {
  const draft = draftProfile();
  const key = selectedProfileName || draft.name;
  if (!key) {
    showToast(t('alerts.profileSelectFirst'), 'error');
    return;
  }
  const res = await fetch(`${API}/api/v1/vpns/${encodeURIComponent(key)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(draft),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.profileUpdateFailed') }));
    showToast(e.error || t('alerts.profileUpdateFailed'), 'error');
    return;
  }
  showToast(t('toast.profileUpdated', { name: draft.name || key }), 'success', 7000);
  await loadProfiles();
  selectedProfileName = draft.name || key;
}

async function deleteProfile() {
  const name = selectedProfileName || document.getElementById('v-name').value.trim();
  if (!name) {
    showToast(t('alerts.profileSelectDelete'), 'error');
    return;
  }
  await deleteProfileByName(name);
}

async function deleteProfileByName(name) {
  if (!confirm(t('confirm.deleteProfile', { name }))) return;
  const res = await fetch(`${API}/api/v1/vpns/${encodeURIComponent(name)}`, { method: 'DELETE' });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.profileDeleteFailed') }));
    showToast(e.error || t('alerts.profileDeleteFailed'), 'error');
    return;
  }
  showToast(t('toast.profileDeleted', { name }), 'info');
  if (selectedProfileName === name) clearProfileForm();
  await loadProfiles();
}

async function deleteProfileByIndex(index) {
  const p = profiles[index];
  if (!p) return;
  await deleteProfileByName(p.name);
}

// ── Groups ──────────────────────────────────────────────────────────────────

let groups = [];
let selectedGroupName = '';

function draftGroup() {
  return {
    name: document.getElementById('g-name').value.trim(),
    profiles: lines('g-profiles'),
  };
}

function fillGroupForm(g) {
  selectedGroupName = g.name;
  document.getElementById('g-name').value = g.name || '';
  document.getElementById('g-profiles').value = (g.profiles || []).join('\n');
}

function clearGroupForm() {
  selectedGroupName = '';
  document.getElementById('g-name').value = '';
  document.getElementById('g-profiles').value = '';
}

function renderGroups() {
  const tbody = document.getElementById('group-list-body');
  if (!Array.isArray(groups) || groups.length === 0) {
    tbody.innerHTML = `<tr><td colspan="3" class="empty">${t('empty.noGroups')}</td></tr>`;
    return;
  }

  tbody.innerHTML = groups.map((g, idx) => `<tr>
      <td>${esc(g.name)}</td>
      <td class="muted">${esc((g.profiles || []).join(', '))}</td>
      <td>
        <div class="policy-row-actions">
          <button class="action" onclick='connectGroupByIndex(${idx})'>${t('actions.connect')}</button>
          <button class="action disconnect" onclick='disconnectGroupByIndex(${idx})'>${t('actions.disconnect')}</button>
          <button class="action" onclick='selectGroupByIndex(${idx})'>${t('actions.edit')}</button>
          <button class="action disconnect" onclick='deleteGroupByIndex(${idx})'>${t('actions.delete')}</button>
        </div>
      </td>
    </tr>`).join('');
}

function selectGroupByIndex(index) {
  const g = groups[index];
  if (!g) return;
  fillGroupForm(g);
}

async function loadGroups() {
  const res = await fetch(`${API}/api/v1/groups`);
  if (!res.ok) return;
  groups = await res.json();
  groups.sort((a, b) => a.name.localeCompare(b.name));
  renderGroups();
}

async function createGroup() {
  const draft = draftGroup();
  const res = await fetch(`${API}/api/v1/groups`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(draft),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.groupCreateFailed') }));
    showToast(e.error || t('alerts.groupCreateFailed'), 'error');
    return;
  }
  showToast(t('toast.groupCreated', { name: draft.name }), 'success');
  await loadGroups();
  selectedGroupName = draft.name;
}

async function updateGroup() {
  const draft = draftGroup();
  const key = selectedGroupName || draft.name;
  if (!key) {
    showToast(t('alerts.groupSelectFirst'), 'error');
    return;
  }
  const res = await fetch(`${API}/api/v1/groups/${encodeURIComponent(key)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(draft),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.groupUpdateFailed') }));
    showToast(e.error || t('alerts.groupUpdateFailed'), 'error');
    return;
  }
  showToast(t('toast.groupUpdated', { name: draft.name || key }), 'success');
  await loadGroups();
  selectedGroupName = draft.name || key;
}

async function deleteGroup() {
  const name = selectedGroupName || document.getElementById('g-name').value.trim();
  if (!name) {
    showToast(t('alerts.groupSelectDelete'), 'error');
    return;
  }
  await deleteGroupByName(name);
}

async function deleteGroupByName(name) {
  if (!confirm(t('confirm.deleteGroup', { name }))) return;
  const res = await fetch(`${API}/api/v1/groups/${encodeURIComponent(name)}`, { method: 'DELETE' });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.groupDeleteFailed') }));
    showToast(e.error || t('alerts.groupDeleteFailed'), 'error');
    return;
  }
  showToast(t('toast.groupDeleted', { name }), 'info');
  if (selectedGroupName === name) clearGroupForm();
  await loadGroups();
}

async function deleteGroupByIndex(index) {
  const g = groups[index];
  if (!g) return;
  await deleteGroupByName(g.name);
}

async function connectGroupByIndex(index) {
  const g = groups[index];
  if (!g) return;
  try {
    await fetch(`${API}/api/v1/groups/${encodeURIComponent(g.name)}/connect`, { method: 'POST' });
    showToast(t('toast.groupConnecting', { name: g.name }), 'info');
  } catch (_) {
    showToast(t('toast.groupActionFailed'), 'error');
  }
}

async function disconnectGroupByIndex(index) {
  const g = groups[index];
  if (!g) return;
  const res = await fetch(`${API}/api/v1/groups/${encodeURIComponent(g.name)}/disconnect`, { method: 'POST' });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('toast.groupActionFailed') }));
    showToast(e.error || t('toast.groupActionFailed'), 'error');
    return;
  }
  showToast(t('toast.groupDisconnecting', { name: g.name }), 'success');
}

setLang(lang);
initLangToggle();
enhanceAllSelects();
loadPolicies();
loadProfiles();
loadGroups();
