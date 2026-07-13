const API = '';

let policies = [];
let selectedName = '';

function esc(s) {
  const d = document.createElement('div');
  d.textContent = s || '';
  return d.innerHTML;
}

function lines(id) {
  return document.getElementById(id).value.split(/\r?\n/).map(v => v.trim()).filter(Boolean);
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
    tbody.innerHTML = '<tr><td colspan="4" class="empty">No policies configured.</td></tr>';
    return;
  }
  tbody.innerHTML = policies.map((p, idx) => {
    const match = [
      ...(p.domains || []),
      ...(p.ip_ranges || []),
      ...(p.apps || []).map(a => `app:${a}`),
    ];
    return `<tr>
      <td>${esc(p.name)}</td>
      <td>${match.length ? esc(match.join(', ')) : '—'}</td>
      <td>${esc(p.via)}</td>
      <td>
        <div class="policy-row-actions">
          <button class="action" onclick='selectPolicyByIndex(${idx})'>Edit</button>
          <button class="action disconnect" onclick='deletePolicyByIndex(${idx})'>Delete</button>
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
    const e = await res.json().catch(() => ({ error: 'Failed to create policy' }));
    alert(e.error || 'Failed to create policy');
    return;
  }
  await loadPolicies();
  selectedName = rule.name;
}

async function updatePolicy() {
  const rule = draftRule();
  const key = selectedName || rule.name;
  if (!key) {
    alert('Select a policy first or provide a name.');
    return;
  }
  const res = await fetch(`${API}/api/v1/policies/${encodeURIComponent(key)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rule),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: 'Failed to update policy' }));
    alert(e.error || 'Failed to update policy');
    return;
  }
  await loadPolicies();
  selectedName = rule.name || key;
}

async function deletePolicy() {
  const name = selectedName || document.getElementById('p-name').value.trim();
  if (!name) {
    alert('Select a policy to delete.');
    return;
  }
  await deletePolicyByName(name);
}

async function deletePolicyByName(name) {
  if (!confirm(`Delete policy "${name}"?`)) return;
  const res = await fetch(`${API}/api/v1/policies/${encodeURIComponent(name)}`, { method: 'DELETE' });
  if (!res.ok) {
    const e = await res.json().catch(() => ({ error: 'Failed to delete policy' }));
    alert(e.error || 'Failed to delete policy');
    return;
  }

  async function deletePolicyByIndex(index) {
    const p = policies[index];
    if (!p) return;
    await deletePolicyByName(p.name);
  }
  if (selectedName === name) clearForm();
  await loadPolicies();
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
    const e = await res.json().catch(() => ({ error: 'Test failed' }));
    box.className = 'resolve-result no-match';
    box.innerHTML = esc(e.error || 'Test failed');
    box.style.display = 'block';
    return;
  }
  const data = await res.json();
  box.style.display = 'block';
  if (data.matched) {
    box.className = 'resolve-result matched';
    box.innerHTML = `Matched → <strong>${esc(data.via)}</strong> (${esc(data.rule)})`;
  } else {
    box.className = 'resolve-result no-match';
    box.innerHTML = `No match (${esc(data.reason || 'rule does not match this input')})`;
  }
}

loadPolicies();
