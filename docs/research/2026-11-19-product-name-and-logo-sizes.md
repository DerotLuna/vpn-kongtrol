<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: product name "Kongtrol" and logo sizes

Version 1.0.0 · 2026-11-19

## Question

For ADR-0047: how to drop "VPN" from the product name without losing search continuity or
breaking installs, and which logo shape to use at which size (favicon, header, footer).

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Google Search Central — Provide a site name to Google Search (`WebSite` `name` + `alternateName`) | https://developers.google.com/search/docs/appearance/site-names | 2026-11-19, **no verificado** (no web access in this session) |
| 2 | Schema.org — `alternateName` | https://schema.org/alternateName | 2026-11-19, **no verificado** |
| 3 | Can I use — SVG favicons | https://caniuse.com/link-icon-svg | 2026-11-19, **no verificado** |
| 4 | web.dev — Building an adaptive favicon (`prefers-color-scheme` inside the SVG) | https://web.dev/articles/building/an-adaptive-favicon | 2026-11-19, **no verificado** |
| 5 | Inno Setup Help — `[Setup]: AppId` | https://jrsoftware.org/ishelp/topic_setup_appid.htm | 2026-11-19, **no verificado** |
| 6 | Code in this repo: `assets/brand/generate.go`, `internal/config/keychain.go`, `internal/uninstall/purge.go`, `packaging/windows/kongtrol.iss` | — | 2026-11-19, verified |

## Findings

1. **Name in search.** Google reads the preferred site name from `WebSite` structured data
   `name` and accepts alternates in `alternateName` [1][2]. So the old name stays only as an
   `alternateName` ("VPN Kongtrol"), the visible name is "Kongtrol". Reasoned, not verified live.
2. **What must not be renamed** (verified in code [6]): the keychain service `vpn-kongtrol`
   (existing passwords would become unreadable; `uninstall --purge` deletes exactly those items),
   the Go module path, the GitHub repo and release URLs (the update check and install scripts
   call them), package names, and Inno `AppId` [5] (upgrade/uninstall identity). `AppPublisher`
   is only a display string in Windows "Apps" (reasoned from [5]; not verified).
3. **Favicon.** All current desktop browsers accept an SVG favicon and honor a
   `prefers-color-scheme` media query inside it [3][4]; a 32 px PNG stays as a fallback for the
   rest. At 16–32 px the line-art mark's concentric lines blur into grey (already seen for the tray,
   ADR-0044), so the derived solid silhouette is the better small mark (verified by rendering the
   generated `favicon.svg` and `logo-kong-32.png` on 2026-11-19).
4. **Lockup.** The full logo `logo.svg` had "VPN" set above "Kongtrol". Its potrace master is one
   `<path>` per glyph with an absolute start point, so the three "VPN" glyphs can be dropped and
   "Kongtrol" moved up exactly (same gap as before), recentered on the 1024 canvas — no redraw.
5. **Sizes.** Landing header 30 → 40 px (36 px under 960 px, matching the 37 px header buttons)
   and footer lockup 64 → 96 px; `width`/`height` on the `<img>` avoid layout shift. Visual check
   of the header at 320/375/768/1280 px in this change.

## Conclusion

Rename the visible product to "Kongtrol" (tagline carries "VPN"), keep every technical
identifier, solid silhouette up to 32 px, line art from 36 px up. Decision: ADR-0047.
