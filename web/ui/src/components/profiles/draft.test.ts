import { describe, expect, it } from 'vitest'
import type { VPNProfile } from '../../data/profiles'
import {
  backupOf,
  checkDraft,
  emptyDraft,
  failoverCycle,
  fieldForErrorKey,
  fromProfile,
  hasEnvReference,
  matchesFilter,
  parseGoDuration,
  toRequest,
  withType,
} from './draft'

function profile(name: string, extra: Partial<VPNProfile> = {}): VPNProfile {
  return {
    name,
    type: 'openvpn',
    priority: 0,
    failover: [],
    has_username_credential: false,
    has_password_credential: false,
    kill_switch_override: 'inherit',
    auto_connect: true,
    ...extra,
  }
}

describe('toRequest', () => {
  it('create sends only non-empty fields of the chosen type', () => {
    const d = { ...emptyDraft('openvpn'), name: ' work ', config: ' ~/.kongtrol/configs/w.ovpn ', host: 'ignored' }
    expect(toRequest(d, 'create')).toEqual({
      name: 'work',
      type: 'openvpn',
      priority: 10,
      auto_connect: true,
      config: '~/.kongtrol/configs/w.ovpn',
      auth_method: 'certificate',
    })
  })

  it('update sends every shown field (empty = clear) and clears attach-only fields of other types', () => {
    const d = { ...emptyDraft('wireguard'), name: 'home', config: 'c.conf', country: 'CH', iface: 'x' }
    const body = toRequest(d, 'update')
    expect(body).toMatchObject({ config: 'c.conf', cert: '', key: '', username: '', auth_method: 'certificate' })
    // Attach-only fields are cleared, a non-location server too; host is not shown, so not sent.
    expect(body).toMatchObject({ country: '', city: '', group: '', interface: '', wait_timeout: '', app_path: '', server: '' })
    expect(body).not.toHaveProperty('host')
    expect(body).not.toHaveProperty('password')
  })

  it('sends the password only when typed and auto_connect=false only where it applies', () => {
    const nord = { ...emptyDraft('nordvpn'), name: 'n', autoConnect: false, password: 'secret' }
    const body = toRequest(nord, 'update')
    expect(body.auto_connect).toBe(false)
    // Providers keep their own session: no sign-in fields at all.
    expect(body).not.toHaveProperty('password')
    expect(body).not.toHaveProperty('auth_method')

    const ovpn = { ...emptyDraft('openvpn'), name: 'o', autoConnect: false, authMethod: 'credentials' as const, password: 'pw' }
    expect(toRequest(ovpn, 'create')).toMatchObject({ auto_connect: true, password: 'pw' })
  })

  it('round-trips a stored profile without ever reading a password back', () => {
    const p = profile('corp', { type: 'forticlient', host: 'vpn.corp', auth_method: 'credentials', username: 'ana', has_password_credential: true, failover: ['b'] })
    const d = fromProfile(p)
    expect(d).toMatchObject({ type: 'forticlient', host: 'vpn.corp', username: 'ana', password: '', failover: ['b'] })
    expect(withType(d, 'openvpn').authMethod).toBe('certificate')
  })
})

