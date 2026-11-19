<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0047: The product name is "Kongtrol"; solid mark for small sizes, line art for large

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Two names were in use: "VPN Kongtrol" in titles, legal pages, the full logo and the tray, and
"Kongtrol" in the UI and the binary `kongtrol` (`PRODUCT.md` listed it as undecided). Separately,
the landing logos were small (header 30 px, footer 64 px) and the favicon used the line-art mark,
whose concentric lines blur into grey at tab size (the same problem ADR-0044 solved for the tray).
Research: [2026-11-19-product-name-and-logo-sizes](../research/2026-11-19-product-name-and-logo-sizes.md).

## Decision Drivers

- One name everywhere a person reads it; "VPN" still findable in search and taglines.
- Never break installs or data: keychain entries, upgrades, URLs and package names stay.
- Brand files stay derived from the masters in `assets/brand/source/` (ADR-0013, ADR-0044).

## Considered Options

1. Keep both names.
2. "VPN Kongtrol" everywhere.
3. **"Kongtrol" everywhere a person reads it, "VPN" only in the tagline.**

## Decision Outcome

Chosen: **option 3** (owner decision, 2026-11-19).

1. **Name.** The product is "Kongtrol". Tagline: ES "Kongtrol · tus VPN bajo control", EN
   "Kongtrol · your VPNs under control" (landing home title, tray tooltip, README). The landing
   JSON-LD keeps "VPN Kongtrol" only in `alternateName`; the terms page says "formerly VPN
   Kongtrol" once. Display names change: Inno `AppPublisher`, GoReleaser `vendor`/`maintainer`,
   tray title and notifications, docs, `llms.txt`, `security.txt`, 404 page.
2. **Not renamed:** GitHub repo `DerotLuna/vpn-kongtrol` and its URLs, Go module path
   `github.com/vpn-kongtrol/kongtrol`, keychain service `vpn-kongtrol`, package / project names
   (`.goreleaser.yaml` `project_name`, npm, Docker), file paths, Inno `AppId`, ADR bodies.
3. **Lockup.** `assets/brand/source/logo.svg` drops the "VPN" glyphs; "Kongtrol" moves up into
   their place and the lockup is recentered (the glyph paths are moved, not redrawn).
   `logo-512.png` is now rasterized from that vector by `generate.go` (`renderLockup`), so the
   raster `source/logo.png` is gone.
4. **Size rule.** Up to 32 px the solid silhouette (`source/logo-kong-solid.svg`, ADR-0044); from
   36 px up the line-art original. Hence `favicon.svg` = the solid mark, transparent, black or
   white by `prefers-color-scheme`, and `logo-kong-32.png` (PNG favicon fallback, name kept for
   old links) = the solid mark on white, both written by `generate.go` (`renderFavicons`; the
   hand-kept `source/favicon.svg` is gone). The panel's 32 px header mark and the 28 px tray
   flyout mark use the solid shape too.
5. **Landing sizes.** Header logo 40 px (36 px under 960 px), footer lockup 96 px, both `<img>`
   with `width`/`height`.

### Consequences

- Good: one name; search keeps the old one as an alias; no data or upgrade path changes.
- Good: the favicon and small marks read as the head instead of a grey disc.
- Bad: the repo, module and keychain still say `vpn-kongtrol`, which readers of the code see.
- Neutral: the PNG favicon fallback cannot switch with the browser theme (white square on dark).

### Confirmation

- `go run -tags brandgen assets/brand/generate.go` regenerates `favicon.svg`,
  `logo-kong-32.png`, `logo.svg`, `logo-512.png`; rendered and checked on 2026-11-19.
- Landing build, hydration check, tests and lint; web/ui lint, tests and build; Go build, vet
  and tests (i18n parity) — see the commit.

## More Information

- Code: `assets/brand/generate.go`, `landing/src/seo.ts`, `landing/src/index.css`,
  `landing/src/components/ui/Nav.tsx`, `Footer.tsx`, `web/ui/src/components/ui/Logo.tsx`,
  `web/ui/src/pages/TrayPage.tsx`, `internal/i18n/i18n.go` (`tray.tooltip`).
- Docs: `PRODUCT.md` (name decided), `DESIGN.md` ("Logo (name and sizes)").
