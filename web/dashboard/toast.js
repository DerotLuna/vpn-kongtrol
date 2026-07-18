// Lightweight toast notifications, used in place of alert()/confirm() result
// messages across the dashboard pages.

function showToast(message, variant = 'info', timeoutMs = 4500) {
  let stack = document.getElementById('toast-stack');
  if (!stack) {
    stack = document.createElement('div');
    stack.id = 'toast-stack';
    stack.className = 'toast-stack';
    document.body.appendChild(stack);
  }

  const el = document.createElement('div');
  el.className = `toast toast-${variant}`;
  el.textContent = message;
  stack.appendChild(el);

  requestAnimationFrame(() => el.classList.add('show'));

  let dismissed = false;
  const dismiss = () => {
    if (dismissed) return;
    dismissed = true;
    el.classList.remove('show');
    setTimeout(() => el.remove(), 220);
  };

  el.addEventListener('click', dismiss);
  setTimeout(dismiss, timeoutMs);
}
