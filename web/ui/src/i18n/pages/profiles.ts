// Kongtrol UI strings — Perfiles (/profiles) and Grupos (/profiles/groups) · v1.0.0 · 2026-11-18
//
// One module per page so parallel work never edits the same file; composed
// in ../messages.ts. ES is the source of the key set; EN must match it
// (the type below + messages.test.ts).
import type { Strings } from '../module'

const es = {
  'page.profiles.lead': 'Las VPN que Kongtrol maneja: cómo se conecta cada una y qué hacer si se cae.',
  'page.profiles.tab.vpns': 'Perfiles VPN',
  'page.profiles.tab.groups': 'Grupos',
  'page.profiles.item1': 'Crear, editar y borrar perfiles VPN',
  'page.profiles.item2': 'Guardar contraseñas en el llavero del sistema',
  'page.profiles.item3': 'Kill switch y respaldo (failover) de cada perfil',
  'page.groups.item1': 'Crear, editar y borrar grupos de VPN',
  'page.groups.item2': 'Conectar, desconectar y recargar un grupo entero',
} as const

const en: Strings<typeof es> = {
  'page.profiles.lead': 'The VPNs Kongtrol manages: how each one connects and what to do if it drops.',
  'page.profiles.tab.vpns': 'VPN profiles',
  'page.profiles.tab.groups': 'Groups',
  'page.profiles.item1': 'Create, edit and delete VPN profiles',
  'page.profiles.item2': 'Keep passwords in the system keychain',
  'page.profiles.item3': 'Kill switch and backup (failover) per profile',
  'page.groups.item1': 'Create, edit and delete VPN groups',
  'page.groups.item2': 'Connect, disconnect and reload a whole group',
}

export default { es, en }
