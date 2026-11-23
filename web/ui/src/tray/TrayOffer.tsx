// Kongtrol UI — tray flyout: a notice's offer · v1.0.0 · 2026-11-23 (ADR-0102)
//
// The flyout shows the latest daemon notice that still offers something
// (src/data/offers.ts): "Activar protección" / "No, gracias" after a VPN
// dropped on its own, "Recuperar internet" while the kill switch blocks for
// a profile, "Agregar <App>". The kill switch is turned on through the same
// API as the panel; "Recuperar internet" is the row's own disconnect;
// "Agregar <App>" asks the tray to open the panel's sheet (bridge message
// per provider) or, in a plain tab, goes there.
// feature:killswitch
import type { SecurityStatus } from '../api/types'
import { usePreferences } from '../app/preferences-context'
import { noticeText, useDaemonNotices, useRecentNotices } from '../data/home'
import { answerOffer, offerLabel, offerOf, offerStillUseful, useAnsweredOffers, type NoticeOffer } from '../data/offers'
import { useSecurityToggle } from '../data/resources'
import { addProviderMessage, trayHost } from './bridge'

const ignore = () => {}

export function TrayOffer({ security, onRelease }: { security: SecurityStatus | null; onRelease: (profile: string) => void }) {
  const { t } = usePreferences()
  useDaemonNotices(ignore)
  const notices = useRecentNotices()
  const answered = useAnsweredOffers()
  const { mutate } = useSecurityToggle('kill_switch')

  let shown: { seq: number; offer: NoticeOffer; text: string } | null = null
  for (const e of notices) {
    const offer = offerOf(e)
    if (offer && !answered.has(e.seq) && offerStillUseful(offer, security)) {
      shown = { seq: e.seq, offer, text: noticeText(e, t) }
      break
    }
  }
  if (!shown) return null
  const { seq, offer, text } = shown

  const run = () => {
    answerOffer(seq)
    if (offer.kind === 'enable_killswitch') mutate(true)
    else if (offer.kind === 'release_block') onRelease(offer.profile)
    else {
      const host = trayHost()
      const type = offer.provider ?? ''
      if (host) host.post(addProviderMessage(type))
      else window.location.assign(`/?add=vpn&type=${encodeURIComponent(type)}`)
    }
  }

  return (
    <div className="tray-notice caution" role="status">
      <p>{text}</p>
      <div className="tray-actions">
        <button type="button" className="tray-btn ghost small" onClick={run}>
          {offerLabel(offer, t)}
        </button>
        {offer.kind !== 'release_block' && (
          <button type="button" className="tray-btn ghost small" onClick={() => answerOffer(seq)}>
            {t('offer.action.noThanks')}
          </button>
        )}
      </div>
    </div>
  )
}
