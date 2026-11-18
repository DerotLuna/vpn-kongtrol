// Kongtrol UI — settings building blocks · v1.0.0 · 2026-11-18
//
// A titled group (flat surface, one task) with label/value rows that stack on
// phones, and the Save / Undo bar of an editable group (form helpers in
// ./form.ts). Shared by /settings and the log settings on /activity.
import type { ReactNode } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { Button } from '../ui/Button'
import './settings.css'

interface SectionProps {
  id: string
  title: string
  what?: ReactNode
  aside?: ReactNode
  children: ReactNode
  busy?: boolean
}

export function SettingsSection({ id, title, what, aside, children, busy }: SectionProps) {
  return (
    <section id={id} className="k-set" aria-labelledby={`${id}-t`} aria-busy={busy || undefined}>
      <header className="k-set__head">
        <div>
          <h2 id={`${id}-t`} className="k-set__title">
            {title}
          </h2>
          {what && <p className="k-set__what">{what}</p>}
        </div>
        {aside}
      </header>
      {children}
    </section>
  )
}

/** A read-only or single-control row: label (and hint) left, value right. */
export function SettingRow({ label, hint, children }: { label: string; hint?: ReactNode; children: ReactNode }) {
  return (
    <div className="k-set__row">
      <div className="k-set__label">
        <span>{label}</span>
        {hint && <span className="k-set__hint">{hint}</span>}
      </div>
      <div className="k-set__value">{children}</div>
    </div>
  )
}

interface FormBarProps {
  dirty: boolean
  saving: boolean
  onRevert: () => void
}

/** Save (submit) + Undo; the unsaved state is written, not only colored. */
export function FormBar({ dirty, saving, onRevert }: FormBarProps) {
  const { t } = usePreferences()
  return (
    <div className="k-set__bar">
      <p className="k-set__dirty" aria-live="polite">
        {dirty ? t('settings.unsaved') : ''}
      </p>
      <Button variant="text" onClick={onRevert} disabled={!dirty || saving}>
        {t('settings.revert')}
      </Button>
      <Button type="submit" variant="tonal" loading={saving} disabled={!dirty && !saving}>
        {t('settings.save')}
      </Button>
    </div>
  )
}
