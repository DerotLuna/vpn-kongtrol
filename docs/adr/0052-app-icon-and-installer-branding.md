<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0052: App icon on an Ink plate, icon in both Windows binaries, branded installer

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner; the visual
choices below were delegated to the implementer and are recorded with their reasons)

## Context and Problem Statement

The app icon was the line-art mark on a white square (`assets/logo.ico`, `logo-kong-512.png`):
hand-assembled, not generated, and its concentric lines turn grey at taskbar size. Only
`kongtrol-tray.exe` carried it; `kongtrol.exe` showed Windows' generic program icon in
Explorer. The installer used Inno Setup's stock box image, and on the Ready page of an update the
description was cut off ("El ayudante de red solo se actualiza si hace…"): the script replaced
`ReadyLabel`'s caption with a longer text without growing the label.
Research: [2026-11-19-app-icon](../research/2026-11-19-app-icon.md).

## Decision Drivers

- Recognizable next to other apps in the taskbar, Start, Dock and Explorer, on light and dark.
- Platform guidance: Apple's icon grid, Fluent's 48 px grid and 3:1 contrast on both themes.
- Every brand raster stays generated from the SVG masters (ADR-0013, ADR-0044); no hand-drawn
  bitmaps, nothing stale.
- Text in the installer follows its language (ES/EN, ADR-0049).
- Keep the pure-Go cross build (`CGO_ENABLED=0`) and the tool the repo already uses.

## Considered Options

1. Keep the line-art icon on white.
2. **Ink squircle plate with the solid mark in Bone; plain mark at the smallest sizes.**
3. Plateless mark at every size.

## Decision Outcome

Option 2 (owner-approved direction, 2026-11-19).

1. **Icon.** Plate in Ink (`text-light` `#1b1813`), mark = `source/logo-kong-solid.svg` in Bone
   Paper (`bg-light` `#f1ede4`), a superellipse (n = 5) plate. Drawn by `generate.go`
   (`renderAppIcons`), vector form `source/app-icon.svg`. **Exception:** this is the only logo
   with a background; every other logo stays transparent (DESIGN.md "Logo").
2. **Geometry per platform.** macOS: plate 824/1024 px centered, the transparent margin is part
   of the icon (Apple grid) → `app-icon-1024.png`, turned into the `.icns` by Apple's own `sips` +
   `iconutil` in `packaging/macos/build-app.sh`. Windows and Linux: plate 44/48 of the canvas,
   mark 68 % of the plate → `assets/logo.ico`, `app-icon-512.png` / `app-icon-256.png`
   (hicolor 512 and 256 in `.deb`/`.rpm`; the archives' `kongtrol.png`, which `install.sh`, AUR
   and Homebrew install). Landing touch icon: full-bleed square without transparency (iOS masks
   it) → `app-icon-180.png`, served as `/apple-touch-icon.png`; `logo-kong-192.png`, used only
   for that, is deleted.
3. **Small sizes (choice, rendered and compared).** `.ico` frames 16, 20, 24, 32, 40, 48, 64,
   256. Up to 24 px the frame is the solid mark **without plate** in mid-tone `#857c6b`;
   from 32 px it has the plate. Reasons: on a 16–24 px plate the head is 10–15 px and the face
   blurs, while the plain mark fills the frame; `#857c6b` (relative luminance ≈ 0.20) keeps
   ≈ 4:1 on a light and ≈ 3.9:1 on a dark taskbar, meeting Fluent's 3:1 on both themes, where
   an Ink plate alone disappears on a dark taskbar. Windows uses 16–24 px for title bars,
   context menus and the 100 % taskbar (same family as the tray glyph, ADR-0044); Start pins at
   100 % (32 px) and the taskbar from 125 % (30 → 32 px) get the plate. The cut is 24/32
   rather than 32/40 so the taskbar at the common 125 % shows the full app icon.
