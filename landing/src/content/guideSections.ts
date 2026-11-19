// Guide table of contents — version 2.0.0 · 2026-11-19.
// Same ids in both languages: other pages link to #install, #update,
// #uninstall, #quickstart and #commands.
export interface GuideSection {
  id: string
  num: string
  label: string
}

export const GUIDE_SECTIONS_ES: GuideSection[] = [
  { id: 'quickstart', num: '0', label: 'Inicio rápido' },
  { id: 'how', num: '1', label: 'Cómo funciona' },
  { id: 'prereqs', num: '2', label: 'Antes de empezar' },
  { id: 'install', num: '3', label: 'Instalar' },
  { id: 'wizard', num: '4', label: 'Tu primera VPN' },
  { id: 'connect', num: '5', label: 'Conectar' },
  { id: 'policies', num: '6', label: 'Reglas por VPN' },
  { id: 'groups', num: '7', label: 'Grupos' },
  { id: 'doctor', num: '8', label: 'Diagnóstico' },
  { id: 'dashboard', num: '9', label: 'El panel' },
  { id: 'daily', num: '10', label: 'Día a día' },
  { id: 'trouble', num: '11', label: 'Si algo falla' },
  { id: 'update', num: '12', label: 'Actualizar' },
  { id: 'uninstall', num: '13', label: 'Desinstalar' },
]

export const GUIDE_SECTIONS_EN: GuideSection[] = [
  { id: 'quickstart', num: '0', label: 'Quickstart' },
  { id: 'how', num: '1', label: 'How it works' },
  { id: 'prereqs', num: '2', label: 'Before you start' },
  { id: 'install', num: '3', label: 'Install' },
  { id: 'wizard', num: '4', label: 'Your first VPN' },
  { id: 'connect', num: '5', label: 'Connect' },
  { id: 'policies', num: '6', label: 'Rules per VPN' },
  { id: 'groups', num: '7', label: 'Groups' },
  { id: 'doctor', num: '8', label: 'Diagnostics' },
  { id: 'dashboard', num: '9', label: 'The panel' },
  { id: 'daily', num: '10', label: 'Day to day' },
  { id: 'trouble', num: '11', label: 'If something fails' },
  { id: 'update', num: '12', label: 'Update' },
  { id: 'uninstall', num: '13', label: 'Uninstall' },
]
