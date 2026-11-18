<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0035: Install scripts pinned to the published tag, then served from the latest release

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

The landing, the guide and the README told people to run
`curl -fsSL https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.sh | sh`
(and `install.ps1` on Windows). The scripts on `main` already expect the next release's assets
(`Kongtrol-macos-universal.zip`, `kongtrol-tray`, `kongtrol.desktop`), while
`releases/latest` is still v0.5.0: the one-liner failed. A script from a branch and binaries
from a release drift apart by design. How do we keep script and assets in step?

## Decision Drivers

- The one-liner must work today and keep working after every release, with no manual sync.
- One place to change the URL (landing rule: external URLs only in `landing/src/links.ts`).
- Verifiable: the script people pipe to a shell should be checksummed with the release.
- Use what the platforms already offer (GitHub "latest" download alias, Cloudflare redirects).

## Considered Options

1. Keep `main` — breaks whenever `main` is ahead of the latest release (now).
2. Pin to a tag in the landing — works, but needs a manual edit on every release.
3. **Pin to the published tag now; from v0.6.0 on publish the scripts as release assets and
   serve `https://kongtrol.app/install.sh` as a 302 to
   `releases/latest/download/install.sh`.**

## Decision Outcome

Chosen: **option 3** (sources in the
[research note](../research/2026-11-18-install-script-delivery.md)).

- `landing/src/links.ts`: `INSTALL_SCRIPT_BASE` is the single base for the install/update
  one-liners (`INSTALL_CMD`), pinned to `https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/v0.5.0/`.
  The v0.5.0 scripts download `kongtrol-{os}-{arch}` + `checksums.txt` from
  `releases/latest/download/`, which exist in v0.5.0 (verified 2026-11-18, HTTP 200).
- Uninstall through the script (`--uninstall` / `-Uninstall`) only exists on `main`; the
  v0.5.0 script ignores the flag and would reinstall, so those lines keep
  `UNINSTALL_SCRIPT_BASE` (`main`) until the switch.
- `.goreleaser.yaml`: `install.sh` and `install.ps1` go in `release.extra_files` and in
  `checksum.extra_files` (release extra files are not added to `checksums.txt` on their own);
  `release.yml` attests them too.
- `landing/public/_redirects`: `/install.sh` and `/install.ps1` → `releases/latest/download/…`,
  302 (the target changes with each release).
- After v0.6.0 final: switch `INSTALL_SCRIPT_BASE` and `UNINSTALL_SCRIPT_BASE` to
  `https://kongtrol.app/` (checklist in `docs/DEVELOPMENT.md`), and README/SETUP with them.

### Consequences

- Good: script and assets always come from the same release; one constant to change.
- Good: the scripts are in `checksums.txt`, signed with Sigstore like the rest.
- Bad: until v0.6.0, install/update shows the pinned tag in the command (longer URL), and the
  macOS one-liner installs only the CLI (v0.5.0 behavior).
- Neutral: `kongtrol.app/install.sh` returns 404 from GitHub until a release carries the asset;
  nothing links to it before the switch.

### Confirmation

`curl -sI https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/v0.5.0/install.sh` and
`…/install.ps1` → 200; `releases/latest` → `tag/v0.5.0`; `releases/latest/download/`
`kongtrol-linux-amd64`, `kongtrol-darwin-arm64`, `kongtrol-windows-amd64.exe`,
`checksums.txt` → 200; `releases/latest/download/install.sh` → 404 (expected before v0.6.0).
The redirect itself is not verified until the landing is deployed (**no verificado**).

## More Information

Code: `landing/src/links.ts`, `landing/public/_redirects`, `.goreleaser.yaml`,
`.github/workflows/release.yml`, `docs/DEVELOPMENT.md` (release checklist).
