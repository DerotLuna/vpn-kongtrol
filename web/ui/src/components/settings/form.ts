// Kongtrol UI — settings form helpers · v1.0.0 · 2026-11-18
import { useEffect } from 'react'

/** After a failed submit, focus the first field marked invalid (once it renders). */
export function focusFirstInvalid(form: HTMLFormElement | null): void {
  setTimeout(() => form?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus(), 0)
}

/**
 * Asks before the tab closes or reloads with unsaved edits (the browser
 * shows its own dialog). In-app navigation is not covered: the declarative
 * router has no blocker (ADR-0038), so the form bar says "Cambios sin guardar".
 */
export function useUnsavedWarning(dirty: boolean): void {
  useEffect(() => {
    if (!dirty) return
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [dirty])
}
