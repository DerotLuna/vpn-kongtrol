// Kongtrol UI strings — Reglas (/rules) and Horarios (/rules/schedule) · v1.0.0 · 2026-11-18
//
// One module per page so parallel work never edits the same file; composed
// in ../messages.ts. ES is the source of the key set; EN must match it
// (the type below + messages.test.ts).
import type { Strings } from '../module'

const es = {
  'page.rules.lead': 'Qué sitios y apps salen por cada VPN, y cuándo se conectan solas.',
  'page.rules.tab.policies': 'Por destino',
  'page.rules.tab.schedule': 'Horarios',
  'page.rules.item1': 'Crear, editar y borrar reglas por IP, dominio o app',
  'page.rules.item2': 'Probar una regla antes de guardarla',
  'page.rules.item3': 'Elegir qué pasa si la VPN de una regla está apagada',
  'page.schedule.item1': 'Conectar y desconectar VPN por horario',
  'page.schedule.item2': 'Activar o pausar cada horario',
} as const

const en: Strings<typeof es> = {
  'page.rules.lead': 'Which sites and apps go through each VPN, and when VPNs connect by themselves.',
  'page.rules.tab.policies': 'By destination',
  'page.rules.tab.schedule': 'Schedules',
  'page.rules.item1': 'Create, edit and delete rules by IP, domain or app',
  'page.rules.item2': 'Test a rule before saving it',
  'page.rules.item3': 'Choose what happens when a rule’s VPN is off',
  'page.schedule.item1': 'Connect and disconnect VPNs on a schedule',
  'page.schedule.item2': 'Turn each schedule on or pause it',
}

export default { es, en }
