import { basename, relative, resolve } from 'node:path'
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

export default defineConfig({
  base: '/',
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
})
