export interface GuideSection {
  id: string
  num: string
  label: string
}

export const GUIDE_SECTIONS_ES: GuideSection[] = [
  { id: 'prereqs', num: '01', label: 'Prerequisitos' },
  { id: 'install', num: '02', label: 'Instalación' },
  { id: 'certs', num: '03', label: 'Certificados' },
  { id: 'wizard', num: '04', label: 'kongtrol init' },
  { id: 'groups', num: '05', label: 'Grupos y políticas' },
  { id: 'doctor', num: '06', label: 'Doctor check' },
  { id: 'connect', num: '07', label: 'Primera conexión' },
  { id: 'dashboard', num: '08', label: 'Dashboard web' },
  { id: 'daily', num: '09', label: 'Uso diario' },
  { id: 'trouble', num: '10', label: 'Solución de problemas' },
]

export const GUIDE_SECTIONS_EN: GuideSection[] = [
  { id: 'prereqs', num: '01', label: 'Prerequisites' },
  { id: 'install', num: '02', label: 'Installation' },
  { id: 'certs', num: '03', label: 'Connection files' },
  { id: 'wizard', num: '04', label: 'kongtrol init' },
  { id: 'groups', num: '05', label: 'Groups & policies' },
  { id: 'doctor', num: '06', label: 'Doctor check' },
  { id: 'connect', num: '07', label: 'First connect' },
  { id: 'dashboard', num: '08', label: 'Web dashboard' },
  { id: 'daily', num: '09', label: 'Daily usage' },
  { id: 'trouble', num: '10', label: 'Troubleshooting' },
]
