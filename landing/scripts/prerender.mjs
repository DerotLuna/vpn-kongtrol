// Static prerender for the built HTML — version 3.1.0 · 2026-11-18.
//
// Why this exists: the landing is a client-rendered React app, so the HTML Vite
// emits ships an empty <div id="root"></div>. Anything that does not execute
// JavaScript — Bing, DuckDuckGo, social preview crawlers, LLM crawlers, and a
// human doing "view source" — would see a blank page with nothing but meta tags.
//
// This renders each entry to HTML with react-dom/server and injects the markup
// into the built page, so the shipped HTML contains the real content.
//
// The client hydrates this markup: each entry calls hydrateRoot(), which keeps
// the prerendered DOM and attaches React to it (ADR-0026). Its first render must
// match this one exactly, so browser-only state (theme, OS, stored language,
// clock) is read with useSyncExternalStore + a server snapshot or in an effect,
// never during render. scripts/check-hydration.mjs (pnpm check:hydration) hydrates
// every built page × language, for a default and a non-default visitor, and
// fails on any hydration error or warning.
//
// It also inlines each page's bundled CSS into a <style> and deletes the now
// unused .css files, so no stylesheet blocks the first paint (ADR-0031).

import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { readdir, readFile, unlink, writeFile } from 'node:fs/promises'
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

  // Inline the page's stylesheets (ADR-0031): the CSS of each page is ~15 KB
  // gzip, and as <link>s it blocked the first paint for extra round trips
  // (Lighthouse render-blocking-insight). CSP style-src already allows
  // 'unsafe-inline'. Fonts stay external files with hashed names.
  const cssFiles = new Set()
  for (const { page } of PAGES) {
    const file = resolve(root, 'dist', page)
    const html = await readFile(file, 'utf8')
    const links = [...html.matchAll(/<link rel="stylesheet"[^>]*href="\/(assets\/[^"]+\.css)"[^>]*>/g)]
    if (links.length === 0) throw new Error(`${page}: no bundled stylesheet to inline`)
    let css = ''
    for (const [, href] of links) {
      cssFiles.add(href)
      css += await readFile(resolve(root, 'dist', href), 'utf8')
    }
    if (css.includes('</style')) throw new Error(`${page}: CSS contains "</style", cannot inline`)
    let out = html
    // at the end of <head>: the font preloads and module scripts above it are
    // discovered before the browser parses ~70 KB of CSS
    for (const [tag] of links) out = out.replace(tag, '')
    out = out.replace('</head>', `<style>${css}</style>\n</head>`)
    await writeFile(file, out, 'utf8')
  }
  // 404.html is a static file (public/): give it the bundled @font-face rules
  // in place of its fonts.css link, so the fonts exist only once, hashed.
  const fontFaces = [...cssFiles].map(f => readFileSync(resolve(root, 'dist', f), 'utf8'))
    .join('').match(/@font-face\{[^}]*\}/g) ?? []
  if (fontFaces.length === 0) throw new Error('no @font-face rules in the bundled CSS')
  const notFound = resolve(root, 'dist', '404.html')
  const nf = await readFile(notFound, 'utf8')
  const FONTS_LINK = '<link rel="stylesheet" href="/fonts/fonts.css" />'
  if (!nf.includes(FONTS_LINK)) throw new Error('404.html: fonts link not found')
  await writeFile(notFound, nf.replace(FONTS_LINK, `<style>${fontFaces.join('')}</style>`), 'utf8')
  // A stylesheet nothing links any more is dead weight in the deploy.
  const shipped = (await Promise.all([...PAGES.map(p => p.page), '404.html'].map(p => readFile(resolve(root, 'dist', p), 'utf8')))).join('')
  const scripts = (await readdir(resolve(root, 'dist', 'assets'))).filter(f => f.endsWith('.js'))
  const js = (await Promise.all(scripts.map(f => readFile(resolve(root, 'dist', 'assets', f), 'utf8')))).join('')
  for (const href of cssFiles) {
    const name = href.split('/').pop()
    if (shipped.includes(name) || js.includes(name)) throw new Error(`${href} is still referenced after inlining`)
    await unlink(resolve(root, 'dist', href))
  }
  console.log(`css: inlined ${cssFiles.size} stylesheets into every page, @font-face into 404.html`)

  // sitemap.xml with every page in both languages + hreflang alternates
  await writeFile(resolve(root, 'dist', 'sitemap.xml'), seo.sitemapXml(), 'utf8')
  console.log(`wrote sitemap.xml — ${PAGES.length} urls`)

  // Every inline executable script must have its hash in the CSP of public/_headers,
  // or the browser blocks it in production (see docs/deploy-cloudflare.md).
  const headers = await readFile(resolve(root, 'dist', '_headers'), 'utf8')
  for (const page of ['404.html', ...PAGES.map(p => p.page)]) {
    const html = await readFile(resolve(root, 'dist', page), 'utf8')
    for (const [, attrs, body] of html.matchAll(/<script\b([^>]*)>([\s\S]*?)<\/script\b[^>]*>/gi)) {
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
