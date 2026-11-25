// SEO head + sitemap, single source for every page and language — version 2.6.1 · 2026-11-19.
//
// Used at build time only (vite.config.ts injects headFor() into each HTML entry,
// scripts/prerender.mjs writes sitemapXml()). Specs followed:
// - hreflang (self + every alternate, reciprocal, x-default):
//   https://developers.google.com/search/docs/specialty/international/localized-versions
// - canonical: https://developers.google.com/search/docs/crawling-indexing/consolidate-duplicate-urls
// - sitemap + xhtml:link alternates: https://www.sitemaps.org/protocol.html
// - Open Graph: https://ogp.me/  · JSON-LD: https://schema.org/SoftwareApplication, /WebPage, /TechArticle
import type { Lang } from './i18n'
import { PATHS, PageId, SITE_UPDATED, SITE_URL, SITE_VERSION, urlFor } from './site'

const REPO = 'https://github.com/DerotLuna/vpn-kongtrol'
const OG_IMAGE = `${SITE_URL}/og-1200x630.jpg`
const LANGS: Lang[] = ['es', 'en']
// Fallback for visitors whose language is neither es nor en: English.
const X_DEFAULT: Lang = 'en'

interface Meta { title: string; description: string; ogDescription?: string; index: boolean }

export const META: Record<PageId, Record<Lang, Meta>> = {
  home: {
    es: {
      title: 'Todas tus VPN bajo Kongtrol',
      description: 'Une todas tus VPN en un solo lugar: reconexión automática, kill switch y reglas por destino. Funciona con NordVPN, Proton, WireGuard y más. Gratis y libre.',
      ogDescription: 'Conecta como siempre; Kongtrol se encarga del resto. 15 VPN, reconexión automática, protección contra fugas de DNS, kill switch y reglas de enrutamiento. Gratis y de código abierto para Windows, macOS y Linux.',
      index: true,
    },
    en: {
      title: 'All your VPNs under Kongtrol',
      description: 'All your VPNs in one place: auto-reconnect, kill switch and per-destination routing. Works with NordVPN, Proton, WireGuard and more. Free and open source.',
      ogDescription: 'Connect the way you always do; Kongtrol handles the rest. 15 VPNs, auto-reconnect, DNS leak protection, kill switch and routing rules. Free and open source for Windows, macOS and Linux.',
      index: true,
    },
  },
  guia: {
    es: {
      title: 'Guía de Kongtrol — instalar, conectar y enrutar',
      description: 'Guía paso a paso de Kongtrol: instala, crea tu primera VPN, conecta y elige qué sale por cada VPN. Sin terminal, con el comando de cada paso. Windows, macOS y Linux.',
      ogDescription: 'Instala Kongtrol, da permiso una vez, crea tu primera VPN en el panel y conecta. Cada paso dice qué hacer y qué vas a ver, para Windows, macOS y Linux.',
      index: true,
    },
    en: {
      title: 'Kongtrol guide — install, connect and route',
      description: 'Step-by-step Kongtrol guide: install, create your first VPN, connect and choose what goes through each VPN. No terminal needed, with the command for each step. Windows, macOS and Linux.',
      ogDescription: 'Install Kongtrol, give permission once, create your first VPN in the panel and connect. Each step says what to do and what you will see, for Windows, macOS and Linux.',
      index: true,
    },
  },
  terminos: {
    es: { title: 'Términos y condiciones — Kongtrol', description: 'Términos y condiciones de Kongtrol: software de código abierto bajo licencia Apache 2.0, sin garantía, en beta activa.', index: true },
    en: { title: 'Terms & Conditions — Kongtrol', description: 'Kongtrol terms and conditions: open-source software under the Apache 2.0 license, no warranty, active beta.', index: true },
  },
  privacidad: {
    es: { title: 'Política de privacidad — Kongtrol', description: 'Qué datos se quedan en tu equipo, qué conexiones hace Kongtrol por su cuenta y a quién, y qué ve este sitio web. Sin telemetría ni analíticas.', index: true },
    en: { title: 'Privacy Policy — Kongtrol', description: 'What data stays on your machine, which connections Kongtrol makes on its own and to whom, and what this website sees. No telemetry, no analytics.', index: true },
  },
  seguridad: {
    es: { title: 'Política de seguridad y divulgación responsable — Kongtrol', description: 'Cómo reportar una vulnerabilidad en Kongtrol de forma privada, qué está en alcance y qué tiempos de respuesta esperar.', index: true },
    en: { title: 'Security Policy & Responsible Disclosure — Kongtrol', description: 'How to privately report a vulnerability in Kongtrol, what is in scope and what response times to expect.', index: true },
  },
}

