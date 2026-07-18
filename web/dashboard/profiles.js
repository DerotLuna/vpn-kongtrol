const API = '';
const LANG_KEY = 'kongtrol-dashboard-lang';

const I18N = {
  es: {
    'nav.overview': 'Resumen',
    'nav.policyStudio': 'Policy Studio',
    'nav.security': 'Seguridad',
    'nav.vpnProfiles': 'Perfiles VPN',
    'nav.auditLog': 'Registro de Auditoría',
    'nav.settings': 'Configuración',
    'page.profiles.title': 'Perfiles VPN',
    'ws.profileEditor': 'editor de perfiles',
    'profileStudio.title': 'Perfiles VPN',
    'profileStudio.restartNote': 'Los cambios aquí actualizan el archivo de config y el keychain del sistema — reinicia el daemon para conectar con un perfil nuevo o editado.',
    'editor.title': 'Editor',
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
    'actions.create': 'Crear',
    'actions.update': 'Actualizar',
    'actions.delete': 'Eliminar',
    'actions.clear': 'Limpiar',
    'actions.edit': 'Editar',
    'existing.title': 'Perfiles existentes',
    'table.name': 'Nombre',
    'table.actions': 'Acciones',
    'pTable.type': 'Tipo',
    'pTable.target': 'Host / Server',
    'pTable.credentials': 'Credenciales',
    'empty.loadingProfiles': 'Cargando perfiles…',
    'empty.noProfiles': 'No hay perfiles configurados.',
    'cred.userPass': 'usuario + contraseña',
    'cred.userOnly': 'solo usuario',
    'cred.passOnly': 'solo contraseña',
    'cred.none': '—',
    'alerts.createFailed': 'Error al crear el perfil',
    'alerts.updateFailed': 'Error al actualizar el perfil',
    'alerts.deleteFailed': 'Error al eliminar el perfil',
    'alerts.selectFirst': 'Selecciona primero un perfil o escribe un nombre.',
    'alerts.selectDelete': 'Selecciona un perfil para eliminar.',
    'confirm.delete': '¿Eliminar el perfil "{name}"? Esto no afecta túneles ya conectados hasta el próximo reinicio.',
    'toast.created': 'Perfil "{name}" creado. Reinicia el daemon para activarlo.',
    'toast.updated': 'Perfil "{name}" actualizado. Reinicia el daemon para aplicar los cambios.',
    'toast.deleted': 'Perfil "{name}" eliminado.',
    'common.refresh': 'Actualizar',
    'groups.title': 'Grupos',
    'groups.editorTitle': 'Editor de Grupo',
    'groups.name': 'Nombre',
    'groups.namePlaceholder': 'trabajo',
    'groups.profiles': 'Perfiles',
    'groups.tip': 'Tip: elige una fila de la tabla para editarla.',
    'groups.existingTitle': 'Grupos existentes',
    'empty.loadingGroups': 'Cargando grupos…',
    'empty.noGroups': 'No hay grupos configurados.',
    'actions.connect': 'Conectar',
    'actions.disconnect': 'Desconectar',
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
    'nav.policyStudio': 'Policy Studio',
    'nav.security': 'Security',
    'nav.vpnProfiles': 'VPN Profiles',
    'nav.auditLog': 'Audit Log',
    'nav.settings': 'Settings',
    'page.profiles.title': 'VPN Profiles',
    'ws.profileEditor': 'profile editor',
    'profileStudio.title': 'VPN Profiles',
    'profileStudio.restartNote': 'Changes here update the config file and OS keychain only — restart the daemon to connect through a new or edited profile.',
    'editor.title': 'Editor',
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
    'actions.create': 'Create',
    'actions.update': 'Update',
    'actions.delete': 'Delete',
    'actions.clear': 'Clear',
    'actions.edit': 'Edit',
    'existing.title': 'Existing Profiles',
    'table.name': 'Name',
    'table.actions': 'Actions',
    'pTable.type': 'Type',
    'pTable.target': 'Host / Server',
    'pTable.credentials': 'Credentials',
    'empty.loadingProfiles': 'Loading profiles…',
    'empty.noProfiles': 'No profiles configured.',
    'cred.userPass': 'username + password',
    'cred.userOnly': 'username only',
    'cred.passOnly': 'password only',
    'cred.none': '—',
    'alerts.createFailed': 'Failed to create profile',
    'alerts.updateFailed': 'Failed to update profile',
    'alerts.deleteFailed': 'Failed to delete profile',
    'alerts.selectFirst': 'Select a profile first or provide a name.',
    'alerts.selectDelete': 'Select a profile to delete.',
    'confirm.delete': 'Delete profile "{name}"? This does not affect already-connected tunnels until the next restart.',
    'toast.created': 'Profile "{name}" created. Restart the daemon to activate it.',
    'toast.updated': 'Profile "{name}" updated. Restart the daemon to apply the change.',
    'toast.deleted': 'Profile "{name}" deleted.',
    'common.refresh': 'Refresh',
    'groups.title': 'Groups',
    'groups.editorTitle': 'Group Editor',
    'groups.name': 'Name',
    'groups.namePlaceholder': 'work',
    'groups.profiles': 'Profiles',
    'groups.tip': 'Tip: pick a row from the table to edit it.',
    'groups.existingTitle': 'Existing Groups',
    'empty.loadingGroups': 'Loading groups…',
    'empty.noGroups': 'No groups configured.',
    'actions.connect': 'Connect',
    'actions.disconnect': 'Disconnect',
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

let profiles = [];
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
  renderProfiles();
  renderGroups();
}

function applyStaticTranslations() {
  document.querySelectorAll('[data-i18n]').forEach((el) => {
    const key = el.getAttribute('data-i18n');
    if (!key) return;
    el.textContent = t(key);
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

  const wsLabel = document.getElementById('ws-label');
  if (wsLabel) wsLabel.textContent = t('ws.profileEditor');
  const indicator = document.getElementById('connection-indicator');
  if (indicator) {
    indicator.className = 'indicator connected';
    indicator.title = 'Profile editor';
  }
}

function initLangToggle() {
  const toggle = document.getElementById('lang-toggle');
  if (!toggle) return;
  toggle.addEventListener('click', () => {
    setLang(lang === 'es' ? 'en' : 'es');
  });
}

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

function fillForm(p) {
  selectedName = p.name;
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
}

function clearForm() {
  selectedName = '';
  ['v-name', 'v-host', 'v-config', 'v-server', 'v-protocol', 'v-cert', 'v-key', 'v-username', 'v-password'].forEach((id) => {
    document.getElementById(id).value = '';
  });
  document.getElementById('v-type').value = 'openvpn';
  document.getElementById('v-priority').value = 10;
  document.getElementById('v-auth-method').value = 'certificate';
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
  fillForm(p);
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
    const e = await res.json().catch(() => ({ error: t('alerts.createFailed') }));
    showToast(e.error || t('alerts.createFailed'), 'error');
    return;
  }
  showToast(t('toast.created', { name: draft.name }), 'success', 7000);
  await loadProfiles();
  selectedName = draft.name;
}

async function updateProfile() {
  const draft = draftProfile();
  const key = selectedName || draft.name;
  if (!key) {
    showToast(t('alerts.selectFirst'), 'error');
    return;
  }
  const res = await fetch(`${API}/api/v1/vpns/${encodeURIComponent(key)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(draft),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.updateFailed') }));
    showToast(e.error || t('alerts.updateFailed'), 'error');
    return;
  }
  showToast(t('toast.updated', { name: draft.name || key }), 'success', 7000);
  await loadProfiles();
  selectedName = draft.name || key;
}

async function deleteProfile() {
  const name = selectedName || document.getElementById('v-name').value.trim();
  if (!name) {
    showToast(t('alerts.selectDelete'), 'error');
    return;
  }
  await deleteProfileByName(name);
}

async function deleteProfileByName(name) {
  if (!confirm(t('confirm.delete', { name }))) return;
  const res = await fetch(`${API}/api/v1/vpns/${encodeURIComponent(name)}`, { method: 'DELETE' });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('alerts.deleteFailed') }));
    showToast(e.error || t('alerts.deleteFailed'), 'error');
    return;
  }
  showToast(t('toast.deleted', { name }), 'info');
  if (selectedName === name) clearForm();
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

function lines(id) {
  return document.getElementById(id).value.split(/\r?\n/).map((v) => v.trim()).filter(Boolean);
}

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
loadProfiles();
loadGroups();
