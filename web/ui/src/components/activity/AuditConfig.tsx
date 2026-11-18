// Kongtrol UI — Actividad: audit log settings · v1.1.0 · 2026-11-18
//
// Path, max size per file, days kept and HMAC signing (PUT /api/v1/settings,
// only these fields; applied live by the daemon, ADR-0042). Shown next to
// the log it configures (parity row "Audit log path, max size MB, HMAC sign
// switch", IA decision: /activity). Sizes and days are checked here with
// the daemon's limits (1–1024 MB, 0–365 days) and refused with a "what to
// do" line; an empty box is never saved as 0 (legacy bug settings.js:201).
import { useRef, useState, type FormEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { useSaveSettings, validateMaxSize, validateRetention, type Invalid, type Settings } from '../../data/settings'
import { useUnsavedChanges } from '../../app/unsaved'
import { focusFirstInvalid } from '../settings/form'
import { FormBar, SettingsSection } from '../settings/SettingsSection'
import { TextField } from '../ui/Field'
import { Switch } from '../ui/Switch'

interface Draft {
  path: string
  size: string
  days: string
  sign: boolean
}

const fromSettings = (s: Settings): Draft => ({
  path: s.audit_log_path,
  size: s.audit_log_max_size_mb ? String(s.audit_log_max_size_mb) : '',
  days: String(s.audit_log_retention_days ?? 0),
  sign: s.audit_log_sign,
})

const same = (a: Draft, b: Draft) => a.path === b.path && a.size === b.size && a.days === b.days && a.sign === b.sign

export function AuditConfig({ settings }: { settings: Settings }) {
  const { t } = usePreferences()
  const save = useSaveSettings()
  const saved = fromSettings(settings)
  const [draft, setDraft] = useState<Draft | null>(null)
  const [sizeError, setSizeError] = useState<Invalid | null>(null)
  const [daysError, setDaysError] = useState<Invalid | null>(null)
  const formRef = useRef<HTMLFormElement>(null)

  const value = draft ?? saved
  const dirty = draft !== null && !same(draft, saved)
  useUnsavedChanges(dirty)

  const edit = (patch: Partial<Draft>) => setDraft({ ...value, ...patch })
  const asError = (e: Invalid | null) => (e ? { message: t(e.message), fix: t(e.fix) } : null)

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    const size = validateMaxSize(value.size)
    const days = validateRetention(value.days)
    setSizeError(size)
    setDaysError(days)
    if (size || days) return focusFirstInvalid(formRef.current)
    save.mutate(
      {
        audit_log_path: value.path.trim(),
        audit_log_max_size_mb: Number(value.size.trim()),
        audit_log_retention_days: Number(value.days.trim()),
        audit_log_sign: value.sign,
      },
      { onSuccess: () => setDraft(null) },
    )
  }

  const days = Number(value.days.trim())
  return (
    <SettingsSection id="log-settings" title={t('activity.config.title')} what={t('activity.config.what')}>
      <form ref={formRef} className="k-set__form" onSubmit={onSubmit} noValidate>
        <TextField
          label={t('activity.config.path')}
          hint={t('activity.config.path.hint')}
          mono
          autoComplete="off"
          spellCheck={false}
          value={value.path}
          onChange={(e) => edit({ path: e.target.value })}
        />
        <div className="k-set__fields">
          <TextField
            label={t('activity.config.size')}
            hint={t('activity.config.size.hint')}
            inputMode="numeric"
            className="k-input--short"
            autoComplete="off"
            value={value.size}
            error={asError(sizeError)}
            onChange={(e) => {
              edit({ size: e.target.value })
              if (sizeError) setSizeError(validateMaxSize(e.target.value))
            }}
            onBlur={(e) => setSizeError(validateMaxSize(e.target.value))}
          />
          <TextField
            label={t('activity.config.retention')}
            hint={
              daysError || !/^\d+$/.test(value.days.trim())
                ? t('activity.config.retention.hint')
                : days === 0
                  ? t('activity.config.retention.zero')
                  : t('activity.config.retention.n', { n: days })
            }
            inputMode="numeric"
            className="k-input--short"
            autoComplete="off"
            value={value.days}
            error={asError(daysError)}
            onChange={(e) => {
              edit({ days: e.target.value })
              if (daysError) setDaysError(validateRetention(e.target.value))
            }}
            onBlur={(e) => setDaysError(validateRetention(e.target.value))}
          />
        </div>
        <Switch
          checked={value.sign}
          onChange={(on) => edit({ sign: on })}
          label={t('activity.config.sign')}
          description={t('activity.config.sign.desc')}
          onText={t('activity.config.sign.on')}
          offText={t('activity.config.sign.off')}
        />
        <FormBar
          dirty={dirty}
          saving={save.isPending}
          onRevert={() => {
            setDraft(null)
            setSizeError(null)
            setDaysError(null)
          }}
        />
      </form>
    </SettingsSection>
  )
}
