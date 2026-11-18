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
  clear(): void {
    queue.set([])
  },
  get: queue.get,
  subscribe: queue.subscribe,
}

export function useToasts(): readonly Toast[] {
  return useStore(queue)
}
