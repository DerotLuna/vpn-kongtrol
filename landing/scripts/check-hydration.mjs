// Hydration check for the built landing — version 1.0.0 · 2026-11-18.
//
// Every entry hydrates the HTML that scripts/prerender.mjs wrote (hydrateRoot,
// ADR-0025). React requires the first client render to match that HTML
// exactly; a mismatch is a bug: React logs it and, for text, throws away the
// prerendered DOM and renders again on the client
// (https://react.dev/reference/react-dom/client/hydrateRoot).
//
// This runs the real entry (src/pages/<page>/main.tsx) against each built page
// in jsdom — every page × language, for three visitors:
//   - default:  no stored choice, light OS theme, Windows, browser in the page language
//   - stored:   dark theme + page language stored, macOS
//   - system:   dark OS theme, Linux, browser in the other language (language banner)
// The inline theme boot script in <head> runs first, as in a browser. The
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
  { name: 'default', ua: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)', dark: false, store: {}, browserLang: lang => lang, theme: 'light', os: 'windows' },
  { name: 'stored', ua: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5)', dark: false, store: lang => ({ kgtheme: 'dark', kglang: lang }), browserLang: lang => lang, theme: 'dark', os: 'darwin' },
  { name: 'system', ua: 'Mozilla/5.0 (X11; Linux x86_64)', dark: true, store: {}, browserLang: lang => (lang === 'es' ? 'en' : 'es'), theme: 'dark', os: 'linux' },
]

// Browser APIs jsdom lacks and the landing touches during hydration/effects.
function stubBrowser(window, visitor, lang) {
  const browserLang = visitor.browserLang(lang)
  const nav = window.navigator
  Object.defineProperty(nav, 'userAgent', { get: () => visitor.ua })
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
        runScripts: 'dangerously', // only the inline theme boot script: external scripts are not loaded
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
      const bootTheme = window.document.documentElement.getAttribute('data-theme')
      if (!firstNode) record('setup', ['no prerendered markup in #root'])
      if (bootTheme !== visitor.theme) record('boot script', [`data-theme=${bootTheme}, expected ${visitor.theme}`])

      vite.moduleGraph.invalidateAll() // fresh app modules (and module state) per run
      await vite.ssrLoadModule(`/src/pages/${page}/main.tsx`)
      await new Promise(r => setTimeout(r, SETTLE_MS))

      if (firstNode && (!firstNode.isConnected || container.firstElementChild !== firstNode)) {
        record('hydration', ['prerendered DOM was replaced instead of hydrated'])
      }
      const theme = window.document.documentElement.getAttribute('data-theme')
      if (theme !== visitor.theme) record('theme', [`data-theme=${theme} after hydration, expected ${visitor.theme}`])
      if (page === 'home' && !container.querySelector('.th-body')?.textContent.includes(`kongtrol-${visitor.os}-`)) {
        record('os', [`hero does not show the ${visitor.os} binary after hydration`])
      }

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
