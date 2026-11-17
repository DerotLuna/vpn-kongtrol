// i18n parity check for the landing — version 1.0.0 · 2026-11-17.
//
// Every bilingual catalog (guide content per OS, guide sections, legal
// pages, SEO meta, generated CLI reference) must have the same shape in ES
// and EN — same keys, same array lengths — and no empty text. Keys are
// already enforced by TypeScript (GuideContent, Record<Lang, …>); this adds
// what types can't see: empty strings/fragments and array lengths.
// Same idea as web/ui's messages.test.ts and internal/i18n's parity test.
//
// Docs/content never block a merge (AGENTS.md): problems are printed as
// GitHub ::warning annotations and the exit code is 0, unless --strict.
import { isValidElement, createElement, Fragment } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { readFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer } from 'vite'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const strict = process.argv.includes('--strict')
const problems = []

const textOf = (el) => renderToStaticMarkup(createElement(Fragment, null, el)).replace(/<[^>]*>/g, '').trim()

function compare(name, es, en, path = '') {
  const where = `${name}${path}`
  if (isValidElement(es) || isValidElement(en) || typeof es === 'string' || typeof en === 'string') {
    for (const [lang, v] of [['es', es], ['en', en]]) {
      const s = typeof v === 'string' ? v.trim() : isValidElement(v) ? textOf(v) : null
      if (s === null) problems.push(`${where}: ${lang} is ${typeof v}, the other language has text`)
      else if (s === '') problems.push(`${where}: ${lang} is empty`)
    }
    return
  }
  if (Array.isArray(es) || Array.isArray(en)) {
    if (!Array.isArray(es) || !Array.isArray(en) || es.length !== en.length) {
      problems.push(`${where}: es has ${es?.length} items, en has ${en?.length}`)
      return
    }
    es.forEach((v, i) => compare(name, v, en[i], `${path}[${i}]`))
    return
  }
  if (es && en && typeof es === 'object' && typeof en === 'object') {
    for (const k of new Set([...Object.keys(es), ...Object.keys(en)])) {
      if (!(k in es) || !(k in en)) problems.push(`${where}.${k}: missing in ${k in es ? 'en' : 'es'}`)
      else compare(name, es[k], en[k], `${path}.${k}`)
    }
  }
}

const vite = await createServer({
  root, appType: 'custom', logLevel: 'warn', server: { middlewareMode: true },
  optimizeDeps: { noDiscovery: true, include: [] },
})
try {
  const load = (p) => vite.ssrLoadModule(p)
  const es = (await load('/src/content/guide.es.tsx')).default
  const en = (await load('/src/content/guide.en.tsx')).default
  for (const os of ['windows', 'macos', 'linux']) compare(`guide(${os})`, es(os), en(os))
  const sections = await load('/src/content/guideSections.ts')
  compare('guideSections', sections.GUIDE_SECTIONS_ES, sections.GUIDE_SECTIONS_EN)
  for (const page of ['privacidad', 'seguridad', 'terminos']) {
    const doc = (await load(`/src/content/legal/${page}.tsx`)).default
    compare(`legal/${page}`, doc.es, doc.en)
  }
  const seo = await load('/src/seo.ts')
  for (const [page, m] of Object.entries(seo.META)) compare(`seo.${page}`, m.es, m.en)
  const cli = JSON.parse(await readFile(resolve(root, '..', 'docs', 'cli.json'), 'utf8'))
  compare('docs/cli.json', cli.langs.es, cli.langs.en)
} finally {
  await vite.close()
}

for (const p of problems) console.log(process.env.GITHUB_ACTIONS === 'true' ? `::warning title=Landing i18n::${p}` : `warning: ${p}`)
console.log(problems.length ? `check-i18n: ${problems.length} problem(s)` : 'check-i18n: ES and EN match')
if (strict && problems.length) process.exit(1)
