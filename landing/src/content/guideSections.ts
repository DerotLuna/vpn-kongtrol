export interface GuideSection {
  id: string
  num: string
  label: string
}

export const GUIDE_SECTIONS_ES: GuideSection[] = [
  { id: 'quickstart', num: '0', label: 'Inicio rápido' },
  { id: 'prereqs', num: '1', label: 'Prerequisitos' },
  { id: 'install', num: '2', label: 'Instalación' },
  { id: 'wizard', num: '3', label: 'kongtrol init' },
  { id: 'groups', num: '4', label: 'Grupos' },
  { id: 'policies', num: '5', label: 'Políticas de routing' },
  { id: 'doctor', num: '6', label: 'Doctor check' },
  { id: 'connect', num: '7', label: 'Primera conexión' },
  { id: 'dashboard', num: '8', label: 'Dashboard web' },
  { id: 'daily', num: '9', label: 'Uso diario' },
  { id: 'trouble', num: '10', label: 'Solución de problemas' },
]

export const GUIDE_SECTIONS_EN: GuideSection[] = [
  { id: 'quickstart', num: '0', label: 'Quickstart' },
  { id: 'prereqs', num: '1', label: 'Prerequisites' },
  { id: 'install', num: '2', label: 'Installation' },
  { id: 'wizard', num: '3', label: 'kongtrol init' },
  { id: 'groups', num: '4', label: 'Groups' },
  { id: 'policies', num: '5', label: 'Routing policies' },
  { id: 'doctor', num: '6', label: 'Doctor check' },
  { id: 'connect', num: '7', label: 'First connect' },
  { id: 'dashboard', num: '8', label: 'Web dashboard' },
  { id: 'daily', num: '9', label: 'Daily usage' },
  { id: 'trouble', num: '10', label: 'Troubleshooting' },
]
