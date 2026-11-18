import { relative, resolve } from 'node:path'
import { defineConfig, Plugin } from 'vite'
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

export default defineConfig({
  base: '/',
  plugins: [react(), seoHead()],
  build: {
    rollupOptions: {
      input: Object.fromEntries(ENTRIES.map(({ page, lang }) => [`${lang}-${page}`, resolve(__dirname, entryFile(page, lang))])),
    },
  },
})
