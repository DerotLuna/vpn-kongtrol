#!/usr/bin/env bash
# Builds Kongtrol.app (universal: arm64 + x86_64) and zips it — version 1.1.0 · 2026-11-19
#
# Runs on macOS only (the tray needs cgo + Cocoa; Apple's toolchain builds
# both architectures from one machine). Used by .github/workflows/release.yml
# (job "macos"); decision and sources in docs/adr/0025-desktop-release-channels-first-run.md.
#
#   packaging/macos/build-app.sh <version> <out-dir>
#
# Output: <out-dir>/Kongtrol-macos-universal.zip containing Kongtrol.app with
# Contents/MacOS/kongtrol-tray (menu bar app, LSUIElement: no Dock icon) and
# Contents/MacOS/kongtrol (CLI + daemon; the tray finds it next to itself).
# The bundle is ad-hoc signed only (no paid Developer ID, no notarization).
set -euo pipefail

version="${1:?version}"
out="${2:?out dir}"
root="$(cd "$(dirname "$0")/../.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

ldflags="-s -w -X main.version=${version}"
cd "$root"
for arch in arm64 amd64; do
  CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" go build -trimpath -ldflags "$ldflags" \
    -o "$work/kongtrol-$arch" ./cmd/kongtrol
  # cgo: cmd/go passes "-arch x86_64"/"-arch arm64" to clang for darwin.
  CGO_ENABLED=1 GOOS=darwin GOARCH="$arch" go build -trimpath -ldflags "$ldflags" \
    -o "$work/kongtrol-tray-$arch" ./cmd/kongtrol-tray
done

app="$work/Kongtrol.app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
lipo -create -output "$app/Contents/MacOS/kongtrol" "$work/kongtrol-arm64" "$work/kongtrol-amd64"
lipo -create -output "$app/Contents/MacOS/kongtrol-tray" "$work/kongtrol-tray-arm64" "$work/kongtrol-tray-amd64"

# App icon from the single brand source (assets/brand): the 1024 px plate on
# Apple's icon grid, transparent margin included (generate.go, ADR-0052).
iconset="$work/Kongtrol.iconset"
mkdir -p "$iconset"
src="assets/brand/app-icon-1024.png"
for size in 16 32 128 256 512; do
  sips -z "$size" "$size" "$src" --out "$iconset/icon_${size}x${size}.png" >/dev/null
done
for size in 16 32 128 256 512; do
  double=$((size * 2))
  sips -z "$double" "$double" "$src" --out "$iconset/icon_${size}x${size}@2x.png" >/dev/null
done
iconutil -c icns -o "$app/Contents/Resources/Kongtrol.icns" "$iconset"

sed "s/__VERSION__/${version}/g" packaging/macos/Info.plist > "$app/Contents/Info.plist"
plutil -lint "$app/Contents/Info.plist"

# Ad-hoc signature (free): lipo output must be re-signed for Apple Silicon to
# run it. Gatekeeper still asks the user to confirm the first launch.
codesign --force --deep --sign - "$app"
codesign --verify --deep --strict "$app"

"$app/Contents/MacOS/kongtrol" version >/dev/null

mkdir -p "$out"
# ditto keeps the bundle's symlinks, permissions and signature (Apple's
# recommended way to zip an app).
ditto -c -k --sequesterRsrc --keepParent "$app" "$out/Kongtrol-macos-universal.zip"
echo "Built $out/Kongtrol-macos-universal.zip"
