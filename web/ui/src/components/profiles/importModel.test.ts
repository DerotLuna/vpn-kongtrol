import { describe, expect, it } from 'vitest'
import { importName, toBase64 } from './importModel'

describe('importName', () => {
  it('uses the profile name, else the file name, as a safe file name', () => {
    expect(importName('Oficina Norte', 'x.conf')).toBe('Oficina-Norte')
    expect(importName('', 'proveedor-españa.ovpn')).toBe('proveedor-espana')
    expect(importName('  ', '../..//evil.conf')).toBe('evil')
    expect(importName('', '.conf')).toBe('vpn')
    expect(importName('a'.repeat(80), 'x.conf')).toHaveLength(64)
  })
})

describe('toBase64', () => {
  it('encodes bytes beyond one chunk', () => {
    const bytes = new Uint8Array(70_000).map((_, i) => i % 251)
    const back = Uint8Array.from(atob(toBase64(bytes)), (c) => c.charCodeAt(0))
    expect(back).toEqual(bytes)
  })
})
