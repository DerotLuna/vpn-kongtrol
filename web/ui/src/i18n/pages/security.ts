// Kongtrol UI strings — Seguridad (/security), including the opt-in public IP card shown on Inicio · v1.0.0 · 2026-11-18
//
// One module per page so parallel work never edits the same file; composed
// in ../messages.ts. ES is the source of the key set; EN must match it
// (the type below + messages.test.ts).
import type { Strings } from '../module'

const es = {
  'publicip.title': 'Salida a internet',
  'publicip.label': 'IP pública:',
  'publicip.pill.on': 'Activada',
  'publicip.pill.off': 'Apagada',
  'publicip.hint': 'Actívalo para ver tu IP pública.',
  'publicip.notice': 'Usa un servicio externo ({host}) que ve tu IP pública',
  'publicip.check': 'Revisar',
  'publicip.checking': 'Revisando…',
  'publicip.turnOn': 'Activar',
  'publicip.turnOff': 'Apagar',
  'publicip.enabled': 'Listo: Kongtrol consultará tu IP al conectar o desconectar y cuando pulses Revisar.',
  'page.security.lead': 'Kill switch, protección DNS y revisión de tu IP pública.',
  'page.security.item1': 'Activar el kill switch y la protección DNS',
  'page.security.item2': 'Kill switch distinto para un perfil',
  'page.security.item3': 'IP pública y prueba de fugas (opcional)',
  'page.security.item4': 'Ajustes finos del kill switch, DNS y DNS dividido',
} as const

const en: Strings<typeof es> = {
  'publicip.title': 'Internet egress',
  'publicip.label': 'Public IP:',
  'publicip.pill.on': 'On',
  'publicip.pill.off': 'Off',
  'publicip.hint': 'Turn it on to see your public IP.',
  'publicip.notice': 'Uses an external service ({host}) that sees your public IP',
  'publicip.check': 'Check',
  'publicip.checking': 'Checking…',
  'publicip.turnOn': 'Turn on',
  'publicip.turnOff': 'Turn off',
  'publicip.enabled': 'Done: Kongtrol will look up your IP when you connect or disconnect and when you press Check.',
  'page.security.lead': 'Kill switch, DNS protection and your public IP check.',
  'page.security.item1': 'Turn on the kill switch and DNS protection',
  'page.security.item2': 'A different kill switch for one profile',
  'page.security.item3': 'Public IP and leak test (optional)',
  'page.security.item4': 'Fine-tuning for the kill switch, DNS and split DNS',
}

export default { es, en }
