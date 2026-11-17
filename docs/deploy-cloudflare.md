<img src="../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Deploying the landing to Cloudflare Pages

Version 1.0.0 · 2026-11-17

How `landing/` is published at **https://kongtrol.app**, plus the free security settings for the
zone. Everything here is on the Cloudflare Free plan.

## 1. Pages project

Dashboard: **Workers & Pages → Create → Pages → Connect to Git** → select this repository.
Docs: https://developers.cloudflare.com/pages/get-started/git-integration/

| Setting | Value |
|---|---|
| Production branch | `main` |
| Framework preset | None |
| Root directory | `landing` |
| Build command | `pnpm run build` |
| Build output directory | `dist` |
| Environment variable | `NODE_VERSION` = `20` (same as `.github/workflows/landing.yml`) (Production and Preview) |

- pnpm is detected from `landing/pnpm-lock.yaml` and the `packageManager` field
  (https://developers.cloudflare.com/pages/configuration/build-image/).
- The build reads `../assets/brand` (copied by `scripts/copy-brand.mjs`), which works because Pages
  clones the whole repository.
- **Build watch paths** (Settings → Builds → Build watch paths): include `landing/**`, `docs/cli.json` and
  `assets/brand/**`, so Go-only commits do not trigger a deploy
  (https://developers.cloudflare.com/pages/configuration/build-watch-paths/).

Pages serves clean URLs on its own (`/guia` → `guia.html`, `/en/guide` → `en/guide.html`) and
`dist/404.html` for unknown paths (https://developers.cloudflare.com/pages/configuration/serving-pages/).
It also reads `dist/_headers` (from `landing/public/_headers`), see section 4.

## 2. Custom domain `kongtrol.app`

1. Add `kongtrol.app` as a zone in Cloudflare and point the registrar's nameservers to Cloudflare.
2. Pages project → **Custom domains → Set up a custom domain** → `kongtrol.app` (Cloudflare creates
   the CNAME). Optionally add `www.kongtrol.app` and redirect it like `kongtrol.dev` below.
   Docs: https://developers.cloudflare.com/pages/configuration/custom-domains/
3. Keep `*.pages.dev` out of search: the canonical tags already point at `kongtrol.app`.

## 3. `kongtrol.dev` → `kongtrol.app` (301)

1. Add `kongtrol.dev` as its own zone (nameservers to Cloudflare).
2. DNS: proxied (orange cloud) `A @ 192.0.2.1` and `AAAA @ 100::` (placeholder targets; the rule
   answers before any origin is contacted), plus the same for `www` if wanted.
3. **Rules → Redirect Rules → Create rule** (Single Redirect):
   - When: *All incoming requests*
   - Then: Dynamic, expression `concat("https://kongtrol.app", http.request.uri.path)`,
     status `301`, preserve query string: on.
   Docs: https://developers.cloudflare.com/rules/url-forwarding/single-redirects/

Path-preserving means `kongtrol.dev/guia` lands on `kongtrol.app/guia`. Never link to `kongtrol.dev`.

## 4. Security settings for the zone (free)

Apply on the `kongtrol.app` zone (and the HTTPS ones on `kongtrol.dev` too).

| Setting | Where | Value | Why |
|---|---|---|---|
| SSL/TLS mode | SSL/TLS → Overview | Full (strict) — Pages is automatic | Encrypted and validated end to end. https://developers.cloudflare.com/ssl/origin-configuration/ssl-modes/ |
| Always Use HTTPS | SSL/TLS → Edge Certificates | On | Redirects every `http://` request to `https://`. https://developers.cloudflare.com/ssl/edge-certificates/additional-options/always-use-https/ |
| HSTS | SSL/TLS → Edge Certificates | On, max-age 12 months, include subdomains, preload on, No-Sniff on | Browsers refuse plain HTTP after the first visit. Note: `.app` and `.dev` are whole-TLD entries on the HSTS preload list, so browsers already force HTTPS for both domains before any visit; the header still covers non-browser clients and documents intent. https://developers.cloudflare.com/ssl/edge-certificates/additional-options/http-strict-transport-security/ |
| Minimum TLS version | SSL/TLS → Edge Certificates | TLS 1.2 | TLS 1.0/1.1 are deprecated (RFC 8996). https://developers.cloudflare.com/ssl/edge-certificates/additional-options/minimum-tls/ |
| TLS 1.3 | SSL/TLS → Edge Certificates | On | Faster handshake, only modern ciphers. https://developers.cloudflare.com/ssl/edge-certificates/additional-options/tls-13/ |
| Automatic HTTPS Rewrites | SSL/TLS → Edge Certificates | On | Rewrites `http://` subresource links to `https://` to avoid mixed content (safety net; the site loads only same-origin assets). https://developers.cloudflare.com/ssl/edge-certificates/additional-options/automatic-https-rewrites/ |
| Bot Fight Mode | Security → Bots | On | Challenges known bad bots for free. It does not block verified search crawlers. https://developers.cloudflare.com/bots/get-started/bot-fight-mode/ |
| DDoS protection | automatic | Nothing to do | L3/4 and HTTP DDoS mitigation is always on, on every plan. https://developers.cloudflare.com/ddos-protection/ |
| WAF managed rules (Free) | Security → WAF → Managed rules | Cloudflare Free Managed Ruleset: enabled (default) | Blocks high-impact, widely exploited vulnerabilities at the edge. https://developers.cloudflare.com/waf/managed-rules/ |

### Security headers (`landing/public/_headers`)

Pages applies `_headers` to every response (https://developers.cloudflare.com/pages/configuration/headers/).
The file sets:

| Header | Value (summary) | Why |
|---|---|---|
| `Content-Security-Policy` | `default-src 'self'`; scripts only from `'self'` + 2 SHA-256 hashes; `style-src 'self' 'unsafe-inline'`; `img-src 'self' data:`; `font-src 'self'`; `object-src 'none'`; `base-uri 'self'`; `form-action 'none'`; `frame-ancestors 'none'`; `upgrade-insecure-requests` | Matches what the site loads: self-hosted fonts, no third parties. Injected scripts cannot run. https://developer.mozilla.org/docs/Web/HTTP/CSP |
| `X-Content-Type-Options` | `nosniff` | Stops MIME sniffing. |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Sends only the origin to other sites. |
| `Permissions-Policy` | camera, microphone, geolocation, payment, usb off | The site uses none of them. |
| `X-Frame-Options` | `DENY` | Clickjacking protection for old browsers (`frame-ancestors` covers modern ones). |
| `Cross-Origin-Opener-Policy` | `same-origin` | Isolates the browsing context from cross-origin popups. |

CSP details:

- The only inline scripts are `THEME_BOOT` (`landing/src/seo.ts`, avoids a theme flash before first
  paint) and the script in `landing/public/404.html`. Both are allowed by hash. If you edit either,
  the build fails (`scripts/prerender.mjs` checks every inline script in `dist/` against
  `_headers`) and prints the new hash to paste.
- JSON-LD blocks (`type="application/ld+json"`) are data, not executed, so CSP does not apply.
- `style-src` keeps `'unsafe-inline'`: React `style={{}}` props are prerendered as `style=""`
  attributes, and `404.html` has a `<style>` block. Inline styles cannot execute code.
- Adding any third-party resource (analytics, CDN, embed) requires updating the CSP first.

How it was verified: `pnpm run build`, then `dist/` served with these headers and every ES/EN route
loaded in headless Chrome: zero CSP console messages. A control run with one hash removed did
report `Executing inline script violates the following Content Security Policy directive`.

## 5. Web Analytics (optional, not enabled)

Cloudflare Web Analytics is cookieless and does not fingerprint visitors
(https://developers.cloudflare.com/web-analytics/about/). It is **not** enabled in the code.
If you enable it:

1. Prefer the automatic setup for Pages projects (Pages → Metrics → Web Analytics), which injects
   the beacon at the edge. https://developers.cloudflare.com/pages/how-to/web-analytics/
2. Allow it in the CSP: `script-src` + `https://static.cloudflareinsights.com`, `connect-src` +
   `https://cloudflareinsights.com`.
3. Update the privacy policy at `/privacidad` and `/en/privacy` (`landing/src/content/legal/privacidad.tsx`):
   today it says "no analytics", which would no longer be true.
