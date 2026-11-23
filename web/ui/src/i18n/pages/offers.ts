// Kongtrol UI strings — notice offers (ADR-0102) · v1.0.0 · 2026-11-23
//
// The extra action a daemon notice may carry (src/data/offers.ts), shown in
// Inicio, toasts, Actividad and the tray flyout. Mirrors internal/i18n
// killswitch_offer.go. ES is the source of the key set; EN must match it
// (the type below + messages.test.ts).
import type { Strings } from '../module'

const es = {
  'offer.notice.killswitch':
    '{0} se cayó sin que lo pidieras · mientras volvía a conectar, tu tráfico salió sin VPN. ¿Quieres que Kongtrol lo impida? Si {0} se cae, Kongtrol bloquea su tráfico hasta que vuelva. Si no vuelve, te aviso y con un clic recuperas internet.',
  'offer.action.enableKillswitch': 'Activar protección',
  'offer.action.noThanks': 'No, gracias',
  'offer.action.releaseBlock': 'Recuperar internet',
  'offer.action.addProvider': 'Agregar {app}',
  'offer.toast.killswitchOn': 'Protección activada: si una VPN se cae, Kongtrol bloquea su tráfico hasta que vuelva. Puedes cambiarlo en Seguridad.',
}

const en: Strings<typeof es> = {
  'offer.notice.killswitch':
    '{0} dropped without you asking · while it was reconnecting, your traffic went out without the VPN. Do you want Kongtrol to prevent that? If {0} drops, Kongtrol blocks its traffic until it is back. If it does not come back, you get a notice and one click brings your internet back.',
  'offer.action.enableKillswitch': 'Turn on protection',
  'offer.action.noThanks': 'No, thanks',
  'offer.action.releaseBlock': 'Get internet back',
  'offer.action.addProvider': 'Add {app}',
  'offer.toast.killswitchOn': 'Protection on: if a VPN drops, Kongtrol blocks its traffic until it is back. You can change it in Security.',
}

export default { es, en }
