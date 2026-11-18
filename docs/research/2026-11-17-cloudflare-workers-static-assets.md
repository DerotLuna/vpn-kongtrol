<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: Cloudflare Workers static assets for the landing

Version 1.0.0 · 2026-11-17

## Question

The Cloudflare dashboard now labels the Pages Git flow "legacy Pages workflow" and offers
Workers first. Can the static landing (`landing/dist`: multi-page HTML, clean URLs, custom
`404.html`, `_headers` with a strict CSP) run as a Worker with static assets, built from this
monorepo on every push to `main`, without an API token stored in GitHub?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Workers — Migrate from Pages | https://developers.cloudflare.com/workers/static-assets/migration-guides/migrate-from-pages/ | 2026-11-17 |
| 2 | Workers — Static Site Generation and custom 404 pages | https://developers.cloudflare.com/workers/static-assets/routing/static-site-generation/ | 2026-11-17 |
| 3 | Workers — HTML handling | https://developers.cloudflare.com/workers/static-assets/routing/advanced/html-handling/ | 2026-11-17 |
| 4 | Workers — Static assets headers (`_headers`) | https://developers.cloudflare.com/workers/static-assets/headers/ | 2026-11-17 |
| 5 | Workers Builds — Configuration | https://developers.cloudflare.com/workers/ci-cd/builds/configuration/ | 2026-11-17 |
| 6 | Workers Builds — Monorepos | https://developers.cloudflare.com/workers/ci-cd/builds/advanced-setups/ | 2026-11-17 |
| 7 | Workers Builds — Build watch paths | https://developers.cloudflare.com/workers/ci-cd/builds/build-watch-paths/ | 2026-11-17 |
| 8 | Wrangler configuration — `assets` | https://developers.cloudflare.com/workers/wrangler/configuration/#assets | 2026-11-17 |
| 9 | Single Redirects | https://developers.cloudflare.com/rules/url-forwarding/single-redirects/ | 2026-11-17 |

## Findings

- Static assets with no Worker script are served without invoking a Worker; such requests are
  not billable [2]. `_headers` and `_redirects` are supported natively, from the assets
  directory, exactly as on Pages [1][4].
- Pages detected SPA vs 404 behavior on its own; Workers makes it explicit:
  `not_found_handling: "404-page"` serves the nearest `404.html` with status 404 [1][2].
- `html_handling: "auto-trailing-slash"` (default): `/file` → `file.html` (200),
  `/file.html` → 307 `/file`; `/folder/` → `folder/index.html` [3]. So `/guia` and `/en/` work
  like on Pages.
- Workers Builds: root directory = folder with `wrangler.jsonc` (monorepo); deploy command
  defaults to `npx wrangler deploy`; Cloudflare generates the API token itself, nothing is
  stored in GitHub [5][6]. Build watch paths include/exclude per project [7].
- `routes: [{ pattern, custom_domain: true }]` attaches a custom domain from the config [8].
- A domain that only redirects needs a proxied DNS record plus a Single Redirect rule [9].

## Verified locally

`pnpm run build` then `npx wrangler@4.164.0 deploy --dry-run` in `landing/`: "Read 50 files
from the assets directory", no bindings. Live behavior (clean URLs, 404 status, CSP headers)
is checked with `curl` after the first deploy.

Side finding: with `core.autocrlf=true` (Windows) `public/404.html` is checked out with CRLF,
which changes its inline-script SHA-256 and fails the CSP check in `scripts/prerender.mjs`.
`.gitattributes` now forces LF for `landing/public/*.html`.
