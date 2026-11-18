// Kongtrol UI strings — Actividad (/activity) · v1.0.0 · 2026-11-18
//
// One module per page so parallel work never edits the same file; composed
// in ../messages.ts. ES is the source of the key set; EN must match it
// (the type below + messages.test.ts).
import type { Strings } from '../module'

const es = {
  'page.activity.lead': 'Lo que hizo Kongtrol y los avisos importantes, en orden.',
  'page.activity.item1': 'Registro de auditoría firmado',
  'page.activity.item2': 'Avisos del sistema',
  'page.activity.item3': 'Dónde se guarda el registro y cuánto ocupa',
} as const

const en: Strings<typeof es> = {
  'page.activity.lead': 'What Kongtrol did and the important notices, in order.',
  'page.activity.item1': 'Signed audit log',
  'page.activity.item2': 'System notices',
  'page.activity.item3': 'Where the log is kept and how big it gets',
}

export default { es, en }
