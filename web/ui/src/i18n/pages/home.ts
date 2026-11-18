// Kongtrol UI strings — Inicio (/) · v1.0.0 · 2026-11-18
//
// One module per page so parallel work never edits the same file; composed
// in ../messages.ts. ES is the source of the key set; EN must match it
// (the type below + messages.test.ts).
import type { Strings } from '../module'

const es = {
  'welcome.eyebrow': 'Primeros pasos',
  'welcome.title': 'Bienvenido a Kongtrol',
  'welcome.lead': 'Controla todas tus VPN desde un solo lugar. Sigue estos pasos para empezar.',
  'welcome.actions.title': '¿Qué quieres hacer?',
  'welcome.actions.add.title': 'Agregar VPN',
  'welcome.actions.add.body': 'Suma la VPN de tu trabajo o de tu proveedor.',
  'welcome.actions.connect.title': 'Conectar',
  'welcome.actions.connect.body': 'Elige una VPN y conéctala con un clic.',
  'welcome.actions.status.title': 'Ver estado',
  'welcome.actions.status.body': 'Mira qué está conectado y cuánto tráfico pasa.',
  'welcome.actions.opensClassic': 'Se abre en el panel clásico.',
} as const

const en: Strings<typeof es> = {
  'welcome.eyebrow': 'Getting started',
  'welcome.title': 'Welcome to Kongtrol',
  'welcome.lead': 'Control all your VPNs from one place. Follow these steps to get started.',
  'welcome.actions.title': 'What do you want to do?',
  'welcome.actions.add.title': 'Add VPN',
  'welcome.actions.add.body': 'Add your work VPN or your provider’s.',
  'welcome.actions.connect.title': 'Connect',
  'welcome.actions.connect.body': 'Pick a VPN and connect it in one click.',
  'welcome.actions.status.title': 'View status',
  'welcome.actions.status.body': 'See what is connected and how much traffic flows.',
  'welcome.actions.opensClassic': 'Opens in the classic dashboard.',
}

export default { es, en }
