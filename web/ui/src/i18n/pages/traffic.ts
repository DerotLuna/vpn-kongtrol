// Kongtrol UI strings — Tráfico (/traffic) · v1.0.0 · 2026-11-18
//
// One module per page so parallel work never edits the same file; composed
// in ../messages.ts. ES is the source of the key set; EN must match it
// (the type below + messages.test.ts).
import type { Strings } from '../module'

const es = {
  'page.traffic.lead': 'Por dónde sale cada conexión y cuánto tráfico pasa por cada VPN.',
  'page.traffic.item1': 'Tráfico en el tiempo',
  'page.traffic.item2': 'Mapa de tráfico y «¿por qué VPN sale este destino?»',
  'page.traffic.item3': 'Rutas activas en vivo y rutas de cada VPN',
} as const

const en: Strings<typeof es> = {
  'page.traffic.lead': 'Where each connection goes out and how much traffic each VPN carries.',
  'page.traffic.item1': 'Traffic over time',
  'page.traffic.item2': 'Traffic map and “which VPN does this destination use?”',
  'page.traffic.item3': 'Live active routes and each VPN’s routes',
}

export default { es, en }
