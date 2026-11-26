import { describe, expect, it } from 'vitest'
import { tunnel } from '../test/fakes'
import type { VPNProfile } from './profiles'
import { profileDescription, profileOrder } from './profileList'
import type { ProfileHistory } from './traffic'

describe('profile list', () => {
  it('distinguishes provider apps from direct configurations without guessing a provider', () => {
    const profile = (name: string, type: VPNProfile['type'], provider?: string) =>
      ({ name, type, provider }) as VPNProfile
    expect(profileDescription(profile('a', 'protonvpn'), 'app')).toBe('Proton VPN · app')
    expect(profileDescription(profile('b', 'wireguard', 'protonvpn'), 'app')).toBe('Proton VPN · WireGuard')
    expect(profileDescription(profile('c', 'openvpn'), 'app')).toBe('OpenVPN')
    expect(profileDescription(undefined, 'app')).toBe('')
  })

  it('puts connected profiles first, then the most recently connected', () => {
    const historyEntry = (name: string, last_connected_at: string): ProfileHistory =>
      ({ name, last_connected_at, samples: 0, connected_samples: 0, uptime_seconds: 0, reconnects: 0, jitter_ms: 0, last_latency_ms: 0 })
    const history = {
      old: historyEntry('old', '2026-11-20T10:00:00Z'),
      recent: historyEntry('recent', '2026-11-24T10:00:00Z'),
    }
    expect(profileOrder([
      tunnel('old', 'disconnected'), tunnel('recent', 'disconnected'),
      tunnel('active', 'connected'), tunnel('never', 'disconnected'),
    ], history)).toEqual(['active', 'recent', 'old', 'never'])
  })
})
