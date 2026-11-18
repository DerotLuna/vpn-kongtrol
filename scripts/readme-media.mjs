#!/usr/bin/env node
// README media generator — version 1.0.0 · 2026-11-18.
//
// Records the README GIFs from the real landing page (English, dark theme), so
// they always show the current design:
//   assets/readme/routing.gif  routing diagram cycling destinations (/en/#map)
//   assets/readme/attach.gif   "your VPN app → Kongtrol → tray" demo (/en/#start)
//   assets/readme/hero.gif     hero terminal boot + live tunnel rows (/en/)
//
// How: builds landing/ (pnpm), serves landing/dist on 127.0.0.1 with a tiny
// built-in static server, records each clip with Playwright's video recorder
// (https://playwright.dev/docs/videos), then converts with ffmpeg using a
// generated palette (palettegen/paletteuse, https://ffmpeg.org/ffmpeg-filters.html#palettegen)
// at 15 fps, at most 720 px wide, looping forever.
//
// Playwright is NOT a project dependency. If `playwright` can't be imported,
// the script installs it once into a cache folder outside the repo
// (<os tmp>/kongtrol-readme-media) and downloads Chromium with Playwright's own
// installer. Requires Node 20+, pnpm and ffmpeg on PATH.
//
//   node scripts/readme-media.mjs                 # build landing + all clips
//   node scripts/readme-media.mjs --no-build      # reuse landing/dist
//   node scripts/readme-media.mjs --only=routing  # routing | attach | hero (comma list)
//
// The server and browser always stop before the script exits.

