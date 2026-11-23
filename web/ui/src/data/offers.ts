// Kongtrol UI — notice offers · v1.0.0 · 2026-11-23 (ADR-0102)
//
// A daemon notice may carry one extra action (internal/monitor/events.go
// `Event.Offer`): "enable_killswitch" (the one-time offer after a VPN
// dropped on its own: "Activar protección" / "No, gracias"),
// "release_block" (every kill switch block: "Recuperar internet", a user
// disconnect of that profile) and "add_provider" ("Agregar <App>": the
// "＋ Agregar" sheet with that provider chosen; the type is the last arg).
// Inicio, toasts, Actividad and the tray flyout render them from here.
// feature:killswitch
import { useCallback } from 'react'
import type { SecurityStatus } from '../api/types'
import { useAddSheet } from '../app/addLink'
import { usePreferences } from '../app/preferences-context'
import type { MessageKey } from '../i18n/messages'
import { isKnownType, specOf } from '../components/profiles/catalog'
import type { DaemonEvent } from './home'
import { useSecurityToggle } from './resources'
import { createStore, useStore } from './store'
import { toasts } from './toast'

export type OfferKind = 'enable_killswitch' | 'release_block' | 'add_provider'

export interface NoticeOffer {
  kind: OfferKind
  profile: string
  /** Adapter type (add_provider). */
  provider?: string
}

type Translate = (key: MessageKey, vars?: Record<string, string | number>) => string

/** The offer of a notice, or null (none, or one this page does not know). */
export function offerOf(e: Pick<DaemonEvent, 'offer' | 'profile' | 'args'>): NoticeOffer | null {
  const profile = e.profile ?? ''
  switch (e.offer) {
    case 'enable_killswitch':
      return { kind: 'enable_killswitch', profile }
    case 'release_block':
      return profile ? { kind: 'release_block', profile } : null
    case 'add_provider': {
      const provider = e.args?.at(-1)
      return provider && isKnownType(provider) ? { kind: 'add_provider', profile, provider } : null
    }
    default:
      return null
  }
}

/** Short brand name for "Agregar <App>": "Agregar Proton" (internal/i18n OfferAppName). */
export function offerAppName(type: string): string {
  if (type === 'protonvpn') return 'Proton'
  return isKnownType(type) ? specOf(type).label : type
}

/** The button text of an offer. */
export function offerLabel(o: NoticeOffer, t: Translate): string {
  switch (o.kind) {
    case 'enable_killswitch':
      return t('offer.action.enableKillswitch')
    case 'release_block':
      return t('offer.action.releaseBlock')
    case 'add_provider':
      return t('offer.action.addProvider', { app: offerAppName(o.provider ?? '') })
  }
}

/** The parts of GET /api/v1/security/status an offer depends on. */
type SecurityView = Pick<SecurityStatus, 'kill_switch_enabled'> & { kill_switch_status?: { blocking?: string[] } }

/**
 * Whether the offer still means something now: the kill switch is still
 * off, the profile is still blocked. Unknown state keeps it.
 */
export function offerStillUseful(o: NoticeOffer, security: SecurityView | null | undefined): boolean {
  if (!security) return true
  if (o.kind === 'enable_killswitch') return !security.kill_switch_enabled
  if (o.kind === 'release_block') {
    const blocking = security.kill_switch_status?.blocking
    return blocking ? blocking.includes(o.profile) : true
  }
  return true
}

// "No, gracias" (or any answered offer) hides it in this page; the daemon
// never offers the kill switch twice (preferences.json), so nothing to save.
const answered = createStore<ReadonlySet<number>>(new Set())

export function answerOffer(seq: number): void {
  const next = new Set(answered.get())
  next.add(seq)
  answered.set(next)
}

export function useAnsweredOffers(): ReadonlySet<number> {
  return useStore(answered)
}

/**
 * Runs an offer in the panel: kill switch on through the security toggle,
 * "Recuperar internet" through the tunnel actions (a user disconnect, so
 * the block lifts), "Agregar <App>" through the shell's "＋ Agregar" sheet.
 * disconnect is passed in so each page keeps its own pending state.
 */
export function useOfferRunner(disconnect: (profile: string) => void): (o: NoticeOffer, seq?: number) => void {
  const { t } = usePreferences()
  const { mutate } = useSecurityToggle('kill_switch')
  const { open } = useAddSheet()
  return useCallback(
    (o: NoticeOffer, seq?: number) => {
      if (seq !== undefined) answerOffer(seq)
      if (o.kind === 'enable_killswitch') {
        mutate(true, { onSuccess: () => toasts.show({ tone: 'ok', message: t('offer.toast.killswitchOn') }) })
      } else if (o.kind === 'release_block') disconnect(o.profile)
      else open(o.provider ? { mode: 'vpn', type: o.provider } : { mode: 'vpn' })
    },
    [mutate, disconnect, open, t],
  )
}
