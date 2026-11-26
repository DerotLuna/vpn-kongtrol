import { describe, expect, it } from 'vitest'
import { addSearch, parseAddParams, withoutAdd } from '../../app/addLink'
// The same table internal/policy/classify_test.go runs against Go's
// ClassifyTargets / AutoName (`kongtrol policy add`), ADR-0055.
import shared from '../../../../../internal/policy/testdata/classify_targets.json?raw'
import {
  autoName,
  appendRulePreset,
  classifyEntry,
  classifyTargets,
  cleanName,
  defaultVia,
  emptyRuleDraft,
  isValidName,
  matchOf,
  nameSource,
  ovpnNeedsCredentials,
  probeOf,
  profileNameFromFile,
  typeForContent,
  typeForFile,
  uniqueName,
} from './model'

interface Table {
  cases: { name: string; in: string[]; subdomains: boolean; domains: string[]; ip_ranges: string[]; apps: string[]; first: string; invalid: string[] }[]
  auto_name: { first: string; via: string; want: string }[]
}
const table = JSON.parse(shared) as Table

describe('"Sitio, IP o programa": same reading as policy.ClassifyTargets (shared table)', () => {
  it('has cases', () => expect(table.cases.length).toBeGreaterThan(5))
  for (const c of table.cases) {
    it(c.name, () => {
      expect(classifyTargets(c.in, c.subdomains)).toEqual({
        domains: c.domains,
        ip_ranges: c.ip_ranges,
        apps: c.apps,
        first: c.first,
        invalid: c.invalid,
      })
    })
  }
  for (const c of table.auto_name) {
    it(`AutoName(${c.first})`, () => expect(autoName(`${nameSource(c.first)} por ${c.via}`, [])).toBe(c.want))
  }
})

describe('"Sitio, IP o programa": what the sheet adds on top', () => {
  it('adds editable presets without losing prior entries or their subdomains', () => {
    const first = appendRulePreset(emptyRuleDraft('EXAMPLE.com\nclaude.ai', 'work'), 'claude')
    expect(first.subs).toBe(false)
    expect(first.input).toContain('*.example.com')
    expect(first.input?.match(/claude\.ai/gi)).toHaveLength(2)
    const second = appendRulePreset({ ...emptyRuleDraft('', 'work'), ...first }, 'chatgpt')
    expect(second.input).toContain('ws.chatgpt.com')
    expect(second.input).toContain('*.oaiusercontent.com')
    const again = appendRulePreset({ ...emptyRuleDraft('', 'work'), ...second }, 'chatgpt')
    expect(again.input).toBe(second.input)
  })

  it('labels for sentences: hosts without "*.", programs by file name', () => {
    expect(classifyEntry('*.acme.test')).toEqual({ ok: true, entry: { kind: 'domain', value: '*.acme.test', label: 'acme.test' } })
    expect(classifyEntry('app:C:\\Program Files\\Teams\\teams.exe')).toEqual({
      ok: true,
      entry: { kind: 'app', value: 'C:\\Program Files\\Teams\\teams.exe', label: 'teams.exe' },
    })
  })

  it('says why an entry is refused, and hints (without refusing) at a bare word or a program', () => {
    expect(classifyEntry('  ')).toMatchObject({ ok: false, problem: 'empty' })
    expect(classifyEntry('app:')).toMatchObject({ ok: false, problem: 'appEmpty' })
    expect(classifyEntry('foo..bar')).toMatchObject({ ok: false, problem: 'invalid' })
    expect(classifyEntry('10.0.0.0/40')).toMatchObject({ ok: false, problem: 'invalid' })
    expect(classifyEntry('intranet')).toMatchObject({ ok: true, hint: 'noDot', entry: { kind: 'domain', value: 'intranet' } })
    expect(classifyEntry('chrome.exe')).toMatchObject({ ok: true, hint: 'exe', entry: { kind: 'domain', value: 'chrome.exe' } })
    expect(classifyEntry('claude.ai')).not.toHaveProperty('hint')
  })

  it('preview asks about a host or the first address of a range', () => {
    const entries = [classifyEntry('claude.ai'), classifyEntry('10.0.0.0/8'), classifyEntry('app:code.exe'), classifyEntry('*.x.test')].flatMap((c) =>
      c.ok ? [c.entry] : [],
    )
    expect(matchOf(entries, false).domains).toEqual(['claude.ai', '*.x.test'])
    expect(probeOf(entries[1]!)).toEqual({ target: '10.0.0.0', app: '' })
    expect(probeOf(entries[2]!)).toEqual({ target: '', app: 'code.exe' })
    expect(probeOf(entries[3]!)).toEqual({ target: 'x.test', app: '' })
  })
})

