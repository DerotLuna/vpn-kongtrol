<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0022: Landing on Cloudflare Workers static assets (Workers Builds)

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

Supersedes the hosting part of [ADR-0012](0012-landing-cloudflare-pages-multilingual.md)
(Cloudflare Pages). The rest of ADR-0012 (domain, ES + EN structure, headers) still applies.

## Context and Problem Statement

`kongtrol.app` and `kongtrol.dev` were added to Cloudflare with nothing deployed. When creating
the project, the dashboard offers Workers first and labels Pages the "legacy Pages workflow".
How should `landing/` be hosted and deployed from this public repository?

## Decision Drivers

- No deploy secret in a public GitHub repository.
- Deploy only when the landing changes ("Build only what changed").
- Keep clean URLs, the custom 404 and the CSP in `_headers` unchanged.
- Follow Cloudflare's current, non-legacy path.

## Considered Options

1. Cloudflare Pages, Git integration (ADR-0012).
2. Workers static assets, Workers Builds Git integration, config in `landing/wrangler.jsonc`.
3. Either one deployed from GitHub Actions with `wrangler deploy` and a
   `CLOUDFLARE_API_TOKEN` repo secret.

## Decision Outcome

Chosen: **option 2**.

- `landing/wrangler.jsonc`: assets only (`./dist`), `html_handling: auto-trailing-slash`,
  `not_found_handling: 404-page`, custom domain `kongtrol.app`, `workers_dev` off.
- Workers Builds: root `landing`, build `pnpm run build`, deploy `npx wrangler deploy`,
  production branch `main`, watch paths `landing/*`, `assets/brand/*`, `docs/cli.json`.
  Cloudflare creates and holds the deploy token; the GitHub App gets access to this repo only.
- `www.kongtrol.app` and `kongtrol.dev` (+ `www`): 301 path-preserving Single Redirects to
  `https://kongtrol.app` (`docs/deploy-cloudflare.md`).
- `landing-ci` keeps the build check on PRs; it does not deploy.

### Consequences

- Good: no secret in GitHub; same URLs and headers as Pages; asset-only requests are not
  billable Worker invocations; current Cloudflare product.
- Bad: one more file (`wrangler.jsonc`); the Git connection and redirect rules still live in the
  Cloudflare dashboard (documented in `docs/deploy-cloudflare.md`).
- Option 3 rejected: a long-lived token in a public repo is avoidable risk (anyone with write
  access could exfiltrate it through a workflow change).

### Confirmation

`npx wrangler deploy --dry-run` reads `dist/`; after deploy, `curl -I` on `/`, `/guia`,
`/en/guide`, an unknown path (404) and `kongtrol.dev/guia` (301 → `kongtrol.app/guia`).

## More Information

- Code: `landing/wrangler.jsonc`, `landing/public/_headers`, `.github/workflows/landing.yml`
- Research: [Cloudflare Workers static assets](../research/2026-11-17-cloudflare-workers-static-assets.md)
