#!/bin/sh
# Installs or upgrades Kongtrol (CLI + tray app) from the latest GitHub release.
# Version 2.4.0 · 2026-11-19 · docs/adr/0025-desktop-release-channels-first-run.md, docs/adr/0056-installers-verify-release-signature.md, docs/adr/0034-uninstall-and-update-notice.md, docs/adr/0046-installers-detect-existing-install.md, docs/adr/0050-installer-resume-tunnels-and-open-request.md
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
# a running Kongtrol is stopped and the VPNs that were connected reconnect on the new version.
# An existing install is detected and the script says "Updating <old> → <new>"
# (or "Reinstalling", or warns before an older version, asking only on a
# terminal; --yes answers for scripts). It reuses the previous folders and
# autostart choice and updates the network helper only when it is outdated.
#
# Env overrides:
#   KONGTROL_INSTALL_DIR   CLI directory (default: /usr/local/bin, falls back to ~/.local/bin)
#   KONGTROL_APP_DIR       macOS app directory (default: ~/Applications)
#   KONGTROL_VERSION       release tag to install, e.g. v1.2.3 (default: latest)
#   KONGTROL_NO_AUTOSTART  1 = no autostart entry (Linux)
#   KONGTROL_NO_LAUNCH     1 = don't open Kongtrol at the end
#   KONGTROL_YES           1 = same as --yes
#   KONGTROL_SKIP_SIGNATURE 1 = don't check the cosign signature of checksums.txt
#                          (only for releases before v0.6.0, which were not signed)
#
# Integrity: the download is checked against checksums.txt. When cosign 2.4+
# is installed, the Sigstore signature of checksums.txt is checked first (it
# must come from this repo's release workflow) and a bad signature stops the
# install. Without cosign the script says so and how to verify by hand.
set -eu
(set -o pipefail) 2>/dev/null && set -o pipefail

REPO="DerotLuna/vpn-kongtrol"
VERSION="${KONGTROL_VERSION:-latest}"

info() { printf '\033[1;34m==>\033[0m %s\n' "$1"; }
warn() { printf '\033[1;33mwarning:\033[0m %s\n' "$1"; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$1" >&2; exit 1; }

# ver_cmp A B prints -1, 0 or 1 (A < B, equal, A > B) with SemVer 2.0.0
# precedence (https://semver.org §11), the same rules as the Windows
# installer: leading "v" and "+build" ignored, X.Y.Z numeric (missing = 0),
# a pre-release is lower than its release, pre-release identifiers dot by
# dot (numbers numerically and below text), fewer identifiers lower.
# 0.6.0-rc.3 < 0.6.0-rc.4 < 0.6.0 < 0.10.0.
ver_cmp() {
  awk -v a="$1" -v b="$2" '
    function isnum(s) { return s ~ /^[0-9]+$/ }
    function cmpid(x, y) {
      if (isnum(x) && isnum(y)) { x += 0; y += 0; return x < y ? -1 : (x > y ? 1 : 0) }
      if (isnum(x)) return -1
      if (isnum(y)) return 1
      return ("" x) < ("" y) ? -1 : (("" x) > ("" y) ? 1 : 0)
    }
    function norm(s) { sub(/^[vV]/, "", s); sub(/\+.*/, "", s); return s }
    BEGIN {
      a = norm(a); b = norm(b); pa = ""; pb = ""
      i = index(a, "-"); if (i) { pa = substr(a, i + 1); a = substr(a, 1, i - 1) }
      i = index(b, "-"); if (i) { pb = substr(b, i + 1); b = substr(b, 1, i - 1) }
      na = split(a, ca, "."); nb = split(b, cb, ".")
      n = na > nb ? na : nb
      for (k = 1; k <= n; k++) {
        x = ca[k] + 0; y = cb[k] + 0
        if (x != y) { print (x < y ? -1 : 1); exit }
      }
      if (pa == "" && pb == "") { print 0; exit }
      if (pa == "") { print 1; exit }
      if (pb == "") { print -1; exit }
      na = split(pa, xa, "."); nb = split(pb, xb, ".")
      for (k = 1; k <= na && k <= nb; k++) { c = cmpid(xa[k], xb[k]); if (c != 0) { print c; exit } }
      print (na < nb ? -1 : (na > nb ? 1 : 0))
    }'
}

# bin_version BIN prints the version BIN reports (`kongtrol version` prints
# only the version when its output is not a terminal), or nothing.
bin_version() {
  v="$("$1" version 2>/dev/null | head -n 1 | tr -d '\r' | sed 's/^[vV]//')" || v=""
  case "$v" in [0-9]*) printf '%s' "$v" ;; esac
}