const AUTHOR = { '@type': 'Person', name: 'DerotLuna', url: 'https://github.com/DerotLuna' }

function jsonLd(page: PageId, lang: Lang): object {
  const m = META[page][lang]
  const url = urlFor(page, lang)
  const site = { '@type': 'WebSite', name: 'Kongtrol', url: urlFor('home', lang) }
  if (page === 'home') {
    return {
      '@context': 'https://schema.org',
      '@type': 'SoftwareApplication',
      name: 'Kongtrol',
      alternateName: ['VPN Kongtrol', 'Kongtrol CLI', 'vpn-kongtrol'],
      applicationCategory: 'SecurityApplication',
      applicationSubCategory: 'VPN manager',
      operatingSystem: 'Windows, Linux, macOS',
      description: m.description,
      inLanguage: lang,
      url,
      image: OG_IMAGE,
      downloadUrl: `${REPO}/releases/latest`,
      softwareHelp: { '@type': 'CreativeWork', url: urlFor('guia', lang) },
      license: 'https://www.apache.org/licenses/LICENSE-2.0',
      isAccessibleForFree: true,
      offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
      author: AUTHOR,
      sameAs: [REPO],
    }
  }
  if (page === 'guia') {
    return {
      '@context': 'https://schema.org',
      '@type': 'TechArticle',
      headline: m.title,
      description: m.description,
      url,
      inLanguage: lang,
      dateModified: SITE_UPDATED,
      author: AUTHOR,
      about: { '@type': 'SoftwareApplication', name: 'Kongtrol', applicationCategory: 'SecurityApplication', operatingSystem: 'Windows, Linux, macOS', sameAs: [REPO] },
      isPartOf: site,
    }
  }
  return {
    '@context': 'https://schema.org',
    '@type': 'WebPage',
    name: m.title,
    description: m.description,
    url,
    inLanguage: lang,
    dateModified: SITE_UPDATED,
    isPartOf: site,
    publisher: AUTHOR,
  }
}

const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
const OG_LOCALE: Record<Lang, string> = { es: 'es_ES', en: 'en_US' }

// Runs before first paint, outside the React tree (<html> attributes only):
// data-device="mobile" on phones/tablets, so CSS can pick the hero's
//    "send to my computer" block in the first paint — no swap, no layout
//    shift after hydration (ADR-0029). Must mirror detectMobile() in
//    hooks/useVisitor.ts. The landing has one designed theme (ADR-0060), so
//    nothing about colour is decided here.
// Editing it changes its CSP hash: update public/_headers (the build says so).
const DEVICE_BOOT = `(function(){var d=document.documentElement;try{var n=navigator,u=n.userAgent,h=n.userAgentData;if((h&&h.mobile)||/Android|iPhone|iPad|iPod/i.test(u)||(/Macintosh/i.test(u)&&(n.maxTouchPoints||0)>1))d.setAttribute('data-device','mobile')}catch(e){}})();`

