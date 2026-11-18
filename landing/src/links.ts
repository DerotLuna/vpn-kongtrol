// Downloads live on GitHub Releases.
export const GITHUB_REPO = 'https://github.com/DerotLuna/vpn-kongtrol'
export const GITHUB_RELEASES = `${GITHUB_REPO}/releases/latest`

// Where the one-line install scripts are fetched from — the ONLY place this is
// written (ADR-0035). Pinned to the published tag: the scripts on `main` expect
// the next release's assets, so serving them while releases/latest is v0.5.0
// breaks the install. The v0.5.0 scripts download releases/latest assets with
// the names below (kongtrol-{os}-{arch}).
// TODO(after v0.6.0 final, docs/DEVELOPMENT.md release checklist): switch to
// 'https://kongtrol.app/' — landing/public/_redirects sends /install.sh and
// /install.ps1 to releases/latest/download/, which ships them from v0.6.0 on.
export const INSTALL_SCRIPT_BASE = 'https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/v0.5.0/'

// Uninstall through the script (--uninstall / -Uninstall) only exists in the
// scripts on `main` (v0.6.0+); the v0.5.0 script ignores the flag and would
// reinstall instead. Same TODO: becomes INSTALL_SCRIPT_BASE after v0.6.0.
export const UNINSTALL_SCRIPT_BASE = 'https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/'

/** Shell one-liners that install (or, with run again, update) Kongtrol. */
export const INSTALL_CMD = {
  unix: `curl -fsSL ${INSTALL_SCRIPT_BASE}install.sh | sh`,
  windows: `iwr ${INSTALL_SCRIPT_BASE}install.ps1 -useb | iex`,
} as const

// Release asset filenames — the ONLY place they are written. When a release
// changes its names (e.g. the Windows installer, issue #32), edit the value
// here and every button, hint and checksum example follows. Names match
// `.goreleaser.yaml` (archives id `cli-binary`: kongtrol-{os}-{arch}) and the
// published v0.5.0 assets.
export const RELEASE_ASSETS = {
  'windows-amd64': 'kongtrol-windows-amd64.exe',
  'windows-arm64': 'kongtrol-windows-arm64.exe',
  'macos-arm64': 'kongtrol-darwin-arm64',
  'macos-amd64': 'kongtrol-darwin-amd64',
  'linux-amd64': 'kongtrol-linux-amd64',
  'linux-arm64': 'kongtrol-linux-arm64',
} as const

export type ReleaseAssetKey = keyof typeof RELEASE_ASSETS

// GitHub's "latest" alias resolves against the most recently published
// (non-draft, non-prerelease) release, so this always serves the current build
// without any API call or CORS issue.
export function releaseAsset(key: ReleaseAssetKey): string {
  return `${GITHUB_REPO}/releases/latest/download/${RELEASE_ASSETS[key]}`
}
