// Kongtrol UI strings — first-run questions · v1.0.0 · 2026-11-23
//
// The question after the first VPN the user connects (ADR-0101), shown on
// Inicio and in the tray flyout. Same texts as internal/i18n/onboarding.go
// (`kongtrol up`). One module so parallel work never edits the same file;
// composed in ../messages.ts.
import type { Strings } from '../module'

const es = {
  'onboarding.restore.title': '¿Volver a conectar tus VPN al encender el equipo?',
  'onboarding.restore.body':
    'Kongtrol se abrirá al encender el equipo y volverá a conectar las VPN que dejaste conectadas. Las que desconectes tú no vuelven. Puedes cambiarlo en Ajustes › Al iniciar.',
  'onboarding.restore.yes': 'Sí, activar',
  'onboarding.restore.no': 'Ahora no',
}

const en: Strings<typeof es> = {
  'onboarding.restore.title': 'Reconnect your VPNs when the computer starts?',
  'onboarding.restore.body':
    'Kongtrol will open when the computer starts and reconnect the VPNs you left connected. The ones you disconnect yourself stay off. You can change this in Settings › When starting.',
  'onboarding.restore.yes': 'Yes, turn on',
  'onboarding.restore.no': 'Not now',
}

export default { es, en }
