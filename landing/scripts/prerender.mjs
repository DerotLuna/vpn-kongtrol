// Static prerender for the built HTML — version 2.1.0 · 2026-11-17.
//
// Why this exists: the landing is a client-rendered React app, so the HTML Vite
// emits ships an empty <div id="root"></div>. Anything that does not execute
// JavaScript — Bing, DuckDuckGo, social preview crawlers, LLM crawlers, and a
// human doing "view source" — would see a blank page with nothing but meta tags.
//
// This renders each entry to HTML with react-dom/server and injects the markup
// into the built page, so the shipped HTML contains the real content.
//
// The client is deliberately unchanged: each entry still calls createRoot(),
// which replaces the injected markup on load. Nothing hydrates, so no hydration
// mismatch can be produced by client-only state (OS detection, stored language
// and theme). Because the server renders the same *default* state the client
// starts from, the swap is invisible for a default visitor.

import { createHash } from 'node:crypto'
import { readFile, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createElement } from 'react'
import { renderToString } from 'react-dom/server'
import { createServer } from 'vite'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')


const EMPTY_ROOT = /<div id="root">\s*<\/div>/

// A page that renders almost nothing means its entry silently produced no
// content — fail the build instead of shipping a blank page.
const MIN_MARKUP_CHARS = 2000

const vite = await createServer({
  root,
  appType: 'custom',
  logLevel: 'warn',
  server: { middlewareMode: true },
  // SSR-only use: skip the dependency pre-bundling scan, which otherwise gets
  // cancelled on close and prints a spurious "build was canceled" error.
  optimizeDeps: { noDiscovery: true, include: [] },
})

let failed = false

try {
  // Every page × language from src/seo.ts: built HTML file + the page's App.
  const seo = await vite.ssrLoadModule('/src/seo.ts')
  const PAGES = seo.ENTRIES.map(({ page, lang }) => ({ page: seo.entryFile(page, lang), entry: `/src/pages/${page}/App.tsx`, lang }))

  for (const { page, entry, lang } of PAGES) {
    const mod = await vite.ssrLoadModule(entry)
    const Component = mod.default

    if (typeof Component !== 'function') {
      throw new Error(`${entry} has no default component export`)
    }

    const markup = renderToString(createElement(Component, { lang }))

    if (markup.length < MIN_MARKUP_CHARS) {
      throw new Error(`${page}: prerendered markup is suspiciously small (${markup.length} chars)`)
    }

    const file = resolve(root, 'dist', page)
    const source = await readFile(file, 'utf8')

    if (!EMPTY_ROOT.test(source)) {
      throw new Error(`${page}: no empty <div id="root"></div> to fill`)
    }

    await writeFile(file, source.replace(EMPTY_ROOT, `<div id="root">${markup}</div>`), 'utf8')
    console.log(`prerendered ${page} — ${markup.length} chars`)
  }

  // sitemap.xml with every page in both languages + hreflang alternates
  await writeFile(resolve(root, 'dist', 'sitemap.xml'), seo.sitemapXml(), 'utf8')
  console.log(`wrote sitemap.xml — ${PAGES.length} urls`)

  // Every inline executable script must have its hash in the CSP of public/_headers,
  // or the browser blocks it in production (see docs/deploy-cloudflare.md).
  const headers = await readFile(resolve(root, 'dist', '_headers'), 'utf8')
  for (const page of ['404.html', ...PAGES.map(p => p.page)]) {
    const html = await readFile(resolve(root, 'dist', page), 'utf8')
    for (const [, attrs, body] of html.matchAll(/<script([^>]*)>([\s\S]*?)<\/script>/g)) {
      if (/\bsrc=|application\/ld\+json/.test(attrs)) continue
      const hash = `'sha256-${createHash('sha256').update(body).digest('base64')}'`
      if (!headers.includes(hash)) throw new Error(`${page}: inline script hash ${hash} missing from public/_headers script-src`)
    }
  }
  console.log('csp: every inline script hash is in _headers')
} catch (err) {
  failed = true
  console.error(`prerender failed: ${err.message}`)
} finally {
  await vite.close()
}

if (failed) process.exit(1)
