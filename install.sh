#!/bin/sh
# Installs or upgrades Kongtrol (CLI + tray app) from the latest GitHub release.
# Version 2.1.0 · 2026-11-18 · docs/adr/0025-desktop-release-channels-first-run.md, docs/adr/0033-uninstall-keeps-data-by-default.md
#
#   curl -fsSL https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.sh | sh
#
# macOS: Kongtrol.app goes to ~/Applications and the kongtrol command is linked
#   into the install dir. Downloaded with curl, so macOS adds no quarantine flag
#   and the ad-hoc signed app opens without a Gatekeeper prompt.
# Linux: kongtrol + kongtrol-tray go to the install dir, plus a menu entry and
#   (unless KONGTROL_NO_AUTOSTART=1) an autostart entry for your user.
# Uninstall: ... | sh -s -- --uninstall   (add --purge to also delete your data)
# Run it again to upgrade: your settings in ~/.kongtrol are never touched, and
# a running Kongtrol is restarted on the new version keeping its tunnels.
#
# Env overrides:
#   KONGTROL_INSTALL_DIR   CLI directory (default: /usr/local/bin, falls back to ~/.local/bin)
#   KONGTROL_APP_DIR       macOS app directory (default: ~/Applications)
#   KONGTROL_VERSION       release tag to install, e.g. v1.2.3 (default: latest)
#   KONGTROL_NO_AUTOSTART  1 = no autostart entry (Linux)
#   KONGTROL_NO_LAUNCH     1 = don't open Kongtrol at the end
set -eu
(set -o pipefail) 2>/dev/null && set -o pipefail

REPO="DerotLuna/vpn-kongtrol"
VERSION="${KONGTROL_VERSION:-latest}"

info() { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$1"; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$1" >&2; exit 1; }

# --uninstall [--purge]: delegate to `kongtrol uninstall`, the same logic the
# app uses (docs/adr/0033-uninstall-keeps-data-by-default.md). Your data stays
# unless you add --purge.
do_uninstall=0
do_purge=0
for arg in "$@"; do
  case "$arg" in
    --uninstall) do_uninstall=1 ;;
    --purge) do_purge=1 ;;
    *) die "unknown option: $arg (use --uninstall, optionally with --purge)" ;;
  esac
done
if [ "$do_purge" = 1 ] && [ "$do_uninstall" = 0 ]; then
  die "--purge only works with --uninstall"
fi
if [ "$do_uninstall" = 1 ]; then
  bin="$(command -v kongtrol 2>/dev/null || true)"
  if [ -z "$bin" ]; then
    for d in "${KONGTROL_INSTALL_DIR:-}" /usr/local/bin "${HOME}/.local/bin"; do
      if [ -n "$d" ] && [ -x "$d/kongtrol" ]; then bin="$d/kongtrol"; break; fi
    done
  fi
  [ -n "$bin" ] || die "kongtrol is not installed here: nothing to remove"
  set -- uninstall
  [ "$do_purge" = 1 ] && set -- uninstall --purge
  # Piped from curl, stdin is the script: ask on the terminal instead.
  if [ -r /dev/tty ]; then exec "$bin" "$@" </dev/tty; fi
  exec "$bin" "$@"
fi

os="$(uname -s)"
case "$os" in
  Linux)  goos="linux" ;;
  Darwin) goos="darwin" ;;
  *) die "unsupported OS: $os (on Windows use the installer from https://kongtrol.app)" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *) die "unsupported architecture: $arch" ;;
esac

if [ "$goos" = "darwin" ]; then
  asset="Kongtrol-macos-universal.zip"
else
  asset="kongtrol_linux_${goarch}.tar.gz"
fi

if [ "$VERSION" = "latest" ]; then
  base_url="https://github.com/${REPO}/releases/latest/download"
else
  base_url="https://github.com/${REPO}/releases/download/${VERSION}"
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

info "Downloading ${asset} (${VERSION})..."
curl -fsSL "${base_url}/${asset}" -o "${tmp_dir}/${asset}" \
  || die "download failed — check that a release has been published: https://github.com/${REPO}/releases"

info "Verifying checksum..."
curl -fsSL "${base_url}/checksums.txt" -o "${tmp_dir}/checksums.txt" \
  || die "could not fetch checksums.txt"
expected="$(grep " ${asset}\$" "${tmp_dir}/checksums.txt" | awk '{print $1}')"
[ -n "$expected" ] || die "no checksum entry found for ${asset}"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${tmp_dir}/${asset}" | awk '{print $1}')"
else
  actual="$(shasum -a 256 "${tmp_dir}/${asset}" | awk '{print $1}')"
fi
[ "$expected" = "$actual" ] || die "checksum mismatch: expected ${expected}, got ${actual}"

