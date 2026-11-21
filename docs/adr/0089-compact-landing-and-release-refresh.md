# ADR-0089: Compact compatibility and refresh the landing on stable releases

Version 1.0.0 · 2026-11-21

Status: Accepted · Date: 2026-11-21

Keep the landing's interactive routing demo. Use explicit playback controls so
someone can recover after selecting a destination, and respect reduced motion.
Show a compact compatibility summary with expandable per-platform details.
Put recurring questions in the guide and link them from home.

Use the existing warm red status color for the author's heart and a restrained
gold-to-cool gradient for the large footer wordmark. Draw the download contours
at full length as soon as the section is visible.

Keep the landing static. Production builds check the current published stable
release first. After publishing a stable release, GitHub Actions can trigger a
fresh Cloudflare Workers build with the `CLOUDFLARE_LANDING_DEPLOY_HOOK` secret.
No request from the visitor's browser to GitHub is needed to show the version.

[Research and deployment notes](../research/2026-11-21-landing-release-refresh.md)

Supersedes the footer heart color in [ADR-0081](0081-author-and-publisher-ruben-luna.md)
and the download stroke animation in [ADR-0070](0070-landing-instrument-terminal-style-contour-edges-and-short-gold-band.md).
