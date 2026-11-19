<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: download buttons, button hierarchy and phone visitors on the landing

Version 1.0.0 · 2026-11-18

## Question

How should the landing present "download" so a non-technical visitor gets the right file for
their computer: how many primary buttons, how to recommend the visitor's OS without a hydration
mismatch, how to tell phones/tablets apart (Kongtrol is a desktop app), and how to send the page
to a computer from a phone?

## Sources

All accessed 2026-11-18.

- Apple HIG, Buttons — https://developer.apple.com/design/human-interface-guidelines/buttons
  (read via its JSON data feed): "use a button that has a prominent visual style for the most
  likely action in a view … Keep the number of prominent buttons to one or two per view";
  "a button needs a hit region of at least 44x44 pt".
- Android Developers, Compose Material 3 buttons —
  https://developer.android.com/develop/ui/compose/components/button: Filled = "High-emphasis
  buttons … for primary actions"; Outlined = "Medium-emphasis buttons … important but not
  primary … alternative, secondary actions"; Text = low emphasis. Links to
  https://m3.material.io/components/buttons/overview (the m3.material.io page itself renders
  client-side and could not be read: **no verificado** beyond the Android summary).
- MDN, `NavigatorUAData.mobile` — https://developer.mozilla.org/en-US/docs/Web/API/NavigatorUAData/mobile:
  "`true` if this is a mobile device"; "not Baseline because it does not work in some of the most
  widely-used browsers" (Chromium only; Firefox and Safari lack `userAgentData`).
- MDN, `Navigator.share()` — https://developer.mozilla.org/en-US/docs/Web/API/Navigator/share and
  `Clipboard.writeText()` — https://developer.mozilla.org/en-US/docs/Web/API/Clipboard/writeText.
- Apple Support 116943, "Mac computers with Apple silicon" — https://support.apple.com/en-us/116943:
  open About This Mac from the Apple menu; Apple silicon shows "Chip", an Intel Mac shows
  "Processor".
- React, `useSyncExternalStore` server rendering —
  https://react.dev/reference/react/useSyncExternalStore#adding-support-for-server-rendering
  (already the basis of ADR-0026).
- GitHub release v0.5.0 assets (`gh release view`) and `.goreleaser.yaml` (`draft: false`,
  archives id `cli-binary`, name `kongtrol-{os}-{arch}`).

## Findings

- **One filled button per view.** HIG (one or two prominent) and Material 3 (filled = primary,
  outlined = secondary) agree. Hero: filled "Descargar para <sistema>" + outlined "Ver en
  GitHub". Download section: one filled button in the recommended card, outlined everywhere else.
  Targets ≥44 px (HIG); the header's 38 px boxes get a 44 px hit region with an inset `::before`.
- **The OS label must wait for hydration.** The prerender can't know the visitor, so the server
  snapshot is "unknown" (`null` → "Descargar gratis") and the client re-renders with the detected
  OS; same for "is mobile" and "can share" (`src/hooks/useVisitor.ts`). The download section
  renders every system as equal cards until then (no false "recommended" in the HTML).
- **Phone/tablet heuristic** (no single API covers all browsers): `userAgentData.mobile`
  (Chromium phones; Android *tablets* report `false`) OR UA contains Android/iPhone/iPad/iPod OR
  a "Macintosh" UA with `maxTouchPoints > 1` (iPadOS 13+ Safari sends a desktop Mac UA by default;
  no Mac has a touchscreen). Known miss: a phone in "desktop site" mode — it just sees the desktop
  buttons. Reasoned from the sources above; the iPadOS desktop-UA default and the Mac touch
  argument are widely documented but **no verificado** against an Apple primary source here.
- **Send to computer.** `navigator.share({ title, url })` opens the system share sheet (AirDrop,
  Nearby Share, mail, chat). It rejects with `AbortError` when the user closes the sheet (do
  nothing); any other failure, or no `share`, falls back to `clipboard.writeText` with a visible
  confirmation in a `role="status"` region (announced politely).
- **Apple Silicon vs Intel** can't be told reliably from the browser (Safari freezes the UA at
  "Intel Mac OS X"; only Chromium's high-entropy `architecture` hint exposes it). So the Mac card
  offers both and explains the About This Mac check.
- **Scroll, don't download, from the hero** until the release ships an installer (issue #32):
  the current assets are raw CLI binaries, and the section explains the unsigned-binary warning
  before the visitor downloads.
- **Header button reveal.** An `IntersectionObserver` on the hero's button wrapper (rootMargin
  minus the sticky header height) shows the header button only once the hero's has scrolled away.
  Hidden = `visibility: hidden` (not focusable, not announced) while keeping its box (no layout
  shift); reduced motion → no transition.

## Confidence

High for the button hierarchy, Web Share/Clipboard behaviour and the Mac check (primary docs).
Medium for the phone/tablet heuristic (combination reasoned from MDN; edge cases above).

## Open questions

- When the Windows installer ships (issue #32), the hero button can link the file directly
  (`RELEASE_ASSETS` in `landing/src/links.ts` is the one place to change names).

## Cited by

- [ADR-0027](../adr/0027-landing-download-ctas.md)
