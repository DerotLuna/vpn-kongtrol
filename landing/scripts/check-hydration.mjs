// Hydration check for the built landing — version 1.4.0 · 2026-11-19.
//
// Every entry hydrates the HTML that scripts/prerender.mjs wrote (hydrateRoot,
// ADR-0026). React requires the first client render to match that HTML
// exactly; a mismatch is a bug: React logs it and, for text, throws away the
// prerendered DOM and renders again on the client
// (https://react.dev/reference/react-dom/client/hydrateRoot).
//
// This runs the real entry (src/pages/<page>/main.tsx) against each built page
// in jsdom — every page × language, for four visitors:
//   - default:  no stored choice, light OS theme, Windows, browser in the page language
//   - stored:   page language stored, dark OS theme, macOS
//   - system:   dark OS theme, Linux, browser in the other language (language banner)
// The landing has one designed theme (ADR-0060): whatever the OS theme, no
// data-theme attribute may appear on <html>.
//   - mobile:   iPhone (touch): boot script marks data-device=mobile, so CSS shows
//               "send to my computer" instead of download buttons (ADR-0027, ADR-0029)
// The inline device boot script in <head> runs first, as in a browser. The
// check FAILS (exit 1) on any console.error/console.warn from React (the
// development build reports every mismatch, text and attributes alike), any
// recoverable or uncaught error, or when the prerendered nodes were replaced
// instead of hydrated. It needs `pnpm build` first; no browser download.
//
// Usage: node scripts/check-hydration.mjs   (pnpm check:hydration)

process.env.NODE_ENV = 'development' // React's dev build reports every mismatch

import { readFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { JSDOM, VirtualConsole } from 'jsdom'
import { createServer } from 'vite'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const SETTLE_MS = 250 // hydration + passive effects + the post-hydration re-render

const VISITORS = [
  { name: 'default', ua: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)', dark: false, store: {}, browserLang: lang => lang, os: 'windows' },
  { name: 'stored', ua: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5)', dark: true, store: lang => ({ kglang: lang }), browserLang: lang => lang, os: 'darwin' },
  { name: 'system', ua: 'Mozilla/5.0 (X11; Linux x86_64)', dark: true, store: {}, browserLang: lang => (lang === 'es' ? 'en' : 'es'), os: 'linux' },
  { name: 'mobile', ua: 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1', dark: false, store: {}, browserLang: lang => lang, os: 'darwin', mobile: true },
]

// hero download button label once the OS is known (src/components/ui/downloadCta.tsx)
const OS_WORD = { windows: 'Windows', darwin: 'Mac', linux: 'Linux' }

// Browser APIs jsdom lacks and the landing touches during hydration/effects.
function stubBrowser(window, visitor, lang) {
  const browserLang = visitor.browserLang(lang)
  const nav = window.navigator
  Object.defineProperty(nav, 'userAgent', { get: () => visitor.ua })
  Object.defineProperty(nav, 'maxTouchPoints', { get: () => (visitor.mobile ? 5 : 0) })
  Object.defineProperty(nav, 'language', { get: () => browserLang })
  Object.defineProperty(nav, 'languages', { get: () => [browserLang] })
  window.matchMedia = query => ({
    matches: query.includes('prefers-color-scheme: dark') ? visitor.dark : false,
    media: query, onchange: null,
    addEventListener() {}, removeEventListener() {}, addListener() {}, removeListener() {},
    dispatchEvent: () => false,
  })
  window.IntersectionObserver = class { observe() {} unobserve() {} disconnect() {} takeRecords() { return [] } }
  window.localStorage.clear()
  const store = typeof visitor.store === 'function' ? visitor.store(lang) : visitor.store
  for (const [k, v] of Object.entries(store)) window.localStorage.setItem(k, v)
}

// Point the browser globals the app and React read at the current jsdom
// window. Getters, so each run swaps the window without reloading React.
let current = null
const FORCED = ['window', 'self', 'document', 'navigator', 'location', 'localStorage', 'sessionStorage', 'matchMedia', 'getComputedStyle', 'requestAnimationFrame', 'cancelAnimationFrame', 'innerWidth', 'innerHeight', 'scrollY', 'IntersectionObserver', 'reportError']
function installGlobals(window) {
  const keys = new Set([...FORCED, ...Object.getOwnPropertyNames(window).filter(k => !(k in globalThis))])
  for (const k of keys) {
    Object.defineProperty(globalThis, k, { configurable: true, get: () => current?.[k] })
  }
}

const problems = []
let runProblems = []
const record = (kind, args) => runProblems.push(`${kind}: ${args.map(a => (a instanceof Error ? a.message : String(a))).join(' ').split('\n')[0].slice(0, 300)}`)
for (const level of ['error', 'warn']) {
  const original = console[level].bind(console)
  console[level] = (...args) => { record(`console.${level}`, args); if (process.env.HYDRATION_VERBOSE) original(...args) }
}
process.on('uncaughtException', err => record('uncaught', [err]))
process.on('unhandledRejection', err => record('unhandled rejection', [err]))

const vite = await createServer({
  root,
  appType: 'custom',
  logLevel: 'warn',
  server: { middlewareMode: true },
  optimizeDeps: { noDiscovery: true, include: [] },
})

let runs = 0
try {
  const seo = await vite.ssrLoadModule('/src/seo.ts')
  const { PATHS } = await vite.ssrLoadModule('/src/site.ts')

  for (const { page, lang } of seo.ENTRIES) {
    const file = seo.entryFile(page, lang)
    const html = await readFile(resolve(root, 'dist', file), 'utf8')

    for (const visitor of VISITORS) {
      runProblems = []
      const virtualConsole = new VirtualConsole()
      virtualConsole.on('jsdomError', err => {
        // The redirect to a stored other language is not part of this check.
        if (!/Not implemented: navigation/.test(err.message)) record('jsdom', [err])
      })
      const dom = new JSDOM(html, {
        url: `https://kongtrol.app${PATHS[page][lang]}`,
        runScripts: 'dangerously', // only the inline device boot script: external scripts are not loaded
        pretendToBeVisual: true,
        virtualConsole,
        beforeParse: window => stubBrowser(window, visitor, lang),
      })
      const { window } = dom
      if (!current) installGlobals(window)
      current = window
      window.addEventListener('error', e => record('window error', [e.error ?? e.message]))

      const container = window.document.getElementById('root')
      const firstNode = container?.firstElementChild
      if (!firstNode) record('setup', ['no prerendered markup in #root'])

      vite.moduleGraph.invalidateAll() // fresh app modules (and module state) per run
      const { root: reactRoot } = await vite.ssrLoadModule(`/src/pages/${page}/main.tsx`)
      await new Promise(r => setTimeout(r, SETTLE_MS))

      if (firstNode && (!firstNode.isConnected || container.firstElementChild !== firstNode)) {
        record('hydration', ['prerendered DOM was replaced instead of hydrated'])
      }
      if (window.document.documentElement.hasAttribute('data-theme')) record('theme', ['data-theme set on <html>: the landing has one theme (ADR-0060)'])
      // Windows gets the installer (links.ts WINDOWS_INSTALLER_FILE), the others their binary
      const osFile = visitor.os === 'windows' ? 'Kongtrol-Setup-windows-' : `kongtrol-${visitor.os}-`
      if (page === 'home' && !container.querySelector('.th-body')?.textContent.includes(osFile)) {
        record('os', [`hero does not show the ${visitor.os} file (${osFile}…) after hydration`])
      }
      if (page === 'home') {
        // The hero carries both variants; CSS shows one by <html data-device>,
        // which the boot script sets before first paint (ADR-0029).
        const heroCta = container.querySelector('#hero-cta')
        const device = window.document.documentElement.getAttribute('data-device')
        if (!heroCta?.querySelector('.th-ctas-mobile[data-send-to-computer]')) record('mobile', ['hero has no "send to my computer" block'])
        if (visitor.mobile && device !== 'mobile') record('mobile', [`data-device=${device} on a phone: hero would show download buttons`])
        if (!visitor.mobile) {
          const primary = heroCta?.querySelector('.th-ctas-desktop a.primary')?.textContent ?? ''
          if (device !== null || !primary.includes(OS_WORD[visitor.os])) {
            record('cta', [`data-device=${device}, hero download button is "${primary}", expected one for ${OS_WORD[visitor.os]}`])
          }
        }
      }

      // Run effect cleanups while this window is still the active DOM. React's
      // scheduler is shared across visitors and can have callbacks queued.
      reactRoot.unmount()
      window.close()
      current = null
      runs++
      const label = `${file} [${visitor.name}]`
      if (runProblems.length) {
        problems.push(...runProblems.map(p => `${label} ${p}`))
        console.log(`FAIL ${label}`)
      } else {
        console.log(`ok   ${label}`)
      }
    }
  }
} catch (err) {
  problems.push(`check crashed: ${err.stack ?? err}`)
} finally {
  await vite.close()
}

if (problems.length) {
  for (const p of problems) process.stderr.write(`hydration: ${p}\n`)
  process.stderr.write(`hydration check failed: ${problems.length} problem(s)\n`)
  process.exit(1)
}
console.log(`hydration: ${runs} runs, no mismatch`)
process.exit(0)
