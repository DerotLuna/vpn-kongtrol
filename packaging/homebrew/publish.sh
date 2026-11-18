#!/usr/bin/env bash
# Renders the Homebrew formula and pushes it to the tap — version 1.0.0 · 2026-11-18.
#
#   packaging/homebrew/publish.sh <version> <dir with release assets>
#
# Needs HOMEBREW_TAP_TOKEN (a token with contents:write on DerotLuna/homebrew-tap)
# and the GitHub CLI. Without the token it only renders the formula and exits 0,
# so a release never fails because the tap is not set up.
set -euo pipefail

version="${1:?version without v}"
assets="${2:?assets dir}"
here="$(cd "$(dirname "$0")" && pwd)"
tap="${HOMEBREW_TAP_REPO:-DerotLuna/homebrew-tap}"
out="$(mktemp -d)/kongtrol.rb"

sha() { sha256sum "$assets/$1" | awk '{print $1}'; }

sed -e "s/__VERSION__/${version}/g" \
    -e "s/__SHA_MACOS__/$(sha Kongtrol-macos-universal.zip)/" \
    -e "s/__SHA_LINUX_AMD64__/$(sha kongtrol_linux_amd64.tar.gz)/" \
    -e "s/__SHA_LINUX_ARM64__/$(sha kongtrol_linux_arm64.tar.gz)/" \
    "$here/kongtrol.rb.tmpl" > "$out"
grep -q "__" "$out" && { echo "unrendered placeholder in formula" >&2; exit 1; }
echo "Rendered $out"

if [ -z "${HOMEBREW_TAP_TOKEN:-}" ]; then
  echo "HOMEBREW_TAP_TOKEN not set: formula not published (tap not configured)."
  exit 0
fi

path="Formula/kongtrol.rb"
current_sha="$(GH_TOKEN="$HOMEBREW_TAP_TOKEN" gh api "repos/$tap/contents/$path" --jq .sha 2>/dev/null || true)"
args=(-X PUT "repos/$tap/contents/$path"
  -f "message=kongtrol ${version}"
  -f "content=$(base64 -w0 "$out")")
[ -n "$current_sha" ] && args+=(-f "sha=$current_sha")
GH_TOKEN="$HOMEBREW_TAP_TOKEN" gh api "${args[@]}" --jq .commit.html_url
