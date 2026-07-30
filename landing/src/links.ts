// Downloads live on GitHub Releases.
export const GITHUB_REPO = 'https://github.com/DerotLuna/vpn-kongtrol'
export const GITHUB_RELEASES = `${GITHUB_REPO}/releases/latest`

// GitHub's "latest" alias resolves against the most recently *published* (non-draft)
// release, so this always serves the current build without any API call or CORS issue.
// Note: goreleaser is configured with draft:true — releases must be published manually
// on GitHub before these URLs resolve to the new version.
export function releaseAsset(filename: string): string {
  return `${GITHUB_REPO}/releases/latest/download/${filename}`
}
