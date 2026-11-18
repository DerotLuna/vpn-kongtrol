// Kongtrol UI — app-wide notices · v0.1.0 · 2026-11-18
//
// Shown once, above every page, inline (never a modal), over the last data
// we had (the cache keeps it while the daemon is away):
// 0. The page is not allowed to talk to Kongtrol (401: opened from a stale
//    tab or another app) → reopen it from the tray or `kongtrol dashboard`.
// 1. Kongtrol stopped answering → what to do + automatic reconnect + Retry.
//    There is no API to start the daemon from a page it no longer serves,
//    so the notice guides instead of offering "Iniciar".
// 2. Helper missing or outdated → the same install flow as everywhere. Not
//    on Inicio, whose helper card already offers it.
import { useQueryClient } from '@tanstack/react-query'
import { useLocation } from 'react-router'
import { useAuthState } from '../../api/auth'
import { useHelper } from '../../api/hooks'
import { usePreferences } from '../../app/preferences-context'
import { useDaemonState } from '../../data/connection'
import { helperUiState } from '../../data/resources'
import { Button } from '../ui/Button'
import { HelperInstallButton, HelperInstallProgress } from '../ui/HelperInstall'
import { RefreshIcon } from '../ui/icons'
import { Banner } from '../ui/States'

export function AppNotices() {
  const { t } = usePreferences()
  const client = useQueryClient()
  const daemon = useDaemonState()
  const helper = useHelper()
  const { pathname } = useLocation()
  const state = helperUiState(helper.status ?? undefined)
  const auth = useAuthState()

  if (auth === 'unauthorized') {
    return (
      <div className="k-notices">
        <Banner tone="bad" urgent title={t('error.unauthorized')}>
          <p>{t('error.unauthorized.fix')}</p>
        </Banner>
      </div>
    )
  }

  if (daemon === 'offline') {
    return (
      <div className="k-notices">
        <Banner
          tone="bad"
          urgent
          title={t('error.offline')}
          actions={
            <Button variant="outlined" size="sm" icon={<RefreshIcon />} onClick={() => void client.refetchQueries()}>
              {t('error.retry')}
            </Button>
          }
        >
          <p>{t('error.offline.fix')}</p>
          <p className="muted">{t('daemon.offline.auto')}</p>
        </Banner>
      </div>
    )
  }

  if ((state === 'missing' || state === 'outdated') && pathname !== '/') {
    const outdated = state === 'outdated'
    return (
      <div className="k-notices">
        <Banner
          tone="caution"
          title={outdated ? t('helper.notice.outdated') : t('helper.notice.missing')}
          actions={<HelperInstallButton helper={helper} outdated={outdated} />}
        >
          <p>{outdated ? t('helper.outdated') : t('helper.needsInstall')}</p>
          <HelperInstallProgress helper={helper} />
        </Banner>
      </div>
    )
  }
  return null
}