import { spawnSync } from 'node:child_process'
import { createReadStream, existsSync, mkdirSync, mkdtempSync, rmSync, statSync } from 'node:fs'
import { createServer } from 'node:http'
import { createRequire } from 'node:module'
import { tmpdir } from 'node:os'
import { extname, join, normalize, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const PLAYWRIGHT_VERSION = process.env.PLAYWRIGHT_VERSION || '1.65.1'
const ROOT = resolve(fileURLToPath(new URL('..', import.meta.url)))
const DIST = join(ROOT, 'landing', 'dist')
const OUT = join(ROOT, 'assets', 'readme')
const FPS = 15
const MAX_WIDTH = 720
const VIEWPORT = { width: 1280, height: 1000 } // tall enough for the whole hero terminal

// Each clip: page, element to crop, how long to record (a whole animation
// cycle, so the GIF loops without a jump), whether to scroll to it, and the
// GIF width / palette size that keep it under ~2 MB.
const CLIPS = {
  // Features.tsx cycles 4 destinations every 2.6 s.
  routing: { path: '/en/', selector: '#map .switchboard', seconds: 10.4, scroll: true, width: 640, colors: 48 },
  // attachdemo.css: one 9 s cycle.
  attach: { path: '/en/', selector: '#start .attach', seconds: 9, scroll: true, width: MAX_WIDTH, colors: 96 },
  // Hero.tsx boot sequence ends at ~3.7 s; keep the live rows on screen a while.
  hero: { path: '/en/', selector: '#motd .terminal', seconds: 6.5, scroll: false, width: 640, colors: 64 },
}

// Hide fixed overlays (header, tmux bar, language suggestion) so they never
// cover the cropped element.
const HIDE_CSS = 'nav, .tmux-bar, .lang-suggest, .skip-link { display: none !important; }'

const args = process.argv.slice(2)
const noBuild = args.includes('--no-build')
const onlyArg = args.find(a => a.startsWith('--only='))
const only = onlyArg ? onlyArg.slice('--only='.length).split(',') : Object.keys(CLIPS)
for (const name of only) {
  if (!CLIPS[name]) fail(`unknown clip "${name}" (use: ${Object.keys(CLIPS).join(', ')})`)
}

function fail(msg) {
  console.error(`readme-media: ${msg}`)
  process.exit(1)
}

// npm and pnpm are .cmd shims on Windows, so they need a shell there; ffmpeg
// and node run without one (the filter graph has quotes and semicolons).
const SHIMS = new Set(['npm', 'pnpm'])

function run(cmd, cmdArgs, opts = {}) {
  const shell = SHIMS.has(cmd) && process.platform === 'win32'
  const r = spawnSync(cmd, cmdArgs, { stdio: 'inherit', shell, ...opts })
  if (r.status !== 0) fail(`${cmd} ${cmdArgs.join(' ')} exited with ${r.status ?? r.error}`)
}

function hasCommand(cmd) {
  return spawnSync(cmd, ['-version'], { stdio: 'ignore' }).status === 0
}

async function loadPlaywright() {
  try {
    return await import('playwright')
  } catch {
    const cache = join(tmpdir(), 'kongtrol-readme-media')
    const require = createRequire(join(cache, 'package.json'))
    let entry
    try {
      entry = require.resolve('playwright')
    } catch {
      console.log(`Installing playwright@${PLAYWRIGHT_VERSION} into ${cache} (outside the repo, one time)…`)
      mkdirSync(cache, { recursive: true })
      run('npm', ['install', '--no-save', '--no-audit', '--no-fund', '--prefix', cache, `playwright@${PLAYWRIGHT_VERSION}`])
      entry = require.resolve('playwright')
    }
    // Chromium for this Playwright version (no-op when already downloaded).
    run(process.execPath, [join(cache, 'node_modules', 'playwright', 'cli.js'), 'install', 'chromium'])
    return require(entry)
  }
}

const TYPES = {
  '.html': 'text/html; charset=utf-8', '.js': 'text/javascript', '.css': 'text/css',
  '.svg': 'image/svg+xml', '.png': 'image/png', '.jpg': 'image/jpeg', '.woff2': 'font/woff2',
  '.json': 'application/json', '.txt': 'text/plain', '.xml': 'application/xml',
}

// Static server for landing/dist with clean URLs (/en/guide → en/guide.html),
// like Cloudflare Pages. Only serves files inside dist.
function serve() {
  const server = createServer((req, res) => {
    const url = decodeURIComponent(new URL(req.url, 'http://x').pathname)
    let file = normalize(join(DIST, url))
    if (file !== DIST && !file.startsWith(DIST + sep)) {
      res.writeHead(403).end()
      return
    }
    if (existsSync(file) && statSync(file).isDirectory()) file = join(file, 'index.html')
    else if (!existsSync(file) && existsSync(file + '.html')) file += '.html'
    if (!existsSync(file)) {
      res.writeHead(404).end()
      return
    }
    res.writeHead(200, { 'Content-Type': TYPES[extname(file)] || 'application/octet-stream' })
    createReadStream(file).pipe(res)
  })
  return new Promise(ok => server.listen(0, '127.0.0.1', () => ok(server)))
}

async function record(browser, base, name, clip, work) {
  const context = await browser.newContext({
    viewport: VIEWPORT,
    deviceScaleFactor: 1,
    colorScheme: 'dark',
    reducedMotion: 'no-preference',
    locale: 'en-US',
    recordVideo: { dir: work, size: VIEWPORT },
  })
  const page = await context.newPage()
  const t0 = Date.now()
  await page.goto(base + clip.path, { waitUntil: 'load' })
  await page.addStyleTag({ content: HIDE_CSS })
  await page.mouse.move(VIEWPORT.width - 1, VIEWPORT.height - 1) // never hover (hover pauses the diagram)
  const el = page.locator(clip.selector).first()
  if (clip.scroll) {
    await el.evaluate(node => node.scrollIntoView({ block: 'center' }))
    await page.waitForTimeout(1500) // reveal transition + in-view trigger
  } else {
    await page.evaluate(() => document.fonts.ready)
  }
  const box = await el.boundingBox()
  if (!box) fail(`${name}: ${clip.selector} not found`)
  const start = clip.scroll ? (Date.now() - t0) / 1000 : 0.3
  await page.waitForTimeout(clip.seconds * 1000 + 300)
  const video = page.video()
  await context.close() // flushes the video file
  return { file: await video.path(), start, box }
}

function toGif(name, clip, { file, start, box }) {
  const even = n => Math.floor(n / 2) * 2
  const x = Math.max(0, Math.round(box.x))
  const y = Math.max(0, Math.round(box.y))
  const w = even(Math.min(box.width, VIEWPORT.width - x))
  const h = even(Math.min(box.height, VIEWPORT.height - y))
  const out = join(OUT, `${name}.gif`)
  const filters =
    `crop=${w}:${h}:${x}:${y},fps=${FPS},scale='min(${clip.width},iw)':-2:flags=lanczos,split[a][b];` +
    `[a]palettegen=max_colors=${clip.colors}:stats_mode=diff[p];` +
    '[b][p]paletteuse=dither=bayer:bayer_scale=5:diff_mode=rectangle'
  run('ffmpeg', ['-v', 'error', '-y', '-ss', start.toFixed(2), '-t', String(clip.seconds), '-i', file,
    '-filter_complex', filters, '-loop', '0', out])
  const kb = Math.round(statSync(out).size / 1024)
  console.log(`${name}.gif  ${kb} KB  (${w}x${h} crop, ${clip.seconds}s)`)
  if (kb > 2048) console.warn(`readme-media: ${name}.gif is over 2 MB — shorten the clip or lower max_colors`)
}

async function main() {
  if (!hasCommand('ffmpeg')) fail('ffmpeg not found on PATH (https://ffmpeg.org/download.html)')
  if (!noBuild) {
    run('pnpm', ['--dir', 'landing', 'install', '--frozen-lockfile'], { cwd: ROOT })
    run('pnpm', ['--dir', 'landing', 'run', 'build'], { cwd: ROOT })
  }
  if (!existsSync(join(DIST, 'en', 'index.html'))) fail('landing/dist is missing: run without --no-build')
  mkdirSync(OUT, { recursive: true })

  const { chromium } = await loadPlaywright()
  const work = mkdtempSync(join(tmpdir(), 'kongtrol-readme-video-'))
  const server = await serve()
  const base = `http://127.0.0.1:${server.address().port}`
  let browser
  try {
    browser = await chromium.launch()
    for (const name of only) {
      const clip = CLIPS[name]
      toGif(name, clip, await record(browser, base, name, clip, work))
    }
  } finally {
    if (browser) await browser.close()
    server.close()
    rmSync(work, { recursive: true, force: true })
  }
}

main().catch(err => fail(err?.stack || String(err)))