# Sigstore identity of a genuine release (docs/adr/0056-installers-verify-release-signature.md):
# checksums.txt is signed keyless by .github/workflows/release.yml on a v* tag.
COSIGN_IDENTITY='^https://github\.com/DerotLuna/vpn-kongtrol/\.github/workflows/release\.yml@refs/tags/v'
COSIGN_ISSUER='https://token.actions.githubusercontent.com'

# cosign_bundle_flag prints the flag `cosign verify-blob` needs to read a
# Sigstore bundle (.sigstore.json): none on cosign 3+, --new-bundle-format on
# 2.4-2.x. Fails when this cosign is older than 2.4 (no bundle support).
cosign_bundle_flag() {
  cv="$(cosign version 2>/dev/null | awk '/GitVersion:/ {print $2; exit}')"
  case "$cv" in v[0-9]*|[0-9]*) ;; *) return 1 ;; esac
  [ "$(ver_cmp "$cv" 2.4.0)" -ge 0 ] || return 1
  [ "$(ver_cmp "$cv" 3.0.0)" -ge 0 ] || printf '%s' "--new-bundle-format"
}

# has_tty: a terminal to ask on, even when the script is piped from curl.
has_tty() { (: </dev/tty) 2>/dev/null; }

# --uninstall [--purge]: delegate to `kongtrol uninstall`, the same logic the
# app uses (docs/adr/0034-uninstall-and-update-notice.md). Your data stays
# unless you add --purge.
do_uninstall=0
do_purge=0
assume_yes=0
[ "${KONGTROL_YES:-0}" = "1" ] && assume_yes=1
for arg in "$@"; do
  case "$arg" in
    --uninstall) do_uninstall=1 ;;
    --purge) do_purge=1 ;;
    --yes|-y) assume_yes=1 ;;
    *) die "unknown option: $arg (use --yes, or --uninstall optionally with --purge)" ;;
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

# Existing install (docs/adr/0046-installers-detect-existing-install.md): the
# manifest an earlier run wrote says where it put things, so an update reuses
# those folders and the autostart choice instead of asking again or leaving
# a second copy behind. Without a manifest, a kongtrol in the default
# folders counts too.
manifest="${HOME}/.kongtrol/install.manifest"
prev_bin=""
prev_dir=""
prev_app_dir=""
prev_no_autostart=""
if [ -f "$manifest" ]; then
  prev_bin="$(sed -n 's|^file=\(.*\)/kongtrol$|\1/kongtrol|p' "$manifest" | head -n 1)"
  [ -n "$prev_bin" ] && prev_dir="$(dirname "$prev_bin")"
  prev_app_dir="$(sed -n 's|^file=\(.*\)/Kongtrol\.app$|\1|p' "$manifest" | head -n 1)"
  if grep -q '/kongtrol-tray$' "$manifest" && ! grep -q '^autostart=' "$manifest"; then
    prev_no_autostart=1
  fi
fi
if [ -z "$prev_bin" ] || [ ! -x "$prev_bin" ]; then
  prev_bin=""
  for d in "${KONGTROL_INSTALL_DIR:-}" /usr/local/bin "${HOME}/.local/bin"; do
    if [ -n "$d" ] && [ -x "$d/kongtrol" ]; then prev_bin="$d/kongtrol"; break; fi
  done
fi
prev_ver=""
[ -n "$prev_bin" ] && prev_ver="$(bin_version "$prev_bin")"
no_autostart="${KONGTROL_NO_AUTOSTART:-${prev_no_autostart:-0}}"

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
verify_hint="cosign verify-blob --bundle checksums.txt.sigstore.json --certificate-identity-regexp '${COSIGN_IDENTITY}' --certificate-oidc-issuer ${COSIGN_ISSUER} checksums.txt (or: gh attestation verify checksums.txt --repo ${REPO})"
if [ "${KONGTROL_SKIP_SIGNATURE:-0}" = "1" ]; then
  warn "signature check skipped (KONGTROL_SKIP_SIGNATURE=1): checked with checksums.txt only."
elif ! command -v cosign >/dev/null 2>&1; then
  info "Signature not checked (cosign is not installed): checked with checksums.txt only. To verify by hand: ${verify_hint}"
elif ! bundle_flag="$(cosign_bundle_flag)"; then
  warn "cosign is too old to check the release signature (needs 2.4 or newer): checked with checksums.txt only. Update cosign, or verify by hand: ${verify_hint}"
