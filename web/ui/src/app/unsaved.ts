// Kongtrol UI — unsaved changes registry · v1.0.0 · 2026-11-18
//
// A form with edits nobody saved calls useUnsavedChanges(dirty). Two guards
// follow from it:
// - Closing or reloading the tab: the browser's own "Leave site?" prompt
//   (beforeunload; browsers ignore custom text there, per the HTML spec).
// - Navigating inside the panel (links, Back/Forward): the shell's
//   UnsavedChangesGuard blocks it with React Router's useBlocker (data
//   router, ADR-0041) and asks "Tienes cambios sin guardar · Descartar /
//   Seguir editando".
// Forms only register here; they never need the router, so they also work
// in tests rendered without one.
import { useEffect, useId } from 'react'
import { createStore, useStore } from '../data/store'

const dirtyForms = createStore<ReadonlySet<string>>(new Set())

function setDirty(id: string, dirty: boolean): void {
  const current = dirtyForms.get()
  if (current.has(id) === dirty) return
  const next = new Set(current)
  if (dirty) next.add(id)
  else next.delete(id)
  dirtyForms.set(next)
}

export const unsavedChanges = {
  /** True while any mounted form has unsaved edits. */
  any: (): boolean => dirtyForms.get().size > 0,
  /** "Descartar": forget every edit so the pending navigation can go on. */
  discardAll: (): void => dirtyForms.set(new Set()),
}

function onBeforeUnload(e: BeforeUnloadEvent): void {
  e.preventDefault()
  // Older Chromium shows the prompt only when returnValue is set.
  e.returnValue = ''
}

/** Registers this form's unsaved state for as long as it is mounted. */
export function useUnsavedChanges(dirty: boolean): void {
  const id = useId()
  useEffect(() => {
    setDirty(id, dirty)
    // One listener per form (the same function would be added only once).
    const listener = (e: BeforeUnloadEvent) => onBeforeUnload(e)
    if (dirty) window.addEventListener('beforeunload', listener)
    return () => {
      setDirty(id, false)
      window.removeEventListener('beforeunload', listener)
    }
  }, [id, dirty])
}

/** Whether any form has unsaved edits (re-renders on change). */
export function useHasUnsavedChanges(): boolean {
  return useStore(dirtyForms).size > 0
}
