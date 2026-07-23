const API = '';
const WS_INDICATOR_KEY = 'ws.security';

const I18N = {
  es: {
    'nav.overview': 'Resumen',
    'nav.studio': 'Estudio',
    'nav.security': 'Seguridad',
    'nav.auditLog': 'Registro de Auditoría',
    'nav.settings': 'Configuración',
    'nav.collapse': 'Contraer',
    'page.security.title': 'Seguridad',
    'ws.security': 'controles en vivo',
    'sections.controls': 'Controles',
    'sections.leakDetection': 'Detección de Fugas',
    'security.killSwitch': 'Kill Switch',
    'security.killSwitchDesc': 'Bloquea todo el tráfico fuera del túnel si una VPN protegida se desconecta inesperadamente.',
    'security.dnsGuard': 'DNS Guard',
    'security.dnsGuardDesc': 'Fuerza la resolución DNS a través de los resolutores del túnel para evitar fugas de DNS.',
    'security.leakCheck': 'Leak Check',
    'security.publicIp': 'IP pública',
    'security.lastChecked': 'Última revisión',
    'security.perProfile': 'Override de Kill Switch por Perfil',
    'security.perProfileNote': 'Sobrescribe el ajuste global de Kill Switch para un perfil VPN específico.',
    'security.override': 'Override',
    'override.inherit': 'Heredar (global)',
    'override.on': 'Forzar activado',
    'override.off': 'Forzar desactivado',
    'table.profile': 'Perfil',
    'empty.loading': 'Cargando…',
    'empty.noProfiles': 'No hay perfiles configurados.',
    'toast.overrideSaved': 'Override de "{name}" actualizado',
    'toast.overrideFailed': 'No se pudo actualizar el override',
    'badge.disabled': 'DESACTIVADO',
    'badge.on': 'ACTIVO',
    'badge.idle': 'INACTIVO',
    'badge.leak': 'FUGA',
    'badge.clean': 'LIMPIO',
    'badge.pending': 'PENDIENTE',
    'badge.error': 'ERROR',
    'badge.unavailable': 'NO DISPONIBLE',
    'confirm.disableKillSwitch': '¿Desactivar el Kill Switch? El tráfico ya no se bloqueará si un túnel protegido se desconecta.',
    'toast.killSwitchOn': 'Kill Switch activado',
    'toast.killSwitchOff': 'Kill Switch desactivado',
    'toast.dnsGuardOn': 'DNS Guard activado',
    'toast.dnsGuardOff': 'DNS Guard desactivado',
    'toast.toggleFailed': 'No se pudo actualizar la configuración',
    'common.refresh': 'Actualizar',
  },
  en: {
    'nav.overview': 'Overview',
    'nav.studio': 'Studio',
    'nav.security': 'Security',
    'nav.auditLog': 'Audit Log',
    'nav.settings': 'Settings',
    'nav.collapse': 'Collapse',
    'page.security.title': 'Security',
    'ws.security': 'live controls',
    'sections.controls': 'Controls',
    'sections.leakDetection': 'Leak Detection',
    'security.killSwitch': 'Kill Switch',
    'security.killSwitchDesc': 'Blocks all network traffic outside the tunnel if a protected VPN drops unexpectedly.',
    'security.dnsGuard': 'DNS Guard',
    'security.dnsGuardDesc': "Forces DNS resolution through the tunnel's resolvers to prevent DNS leaks.",
    'security.leakCheck': 'Leak Check',
    'security.publicIp': 'Public IP',
    'security.lastChecked': 'Last checked',
    'security.perProfile': 'Per-Profile Kill Switch Override',
    'security.perProfileNote': 'Overrides the global Kill Switch setting for a specific VPN profile.',
    'security.override': 'Override',
    'override.inherit': 'Inherit (global)',
    'override.on': 'Force on',
    'override.off': 'Force off',
    'table.profile': 'Profile',
    'empty.loading': 'Loading…',
    'empty.noProfiles': 'No profiles configured.',
    'toast.overrideSaved': 'Override for "{name}" updated',
    'toast.overrideFailed': 'Failed to update override',
    'badge.disabled': 'DISABLED',
    'badge.on': 'ON',
    'badge.idle': 'IDLE',
    'badge.leak': 'LEAK',
    'badge.clean': 'CLEAN',
    'badge.pending': 'PENDING',
    'badge.error': 'ERROR',
    'badge.unavailable': 'UNAVAILABLE',
    'confirm.disableKillSwitch': 'Disable the Kill Switch? Traffic will no longer be blocked if a protected tunnel drops.',
    'toast.killSwitchOn': 'Kill Switch enabled',
    'toast.killSwitchOff': 'Kill Switch disabled',
    'toast.dnsGuardOn': 'DNS Guard enabled',
    'toast.dnsGuardOff': 'DNS Guard disabled',
    'toast.toggleFailed': 'Failed to update setting',
    'common.refresh': 'Refresh',
  },
};

let lastPublicIP = '';

// t/setLang/resolveInitialLang/applyStaticTranslations/initLangToggle/esc
// live in the shared i18n.js — this page only supplies the I18N dict and
// WS_INDICATOR_KEY above.

function setBadge(id, label, stateClass) {
  const el = document.getElementById(id);
  if (!el) return;
  el.textContent = label;
  el.className = 'badge ' + stateClass;
}

