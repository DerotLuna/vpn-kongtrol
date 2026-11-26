// Kongtrol UI — toast / snackbar queue · v0.1.0 · 2026-11-18
//
// One toast on screen at a time, the rest wait in order (Material 3
// snackbars). Short confirmations close by themselves; errors stay until
// dismissed, and every timer pauses while the pointer or focus is on the
// toast (WCAG 2.2.1 Timing Adjustable). Rendered by components/ui/Toaster.
import { createStore, useStore } from './store'

export type ToastTone = 'ok' | 'info' | 'bad'

export interface ToastAction {
  label: string
  run: () => void
}

export interface Toast {
  id: number
  tone: ToastTone
  message: string
  /** The "what to do" line for errors, or a short detail. */
  detail?: string
  action?: ToastAction
  /** ms; 0 = stays until dismissed. */
  duration: number
  notice?: { profile: string; key: string; kind: string; offer?: string; args?: string[]; connectAction?: boolean }
  actionHidden?: boolean
}

export type ToastInput = Omit<Toast, 'id' | 'duration'> & { duration?: number }

/** Errors stay until dismissed; a toast with an action gets 10s to reach it. */
export const TOAST_MS = { ok: 4_000, info: 4_000, bad: 0, withAction: 10_000 } as const

const queue = createStore<readonly Toast[]>([])
let nextId = 1

export const toasts = {
  show(input: ToastInput): number {
    const id = nextId++
    const fallback = input.tone === 'bad' ? TOAST_MS.bad : input.action ? TOAST_MS.withAction : TOAST_MS[input.tone]
    const toast: Toast = { duration: fallback, ...input, id }
    // The same message twice in a row is one toast, not a stack.
    const list = queue.get().filter((t) => !(t.message === toast.message && t.tone === toast.tone))
    queue.set([...list, toast])
    return id
  },
  dismiss(id: number): void {
    queue.set(queue.get().filter((t) => t.id !== id))
  },
  dismissWhere(predicate: (toast: Toast) => boolean): void {
    const list = queue.get()
    const kept = list.filter((toast) => !predicate(toast))
    if (kept.length !== list.length) queue.set(kept)
  },
  setConnectActionHidden(profile: string, hidden: boolean): void {
    const list = queue.get()
    let changed = false
    const next = list.map((toast) => {
      if (toast.notice?.profile !== profile || !toast.notice.connectAction || !!toast.actionHidden === hidden) return toast
      changed = true
      return { ...toast, actionHidden: hidden }
    })
    if (changed) queue.set(next)
  },
  resolveNotice(profile: string, recovery: 'connected' | 'responding'): void {
    this.dismissWhere((toast) => {
      const notice = toast.notice
      if (!notice || notice.profile !== profile) return false
      if (recovery === 'responding') return notice.key === 'watchdog.event.no_response'
      return notice.kind === 'policy_tunnel_down' ||
        /^watchdog\.event\.(no_response|manual_reconnect|gave_up|gave_up_blocking|gave_up_blocking_full|reconnect_blocked|reconnect_blocked_proton|reconnect_blocked_address)$/.test(notice.key)
    })
  },
  clear(): void {
    queue.set([])
  },
  get: queue.get,
  subscribe: queue.subscribe,
}

export function useToasts(): readonly Toast[] {
  return useStore(queue)
}