4. **Both Windows binaries.** `akavel/rsrc@v0.10.2` (already used for the tray) now also writes
   `cmd/kongtrol/rsrc_windows_<arch>.syso` from `assets/logo.ico`, at the same three places as
   the tray (`make build` / `build-tray-native` on Windows, `release.yml`, `.goreleaser.yaml`
   hooks for amd64 + arm64); the `.syso` stay gitignored, generated before each build. Pure Go,
   so `CGO_ENABLED=0` cross builds keep working. No version info: `rsrc` only embeds icons and
   manifests; switching tools (`go-winres`) for version info is left out (not cheap: a second
   tool plus a manifest change for the tray).
5. **Installer.** `WizardSmallImageFile` = the app icon at 58/77/97/124/159 px and
   `WizardImageFile` = the lockup (line-art mark large, wordmark under it) in Bone on Ink at
   202×386, 269×515, 336×643, 430×824, 534×1022: Inno Setup 6.6+ image areas at
   100/125/150/200/250 % (Setup picks the closest, no blur), PNG, from `generate.go`
   (`renderInstallerImages`, `assets/brand/installer/`).
   - **No words in the image (choice).** One image cannot follow the installer's language, so the
     tagline is text on the finished page: `FinishedHeadingLabel` "Kongtrol está listo" /
     "Kongtrol is ready" and `FinishedLabel` starting with "Todas tus VPN bajo Kongtrol." /
     "All your VPNs under Kongtrol." (ADR-0048). `ClickFinish` (ES) uses "tú" like the rest.
   - **Welcome page stays off (choice).** Inno's default (`DisableWelcomePage=yes`) avoids a
     page with nothing to decide; the side panel shows on the finished page.
6. **Cut-off text fixed.** `FitLabel` in `[Code]` grows a label whose caption the script replaces
   (`TWizardForm.AdjustLabelHeight`) and moves the control below it: the Ready memo shrinks
   (`IncTopDecHeight`), the finished page's checkbox list moves down. Applied to `ReadyLabel`
   and `FinishedLabel`. Header texts that cannot grow are shortened: `ReadyKeep` fits two lines,
   `TitleDowngrade` one ("Volver a una versión anterior (%1 → %2)").

This refines ADR-0044 point 2 and ADR-0047 point 4 only where they named the app icons as line
art; both stay Accepted.

### Consequences

- Good: one recognizable icon on every platform and in Explorer for both exes and the setup;
  the installer looks like Kongtrol; no truncated text on update pages.
- Neutral: 15 generated files more in `assets/brand/` (≈ 450 KB with `logo.ico`, mostly the wizard panels);
  a change to them also triggers the landing build (`assets/brand/**`), as any brand change does.
- Bad: the icon style changes between 24 and 32 px on Windows (plain mark vs plate), as many
  Windows apps simplify their small icons.
- Bad / open: macOS 26 prefers layered Icon Composer icons; how it shows this classic `.icns`
  (own plate inside Apple's margin) is not verified. Moving to Icon Composer needs Xcode's
  `actool` in the macOS job; a later decision if the Dock shows the icon inside a system plate.

### Confirmation

- `go run -tags brandgen assets/brand/generate.go` twice: identical output.
- Windows build with both `.syso`; `ExtractAssociatedIcon` shows the icon for `kongtrol.exe`,
  `kongtrol-tray.exe` and the setup.
- Inno Setup 6.7.4 (the CI version) compiled the script locally without warnings; the update
  Ready page (ES and EN) and a no-op copy of the finished page were captured: see the preview
  linked in the commit.

## More Information

- Code: `assets/brand/generate.go`, `packaging/windows/kongtrol.iss`,
  `packaging/macos/build-app.sh`, `.goreleaser.yaml`, `.github/workflows/release.yml`,
  `Makefile`, `landing/scripts/copy-brand.mjs`, `landing/src/seo.ts`.
- Docs: `DESIGN.md` ("Logo"), `AGENTS.md` (assets row), `docs/ARCHITECTURE.md`.
- Refines: [ADR-0044](0044-tray-glyph-relleno.md) point 2,
  [ADR-0047](0047-product-name-kongtrol-and-logo-sizes.md) point 4.
