import { describe, expect, it } from 'vitest'
import { browserLang } from './lang'
import { format, messages, MODULES } from './messages'

describe('messages', () => {
  it('ES and EN have exactly the same keys', () => {
    expect(Object.keys(messages.en).sort()).toEqual(Object.keys(messages.es).sort())
  })

  it('no message is empty and placeholders match across languages', () => {
    const placeholders = (s: string) => (s.match(/\{\w+\}/g) ?? []).sort()
    for (const [key, es] of Object.entries(messages.es)) {
      const en = messages.en[key as keyof typeof messages.en]
      expect(es.trim(), key).not.toBe('')
      expect(en.trim(), key).not.toBe('')
      expect(placeholders(en), key).toEqual(placeholders(es))
    }
  })

  it('no key is defined in two page modules (a later spread would hide it)', () => {
    const seen = new Map<string, string>()
    for (const [name, mod] of Object.entries(MODULES)) {
      for (const key of Object.keys(mod.es)) {
        expect(seen.get(key), `${key} in ${name}`).toBeUndefined()
        seen.set(key, name)
      }
      expect(Object.keys(mod.en).sort(), name).toEqual(Object.keys(mod.es).sort())
    }
    expect(seen.size).toBe(Object.keys(messages.es).length)
  })

  it('format fills known placeholders and keeps unknown ones', () => {
    expect(format('{a} de {b} {c}', { a: 1, b: 'x' })).toBe('1 de x {c}')
  })
})

describe('browserLang', () => {
  it('uses the first preferred language: es* -> es, else en', () => {
    expect(browserLang({ languages: ['es-AR', 'en'], language: 'es-AR' })).toBe('es')
    expect(browserLang({ languages: ['en-US', 'es'], language: 'en-US' })).toBe('en')
    expect(browserLang({ languages: [], language: 'fr' })).toBe('en')
  })
})
