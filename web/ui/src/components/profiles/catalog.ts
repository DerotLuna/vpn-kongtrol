// Kongtrol UI — the VPN types and what each one needs · v1.0.0 · 2026-11-18
//
// One table drives the type picker, the guided form (which fields each type
// shows) and the list's "works on this system" chip.
// Sources: fields per type = legacy studio.html:145-204 (data-types) and
// internal/config/semantics.go validateProvider (location only for nordvpn
// and protonvpn, group only nordvpn, attach fields only for providers);
// default auth = configs/example.yaml; OS support = the landing's VPN matrix
// (landing/src/components/sections/Providers.tsx, checked against each
// adapter's platform files). Brand names are proper nouns, not translated;
// monograms are our own marks (ADR-0023).
import type { Platform } from '../../lib/platform'
import type { AdapterType } from '../../data/profiles'

/** yes: works · auto: Kongtrol connects through the official CLI · manual: you connect in the app · partial: works with a limit. */
export type Support = 'yes' | 'auto' | 'manual' | 'partial'
type OS = 'windows' | 'mac' | 'linux'

/** Where the auth fields sit: main section, under "Más opciones", or none (the app holds the session). */
export type AuthPlacement = 'main' | 'advanced' | 'none'

export interface AdapterSpec {
  type: AdapterType
  label: string
  mark: string
  /** attach = works with the provider's official app; direct = Kongtrol brings the tunnel up. */
  group: 'attach' | 'direct'
  os: Partial<Record<OS, Support>>
  /** Path to a .conf / .ovpn file (required). */
  config?: 'ovpn' | 'conf'
  /** Gateway host: required for corporate clients, optional exit node for Tailscale. */
  host?: 'required' | 'optional'
  /** Country / city / exact server (the official CLI can pick a location). */
  location?: boolean
  /** Exact region name or ID passed to the official CLI. */
  exactRegion?: boolean
  /** NordVPN server group. */
  serverGroup?: boolean
  /** Proton protocol. */
  protocol?: boolean
  /** "Connect automatically" (official CLI or, for FortiClient, GUI automation). */
  autoConnect?: boolean
  /** Attach-mode fields: interface, wait_timeout, app_path. */
  attach?: boolean
  auth: AuthPlacement
  defaultAuth: AuthMethod
}

export type AuthMethod = 'certificate' | 'credentials' | 'certificate+credentials'
export const AUTH_METHODS: AuthMethod[] = ['certificate', 'credentials', 'certificate+credentials']

export const CATALOG: readonly AdapterSpec[] = [
  // Attach group: the provider's official app (internal/vpn/appattach).
  {
    type: 'nordvpn', label: 'NordVPN', mark: 'Nv', group: 'attach',
    os: { windows: 'auto', mac: 'manual', linux: 'auto' },
    location: true, serverGroup: true, autoConnect: true, attach: true, auth: 'none', defaultAuth: 'credentials',
  },
  {
    type: 'protonvpn', label: 'Proton VPN', mark: 'Pv', group: 'attach',
    os: { windows: 'manual', mac: 'manual', linux: 'auto' },
    location: true, protocol: true, autoConnect: true, attach: true, auth: 'none', defaultAuth: 'credentials',
  },
  {
    type: 'surfshark', label: 'Surfshark', mark: 'Ss', group: 'attach',
    os: { windows: 'manual', mac: 'manual', linux: 'manual' },
    attach: true, auth: 'none', defaultAuth: 'credentials',
  },
  {
    type: 'forticlient', label: 'FortiClient', mark: 'Fc', group: 'attach',
    os: { windows: 'auto', mac: 'yes', linux: 'yes' },
    host: 'required', autoConnect: true, attach: true, auth: 'main', defaultAuth: 'credentials',
  },
  {
    type: 'mullvad', label: 'Mullvad VPN', mark: 'Mv', group: 'attach',
    os: { windows: 'auto', mac: 'partial', linux: 'auto' },
    location: true, autoConnect: true, attach: true, auth: 'none', defaultAuth: 'credentials',
  },
  {
    type: 'pia', label: 'Private Internet Access', mark: 'Pa', group: 'attach',
    os: { windows: 'auto', mac: 'partial', linux: 'auto' },
    exactRegion: true, autoConnect: true, attach: true, auth: 'none', defaultAuth: 'credentials',
  },
  {
    type: 'expressvpn', label: 'ExpressVPN', mark: 'Ev', group: 'attach',
    os: { windows: 'manual', mac: 'partial', linux: 'auto' },
    exactRegion: true, autoConnect: true, attach: true, auth: 'none', defaultAuth: 'credentials',
  },
  {
    type: 'windscribe', label: 'Windscribe', mark: 'Ws', group: 'attach',
    os: { windows: 'auto', mac: 'partial', linux: 'auto' },
    exactRegion: true, autoConnect: true, attach: true, auth: 'none', defaultAuth: 'credentials',
  },
  {
    type: 'cyberghost', label: 'CyberGhost VPN', mark: 'Cg', group: 'attach',
    os: { windows: 'manual', mac: 'manual', linux: 'partial' },
    attach: true, auth: 'none', defaultAuth: 'credentials',
  },
  // Direct group: Kongtrol brings the tunnel up itself or drives the vendor CLI.
  {
    type: 'wireguard', label: 'WireGuard', mark: 'Wg', group: 'direct',
    os: { windows: 'yes', mac: 'yes', linux: 'yes' },
    config: 'conf', auth: 'advanced', defaultAuth: 'certificate',
  },
  {
    type: 'openvpn', label: 'OpenVPN', mark: 'Ov', group: 'direct',
    os: { windows: 'yes', mac: 'partial', linux: 'yes' },
    config: 'ovpn', auth: 'main', defaultAuth: 'certificate',
  },
  {
    type: 'tailscale', label: 'Tailscale', mark: 'Ts', group: 'direct',
    os: { windows: 'yes', mac: 'yes', linux: 'yes' },
    host: 'optional', auth: 'advanced', defaultAuth: 'credentials',
  },
  {
    type: 'cloudflarewarp', label: 'Cloudflare WARP', mark: 'Cw', group: 'direct',
    os: { windows: 'yes', mac: 'yes', linux: 'yes' },
    auth: 'advanced', defaultAuth: 'credentials',
  },
  {
    type: 'ciscoanyconnect', label: 'Cisco AnyConnect', mark: 'Ca', group: 'direct',
    os: { windows: 'yes', mac: 'yes', linux: 'yes' },
    host: 'required', auth: 'main', defaultAuth: 'credentials',
  },
  {
    type: 'globalprotect', label: 'GlobalProtect', mark: 'Gp', group: 'direct',
    os: { windows: 'yes', mac: 'yes' },
    host: 'required', auth: 'main', defaultAuth: 'credentials',
  },
]

const BY_TYPE = new Map(CATALOG.map((s) => [s.type, s]))

export function specOf(type: string): AdapterSpec {
  return BY_TYPE.get(type as AdapterType) ?? (BY_TYPE.get('openvpn') as AdapterSpec)
}

export function isKnownType(type: string): type is AdapterType {
  return BY_TYPE.has(type as AdapterType)
}

/** Support on this computer's OS; 'none' = not available; undefined = OS unknown. */
export function supportOn(spec: AdapterSpec, platform: Platform): Support | 'none' | undefined {
  if (platform === 'other') return undefined
  return spec.os[platform] ?? 'none'
}
