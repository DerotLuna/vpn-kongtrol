// Kongtrol UI — Actividad: audit log settings · v1.2.0 · 2026-11-19
//
// Path, max size per file, days kept, total space and HMAC signing
// (PUT /api/v1/settings, only these fields; applied live by the daemon,
// ADR-0042, ADR-0064). Shown next to the log it configures (parity row
// "Audit log path, max size MB, HMAC sign switch", IA decision: /activity).
// Values are checked here with the daemon's limits (1–1024 MB per file,
// 1–365 days, 1–10240 MB in total and never below one file) and refused
// with a "what to do" line; an empty box is never saved as 0 (legacy bug
// settings.js:201). The daemon sends the values in use, never a 0.
import { useRef, useState, type FormEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import {
  useSaveSettings,
  validateMaxSize,
  validateMaxTotal,
  validateRetention,
  type Invalid,
  type Settings,
} from '../../data/settings'
import { useUnsavedChanges } from '../../app/unsaved'
import { focusFirstInvalid } from '../settings/form'
import { FormBar, SettingsSection } from '../settings/SettingsSection'
import { TextField } from '../ui/Field'
import { Switch } from '../ui/Switch'

interface Draft {
  path: string
  size: string
  days: string
  total: string
  sign: boolean
}

const fromSettings = (s: Settings): Draft => ({
  path: s.audit_log_path,
  size: s.audit_log_max_size_mb ? String(s.audit_log_max_size_mb) : '',
  days: s.audit_log_retention_days ? String(s.audit_log_retention_days) : '',
  total: s.audit_log_max_total_mb ? String(s.audit_log_max_total_mb) : '',
  sign: s.audit_log_sign,
})

const same = (a: Draft, b: Draft) =>
  a.path === b.path && a.size === b.size && a.days === b.days && a.total === b.total && a.sign === b.sign

export function AuditConfig({ settings }: { settings: Settings }) {
  const { t } = usePreferences()
  const save = useSaveSettings()
  const saved = fromSettings(settings)
  const [draft, setDraft] = useState<Draft | null>(null)
  const [sizeError, setSizeError] = useState<Invalid | null>(null)
  const [daysError, setDaysError] = useState<Invalid | null>(null)
  const [totalError, setTotalError] = useState<Invalid | null>(null)
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
    const total = validateMaxTotal(value.total, value.size)
    setSizeError(size)
    setDaysError(days)
    setTotalError(total)
    if (size || days || total) return focusFirstInvalid(formRef.current)
    save.mutate(
      {
        audit_log_path: value.path.trim(),
        audit_log_max_size_mb: Number(value.size.trim()),
        audit_log_retention_days: Number(value.days.trim()),
        audit_log_max_total_mb: Number(value.total.trim()),
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
              daysError || !/^\d+$/.test(value.days.trim()) || days < 1
                ? t('activity.config.retention.hint')
                : days === 1
                  ? t('activity.config.retention.one')
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
          <TextField
            label={t('activity.config.total')}
            hint={t('activity.config.total.hint')}
            inputMode="numeric"
            className="k-input--short"
            autoComplete="off"
            value={value.total}
            error={asError(totalError)}
            onChange={(e) => {
              edit({ total: e.target.value })
              if (totalError) setTotalError(validateMaxTotal(e.target.value, value.size))
            }}
            onBlur={(e) => setTotalError(validateMaxTotal(e.target.value, value.size))}
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
            setTotalError(null)
          }}
        />
      </form>
    </SettingsSection>
  )
}
