<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: Cloudflare Pages build configuration and security headers

Version 1.0.0 · 2026-11-17

## Question

How to build only `landing/` from a monorepo on Cloudflare Pages, and how to set security
headers and redirects.

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Cloudflare Pages — Build configuration | https://developers.cloudflare.com/pages/configuration/build-configuration/ | 2026-11-17 |
| 2 | Cloudflare Pages — Headers | https://developers.cloudflare.com/pages/configuration/headers/ | 2026-11-17 |
| 3 | Cloudflare Pages — Redirects | https://developers.cloudflare.com/pages/configuration/redirects/ | 2026-11-17 |
| 4 | Cloudflare Pages — Build watch paths | https://developers.cloudflare.com/pages/configuration/build-watch-paths/ | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- [1]: "The root directory is where your site's content lives"; the build output directory is
  where the build command outputs the built site.
- [2]: `_headers` file; "You may define up to 100 header rules"; "Each line in the `_headers`
  file has a 2,000 character limit."
- [3]: `_redirects` "limited to 2,000 static redirects and 100 dynamic redirects"; 1,000
  characters per line.
- [4]: "Paths satisfying excludes conditions are ignored first"; `*` "matches zero or more
  characters, including path separators (/)".

## Applied (see `docs/deploy-cloudflare.md`, `landing/public/_headers`)

- Root `landing`, build `pnpm run build`, output `dist`; watch paths include `landing/**`.
- `_headers`: strict CSP (`default-src 'self'`, hashed inline scripts), `X-Frame-Options: DENY`,
  `nosniff`, `Referrer-Policy`, `Permissions-Policy`, COOP. One rule, well under the limits.
- `kongtrol.dev` → `kongtrol.app` via a zone-level Single Redirect (path-preserving 301), not
  `_redirects` (different domain).

## Confidence

High (Cloudflare docs).

## Open questions

- None.

## Used by

- [ADR-0012](../adr/0012-landing-cloudflare-pages-multilingual.md), [ADR-0015](../adr/0015-ci-scoped-builds-generated-docs.md)
