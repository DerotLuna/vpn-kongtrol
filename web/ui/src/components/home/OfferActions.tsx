// Kongtrol UI — a notice's offer buttons · v1.0.0 · 2026-11-23 (ADR-0102)
//
// The extra action a daemon notice carries (src/data/offers.ts), in Inicio's
// notices and in Actividad: "Activar protección" + "No, gracias",
// "Recuperar internet" or "Agregar <App>". An answered offer, or one that no
// longer means anything (kill switch already on, profile no longer
// blocked), shows nothing.
// feature:killswitch
import { usePreferences } from '../../app/preferences-context'
import type { DaemonEvent } from '../../data/home'
import {
  answerOffer,
  offerLabel,
  offerOf,
  offerStillUseful,
  useAnsweredOffers,
  useOfferRunner,
  type NoticeOffer,
} from '../../data/offers'
import { useSecurityQuery } from '../../data/resources'
import { Button } from '../ui/Button'

interface Props {
  event: Pick<DaemonEvent, 'seq' | 'offer' | 'profile' | 'args'>
  onDisconnect: (profile: string) => void
}

export function OfferActions({ event, onDisconnect }: Props) {
  const offer = offerOf(event)
  // Most notices have no offer: no hooks for them (Actividad lists 200).
  return offer ? <OfferButtons offer={offer} seq={event.seq} onDisconnect={onDisconnect} /> : null
}

function OfferButtons({ offer, seq, onDisconnect }: { offer: NoticeOffer; seq: number; onDisconnect: (profile: string) => void }) {
  const { t } = usePreferences()
  const answered = useAnsweredOffers()
  const security = useSecurityQuery().data
  const run = useOfferRunner(onDisconnect)
  if (answered.has(seq) || !offerStillUseful(offer, security)) return null
  return (
    <>
      <Button variant="tonal" size="sm" onClick={() => run(offer, seq)}>
        {offerLabel(offer, t)}
      </Button>
      {offer.kind !== 'release_block' && (
        <Button variant="text" size="sm" onClick={() => answerOffer(seq)}>
          {t('offer.action.noThanks')}
        </Button>
      )}
    </>
  )
}
