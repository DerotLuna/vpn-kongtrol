// Progressively enhances native <select> elements into a custom-styled
// dropdown that matches the rest of the dashboard (native <select> menus
// can't be themed consistently across browsers/OSes). The original <select>
// stays in the DOM as the source of truth — it's visually hidden but still
// receives .value/.dispatchEvent('change'), so existing code that reads
// select.value or listens for 'change' keeps working unmodified.
//
// Usage:
//   enhanceAllSelects()         — call once after a page's static <select>s exist
//   enhanceSelect(selectEl)     — call after populating/rebuilding one select's
//                                 <option>s at runtime (e.g. sel.innerHTML = ...)

function enhanceSelect(selectEl) {
  if (!selectEl || selectEl.tagName !== 'SELECT') return;

  let wrap = selectEl.closest('.ks-select');
  if (!wrap) {
    wrap = document.createElement('div');
    wrap.className = 'ks-select';
    selectEl.parentNode.insertBefore(wrap, selectEl);
    wrap.appendChild(selectEl);

    const trigger = document.createElement('button');
    trigger.type = 'button';
    trigger.className = 'ks-select-trigger';
    trigger.innerHTML = '<span class="ks-select-label"></span><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="6 9 12 15 18 9"/></svg>';
    wrap.appendChild(trigger);

    const listbox = document.createElement('div');
    listbox.className = 'ks-select-listbox';
    listbox.setAttribute('role', 'listbox');
    wrap.appendChild(listbox);

    const close = () => wrap.classList.remove('open');
    const open = () => {
      document.querySelectorAll('.ks-select.open').forEach((o) => { if (o !== wrap) o.classList.remove('open'); });
      wrap.classList.add('open');
      const active = listbox.querySelector('.ks-select-option.selected') || listbox.querySelector('.ks-select-option');
      if (active) active.focus();
    };

    trigger.addEventListener('click', (e) => {
      e.stopPropagation();
      if (selectEl.disabled) return;
      wrap.classList.contains('open') ? close() : open();
    });
    trigger.addEventListener('keydown', (e) => {
      if (e.key === 'ArrowDown' || e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        open();
      }
    });
    document.addEventListener('click', close);
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') close();
    });

    wrap._ksClose = close;
  }

  const trigger = wrap.querySelector('.ks-select-trigger');
  const label = wrap.querySelector('.ks-select-label');
  const listbox = wrap.querySelector('.ks-select-listbox');

  const selectOption = (index) => {
    selectEl.selectedIndex = index;
    selectEl.dispatchEvent(new Event('change', { bubbles: true }));
    wrap._ksClose();
    trigger.focus();
  };

  const renderOptions = () => {
    listbox.innerHTML = '';
    Array.from(selectEl.options).forEach((opt, i) => {
      const item = document.createElement('div');
      item.className = 'ks-select-option' + (opt.selected ? ' selected' : '');
      item.textContent = opt.textContent;
      item.setAttribute('role', 'option');
      item.tabIndex = -1;
      item.addEventListener('click', () => selectOption(i));
      item.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selectOption(i); }
        else if (e.key === 'ArrowDown') { e.preventDefault(); (item.nextElementSibling || listbox.firstElementChild)?.focus(); }
        else if (e.key === 'ArrowUp') { e.preventDefault(); (item.previousElementSibling || listbox.lastElementChild)?.focus(); }
      });
      listbox.appendChild(item);
    });
  };

  const sync = () => {
    const opt = selectEl.options[selectEl.selectedIndex];
    label.textContent = opt ? opt.textContent : '';
    listbox.querySelectorAll('.ks-select-option').forEach((el, i) => {
      el.classList.toggle('selected', i === selectEl.selectedIndex);
    });
    trigger.classList.toggle('disabled', !!selectEl.disabled);
  };

  renderOptions();
  sync();
  if (!selectEl._ksBound) {
    selectEl._ksBound = true;
    selectEl.addEventListener('change', sync);
  }
}

function enhanceAllSelects(root) {
  (root || document).querySelectorAll('select').forEach(enhanceSelect);
}
