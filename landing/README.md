# VPN Kongtrol Landing (`landing/`)

Version 2.4.0 · 2026-11-17

Public marketing/docs site for **VPN Kongtrol** (short name: *Kongtrol*, CLI binary `kongtrol`),
separate from the embedded dashboard (`web/dashboard`).

Five pages, each in Spanish (root URLs) and English (`/en/...`) — a Vite multi-page app
with 10 HTML entries:

- `/` — landing: the hero is an SSH session (ASCII motd, auto boot, then an
  interactive demo shell — try `help`, `status`, `map claude.ai`, `sudo rm -rf /`),
  then the plain-language promise with the attach-mode demo and the two paths
  (everyone / devs), how to get started, providers, policy switchboard, security,
  under the hood, download, tmux-style status bar
- `/guia` — full setup guide (ES/EN, per-OS tabs)
- `/terminos` — terms (Apache-2.0, no warranty, beta)
- `/privacidad` — privacy policy (what stays local, what Kongtrol sends and to whom)
- `/seguridad` — security policy and responsible disclosure (mirrors `/SECURITY.md` and `/.well-known/security.txt`)

English twins (localized slugs): `/en/`, `/en/guide`, `/en/terms`, `/en/privacy`, `/en/security`.

Language and theme follow the browser until the visitor picks one; an explicit pick is stored
(`kglang`, `kgtheme`) and wins. There is no automatic language redirect on a guess (Google:
https://developers.google.com/search/docs/specialty/international/managing-multi-regional-sites):
a visitor whose browser language differs from the page gets a dismissible "View in English /
Ver en español" bar; only a stored explicit pick switches URL client-side.

## Prerequisites

- Node.js 20+
- pnpm 10+

## Local development

```bash
cd landing
pnpm install
pnpm run dev
```

## Build

```bash
cd landing
pnpm run build
```

The build runs four steps: `node scripts/copy-brand.mjs` (logos from `../assets/brand`) →
`tsc` (typecheck) → `vite build` → `node scripts/prerender.mjs` (static HTML + `sitemap.xml`).

The prerender step matters for search visibility. The app is client-rendered, so Vite
emits an empty `<div id="root"></div>`; anything that does not execute JavaScript —
Bing, DuckDuckGo, social preview crawlers, LLM crawlers — would otherwise index a
blank page. `scripts/prerender.mjs` server-renders each entry with `react-dom/server`
and injects the markup into the built HTML. The client is unchanged and still boots
with `createRoot()`, which replaces the injected markup on load.

Do not remove the prerender step from `build`; the site would silently go back to
shipping empty pages.

## Quality check

```bash
cd landing
pnpm run check
```

## Project structure

```text
landing/
├── index.html … seguridad.html   # Spanish entries (thin; <head> injected from src/seo.ts)
├── en/*.html                     # English entries
├── vite.config.ts                # MPA inputs + seo head plugin
├── scripts/
│   ├── copy-brand.mjs            # copies ../assets/brand into public/ (git-ignored copies)
│   └── prerender.mjs             # post-build static prerender + sitemap.xml
├── public/                       # robots.txt, llms.txt, 404.html, fonts/, .well-known/security.txt
└── src/
    ├── site.ts                   # routes per language, site version/date
    ├── seo.ts                    # titles, descriptions, canonical, hreflang, OG, JSON-LD, sitemap
    ├── pages/<page>/{main,App}.tsx   # home, guia, terminos, privacidad, seguridad
    ├── pages/legal.tsx           # shared shell for the legal pages
    ├── components/sections/      # page sections (Hero, Guide, LegalDoc, …)
    ├── components/ui/            # reusable pieces (Nav, Footer, TmuxBar, CodeBlock, OsTabs, LangSuggest, …)
    ├── content/                  # words: guide.{es,en}.tsx, legal/*.tsx, guideSections.ts
    └── hooks/                    # preferences (theme/language), OS detection, Konami
```

The guide markup lives once in `components/sections/Guide.tsx`; `content/guide.es.tsx` and
`content/guide.en.tsx` hold only the words, with the same keys (enforced by the
`GuideContent` type), so both languages cannot drift structurally.

Multi-page Vite build (`index.html` + `guia.html` + `terminos.html`) served from the
root of **https://kongtrol.app** on Cloudflare Pages (`base: '/'`). Pages maps clean
URLs (`/guia` → `guia.html`, `/en/guide` → `en/guide.html`) and serves `public/404.html` for unknown paths on its own,
and applies the security headers in `public/_headers` (see `docs/deploy-cloudflare.md`). Cloudflare project settings: root directory
`landing`, build command `pnpm run build`, output directory `dist`. The build reads
`../assets/brand` (the repo-wide brand source); Cloudflare clones the whole repository and runs
the build command inside the root directory, so the parent folder is available.

The old GitHub Pages address (`derotluna.github.io/vpn-kongtrol/`) only serves redirect
stubs to kongtrol.app, deployed by `.github/workflows/landing.yml`.

## SEO

- `src/seo.ts` is the single source: per page and language it emits title, description,
  canonical, `hreflang` es/en/x-default (x-default → English), Open Graph/Twitter cards and
  JSON-LD (`SoftwareApplication` on home, `TechArticle` on the guide, `WebPage` on legal pages).
- `sitemap.xml` is generated at build time with every URL and its `xhtml:link` alternates
  (https://www.sitemaps.org/protocol.html,
  https://developers.google.com/search/docs/specialty/international/localized-versions).
- `public/robots.txt` allows everything and points at the sitemap; `public/llms.txt` is the
  plain-text summary for LLM crawlers.
- Fonts are self-hosted in `public/fonts/` (no request to Google).

The canonical host is `https://kongtrol.app/`. If it ever changes, update `SITE_URL` in
`src/site.ts`, `robots.txt`, `llms.txt`, `.well-known/security.txt` and the redirect stubs in
`.github/workflows/landing.yml` together.

## Downloads

Binaries are **no longer served by the landing site**. They are published as assets on
GitHub Releases:

- https://github.com/DerotLuna/vpn-kongtrol/releases

Expected release assets: the 5 platform binaries + `checksums.txt`
(produced by `make build-all-cli`).

## Easter egg

Konami code (`↑↑↓↓←→←→BA`) toggles green-phosphor CRT mode.
