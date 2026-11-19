<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Delivering the one-line install scripts

Version 1.0.0 · 2026-11-18

Used by: [ADR-0035](../adr/0035-install-scripts-pinned-and-release-assets.md). All sources
accessed 2026-11-18.

## Findings

- **GitHub "latest" download alias.** `https://github.com/<owner>/<repo>/releases/latest/download/<asset>`
  redirects to the asset of the most recent non-prerelease, non-draft release.
  https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases
  Checked live: `releases/latest` → `tag/v0.5.0`; v0.5.0 assets answer 200; `install.sh` is
  not an asset yet (404).
- **GoReleaser `release.extra_files`.** List of `glob` (+ optional `name_template` when the
  glob matches one file); the uploaded name is the path's base.
  https://goreleaser.com/customization/release/
- **GoReleaser `checksum.extra_files`.** Same syntax. The checksum page says only `checksum.ids`
  is included automatically (binaries, archives, Linux packages, source archives); it does not
  say release extra files are, so they are listed explicitly.
  https://goreleaser.com/customization/package/checksum/
- **Cloudflare Workers static assets `_redirects`.** Plain-text file in the assets directory
  (Vite copies `public/` to `dist/`), not served as an asset; one rule per line
  `source destination [code]`; default 302; 301/302/303/307/308; absolute destinations allowed
  (only 200 proxying must be relative); 2,000 static + 100 dynamic rules; top-most match wins;
  not applied to requests handled by Worker code (this site has none).
  https://developers.cloudflare.com/workers/static-assets/redirects/
- **Same pattern elsewhere.** Projects serving installers from their own domain redirect to
  the release asset (e.g. `https://get.docker.com`, `https://sh.rustup.rs`): reasoned
  comparison, their redirect mechanics **no verificado**.

## Why 302, not 301

The target moves with every release; a 301 may be cached by clients indefinitely
(RFC 9110 §15.4.2: https://www.rfc-editor.org/rfc/rfc9110#section-15.4.2), a 302 is not
cached unless explicitly marked (§15.4.3).