describe('names', () => {
  it('validates like config.ValidateName', () => {
    for (const ok of ['oficina', 'Oficina VPN', 'casa-2', 'trabajo ñandú', 'claude.ai por Trabajo']) expect(isValidName(ok), ok).toBe(true)
    for (const bad of ['', '   ', 'a<b', 'a"b', "a'b", 'a`b', 'a&b', 'a\\b', 'a\nb', `a${String.fromCharCode(0x2028)}b`, 'x'.repeat(65)]) {
      expect(isValidName(bad), JSON.stringify(bad)).toBe(false)
    }
  })

  it('cleans like policy.CleanName and dedupes like SuggestName (case-insensitive)', () => {
    expect(cleanName(' Nord<VPN> "Pro" ')).toBe('NordVPN Pro')
    expect(isValidName(cleanName('a\tb&c'))).toBe(true)
    expect([...cleanName('x'.repeat(80))].length).toBe(60)
    expect(uniqueName('NordVPN', ['nordvpn'])).toBe('NordVPN 2')
    expect(uniqueName('NordVPN', ['NordVPN', 'NordVPN 2'])).toBe('NordVPN 3')
    expect(uniqueName('Proton VPN', ['work'])).toBe('Proton VPN')
    expect(autoName('a.com via v', ['A.COM VIA V'])).toBe('a.com via v 2')
  })

  it('a profile name from a file: valid for the stored copy too, never taken', () => {
    expect(profileNameFromFile('Oficina Madrid.ovpn', [])).toBe('Oficina-Madrid')
    expect(profileNameFromFile('wg0.conf', ['wg0'])).toBe('wg0-2')
    expect(profileNameFromFile('ñandú.conf', [])).toBe('nandu')
  })
})

describe('files', () => {
  it('type from the extension, like profileimport.TypeForFile', () => {
    expect(typeForFile('a.conf')).toBe('wireguard')
    expect(typeForFile('A.OVPN')).toBe('openvpn')
    expect(typeForFile('a.txt')).toBeNull()
    expect(typeForFile('conf')).toBeNull()
    // a .conf is decided by its content (OpenVPN GUI on Windows uses .conf)
    expect(typeForContent('wg0.conf', '[Interface]\nPrivateKey = x\n')).toBe('wireguard')
    expect(typeForContent('aws.conf', 'client\r\ndev tun\r\nremote vpn 1194\r\nca ca.crt\r\n')).toBe('openvpn')
    expect(typeForContent('x.conf', '[Interface]\nremote = no\n')).toBe('wireguard')
    expect(typeForContent('x.ovpn', '[Interface]')).toBe('openvpn')
    expect(typeForContent('x.txt', 'client')).toBeNull()
  })

  it('auth-user-pass outside a comment asks for credentials', () => {
    expect(ovpnNeedsCredentials('client\nauth-user-pass\n')).toBe(true)
    expect(ovpnNeedsCredentials('client\n  auth-user-pass creds.txt\n')).toBe(true)
    expect(ovpnNeedsCredentials('client\n# auth-user-pass\n; auth-user-pass\n')).toBe(false)
    expect(ovpnNeedsCredentials('auth-user-pass-verify x\n')).toBe(false)
  })
})

describe('which VPN', () => {
  const tunnels = [
    { Name: 'home', Status: 'disconnected' },
    { Name: 'work', Status: 'connected' },
  ]
  it('deep link, else the only profile, else the first connected, else none', () => {
    expect(defaultVia(['home', 'work'], tunnels, 'home')).toBe('home')
    expect(defaultVia(['home', 'work'], tunnels, 'gone')).toBe('work')
    expect(defaultVia(['solo'], tunnels)).toBe('solo')
    expect(defaultVia(['home', 'work'], tunnels)).toBe('work')
    expect(defaultVia(['home', 'tv'], tunnels)).toBe('')
    expect(defaultVia([], tunnels)).toBe('')
  })
})

describe('deep link', () => {
  it('parses ?add= with its prefill, trims, and opens the menu for unknown modes', () => {
    expect(parseAddParams(new URLSearchParams(''))).toBeNull()
    expect(parseAddParams(new URLSearchParams('add=rule&via=Trabajo&target=claude.ai'))).toEqual({ mode: 'rule', via: 'Trabajo', target: 'claude.ai' })
    expect(parseAddParams(new URLSearchParams('add=vpn&type=nordvpn&app='))).toEqual({ mode: 'vpn', type: 'nordvpn' })
    expect(parseAddParams(new URLSearchParams('add=file'))).toEqual({ mode: 'file' })
    expect(parseAddParams(new URLSearchParams('add=whatever'))).toEqual({ mode: 'menu' })
    expect(parseAddParams(new URLSearchParams('add=rule&app=%20teams.exe%20'))).toEqual({ mode: 'rule', app: 'teams.exe' })
    expect(parseAddParams(new URLSearchParams(`add=rule&target=${'a'.repeat(400)}`))?.target).toHaveLength(260)
  })

  it('keeps the page own parameters when opening and closing', () => {
    const q = new URLSearchParams('tab=x&add=menu&via=old')
    expect(addSearch(q, { mode: 'rule', via: 'work' })).toBe('?tab=x&add=rule&via=work')
    expect(withoutAdd(new URLSearchParams('tab=x&add=rule&via=work&target=a.b'))).toBe('?tab=x')
    expect(withoutAdd(new URLSearchParams('add=menu'))).toBe('')
  })
})
