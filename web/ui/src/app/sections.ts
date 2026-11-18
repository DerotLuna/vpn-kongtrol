// Kongtrol UI — information architecture · v0.1.0 · 2026-11-18
//
// The sections of the panel, in navigation order (docs/dashboard-parity.md
// maps every legacy feature to one of them; ADR-0038). `mobile` marks the
// four destinations in the phone bottom bar; the rest live under "Más".
import type { ComponentType } from 'react'
import {
  ActivityIcon,
  HomeIcon,
  ProfilesIcon,
  RulesIcon,
  SettingsIcon,
  ShieldIcon,
  TrafficIcon,
} from '../components/ui/icons'
import type { MessageKey } from '../i18n/messages'

export type SectionId = 'home' | 'profiles' | 'rules' | 'traffic' | 'security' | 'activity' | 'settings'

export interface Section {
  id: SectionId
  path: string
  label: MessageKey
  icon: ComponentType
  mobile: boolean
}

export const SECTIONS: readonly Section[] = [
  { id: 'home', path: '/', label: 'nav.home', icon: HomeIcon, mobile: true },
  { id: 'profiles', path: '/profiles', label: 'nav.profiles', icon: ProfilesIcon, mobile: true },
  { id: 'rules', path: '/rules', label: 'nav.rules', icon: RulesIcon, mobile: true },
  { id: 'traffic', path: '/traffic', label: 'nav.traffic', icon: TrafficIcon, mobile: false },
  { id: 'security', path: '/security', label: 'nav.security', icon: ShieldIcon, mobile: true },
  { id: 'activity', path: '/activity', label: 'nav.activity', icon: ActivityIcon, mobile: false },
  { id: 'settings', path: '/settings', label: 'nav.settings', icon: SettingsIcon, mobile: false },
]
