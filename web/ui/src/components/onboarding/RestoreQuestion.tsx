// Kongtrol UI — restore question · v1.0.0 · 2026-11-23
//
// "¿Volver a conectar tus VPN al encender el equipo?" (ADR-0101): asked
// once, after the first VPN the user connects. The daemon decides when it
// is pending (GET /api/v1/startup `ask`); either button answers it, so it
// disappears everywhere (panel, tray flyout, `kongtrol up`) at once.
// Inicio shows it as a banner; the tray flyout with its own notice style.
// feature:restore_at_start
import { usePreferences } from '../../app/preferences-context'
import { useAnswerRestore, useStartupQuery } from '../../data/startup'
import { Button } from '../ui/Button'
import { Banner } from '../ui/States'
import './onboarding.css'

export function RestoreQuestion({ variant = 'panel' }: { variant?: 'panel' | 'tray' }) {
  const { t } = usePreferences()
  const query = useStartupQuery()
  const answer = useAnswerRestore()
  if (!query.data?.ask) return null
  const busy = answer.isPending
  const yes = () => answer.mutate(true)
  const no = () => answer.mutate(false)

  if (variant === 'tray') {
    return (
      <div className="tray-notice info" role="status">
        <p>
          <strong>{t('onboarding.restore.title')}</strong> {t('onboarding.restore.body')}
        </p>
        <div className="tray-actions">
          <button type="button" className="tray-btn primary small" onClick={yes} disabled={busy}>
            {t('onboarding.restore.yes')}
          </button>
          <button type="button" className="tray-btn ghost small" onClick={no} disabled={busy}>
            {t('onboarding.restore.no')}
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="home-banner">
      <Banner
        tone="info"
        title={t('onboarding.restore.title')}
        actions={
          <>
            <Button size="sm" onClick={yes} loading={busy && answer.variables === true} disabled={busy}>
              {t('onboarding.restore.yes')}
            </Button>
            <Button size="sm" variant="text" onClick={no} loading={busy && answer.variables === false} disabled={busy}>
              {t('onboarding.restore.no')}
            </Button>
          </>
        }
      >
        <p>{t('onboarding.restore.body')}</p>
      </Banner>
    </div>
  )
}