else
  info "Verifying the signature of checksums.txt (cosign)..."
  curl -fsSL "${base_url}/checksums.txt.sigstore.json" -o "${tmp_dir}/checksums.txt.sigstore.json" \
    || die "this release has no signature (checksums.txt.sigstore.json), so nothing was installed. Releases before v0.6.0 were not signed: to install one anyway, run again with KONGTROL_SKIP_SIGNATURE=1. Otherwise report it: https://github.com/${REPO}/issues"
  # $bundle_flag is empty or one word, so it stays unquoted on purpose.
  # shellcheck disable=SC2086
  if ! cosign verify-blob $bundle_flag \
      --bundle "${tmp_dir}/checksums.txt.sigstore.json" \
      --certificate-identity-regexp "$COSIGN_IDENTITY" \
      --certificate-oidc-issuer "$COSIGN_ISSUER" \
      "${tmp_dir}/checksums.txt" >"${tmp_dir}/cosign.log" 2>&1; then
    die "the signature of checksums.txt could not be verified, so nothing was installed (cosign: $(grep -v '^error during' "${tmp_dir}/cosign.log" | tail -n 1)). If you are offline, connect and try again; otherwise do not use this download and report it: https://github.com/${REPO}/security"
  fi
  info "Signature OK: checksums.txt comes from the Kongtrol release workflow."
fi
expected="$(grep " ${asset}\$" "${tmp_dir}/checksums.txt" | awk '{print $1}')"
[ -n "$expected" ] || die "no checksum entry found for ${asset}"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${tmp_dir}/${asset}" | awk '{print $1}')"
else
  actual="$(shasum -a 256 "${tmp_dir}/${asset}" | awk '{print $1}')"
fi
[ "$expected" = "$actual" ] || die "checksum mismatch: expected ${expected}, got ${actual}"

# Unpack first: the new binary says which version this is, so an update can
# be told apart from a reinstall or an older version before anything changes.
if [ "$goos" = "darwin" ]; then
  ditto -x -k "${tmp_dir}/${asset}" "${tmp_dir}/app"
  new_bin="${tmp_dir}/app/Kongtrol.app/Contents/MacOS/kongtrol"
else
  mkdir -p "${tmp_dir}/x"
  tar -xzf "${tmp_dir}/${asset}" -C "${tmp_dir}/x"
  new_bin="${tmp_dir}/x/kongtrol"
  chmod +x "$new_bin"
fi
new_ver="$(bin_version "$new_bin")"

kind=install
if [ -n "$prev_bin" ]; then
  kind=update
  if [ -n "$prev_ver" ] && [ -n "$new_ver" ]; then
    case "$(ver_cmp "$prev_ver" "$new_ver")" in
      0) kind=repair ;;
      1) kind=downgrade ;;
    esac
  fi
fi
case "$kind" in
  update) info "Updating Kongtrol ${prev_ver:-} → ${new_ver:-${VERSION}} (your VPNs and settings are kept)..." ;;
  repair) info "Reinstalling Kongtrol ${new_ver} (repair: your VPNs and settings are kept)..." ;;
  downgrade)
    warn "This installs an OLDER version (${prev_ver} → ${new_ver}). Your VPNs and settings are kept."
    if [ "$assume_yes" != 1 ] && has_tty; then
      printf 'Install the older version? [y/N] ' >/dev/tty
      answer=""
      read -r answer </dev/tty || answer=""
      case "$answer" in
        y|Y|yes|YES|s|S|si|SI) ;;
        *) info "Nothing changed. To install it anyway, run the same command with: sh -s -- --yes"; exit 0 ;;
      esac
    fi
    ;;
esac

install_dir="${KONGTROL_INSTALL_DIR:-${prev_dir:-/usr/local/bin}}"
if [ ! -w "$install_dir" ] 2>/dev/null; then
  if [ -z "${KONGTROL_INSTALL_DIR:-}" ] && [ -z "$prev_dir" ]; then
    install_dir="${HOME}/.local/bin"
    mkdir -p "$install_dir"
  elif [ -n "${KONGTROL_INSTALL_DIR:-}" ]; then
    die "no write permission to ${install_dir}"
  fi
  # The previous install's folder: sudo below, so no second copy is left.
fi
SUDO=""
[ -w "$install_dir" ] || { info "sudo required to write to ${install_dir}"; SUDO="sudo"; }