install_dir="${KONGTROL_INSTALL_DIR:-/usr/local/bin}"
if [ ! -w "$install_dir" ] 2>/dev/null; then
  if [ -z "${KONGTROL_INSTALL_DIR:-}" ]; then
    install_dir="${HOME}/.local/bin"
    mkdir -p "$install_dir"
  else
    die "no write permission to ${install_dir}"
  fi
fi
SUDO=""
[ -w "$install_dir" ] || { info "sudo required to write to ${install_dir}"; SUDO="sudo"; }


if [ "$goos" = "darwin" ]; then
  app_dir="${KONGTROL_APP_DIR:-${HOME}/Applications}"
  mkdir -p "$app_dir"
  ditto -x -k "${tmp_dir}/${asset}" "${tmp_dir}/app"
  # Quit the old tray (the daemon keeps running until the restart below).
  pkill -x kongtrol-tray 2>/dev/null || true
  rm -rf "${app_dir}/Kongtrol.app.old"
  [ -d "${app_dir}/Kongtrol.app" ] && mv "${app_dir}/Kongtrol.app" "${app_dir}/Kongtrol.app.old"
  mv "${tmp_dir}/app/Kongtrol.app" "${app_dir}/Kongtrol.app"
  rm -rf "${app_dir}/Kongtrol.app.old"
  $SUDO ln -sfn "${app_dir}/Kongtrol.app/Contents/MacOS/kongtrol" "${install_dir}/kongtrol"
  info "Installed ${app_dir}/Kongtrol.app and ${install_dir}/kongtrol"
else
  mkdir -p "${tmp_dir}/x"
  tar -xzf "${tmp_dir}/${asset}" -C "${tmp_dir}/x"
  pkill -x kongtrol-tray 2>/dev/null || true
  for b in kongtrol kongtrol-tray; do
    chmod +x "${tmp_dir}/x/${b}"
    # mv replaces the file: a running daemon keeps its old copy until restarted.
    $SUDO mv -f "${tmp_dir}/x/${b}" "${install_dir}/${b}"
  done
  apps="${XDG_DATA_HOME:-${HOME}/.local/share}"
  mkdir -p "${apps}/applications" "${apps}/icons/hicolor/512x512/apps"
  cp "${tmp_dir}/x/kongtrol.png" "${apps}/icons/hicolor/512x512/apps/kongtrol.png"
  sed "s|^Exec=kongtrol-tray|Exec=${install_dir}/kongtrol-tray|" "${tmp_dir}/x/kongtrol.desktop" \
    > "${apps}/applications/kongtrol.desktop"
  if [ "${KONGTROL_NO_AUTOSTART:-0}" != "1" ]; then
    autostart="${XDG_CONFIG_HOME:-${HOME}/.config}/autostart"
    mkdir -p "$autostart"
    cp "${apps}/applications/kongtrol.desktop" "${autostart}/kongtrol.desktop"
  fi
  info "Installed ${install_dir}/kongtrol and ${install_dir}/kongtrol-tray (+ menu entry)"
fi

# What this script installed, so `kongtrol uninstall` removes exactly that
# and nothing a package manager owns (docs/adr/0033-…).
mkdir -p "${HOME}/.kongtrol"
{
  echo "# Written by install.sh: files kongtrol uninstall may remove."
  echo "file=${install_dir}/kongtrol"
  if [ "$goos" = "darwin" ]; then
    echo "file=${app_dir}/Kongtrol.app"
  else
    echo "file=${install_dir}/kongtrol-tray"
    echo "file=${apps}/applications/kongtrol.desktop"
    echo "file=${apps}/icons/hicolor/512x512/apps/kongtrol.png"
    if [ "${KONGTROL_NO_AUTOSTART:-0}" != "1" ]; then echo "autostart=${autostart}/kongtrol.desktop"; fi
  fi
} > "${HOME}/.kongtrol/install.manifest"

case ":$PATH:" in
  *":${install_dir}:"*) ;;
  *) warn "${install_dir} is not on your PATH — add it to your shell profile." ;;
esac

# Upgrade: a running daemon (old version) is restarted on the new binary with
# its connected VPNs; with no daemon running this does nothing.
if "${install_dir}/kongtrol" restart --quiet >/dev/null 2>&1; then
  info "Kongtrol restarted on the new version (your connected VPNs reconnected)."
fi

if [ "${KONGTROL_NO_LAUNCH:-0}" != "1" ]; then
  if [ "$goos" = "darwin" ]; then
    open "${app_dir}/Kongtrol.app" || true
  elif [ -n "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ]; then
    nohup "${install_dir}/kongtrol-tray" >/dev/null 2>&1 &
  fi
  info "Kongtrol is open: look for its icon in the menu bar / system tray."
fi
"${install_dir}/kongtrol" version 2>/dev/null || true
