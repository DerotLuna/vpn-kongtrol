// Kongtrol UI strings — Proton account and shared tunnel address (ADR-0103) · v1.0.0 · 2026-11-23
//
// A direct profile from a Proton account next to Proton's own app, and two
// direct profiles with the same tunnel address: the Add sheet's import
// hint and the daemon notices (Inicio toasts, Actividad). Mirrors
// internal/i18n/proton.go; {0}, {1}… are the notice's args. One module so
// parallel work never edits the same file; composed in ../messages.ts.
import type { Strings } from '../module'

const es = {
  'add.done.protonHint': 'Este archivo es de tu cuenta de Proton. No lo conectes mientras la app de Proton esté conectada: chocan.',

  'home.notice.watchdog.protonHandover':
    'Conectaste Proton VPN en su app. Desconecté {0} para que no choquen; tu tráfico sale por Proton ({1}). Para volver a {0}, desconecta Proton en su app y pulsa Reintentar.',
  'home.notice.watchdog.protonHandoverAdd':
    'Conectaste Proton VPN en su app. Desconecté {0} para que no choquen. Tu tráfico sale por la app de Proton, pero Kongtrol no la ve: agrégala como perfil para verla aquí y usarla en tus reglas.',
  'home.notice.watchdog.protonBlocksOther':
    '{0} no responde mientras Proton está conectado en su app. Si en Proton tienes el "kill switch avanzado", bloquea otras VPN: desactívalo o agrega {0} en "Split tunneling".',
  'home.notice.watchdog.reconnectBlockedProton':
    '{0} no se reconectó: Proton VPN está conectado en su app y {0} usa tu misma cuenta de Proton · desconecta Proton en su app y pulsa Reintentar',
  'home.notice.watchdog.reconnectBlockedAddress':
    '{0} no se reconectó: {1} usa la misma dirección interna ({2}) · desconecta {1} y pulsa Reintentar',

  'activity.event.watchdog.event.proton_handover':
    'Conectaste Proton VPN en su app. Desconecté {0} para que no choquen; tu tráfico sale por Proton ({1}). Para volver a {0}, desconecta Proton en su app y pulsa Reintentar.',
  'activity.event.watchdog.event.proton_handover_add':
    'Conectaste Proton VPN en su app. Desconecté {0} para que no choquen. Tu tráfico sale por la app de Proton, pero Kongtrol no la ve: agrégala como perfil para verla aquí y usarla en tus reglas.',
  'activity.event.watchdog.event.proton_blocks_other':
    '{0} no responde mientras Proton está conectado en su app. Si en Proton tienes el "kill switch avanzado", bloquea otras VPN: desactívalo o agrega {0} en "Split tunneling".',
  'activity.event.watchdog.event.reconnect_blocked_proton':
    '{0} no se reconectó: Proton VPN está conectado en su app y {0} usa tu misma cuenta de Proton · desconecta Proton en su app y pulsa Reintentar',
  'activity.event.watchdog.event.reconnect_blocked_address':
    '{0} no se reconectó: {1} usa la misma dirección interna ({2}) · desconecta {1} y pulsa Reintentar',
}

const en: Strings<typeof es> = {
  'add.done.protonHint': 'This file is from your Proton account. Do not connect it while the Proton app is connected: they clash.',

  'home.notice.watchdog.protonHandover':
    'You connected Proton VPN in its app. I disconnected {0} so they do not clash; your traffic goes through Proton ({1}). To go back to {0}, disconnect Proton in its app and press Retry.',
  'home.notice.watchdog.protonHandoverAdd':
    'You connected Proton VPN in its app. I disconnected {0} so they do not clash. Your traffic goes through the Proton app, but Kongtrol cannot see it: add it as a profile to see it here and use it in your rules.',
  'home.notice.watchdog.protonBlocksOther':
    '{0} is not responding while Proton is connected in its app. If Proton\'s "advanced kill switch" is on, it blocks other VPNs: turn it off or add {0} under "Split tunneling".',
  'home.notice.watchdog.reconnectBlockedProton':
    '{0} did not reconnect: Proton VPN is connected in its app and {0} uses the same Proton account · disconnect Proton in its app and press Retry',
  'home.notice.watchdog.reconnectBlockedAddress':
    '{0} did not reconnect: {1} uses the same internal address ({2}) · disconnect {1} and press Retry',

  'activity.event.watchdog.event.proton_handover':
    'You connected Proton VPN in its app. I disconnected {0} so they do not clash; your traffic goes through Proton ({1}). To go back to {0}, disconnect Proton in its app and press Retry.',
  'activity.event.watchdog.event.proton_handover_add':
    'You connected Proton VPN in its app. I disconnected {0} so they do not clash. Your traffic goes through the Proton app, but Kongtrol cannot see it: add it as a profile to see it here and use it in your rules.',
  'activity.event.watchdog.event.proton_blocks_other':
    '{0} is not responding while Proton is connected in its app. If Proton\'s "advanced kill switch" is on, it blocks other VPNs: turn it off or add {0} under "Split tunneling".',
  'activity.event.watchdog.event.reconnect_blocked_proton':
    '{0} did not reconnect: Proton VPN is connected in its app and {0} uses the same Proton account · disconnect Proton in its app and press Retry',
  'activity.event.watchdog.event.reconnect_blocked_address':
    '{0} did not reconnect: {1} uses the same internal address ({2}) · disconnect {1} and press Retry',
}

export default { es, en }
