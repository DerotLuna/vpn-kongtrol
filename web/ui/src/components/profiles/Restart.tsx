// Kongtrol UI — "Reiniciar ahora" after profile changes · v1.0.0 · 2026-11-18
//
// The daemon builds its VPN adapters at start, so a new or edited profile
// connects only after a restart (AGENTS.md "Dashboard REST API"). One banner
// says so and offers the restart; the same action is reused by toasts and by
// the first run (which restarts on its own and then goes to Inicio). While
// it runs, the banner says what is happening; the result is one toast.
import { usePreferences } from '../../app/preferences-context'
import { useRestartPending, type RestartPhase } from '../../data/profiles'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { RefreshIcon } from '../ui/icons'
import { Banner } from '../ui/States'
import { useRestartAction } from './useRestartAction'

const PHASE_TEXT: Partial<Record<RestartPhase, MessageKey>> = {
  requesting: 'profiles.restart.phase.requesting',
  stopping: 'profiles.restart.phase.stopping',
  starting: 'profiles.restart.phase.starting',
}

/** Shown after any profile change until Kongtrol restarts (also the daemon's own restart_pending). */
export function RestartBanner() {
  const { t } = usePreferences()
  const pending = useRestartPending()
  const { restart, busy, phase } = useRestartAction()
  if (!pending && !busy) return null
  const progress = PHASE_TEXT[phase]
  return (
    <Banner
      tone="info"
      title={t('profiles.restart.pending')}
      actions={
        <Button icon={<RefreshIcon />} loading={busy} onClick={() => void restart()}>
          {t('profiles.restart.button')}
        </Button>
      }
    >
      <span>{progress ? t(progress) : t('profiles.restart.pendingHint')}</span>
    </Banner>
  )
}
