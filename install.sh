#!/usr/bin/env bash
# Installs the kongtrol CLI from the latest GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.sh | sh
#
# Env overrides:
#   KONGTROL_INSTALL_DIR   install directory (default: /usr/local/bin, falls back to ~/.local/bin)
#   KONGTROL_VERSION       release tag to install, e.g. v1.2.3 (default: latest)
set -euo pipefail

REPO="DerotLuna/vpn-kongtrol"
VERSION="${KONGTROL_VERSION:-latest}"

info() { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$1" >&2; exit 1; }

os="$(uname -s)"
case "$os" in
  Linux)  goos="linux" ;;
  Darwin) goos="darwin" ;;
  *) die "unsupported OS: $os (use install.ps1 on Windows, or build from source)" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *) die "unsupported architecture: $arch" ;;
esac

binary="kongtrol-${goos}-${goarch}"

if [ "$VERSION" = "latest" ]; then
  base_url="https://github.com/${REPO}/releases/latest/download"
else
  base_url="https://github.com/${REPO}/releases/download/${VERSION}"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

info "Downloading ${binary} (${VERSION})..."
curl -fsSL "${base_url}/${binary}" -o "${tmp_dir}/kongtrol" \
  || die "download failed — check that a release has been published: https://github.com/${REPO}/releases"

info "Verifying checksum..."
curl -fsSL "${base_url}/checksums.txt" -o "${tmp_dir}/checksums.txt" \
  || die "could not fetch checksums.txt"

expected="$(grep " ${binary}\$" "${tmp_dir}/checksums.txt" | awk '{print $1}')"
[ -n "$expected" ] || die "no checksum entry found for ${binary}"

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${tmp_dir}/kongtrol" | awk '{print $1}')"
else
  actual="$(shasum -a 256 "${tmp_dir}/kongtrol" | awk '{print $1}')"
fi

[ "$expected" = "$actual" ] || die "checksum mismatch: expected ${expected}, got ${actual}"

chmod +x "${tmp_dir}/kongtrol"

install_dir="${KONGTROL_INSTALL_DIR:-/usr/local/bin}"
if [ ! -w "$install_dir" ] 2>/dev/null; then
  if [ -z "${KONGTROL_INSTALL_DIR:-}" ]; then
    install_dir="${HOME}/.local/bin"
    mkdir -p "$install_dir"
  else
    die "no write permission to ${install_dir}"
  fi
fi

if [ -w "$install_dir" ]; then
  mv "${tmp_dir}/kongtrol" "${install_dir}/kongtrol"
else
  info "sudo required to write to ${install_dir}"
  sudo mv "${tmp_dir}/kongtrol" "${install_dir}/kongtrol"
fi

info "Installed to ${install_dir}/kongtrol"

case ":$PATH:" in
  *":${install_dir}:"*) ;;
  *) printf '\033[1;33mwarning:\033[0m %s is not on your PATH — add it to your shell profile.\n' "$install_dir" ;;
esac

"${install_dir}/kongtrol" version 2>/dev/null || true
