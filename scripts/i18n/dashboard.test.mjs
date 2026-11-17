// i18n parity for the legacy dashboard (web/dashboard) — version 1.0.0 · 2026-11-17.
//
// Every top-level `const X = { es: {...}, en: {...} }` dictionary in
// web/dashboard/*.js must have the same keys in both languages, no empty
// text and the same {placeholders}. Same rules as web/ui's
// messages.test.ts and internal/i18n's parity test.
// Run: node --test 'scripts/i18n/*.test.mjs' (Node's built-in runner, no deps:
// https://nodejs.org/api/test.html). It lives outside web/dashboard
// because that folder is embedded into the binary as is.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import vm from 'node:vm'

const dir = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..', 'web', 'dashboard')

// Finds `const NAME = {` whose first property is `es: {`, up to the
// matching closing brace, and evaluates only that object literal.
function dictionaries(file) {
  const src = readFileSync(join(dir, file), 'utf8')
  const out = []
  for (const m of src.matchAll(/^const (\w+) = \{\s*\n\s*es: \{/gm)) {
    const start = src.indexOf('{', m.index)
    let depth = 0, end = start, quote = null
    for (let i = start; i < src.length; i++) {
      const c = src[i]
      if (quote) {
        if (c === '\\') i++
        else if (c === quote) quote = null
        continue
      }
      if (c === '/' && src[i + 1] === '/') i = src.indexOf('\n', i)
      else if (c === "'" || c === '"' || c === '`') quote = c
      else if (c === '{') depth++
      else if (c === '}' && --depth === 0) { end = i; break }
    }
    out.push({ name: `${file}:${m[1]}`, dict: vm.runInNewContext(`(${src.slice(start, end + 1)})`) })
  }
  return out
}

const placeholders = (s) => (String(s).match(/\{\w+\}/g) ?? []).sort()
const all = readdirSync(dir).filter((f) => f.endsWith('.js')).flatMap(dictionaries)

test('found the dashboard dictionaries', () => {
  assert.ok(all.length >= 5, `only ${all.length} dictionaries found`)
})

for (const { name, dict } of all) {
  test(`${name}: ES and EN match`, () => {
    assert.deepEqual(Object.keys(dict.en).sort(), Object.keys(dict.es).sort(), 'same keys in es and en')
    for (const [key, es] of Object.entries(dict.es)) {
      const en = dict.en[key]
      assert.notEqual(String(es).trim(), '', `es ${key} is empty`)
      assert.notEqual(String(en).trim(), '', `en ${key} is empty`)
      assert.deepEqual(placeholders(en), placeholders(es), `${key}: placeholders differ`)
    }
  })
}
