// Copy brand assets from the repo-wide single source (../assets/brand) into
// public/ before dev/build — version 1.1.0 · 2026-11-17.
//
// Why: the logo used to be committed three times (assets/, web/dashboard/,
// landing/public/). assets/brand/ is now the only committed copy; the landing
// receives its files at build time and public/ copies are git-ignored.
//
// Cloudflare Workers Builds uses root directory `landing`: the build command
// runs inside landing/ of the cloned repository, so ../assets/brand resolves.
// If it is missing the build fails loudly instead of shipping a page without
// logos.

import { copyFile, cp, mkdir, stat } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const landing = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const brand = resolve(landing, '..', 'assets', 'brand')
const pub = resolve(landing, 'public')

// brand file -> path served by the landing
const FILES = {
  'favicon.svg': 'favicon.svg',
  'logo.svg': 'logo.svg',
  'logo-kong.svg': 'logo-kong.svg',
  'logo-kong-32.png': 'logo-kong-32.png',
  'logo-kong-192.png': 'logo-kong-192.png',
  'og-1200x630.jpg': 'og-1200x630.jpg',
  // legacy URLs kept alive for old shares/hotlinks, now the optimized rasters
  'logo-kong-512.png': 'logo-kong.png',
  'logo-512.png': 'logo.png',
}

try {
  await stat(brand)
} catch {
  console.error(`copy-brand: ${brand} not found — the landing needs the repo's assets/brand/ next to landing/`)
  process.exit(1)
}

await mkdir(pub, { recursive: true })
for (const [from, to] of Object.entries(FILES)) {
  await copyFile(resolve(brand, from), resolve(pub, to))
}
// Self-hosted fonts (fonts.css + woff2), shared with the dashboard and web/ui.
await cp(resolve(brand, 'fonts'), resolve(pub, 'fonts'), { recursive: true })
console.log(`copy-brand: ${Object.keys(FILES).length} files + fonts/ from assets/brand -> public/`)
