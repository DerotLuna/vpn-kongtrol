#!/usr/bin/env bash
# Version 1.0.0 · 2026-11-19 · docs/adr/0063-kill-switch-lan-protected-read-and-release-hardening.md
#
# Offline regression checks for install.sh and install.ps1 (CI packaging job):
#   1. A folder read back from ~/.kongtrol/install.manifest is reused only when
#      it is a default folder (the install may sudo / prompt there), unless
#      KONGTROL_INSTALL_DIR names it.
#   2. A truncated download (curl | sh, iwr | iex) runs nothing.
# Downloads are stubbed to fail, so nothing is fetched or installed.
#
#   bash packaging/tests/install-scripts.sh
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
t="$(mktemp -d)"
trap 'rm -rf "$t"' EXIT
fail=0
pass() { echo "ok   $1"; }
bad() { echo "FAIL $1"; fail=1; }

# Stubs: every download fails; uname says Linux x86_64 (also on a dev box).
mkdir -p "$t/stub"
printf '#!/bin/sh\nexit 22\n' > "$t/stub/curl"
# shellcheck disable=SC2016 # the stub's own $1, expanded when it runs
printf '#!/bin/sh\ncase "${1:-}" in -m) echo x86_64 ;; *) echo Linux ;; esac\n' > "$t/stub/uname"
chmod +x "$t/stub/curl" "$t/stub/uname"

mkdir -p "$t/home/.kongtrol" "$t/elsewhere"
printf 'file=%s/elsewhere/kongtrol\n' "$t" > "$t/home/.kongtrol/install.manifest"
printf '#!/bin/sh\necho 0.1.0\n' > "$t/elsewhere/kongtrol"
chmod +x "$t/elsewhere/kongtrol"

run_sh() { env HOME="$t/home" PATH="$t/stub:$PATH" "$@" 2>&1 || true; }

out="$(run_sh sh "$root/install.sh")"
if grep -q "not a default folder, so it is not reused" <<<"$out"; then pass "install.sh ignores a non-default manifest folder"; else bad "install.sh reused a manifest folder: $out"; fi
out="$(run_sh env KONGTROL_INSTALL_DIR="$t/elsewhere" sh "$root/install.sh")"
if grep -q "not a default folder" <<<"$out"; then bad "install.sh warned with KONGTROL_INSTALL_DIR set"; else pass "install.sh accepts an explicit KONGTROL_INSTALL_DIR"; fi

head -c "$(( $(wc -c <"$root/install.sh") * 6 / 10 ))" "$root/install.sh" > "$t/cut.sh"
out="$(run_sh sh "$t/cut.sh")"
if grep -q "==>" <<<"$out"; then bad "truncated install.sh ran: $out"; else pass "truncated install.sh runs nothing"; fi

if command -v pwsh >/dev/null 2>&1; then
  # Git Bash on Windows: pwsh needs Windows paths.
  if command -v cygpath >/dev/null 2>&1; then root="$(cygpath -w "$root")"; t="$(cygpath -w "$t")"; fi
  mkdir -p "$t/lad"
  printf 'file=%s/elsewhere/kongtrol.exe\n' "$t" > "$t/home/.kongtrol/install.manifest"
  # Invoke-WebRequest is replaced by a function that fails (no network).
  out="$(env HOME="$t/home" USERPROFILE="$t/home" LOCALAPPDATA="$t/lad" PROCESSOR_ARCHITECTURE=AMD64 \
    pwsh -NoProfile -Command "function Invoke-WebRequest { throw 'offline' }; & '$root/install.ps1'" 2>&1 || true)"
  if grep -q "not the default folder, so it is not reused" <<<"$out"; then pass "install.ps1 ignores a non-default manifest folder"; else bad "install.ps1 reused a manifest folder: $out"; fi
  # shellcheck disable=SC2016 # PowerShell code, expanded by pwsh
  out="$(env ROOT="$root" pwsh -NoProfile -Command '$s = Get-Content -Raw (Join-Path $env:ROOT "install.ps1"); $e = $null; $null = [System.Management.Automation.Language.Parser]::ParseInput($s.Substring(0, [int]($s.Length * 0.6)), [ref]$null, [ref]$e); $e.Count')"
  out="${out//[[:space:]]/}"
  if [[ "$out" =~ ^[1-9][0-9]*$ ]]; then pass "truncated install.ps1 does not parse (runs nothing)"; else bad "truncated install.ps1 parses or could not be read: $out"; fi
else
  echo "skip install.ps1 (pwsh not found)"
fi

exit "$fail"
