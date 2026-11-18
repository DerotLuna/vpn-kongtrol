// Kongtrol UI — Actividad: audit log settings · v1.0.0 · 2026-11-18
//
// Path, max size and HMAC signing (PUT /api/v1/settings, only these three
// fields). Shown next to the log it configures (parity row "Audit log path,
// max size MB, HMAC sign switch", IA decision: /activity). Fix of the legacy
// bug settings.js:201: an empty or invalid size is refused here with a
// "what to do" line instead of being saved as 0.
import { useRef, useState, type FormEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { useSaveSettings, validateMaxSize, type Invalid, type Settings } from '../../data/settings'
import { useUnsavedChanges } from '../../app/unsaved'
import { focusFirstInvalid } from '../settings/form'
import { FormBar, SettingsSection } from '../settings/SettingsSection'
import { TextField } from '../ui/Field'
import { Switch } from '../ui/Switch'

interface Draft {
  path: string
  size: string
  sign: boolean
}

const fromSettings = (s: Settings): Draft => ({
  path: s.audit_log_path,
  size: s.audit_log_max_size_mb ? String(s.audit_log_max_size_mb) : '',
  sign: s.audit_log_sign,
})

export function AuditConfig({ settings }: { settings: Settings }) {
  const { t } = usePreferences()
  const save = useSaveSettings()
  const saved = fromSettings(settings)
  const [draft, setDraft] = useState<Draft | null>(null)
  const [sizeError, setSizeError] = useState<Invalid | null>(null)
  const formRef = useRef<HTMLFormElement>(null)

  const value = draft ?? saved
  const dirty = draft !== null && (draft.path !== saved.path || draft.size !== saved.size || draft.sign !== saved.sign)
  useUnsavedChanges(dirty)

  const edit = (patch: Partial<Draft>) => setDraft({ ...value, ...patch })

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    const invalid = validateMaxSize(value.size)
    setSizeError(invalid)
    if (invalid) return focusFirstInvalid(formRef.current)
    save.mutate(
      {
        audit_log_path: value.path.trim(),
        audit_log_max_size_mb: Number(value.size.trim()),
        audit_log_sign: value.sign,
      },
      { onSuccess: () => setDraft(null) },
    )
  }

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
            error={sizeError ? { message: t(sizeError.message), fix: t(sizeError.fix) } : null}
            onChange={(e) => {
              edit({ size: e.target.value })
              if (sizeError) setSizeError(validateMaxSize(e.target.value))
            }}
            onBlur={(e) => setSizeError(validateMaxSize(e.target.value))}
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
          }}
        />
      </form>
    </SettingsSection>
  )
}
