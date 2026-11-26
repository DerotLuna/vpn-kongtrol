// Kongtrol UI — live notices (Inicio) · v1.0.0 · 2026-11-18
//
// The daemon's notices (GET /api/v1/events, legacy notices.js): a tunnel the
// watchdog cannot reconnect, a backup that took over, a server that stopped
// answering, traffic of a rule whose VPN is off ("claude.ai sale por
// internet normal · brainbox está apagado") and the opt-in update notice.
// Each new one is a toast (WARN stays until dismissed; one with "Conectar
// <perfil>" waits 60 s) and stays in this list, so a notice is never only a
// toast. The full history is /activity.
// A notice may also offer one more action (ADR-0102, OfferActions).
import { useCallback } from 'react'
import { Link } from 'react-router'
import { OfferActions } from './OfferActions'
import { usePreferences } from '../../app/preferences-context'
import { noticeActionLabel, noticeText, useRecentNotices, useTunnelActions } from '../../data/home'
import { tunnelState, useTunnelsQuery } from '../../data/resources'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { AlertIcon, InfoIcon } from '../ui/icons'
import { formatClock } from './format'

export function NoticesCard() {
  const { t, lang } = usePreferences()
  const notices = useRecentNotices()
  const tunnels = useTunnelsQuery().data
  const run = useTunnelActions()
  const disconnect = useCallback((profile: string) => run(profile, 'disconnect'), [run])

  return (
    <Card className="home-notices" title={t('home.notices.title')} aside={<Link to="/activity">{t('home.notices.seeAll')}</Link>}>
      {notices.length === 0 ? (
        <p className="muted">{t('home.notices.empty')}</p>
      ) : (
        <ul className="home-notices__list" aria-live="polite">
          {notices.map((e) => (
            <li key={e.seq} className={e.level === 'WARN' ? 'home-notice home-notice--warn' : 'home-notice'}>
              <span className="home-notice__icon">{e.level === 'WARN' ? <AlertIcon /> : <InfoIcon />}</span>
              <div className="home-notice__text">
                <p>{noticeText(e, t)}</p>
                <time className="home-dim small" dateTime={e.time}>
                  {formatClock(e.time, lang)}
                </time>
              </div>
              {e.can_connect && e.profile && !tunnels?.some((tn) => tn.Name === e.profile && (tunnelState(tn) === 'connected' || tunnelState(tn) === 'connecting')) && (
                <Button variant="tonal" size="sm" onClick={() => run(e.profile ?? '', 'connect')}>
                  {noticeActionLabel(e, t)}
                </Button>
              )}
              <OfferActions event={e} onDisconnect={disconnect} />
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}
