// Downloads live on GitHub Releases.
export const GITHUB_REPO = 'https://github.com/DerotLuna/vpn-kongtrol'
export const GITHUB_RELEASES = `${GITHUB_REPO}/releases/latest`

// Where the one-line install scripts are fetched from — the ONLY place this is
// written (ADR-0035). landing/public/_redirects sends /install.sh and
// /install.ps1 to the copies attached to the latest GitHub release, so the
// script always matches that release's asset names.
export const INSTALL_SCRIPT_BASE = 'https://kongtrol.app/'

// The same scripts uninstall (--uninstall / -Uninstall).
export const UNINSTALL_SCRIPT_BASE = INSTALL_SCRIPT_BASE

/** Shell one-liners that install (or, with run again, update) Kongtrol. */
export const INSTALL_CMD = {
  unix: `curl -fsSL ${INSTALL_SCRIPT_BASE}install.sh | sh`,
  windows: `iwr ${INSTALL_SCRIPT_BASE}install.ps1 -useb | iex`,
} as const

// Release asset filenames — the ONLY place they are written. When a release
// changes its names (e.g. the Windows installer, issue #32), edit the value
// here and every button, hint and checksum example follows. Names match
// `.goreleaser.yaml` (archives id `cli-binary`: kongtrol-{os}-{arch}).
export const RELEASE_ASSETS = {
  'windows-amd64': 'kongtrol-windows-amd64.exe',
  'windows-arm64': 'kongtrol-windows-arm64.exe',
  'macos-arm64': 'kongtrol-darwin-arm64',
  'macos-amd64': 'kongtrol-darwin-amd64',
  'linux-amd64': 'kongtrol-linux-amd64',
  'linux-arm64': 'kongtrol-linux-arm64',
} as const

export type ReleaseAssetKey = keyof typeof RELEASE_ASSETS

// The Windows installer (Inno Setup, per user; upgrades in place keep data,
// ADR-0025) — the ONLY place its name and URL are written. Every "Descargar
// para Windows" button downloads it from the latest release.
export const WINDOWS_INSTALLER_FILE = 'Kongtrol-Setup-windows-amd64.exe'
export const WINDOWS_INSTALLER_URL = `${GITHUB_REPO}/releases/latest/download/${WINDOWS_INSTALLER_FILE}`

// The macOS app (Kongtrol.app: tray + panel + the kongtrol command,
// universal for Apple Silicon and Intel), zipped by release.yml — the ONLY
// place its name and URL are written (ADR-0077).
export const MACOS_APP_FILE = 'Kongtrol-macos-universal.zip'
export const MACOS_APP_URL = `${GITHUB_REPO}/releases/latest/download/${MACOS_APP_FILE}`

// GitHub's "latest" alias resolves against the most recently published
// (non-draft, non-prerelease) release, so this always serves the current build
// without any API call or CORS issue.
export function releaseAsset(key: ReleaseAssetKey): string {
  return `${GITHUB_REPO}/releases/latest/download/${RELEASE_ASSETS[key]}`
}

// Microsoft's form to report a Defender false positive (the guide's Windows
// install step, ADR-0073).
export const DEFENDER_FALSE_POSITIVE_URL = 'https://www.microsoft.com/wdsi/filesubmission'
