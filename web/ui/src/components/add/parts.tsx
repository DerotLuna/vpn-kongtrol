// Kongtrol UI — "＋ Agregar": small shared pieces · v1.2.0 · 2026-11-25
import type { ReactNode } from 'react'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { Button } from '../ui/Button'
import { SelectField } from '../ui/Field'
import { Banner } from '../ui/States'
import { ArrowLeftIcon } from '../profiles/icons'
import { ON_TUNNEL_DOWN, type OnTunnelDown } from '../rules/model'
import type { ActivatePhase, SaveIntent } from './activate'

export function ProfileDownField({ value, onChange }: { value: OnTunnelDown; onChange: (value: OnTunnelDown) => void }) {
  const { t } = usePreferences()
  return (
    <SelectField
      label={t('profiles.onDown.label')}
      hint={<>{t(`rules.onDown.${value}.hint`)} {t('profiles.onDown.hint')}</>}
      value={value}
      options={ON_TUNNEL_DOWN.map((option) => ({ value: option, label: t(`rules.onDown.${option}`) }))}
      onChange={(option) => onChange(option as OnTunnelDown)}
    />
  )
}

/**
 * Footer of the profile steps: what saving does, then Back, "Guardar"
 * (outlined: save only) and "Guardar y activar" (the one primary, the form's
 * submit: save and connect; ADR-0115). The spinner shows on the button that
 * was pressed; both wait while it runs.
 */
export function ActivateFooter({
  phase,
  intent,
  onBack,
  onRetry,
  onSaveOnly,
  blocked,
}: {
  phase: ActivatePhase
  intent: SaveIntent
  onBack: () => void
  onRetry: () => void
  onSaveOnly: () => void
  /** Why saving is not possible yet (the file step's check, ADR-0118): both buttons off, the reason shown; null = can save. */
  blocked?: string | null | undefined
}) {
  const { t } = usePreferences()
  const busy = phase === 'saving' || phase === 'restarting'
  const off = !busy && !!blocked
  const label = phase === 'saving' ? t('add.saving') : phase === 'restarting' ? t('add.restarting') : null
  return (
    <>
      {phase === 'restartFailed' ? (
        <Banner
          tone="caution"
          title={t('add.restartFailed')}
          urgent
          actions={
            <Button variant="outlined" size="sm" onClick={onRetry}>
              {t('add.restartRetry')}
            </Button>
          }
        >
          <span>{t('add.restartFailed.fix')}</span>
        </Banner>
      ) : (
        <p className="k-add__note" aria-live="polite">
          {phase === 'restarting' ? t('add.restarting') : t('add.restart.what')}
        </p>
      )}
      {/* Always there for a step that blocks (its line keeps its height: no jump when the check ends). */}
      {blocked !== undefined && phase !== 'restartFailed' && (
        <p className="k-add__note k-add__small k-add__blocked" aria-live="polite">
          {off ? blocked : ' '}
        </p>
      )}
      <footer className="k-dialog__actions k-add__actions">
        <Button variant="text" className="k-add__back" icon={<ArrowLeftIcon />} onClick={onBack} disabled={busy}>
          {t('add.back')}
        </Button>
        {phase !== 'restartFailed' && (
          <>
            <Button variant="outlined" onClick={onSaveOnly} loading={busy && intent === 'save'} disabled={busy || off}>
              {busy && intent === 'save' && label ? label : t('add.saveOnly')}
            </Button>
            <Button type="submit" loading={busy && intent === 'activate'} disabled={(busy && intent === 'save') || off}>
              {busy && intent === 'activate' && label ? label : t('add.save')}
            </Button>
          </>
        )}
      </footer>
    </>
  )
}

/** A refusal above the form: what happened + what to do (+ an action). */
export function ErrorBanner({ error, actions }: { error: unknown; actions?: ReactNode }) {
  const { t } = usePreferences()
  const { message, fix } = describeError(error, t)
  return (
    <Banner tone="bad" title={message} urgent {...(actions ? { actions } : {})}>
      <span>{fix}</span>
    </Banner>
  )
}

/** Big choice row (menu, done steps). */
export function Choice({
  icon,
  title,
  body,
  onClick,
  autoFocus = false,
}: {
  icon: ReactNode
  title: string
  body?: ReactNode
  onClick: () => void
  autoFocus?: boolean
}) {
  return (
    <li>
      <button type="button" className="k-add-choice" onClick={onClick} data-autofocus={autoFocus || undefined}>
        <span className="k-add-choice__icon" aria-hidden="true">
          {icon}
        </span>
        <span className="k-add-choice__text">
          <span className="k-add-choice__title">{title}</span>
          {body && <span className="k-add-choice__body">{body}</span>}
        </span>
      </button>
    </li>
  )
}
