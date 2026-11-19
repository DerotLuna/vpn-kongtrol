// Kongtrol UI — "＋ Agregar": small shared pieces · v1.0.0 · 2026-11-19
import type { ReactNode } from 'react'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { Button } from '../ui/Button'
import { Banner } from '../ui/States'
import { ArrowLeftIcon } from '../profiles/icons'
import type { ActivatePhase } from './activate'

/** Footer of the profile steps: what saving does, then Back + "Guardar y activar". */
export function ActivateFooter({ phase, onBack, onRetry }: { phase: ActivatePhase; onBack: () => void; onRetry: () => void }) {
  const { t } = usePreferences()
  const busy = phase === 'saving' || phase === 'restarting'
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
      <footer className="k-dialog__actions k-add__actions">
        <Button variant="outlined" icon={<ArrowLeftIcon />} onClick={onBack} disabled={busy}>
          {t('add.back')}
        </Button>
        {phase !== 'restartFailed' && (
          <Button type="submit" loading={busy}>
            {phase === 'saving' ? t('add.saving') : phase === 'restarting' ? t('add.restarting') : t('add.save')}
          </Button>
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
