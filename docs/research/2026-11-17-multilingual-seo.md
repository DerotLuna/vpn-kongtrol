<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: multilingual SEO (hreflang, localized URLs, no auto-redirect)

Version 1.0.0 · 2026-11-17

## Question

How should a bilingual (ES default, EN) static site expose its languages to search engines and
visitors?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Google Search Central — Tell Google about localized versions of your page | https://developers.google.com/search/docs/specialty/international/localized-versions | 2026-11-17 |
| 2 | Google Search Central — Managing multi-regional and multilingual sites | https://developers.google.com/search/docs/specialty/international/managing-multi-regional-sites | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- [1]: "Each language version must list itself as well as all other language versions."
  `x-default` matches "any language not explicitly listed by an hreflang tag on the page".
- [2]: "Google recommends using different URLs for each language version of a page". The page
  advises against automatically redirecting users based on perceived language (the fetch tool
  flagged its own wording as possibly paraphrased — re-check exact text before quoting).

## Conclusions (applied in `landing/`)

- One URL per language: Spanish at `/…`, English under `/en/` with localized slugs.
- Every page lists itself and its twin via `hreflang` (+ `x-default`).
- No automatic redirect on a guessed language; a dismissible "View in English / Ver en español"
  bar instead; only a stored explicit pick switches URL (`landing/README.md`).

## Confidence

High (Google's own documentation).

## Open questions

- None.

## Used by

- [ADR-0012](../adr/0012-landing-cloudflare-pages-multilingual.md)