if [ "$goos" = "darwin" ]; then
  app_dir="${KONGTROL_APP_DIR:-${prev_app_dir:-${HOME}/Applications}}"
  mkdir -p "$app_dir"
  # Quit the old tray (the daemon keeps running until the stop below).
  pkill -x kongtrol-tray 2>/dev/null || true
  rm -rf "${app_dir}/Kongtrol.app.old"
  [ -d "${app_dir}/Kongtrol.app" ] && mv "${app_dir}/Kongtrol.app" "${app_dir}/Kongtrol.app.old"
  mv "${tmp_dir}/app/Kongtrol.app" "${app_dir}/Kongtrol.app"
  rm -rf "${app_dir}/Kongtrol.app.old"
  $SUDO ln -sfn "${app_dir}/Kongtrol.app/Contents/MacOS/kongtrol" "${install_dir}/kongtrol"
  info "Installed ${app_dir}/Kongtrol.app and ${install_dir}/kongtrol"
else
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
  if [ "$no_autostart" != "1" ]; then
    autostart="${XDG_CONFIG_HOME:-${HOME}/.config}/autostart"
    mkdir -p "$autostart"
    cp "${apps}/applications/kongtrol.desktop" "${autostart}/kongtrol.desktop"
  fi
  info "Installed ${install_dir}/kongtrol and ${install_dir}/kongtrol-tray (+ menu entry)"
fi

# What this script installed, so `kongtrol uninstall` removes exactly that
# and nothing a package manager owns (docs/adr/0034-uninstall-and-update-notice.md).
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
    if [ "$no_autostart" != "1" ]; then echo "autostart=${autostart}/kongtrol.desktop"; fi
  fi
} > "${HOME}/.kongtrol/install.manifest"

case ":$PATH:" in
  *":${install_dir}:"*) ;;
  *) warn "${install_dir} is not on your PATH — add it to your shell profile." ;;
esac

# Update: stop the old Kongtrol and save which VPNs were connected, BEFORE
# the helper update (a helper restart could drop a VPN and it would then be
# forgotten); `resume` below reconnects exactly those on the new version
# (docs/adr/0050-installer-resume-tunnels-and-open-request.md).
if [ "$kind" != install ]; then
  info "Stopping Kongtrol for the update (connected VPNs reconnect at the end)..."
  "${install_dir}/kongtrol" down --all --quiet --remember >/dev/null 2>&1 || true
fi

# Update: the network helper is updated only when it speaks an older
# protocol (`service install --if-outdated`: a missing or current helper is
# left alone and nothing is asked). It needs the admin password, so only on
# a terminal; otherwise the tray offers "Actualizar ayudante (una vez)".
# Before the resume below, so the VPNs reconnect through a current helper.
if [ "$kind" != install ] && has_tty; then
  info "Checking the network helper (asks for your admin password only if it is out of date)..."
  "${install_dir}/kongtrol" service install --if-outdated --quiet </dev/tty \
    || warn "the network helper was not updated: open Kongtrol and choose \"Actualizar ayudante (una vez)\", or run: kongtrol service install"
fi

# Update: reconnect the VPNs saved above (nothing when none was connected).
if [ "$kind" != install ]; then
  if "${install_dir}/kongtrol" resume --quiet >/dev/null 2>&1; then
    info "Kongtrol is reconnecting the VPNs you had connected (if any). Check with: kongtrol status"
  else
    warn "your VPNs were not reconnected: connect them from the Kongtrol icon, or run: kongtrol up"
  fi
fi

if [ "${KONGTROL_NO_LAUNCH:-0}" != "1" ]; then
  if [ "$goos" = "darwin" ]; then
    open "${app_dir}/Kongtrol.app" || true
  elif [ -n "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ]; then
    nohup "${install_dir}/kongtrol-tray" >/dev/null 2>&1 &
  fi
  info "Kongtrol is open: look for its icon in the menu bar / system tray."
fi
case "$kind" in
  downgrade) info "Kongtrol ${new_ver} installed: your VPNs and settings are kept. Next: connect from the Kongtrol icon (or: kongtrol up)." ;;
  update) info "Kongtrol updated to ${new_ver:-the new version}: your VPNs and settings are kept. Next: connect from the Kongtrol icon (or: kongtrol up)." ;;
  repair) info "Kongtrol ${new_ver} reinstalled: your VPNs and settings are kept. Next: connect from the Kongtrol icon (or: kongtrol up)." ;;
esac
"${install_dir}/kongtrol" version 2>/dev/null || true
