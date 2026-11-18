// Kongtrol UI — settings form helpers · v1.1.0 · 2026-11-18
// Unsaved edits are guarded app-wide by app/unsaved.ts (useUnsavedChanges).

/** After a failed submit, focus the first field marked invalid (once it renders). */
export function focusFirstInvalid(form: HTMLFormElement | null): void {
  setTimeout(() => form?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus(), 0)
}