export function headFor(page: PageId, lang: Lang): string {
  const m = META[page][lang]
  const url = urlFor(page, lang)
  const og = m.ogDescription ?? m.description
  const other = LANGS.find(l => l !== lang)!
  const lines = [
    `<meta charset="UTF-8" />`,
    `<meta name="viewport" content="width=device-width, initial-scale=1.0" />`,
    `<title>${esc(m.title)}</title>`,
    `<meta name="description" content="${esc(m.description)}" />`,
    `<meta name="robots" content="${m.index ? 'index, follow, max-image-preview:large, max-snippet:-1' : 'noindex, follow'}" />`,
    `<meta name="version" content="${SITE_VERSION.replace(/^v/, '')}" />`,
    `<meta name="date" content="${SITE_UPDATED}" />`,
    `<link rel="canonical" href="${url}" />`,
    ...LANGS.map(l => `<link rel="alternate" hreflang="${l}" href="${urlFor(page, l)}" />`),
    `<link rel="alternate" hreflang="x-default" href="${urlFor(page, X_DEFAULT)}" />`,
    `<link rel="icon" type="image/svg+xml" href="/favicon.svg" />`,
    `<link rel="icon" type="image/png" sizes="32x32" href="/logo-kong-32.png" />`,
    `<link rel="apple-touch-icon" href="/apple-touch-icon.png" />`,
    `<link type="text/plain" rel="author" href="/humans.txt" />`,
    // Fonts: bundled by src/index.css (@import '@brand/fonts/fonts.css') with
    // hashed names; vite.config.ts fontPreloads() adds their preloads (ADR-0031).
    // the header is always Ink (ADR-0060)
    `<meta name="theme-color" content="#0a0c10" />`,
    // keep the view-transition snapshot of a page change until the whole
    // prerendered page is parsed, so the browser has already scrolled to the
    // #section the language switch carries (Chrome: render-blocking link)
    `<link rel="expect" href="#site-footer" blocking="render" />`,
    `<meta property="og:type" content="${page === 'home' ? 'website' : 'article'}" />`,
    `<meta property="og:site_name" content="Kongtrol" />`,
    `<meta property="og:title" content="${esc(m.title)}" />`,
    `<meta property="og:description" content="${esc(og)}" />`,
    `<meta property="og:url" content="${url}" />`,
    `<meta property="og:image" content="${OG_IMAGE}" />`,
    `<meta property="og:image:width" content="1200" />`,
    `<meta property="og:image:height" content="630" />`,
    `<meta property="og:image:alt" content="Kongtrol" />`,
    `<meta property="og:locale" content="${OG_LOCALE[lang]}" />`,
    `<meta property="og:locale:alternate" content="${OG_LOCALE[other]}" />`,
    `<meta name="twitter:card" content="summary_large_image" />`,
    `<meta name="twitter:title" content="${esc(m.title)}" />`,
    `<meta name="twitter:description" content="${esc(og)}" />`,
    `<meta name="twitter:image" content="${OG_IMAGE}" />`,
    `<script>${DEVICE_BOOT}</script>`,
    `<script type="application/ld+json">${JSON.stringify(jsonLd(page, lang))}</script>`,
  ]
  return lines.map(l => '    ' + l).join('\n')
}

// HTML entry file (relative to landing/) -> page + language.
export function entryFor(file: string): { page: PageId; lang: Lang } | null {
  const f = file.replace(/\\/g, '/').replace(/^\.?\//, '')
  for (const page of Object.keys(PATHS) as PageId[]) {
    for (const lang of LANGS) if (entryFile(page, lang) === f) return { page, lang }
  }
  return null
}

// The file mirrors the localized URL: /guia -> guia.html, /en/guide -> en/guide.html.
export function entryFile(page: PageId, lang: Lang): string {
  const p = PATHS[page][lang].replace(/^\//, '')
  return (p === '' || p.endsWith('/') ? p + 'index' : p) + '.html'
}

export const ENTRIES = (Object.keys(PATHS) as PageId[]).flatMap(page => LANGS.map(lang => ({ page, lang })))

export function sitemapXml(): string {
  const urls = ENTRIES.filter(e => META[e.page][e.lang].index).map(({ page, lang }) => {
    const alts = [...LANGS.map(l => [l, urlFor(page, l)]), ['x-default', urlFor(page, X_DEFAULT)]]
      .map(([h, u]) => `    <xhtml:link rel="alternate" hreflang="${h}" href="${u}"/>`).join('\n')
    return `  <url>\n    <loc>${urlFor(page, lang)}</loc>\n    <lastmod>${SITE_UPDATED}</lastmod>\n${alts}\n  </url>`
  })
  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">\n${urls.join('\n')}\n</urlset>\n`
}
