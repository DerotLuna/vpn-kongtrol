// Kongtrol UI — Seguridad: small shared pieces · v1.0.0 · 2026-11-18
//
// ConfirmDialog (kit Dialog as alertdialog: Cancel first, the action names
// what happens), TechDetails (technical detail on demand, native
// <details>), StateLine (pill + plain sentence) and SettingForm (one text
// setting with Save and an inline "problem + what to do" error).
import { useState, type FormEvent, type ReactNode } from 'react'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { useUnsavedChanges } from '../../app/unsaved'
import type { SettingsPatch } from '../../data/security'
import { useSettingsSave } from '../../data/security'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { TextAreaField, TextField, type FieldError } from '../ui/Field'
import { StatusPill } from '../ui/StatusPill'
import type { StateView } from './model'

export function ConfirmDialog({
  open,
  title,
  children,
  action,
  danger = true,
  onCancel,
  onConfirm,
}: {
  open: boolean
  title: string
  children: ReactNode
  action: string
  danger?: boolean
  onCancel: () => void
  onConfirm: () => void
}) {
  const { t } = usePreferences()
  return (
    <Dialog
      open={open}
      onClose={onCancel}
      title={title}
      alert
      description={children}
      actions={
        <>
          <Button variant="outlined" onClick={onCancel}>
            {t('security.cancel')}
          </Button>
          <Button variant={danger ? 'danger' : 'filled'} onClick={onConfirm}>
            {action}
          </Button>
        </>
      }
    />
  )
}

export function TechDetails({ children }: { children: ReactNode }) {
  const { t } = usePreferences()
  return (
    <details className="sec-tech">
      <summary>{t('security.tech')}</summary>
      <div className="sec-tech__body">{children}</div>
    </details>
  )
}

/** Pill + one sentence saying what the state means for the user. */
export function StateLine({ view, children }: { view: StateView; children?: ReactNode }) {
  const { t } = usePreferences()
  return (
    <div className="sec-state" role="status">
      <StatusPill tone={view.tone}>{t(view.pill)}</StatusPill>
      <div className="sec-state__text">
        <p>{t(view.line, view.vars)}</p>
        {children}
      </div>
    </div>
  )
}

interface SettingFormProps {
  label: string
  hint: ReactNode
  initial: string
  /** Client-side check; null = fine. */
  validate: (value: string) => FieldError | null
  toPatch: (value: string) => SettingsPatch
  multiline?: boolean
  placeholder?: string
}

/**
 * One text setting. Validates on Save (not per keystroke); a daemon refusal
 * lands under the field with its translated hint. Remount with a new `key`
 * when the stored value changes elsewhere.
 */
export function SettingForm({ label, hint, initial, validate, toPatch, multiline = false, placeholder }: SettingFormProps) {
  const { t } = usePreferences()
  const save = useSettingsSave()
  const [value, setValue] = useState(initial)
  const [local, setLocal] = useState<FieldError | null>(null)
  const dirty = value.trim() !== initial.trim()
  useUnsavedChanges(dirty)
  const remote = save.error ? describeError(save.error, t) : null
  const error = local ?? (remote ? { message: remote.message, fix: remote.fix } : null)

  const submit = (e: FormEvent) => {
    e.preventDefault()
    const problem = validate(value)
    setLocal(problem)
    if (problem) return
    save.mutate(toPatch(value))
  }

  const common = {
    label,
    hint,
    error,
    value,
    placeholder,
    mono: true,
    spellCheck: false,
    autoComplete: 'off',
  }
  return (
    <form className="sec-form" onSubmit={submit} noValidate>
      {multiline ? (
        <TextAreaField {...common} rows={3} onChange={(e) => setValue(e.target.value)} />
      ) : (
        <TextField {...common} onChange={(e) => setValue(e.target.value)} />
      )}
      <div className="sec-form__actions">
        <Button type="submit" variant="outlined" size="sm" loading={save.isPending} disabled={!dirty && !error}>
          {t('security.save')}
        </Button>
        <span className="sec-form__done" aria-live="polite">
          {save.isSuccess && !dirty ? t('security.saved') : ''}
        </span>
      </div>
    </form>
  )
}