async function refreshSecurity() {
  try {
    const res = await fetch(`${API}/api/v1/security/status`);
    if (!res.ok) return;
    const data = await res.json();

    const ksToggle = document.getElementById('ks-toggle');
    if (ksToggle && document.activeElement !== ksToggle) ksToggle.checked = !!data.kill_switch_enabled;
    const dnsToggle = document.getElementById('dns-toggle');
    if (dnsToggle && document.activeElement !== dnsToggle) dnsToggle.checked = !!data.dns_guard_enabled;
    setRuntimeBadge('ks-runtime-status', data.kill_switch_state);
    setRuntimeBadge('dns-runtime-status', data.dns_guard_state);

    if (!data.leak_check_available) {
      setBadge(
        'leak-status',
        data.leak_detection_configured ? t('badge.unavailable') : t('badge.disabled'),
        data.leak_detection_configured ? 'leak' : 'disabled',
      );
      document.getElementById('public-ip').textContent = lastPublicIP || '—';
      document.getElementById('leak-time').textContent = '—';
      return;
    }

    if (data.leak_check) {
      const lk = data.leak_check;
      if (lk.state === 'error') {
        setBadge('leak-status', t('badge.error'), 'leak');
      } else {
        setBadge('leak-status', lk.has_leak ? t('badge.leak') : t('badge.clean'), lk.has_leak ? 'leak' : 'ok');
      }
      lastPublicIP = lk.public_ip || '';
      document.getElementById('public-ip').textContent = lastPublicIP || '—';
      if (lk.checked_at) {
        document.getElementById('leak-time').textContent = new Date(lk.checked_at).toLocaleTimeString(lang);
      } else {
        document.getElementById('leak-time').textContent = '—';
      }

      function setRuntimeBadge(id, state) {
        if (state === 'disabled') {
          setBadge(id, t('badge.disabled'), 'disabled');
        } else if (state === 'armed' || state === 'active') {
          setBadge(id, t('badge.on'), 'on');
        } else {
          setBadge(id, t('badge.idle'), 'pending');
        }
      }
      return;
    }

    setBadge('leak-status', t('badge.pending'), 'pending');
    document.getElementById('leak-time').textContent = '—';
  } catch (_) {}
}

async function postToggle(path, enabled) {
  const res = await fetch(`${API}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled }),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('toast.toggleFailed') }));
    throw new Error(e.error || t('toast.toggleFailed'));
  }
}

async function onToggleKillSwitch(input) {
  const enabled = input.checked;
  if (!enabled && !confirm(t('confirm.disableKillSwitch'))) {
    input.checked = true;
    return;
  }
  input.disabled = true;
  try {
    await postToggle('/api/v1/security/killswitch', enabled);
    showToast(enabled ? t('toast.killSwitchOn') : t('toast.killSwitchOff'), enabled ? 'success' : 'info');
  } catch (err) {
    input.checked = !enabled;
    showToast(err.message, 'error');
  } finally {
    input.disabled = false;
    refreshSecurity();
  }
}

async function onToggleDNSGuard(input) {
  const enabled = input.checked;
  input.disabled = true;
  try {
    await postToggle('/api/v1/security/dnsguard', enabled);
    showToast(enabled ? t('toast.dnsGuardOn') : t('toast.dnsGuardOff'), enabled ? 'success' : 'info');
  } catch (err) {
    input.checked = !enabled;
    showToast(err.message, 'error');
  } finally {
    input.disabled = false;
    refreshSecurity();
  }
}

// ── Per-profile kill switch override ───────────────────────────────────────

function renderProfileOverrides(profiles) {
  const tbody = document.getElementById('profile-ks-body');
  if (!Array.isArray(profiles) || profiles.length === 0) {
    tbody.innerHTML = `<tr><td colspan="2" class="empty">${t('empty.noProfiles')}</td></tr>`;
    return;
  }

  tbody.innerHTML = profiles.map((p) => `<tr>
      <td>${esc(p.name)}</td>
      <td>
        <select onchange='onChangeProfileOverride(${JSON.stringify(p.name)}, this.value)'>
          <option value="inherit" ${p.kill_switch_override === 'inherit' ? 'selected' : ''}>${t('override.inherit')}</option>
          <option value="on" ${p.kill_switch_override === 'on' ? 'selected' : ''}>${t('override.on')}</option>
          <option value="off" ${p.kill_switch_override === 'off' ? 'selected' : ''}>${t('override.off')}</option>
        </select>
      </td>
    </tr>`).join('');
  enhanceAllSelects(tbody);
}

async function refreshProfileOverrides() {
  try {
    const res = await fetch(`${API}/api/v1/vpns`);
    if (!res.ok) return;
    const profiles = await res.json();
    profiles.sort((a, b) => a.name.localeCompare(b.name));
    renderProfileOverrides(profiles);
  } catch (_) {}
}

async function onChangeProfileOverride(name, override) {
  const res = await fetch(`${API}/api/v1/vpns/${encodeURIComponent(name)}/killswitch`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ override }),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: t('toast.overrideFailed') }));
    showToast(e.error || t('toast.overrideFailed'), 'error');
    refreshProfileOverrides();
    return;
  }
  showToast(t('toast.overrideSaved', { name }), 'success');
}

setLang(lang);
initLangToggle();
enhanceAllSelects();
refreshSecurity();
refreshProfileOverrides();
setInterval(refreshSecurity, 15_000);