describe('checkDraft', () => {
  const others = [profile('a', { failover: ['b'] }), profile('b')]

  it('flags missing name, taken name and the file each direct type needs', () => {
    expect(checkDraft(emptyDraft('openvpn'), { profiles: others })).toMatchObject({
      name: { key: 'profiles.check.nameRequired' },
      config: { key: 'profiles.check.ovpnRequired' },
    })
    const taken = { ...emptyDraft('wireguard'), name: 'a', config: 'x' }
    expect(checkDraft(taken, { profiles: others }).name?.key).toBe('profiles.check.nameTaken')
    // Editing keeps its own name.
    expect(checkDraft(taken, { profiles: others, editing: 'a' }).name).toBeUndefined()
    // Renaming onto another profile's name is caught before the daemon's 409.
    expect(checkDraft(taken, { profiles: others, editing: 'b' }).name?.key).toBe('profiles.check.nameTaken')
    expect(checkDraft({ ...emptyDraft('forticlient'), name: 'f' }, { profiles: others }).host?.key).toBe('profiles.check.hostRequired')
  })

  it('checks provider fields like config.validateProvider', () => {
    const nord = { ...emptyDraft('nordvpn'), name: 'n', server: 'us1', country: 'US', waitTimeout: '5s' }
    const p = checkDraft(nord, { profiles: [] })
    expect(p.server?.key).toBe('profiles.check.serverCombo')
    expect(p.waitTimeout?.key).toBe('profiles.check.waitTimeout')
    expect(checkDraft({ ...nord, server: '', waitTimeout: '1m30s' }, { profiles: [] })).toEqual({})
  })

  it('rejects $VAR references and a non-numeric priority', () => {
    const d = { ...emptyDraft('openvpn'), name: 'o', config: '$HOME/x.ovpn', priority: 'high' }
    expect(checkDraft(d, { profiles: [] })).toMatchObject({ config: { key: 'profiles.check.envRef' }, priority: { key: 'profiles.check.priority' } })
  })

  it('validates failover: self, unknown, duplicate, cycle', () => {
    const base = { ...emptyDraft('wireguard'), name: 'b', config: 'x' }
    expect(checkDraft({ ...base, failover: ['b'] }, { profiles: others, editing: 'b' }).failover?.key).toBe('profiles.check.failoverSelf')
    expect(checkDraft({ ...base, failover: ['zz'] }, { profiles: others, editing: 'b' }).failover?.key).toBe('profiles.check.failoverUnknown')
    expect(checkDraft({ ...base, failover: ['a', 'a'] }, { profiles: others, editing: 'b' }).failover?.key).toBe('profiles.check.failoverDuplicate')
    // a → b already; b → a closes the loop.
    expect(checkDraft({ ...base, failover: ['a'] }, { profiles: others, editing: 'b' }).failover).toEqual({
      key: 'profiles.check.failoverCycle',
      vars: { cycle: 'b → a → b' },
    })
    expect(failoverCycle('c', ['a'], others)).toBe('')
  })
})

describe('helpers', () => {
  it('parses Go durations', () => {
    expect(parseGoDuration('5m')).toBe(300)
    expect(parseGoDuration('1h2m3s')).toBe(3723)
    expect(parseGoDuration('90')).toBeNaN()
    expect(parseGoDuration('5 min')).toBeNaN()
  })

  it('detects env references like the daemon', () => {
    expect(hasEnvReference('${X}')).toBe(true)
    expect(hasEnvReference('$_x')).toBe(true)
    expect(hasEnvReference('price $5')).toBe(false)
  })

  it('maps daemon validation keys to fields and finds who uses a profile as backup', () => {
    expect(fieldForErrorKey('config.err.failover_cycle')).toBe('failover')
    expect(fieldForErrorKey('config.err.wait_timeout')).toBe('waitTimeout')
    expect(fieldForErrorKey('config.err.file_root_only')).toBe('config')
    expect(fieldForErrorKey('config.err.server_combo')).toBe('server')
    expect(fieldForErrorKey('api.err.vpn_rename_busy')).toBe('name')
    expect(fieldForErrorKey('api.err.vpn_exists')).toBe('name')
    expect(fieldForErrorKey('config.err.other')).toBeUndefined()
    expect(backupOf('b', [profile('a', { failover: ['b'] }), profile('b')])).toEqual(['a'])
  })

  it('filters by name, type, brand, host and server', () => {
    const p = profile('office', { type: 'nordvpn', server: 'us3710' })
    expect(matchesFilter(p, 'OFF')).toBe(true)
    expect(matchesFilter(p, 'nord')).toBe(true)
    expect(matchesFilter(p, 'us37')).toBe(true)
    expect(matchesFilter(p, 'zzz')).toBe(false)
  })
})
