import { execFileSync } from 'node:child_process'
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { basename, dirname, relative, resolve } from 'node:path'
import { defineConfig, HtmlTagDescriptor, Plugin, Rollup } from 'vite'
import react from '@vitejs/plugin-react'
import { ENTRIES, entryFile, entryFor, headFor } from './src/seo'

// Multi-page app (https://vite.dev/guide/build#multi-page-app): every page in
// Spanish at the root and in English under /en/ — 10 thin HTML entries.
// Served from the root of https://kongtrol.app (Workers static assets map clean
// URLs like /guia to guia.html and /en/guide to en/guide.html; landing/wrangler.jsonc).

// Fills each entry's <!--kongtrol:head--> with its title, meta, canonical,
// hreflang, Open Graph and JSON-LD from src/seo.ts, in dev and build alike.
function seoHead(): Plugin {
  let root = __dirname
  return {
    name: 'kongtrol-seo-head',
    configResolved(c) { root = c.root },
    transformIndexHtml: {
      order: 'pre',
      handler(html, ctx) {
        const file = relative(root, ctx.filename)
        const entry = entryFor(file)
        if (!entry) throw new Error(`seo: ${file} is not a known page entry (src/seo.ts)`)
        return html.replace('<!--kongtrol:head-->', headFor(entry.page, entry.lang).trimStart())
      },
    },
  }
}

// The faces the first screen paints with (hero terminal = Plex Mono, the
// plain-language line = Inter). Fonts are bundled from assets/brand/fonts with
// content-hashed names (cached as immutable, public/_headers), so their URLs
// exist only after bundling: the preloads are added then (ADR-0031).
// Preloaded fonts need crossorigin even same-origin:
// https://web.dev/articles/preload-critical-assets
const PRELOAD_FONTS = ['ibm-plex-mono-latin-400.woff2', 'inter-latin-var.woff2']
function fontPreloads(): Plugin {
  return {
    name: 'kongtrol-font-preloads',
    apply: 'build',
    transformIndexHtml: {
      order: 'post',
      handler(html, ctx) {
        const assets = Object.values(ctx.bundle ?? {}).filter((o): o is Rollup.OutputAsset => o.type === 'asset')
        const tags: HtmlTagDescriptor[] = PRELOAD_FONTS.map(font => {
          const asset = assets.find(a => a.originalFileNames.some(f => basename(f) === font))
          if (!asset) throw new Error(`font preload: ${font} is not in the bundle (src/index.css @import)`)
          return { tag: 'link', attrs: { rel: 'preload', href: `/${asset.fileName}`, as: 'font', type: 'font/woff2', crossorigin: true }, injectTo: 'head' }
        })
        return { html, tags }
      },
    },
  }
}

// The latest stable Kongtrol release (the footer's `kongtrol version` line),
// resolved at BUILD time so visitors never call a third party (CSP
// connect-src 'self', /privacidad): 1) KONGTROL_RELEASE, 2) the GitHub
// Releases API ("latest" skips drafts and pre-releases;
// https://docs.github.com/en/rest/releases/releases#get-the-latest-release),
// 5 s timeout, 3) the newest stable git tag, 4) none (the line is left out).
// Cached for an hour in node_modules/.cache so build, prerender and the
// hydration check (three processes) bake the same value. A new release shows
// up on the next landing build.
const RELEASE_CACHE = resolve(__dirname, 'node_modules', '.cache', 'kongtrol-latest-release.json')
const STABLE_TAG = /^v\d+\.\d+\.\d+$/
async function latestRelease(): Promise<string | null> {
  const env = process.env.KONGTROL_RELEASE
  if (env && STABLE_TAG.test(env)) return env
  try {
    const c = JSON.parse(readFileSync(RELEASE_CACHE, 'utf8')) as { tag: string | null; at: number }
    if (Date.now() - c.at < 3_600_000) return c.tag
  } catch { /* no cache yet */ }
  let tag: string | null = null
  try {
    const res = await fetch('https://api.github.com/repos/DerotLuna/vpn-kongtrol/releases/latest', {
      headers: { Accept: 'application/vnd.github+json', 'User-Agent': 'kongtrol-landing-build' },
      signal: AbortSignal.timeout(5000),
    })
    if (res.ok) {
      const t = ((await res.json()) as { tag_name?: string }).tag_name
      if (t && STABLE_TAG.test(t)) tag = t
    }
  } catch { /* offline or rate limited: fall back to git */ }
  if (!tag) {
    try {
      const t = execFileSync('git', ['describe', '--tags', '--abbrev=0', '--exclude', '*-*'], { cwd: __dirname, stdio: ['ignore', 'pipe', 'ignore'] }).toString().trim()
      if (STABLE_TAG.test(t)) tag = t
    } catch { /* shallow clone without tags */ }
  }
  try {
    mkdirSync(dirname(RELEASE_CACHE), { recursive: true })
    writeFileSync(RELEASE_CACHE, JSON.stringify({ tag, at: Date.now() }))
  } catch { /* read-only node_modules: no cache */ }
  return tag
}

export default defineConfig(async () => ({
  base: '/',
  define: { __LATEST_RELEASE__: JSON.stringify(await latestRelease()) },
  plugins: [react(), seoHead(), fontPreloads()],
  resolve: {
    // single source of brand assets (assets/brand), shared with web/ui and the dashboard
    alias: { '@brand': resolve(__dirname, '..', 'assets', 'brand') },
  },
  build: {
    rollupOptions: {
      input: Object.fromEntries(ENTRIES.map(({ page, lang }) => [`${lang}-${page}`, resolve(__dirname, entryFile(page, lang))])),
    },
  },
}))
