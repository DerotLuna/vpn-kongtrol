// Kongtrol UI — Ajustes: health check interval + timeout · v1.0.0 · 2026-11-18
//
// PUT /api/v1/settings with only health_check_interval / _timeout (the
// daemon checks time.ParseDuration and config.Validate). Validated on blur
// and on save; the first invalid field gets focus.
import { useRef, useState, type FormEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { useSaveSettings, validateDuration, type Invalid, type Settings } from '../../data/settings'
import { TextField, type FieldError } from '../ui/Field'
import { focusFirstInvalid, useUnsavedWarning } from './form'
import { FormBar, SettingsSection } from './SettingsSection'

type Draft = { interval: string; timeout: string }

const fromSettings = (s: Settings): Draft => ({ interval: s.health_check_interval, timeout: s.health_check_timeout })

export function HealthSection({ settings }: { settings: Settings }) {
  const { t } = usePreferences()
  const save = useSaveSettings()
  const saved = fromSettings(settings)
  const [draft, setDraft] = useState<Draft | null>(null)
  const [errors, setErrors] = useState<Partial<Record<keyof Draft, Invalid | null>>>({})
  const formRef = useRef<HTMLFormElement>(null)

  // No local edits = show the daemon's values (and follow them after a save).
  const value = draft ?? saved
  const dirty = draft !== null && (draft.interval !== saved.interval || draft.timeout !== saved.timeout)
  useUnsavedWarning(dirty)

  const err = (k: keyof Draft): FieldError | null => {
    const e = errors[k]
    return e ? { message: t(e.message), fix: t(e.fix) } : null
  }
  const check = (k: keyof Draft, v: string) => setErrors((prev) => ({ ...prev, [k]: validateDuration(v) }))
  const edit = (k: keyof Draft, v: string) => {
    setDraft({ ...value, [k]: v })
    if (errors[k]) check(k, v)
  }

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    const next = { interval: validateDuration(value.interval), timeout: validateDuration(value.timeout) }
    setErrors(next)
    if (next.interval || next.timeout) return focusFirstInvalid(formRef.current)
    save.mutate(
      { health_check_interval: value.interval.trim(), health_check_timeout: value.timeout.trim() },
      { onSuccess: () => setDraft(null) },
    )
  }

  return (
    <SettingsSection id="health" title={t('settings.health.title')} what={t('settings.health.what')}>
      <form ref={formRef} className="k-set__form" onSubmit={onSubmit} noValidate>
        <div className="k-set__fields">
          <TextField
            label={t('settings.health.interval')}
            hint={t('settings.health.interval.hint')}
            mono
            autoComplete="off"
            spellCheck={false}
            value={value.interval}
            error={err('interval')}
            onChange={(e) => edit('interval', e.target.value)}
            onBlur={(e) => check('interval', e.target.value)}
          />
          <TextField
            label={t('settings.health.timeout')}
            hint={t('settings.health.timeout.hint')}
            mono
            autoComplete="off"
            spellCheck={false}
            value={value.timeout}
            error={err('timeout')}
            onChange={(e) => edit('timeout', e.target.value)}
            onBlur={(e) => check('timeout', e.target.value)}
          />
        </div>
        <FormBar
          dirty={dirty}
          saving={save.isPending}
          onRevert={() => {
            setDraft(null)
            setErrors({})
          }}
        />
      </form>
    </SettingsSection>
  )
}
