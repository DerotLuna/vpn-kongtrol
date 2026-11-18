// Kongtrol UI — backup profiles (failover), in order · v1.0.0 · 2026-11-18
//
// Legacy took a comma-separated text (docs/dashboard-parity.md §5.1 item 11);
// here the user picks existing profiles only, sees the order the watchdog
// will try them in, and can move or remove each one. Self, duplicates and
// cycles are caught by checkDraft before saving (ADR-0018: only declared
// backups ever connect).
import { useId, useState } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { IconButton } from '../ui/Button'
import { Button } from '../ui/Button'
import { CloseIcon, PlusIcon } from '../ui/icons'
import type { FieldError } from '../ui/Field'
import { ArrowUpIcon } from './icons'

interface Props {
  value: string[]
  onChange: (next: string[]) => void
  /** Every profile name, this one excluded. */
  candidates: string[]
  error?: FieldError | null
}

export function FailoverEditor({ value, onChange, candidates, error }: Props) {
  const { t } = usePreferences()
  const id = useId()
  const free = candidates.filter((c) => !value.includes(c))
  const [pick, setPick] = useState('')
  const chosen = free.includes(pick) ? pick : (free[0] ?? '')
  const hintId = `${id}-hint`
  const errId = `${id}-err`

  const move = (i: number) => {
    if (i === 0) return
    const next = [...value]
    const [item] = next.splice(i, 1)
    if (item !== undefined) next.splice(i - 1, 0, item)
    onChange(next)
  }

  return (
    <fieldset className="kp-failover" aria-describedby={[error ? errId : '', hintId].filter(Boolean).join(' ')}>
      <legend className="k-field__label">{t('profiles.field.failover')}</legend>
      {value.length > 0 ? (
        <ol className="kp-failover__list">
          {value.map((name, i) => (
            <li key={name}>
              <span className="kp-failover__pos" aria-hidden="true">
                {i + 1}
              </span>
              <span className="mono kp-failover__name">{name}</span>
              <IconButton
                label={t('profiles.failover.up', { name })}
                icon={<ArrowUpIcon />}
                disabled={i === 0}
                onClick={() => move(i)}
              />
              <IconButton
                label={t('profiles.failover.remove', { name })}
                icon={<CloseIcon />}
                onClick={() => onChange(value.filter((v) => v !== name))}
              />
            </li>
          ))}
        </ol>
      ) : (
        <p className="kp-muted kp-small">{t('profiles.failover.none')}</p>
      )}
      {free.length > 0 && (
        <div className="kp-failover__add">
          <label className="visually-hidden" htmlFor={`${id}-add`}>
            {t('profiles.failover.pick')}
          </label>
          <select id={`${id}-add`} className="k-input k-select" value={chosen} onChange={(e) => setPick(e.target.value)}>
            {free.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
          <Button variant="outlined" icon={<PlusIcon />} onClick={() => chosen && onChange([...value, chosen])}>
            {t('profiles.failover.add')}
          </Button>
        </div>
      )}
      {error && (
        <p id={errId} className="k-field__error" role="alert">
          <strong>{error.message}</strong> {error.fix && <span>{error.fix}</span>}
        </p>
      )}
      <p id={hintId} className="k-field__hint">
        {candidates.length === 0 ? t('profiles.failover.onlyOne') : t('profiles.hint.failover')}
      </p>
    </fieldset>
  )
}
