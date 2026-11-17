<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0012: Landing on Cloudflare Pages, `kongtrol.app`, ES + EN

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

The public site was a single page on GitHub Pages. It now needs a guide, terms, privacy and
security pages in Spanish (default) and English, a custom domain, security headers and a
`security.txt`, without third-party trackers or fonts.

## Decision Drivers

- Free hosting with custom headers (CSP) and redirects.
- Correct multilingual SEO.
- No third-party requests from the site (consistent with "Local only" spirit).

## Considered Options

1. Stay on GitHub Pages — no custom response headers (**no verificado** in this research; not
   re-checked), one domain.
2. Cloudflare Pages, Git integration, root `landing`.
3. A JS-redirect single page choosing language on load — discouraged by Google.

## Decision Outcome

Chosen: **option 2**.

- Cloudflare Pages: root `landing`, build `pnpm run build`, output `dist`, watch paths
  `landing/**` (`docs/deploy-cloudflare.md`).
- Canonical domain `kongtrol.app`. `kongtrol.dev` → 301 path-preserving Single Redirect; old
  GitHub Pages URL gets redirect stubs (meta refresh + canonical + link) from
  `.github/workflows/landing.yml`. Never link to the old URLs.
- Multi-page Vite + React + TS (10 HTML entries): ES at `/…` (`/guia`, `/terminos`,
  `/privacidad`, `/seguridad`), EN under `/en/` with localized slugs (`/en/guide`, `/en/terms`,
  `/en/privacy`, `/en/security`); `hreflang` on each page listing itself and its twin.
- No automatic language redirect on a guess; a dismissible language bar instead (Google Search
  Central advises against auto-redirecting; verified 2026-11-17, exact wording to re-check).
- Self-hosted fonts (`landing/public/fonts/`: Inter, Space Grotesk, IBM Plex Mono).
- `_headers`: strict CSP (`default-src 'self'`, hashed inline scripts, checked by
  `scripts/prerender.mjs`), `X-Frame-Options: DENY`, `nosniff`, `Referrer-Policy`,
  `Permissions-Policy`, COOP.
- `/.well-known/security.txt` per RFC 9116 ([ADR-0016](0016-legal-contact-model.md)).

### Consequences

- Good: fast static site, no third parties, correct language indexing.
- Bad: Cloudflare account/zone setup lives outside the repo (documented in
  `docs/deploy-cloudflare.md`); two language trees to keep in sync.

### Confirmation

`landing-ci` workflow builds on every landing change; prerender fails on unhashed inline scripts.

## More Information

- Code: `landing/`, `landing/public/_headers`, `landing/src/seo.ts`, `.github/workflows/landing.yml`
- Research: [multilingual SEO](../research/2026-11-17-multilingual-seo.md),
  [Cloudflare Pages](../research/2026-11-17-cloudflare-pages.md),
  [security.txt](../research/2026-11-17-security-txt.md)
