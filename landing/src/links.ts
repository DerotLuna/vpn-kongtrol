// Downloads live on GitHub Releases.
export const GITHUB_REPO = 'https://github.com/DerotLuna/vpn-kongtrol'
export const GITHUB_RELEASES = `${GITHUB_REPO}/releases/latest`

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
