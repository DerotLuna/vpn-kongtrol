#!/usr/bin/env bash
# Version 1.1.0 · 2026-11-19 · docs/adr/0063-kill-switch-lan-protected-read-and-release-hardening.md
#
# Offline regression checks for install.sh and install.ps1 (CI packaging job):
#   1. A folder read back from ~/.kongtrol/install.manifest is reused when it
#      is a default folder or this user can write it without sudo/admin (an
#      update replaces the copy in place, stops and resumes Kongtrol); a
#      folder that would need sudo is ignored with a warning, unless
#      KONGTROL_INSTALL_DIR names it.
#   2. A truncated download (curl | sh, iwr | iex) runs nothing.
# curl serves a fake release from a local folder (or fails); nothing is
# fetched, and nothing is written outside a temp folder.
#
#   bash packaging/tests/install-scripts.sh
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
t="$(mktemp -d)"
trap 'chmod -R u+w "$t" 2>/dev/null; rm -rf "$t"' EXIT
fail=0
pass() { echo "ok   $1"; }
bad() { echo "FAIL $1"; fail=1; }
skip() { echo "skip $1"; }

# Stubs. curl copies <url basename> from $KT_FIX, or fails when KT_FIX is
# empty; uname says Linux x86_64 (also on a dev box); sudo always fails.
mkdir -p "$t/stub"
# shellcheck disable=SC2016 # the stubs' own variables, expanded when they run
{
  printf '#!/bin/sh\nout=""; url=""\n'
  printf 'while [ $# -gt 0 ]; do case "$1" in -o) out="$2"; shift 2 ;; -*) shift ;; *) url="$1"; shift ;; esac; done\n'
  printf '[ -n "${KT_FIX:-}" ] && [ -f "$KT_FIX/${url##*/}" ] || exit 22\ncp "$KT_FIX/${url##*/}" "$out"\n'
} > "$t/stub/curl"
# shellcheck disable=SC2016
printf '#!/bin/sh\ncase "${1:-}" in -m) echo x86_64 ;; *) echo Linux ;; esac\n' > "$t/stub/uname"
printf '#!/bin/sh\necho "sudo $*" >&2\nexit 1\n' > "$t/stub/sudo"
chmod +x "$t/stub/curl" "$t/stub/uname" "$t/stub/sudo"

# A fake release 0.2.0: its kongtrol logs every call to $KT_LOG.
mkdir -p "$t/rel" "$t/fix"
# shellcheck disable=SC2016
printf '#!/bin/sh\necho "$*" >> "$KT_LOG"\n[ "${1:-}" = version ] && echo 0.2.0\nexit 0\n' > "$t/rel/kongtrol"
cp "$t/rel/kongtrol" "$t/rel/kongtrol-tray"
printf 'png' > "$t/rel/kongtrol.png"
printf '[Desktop Entry]\nExec=kongtrol-tray\n' > "$t/rel/kongtrol.desktop"
chmod +x "$t/rel/kongtrol" "$t/rel/kongtrol-tray"
tar -czf "$t/fix/kongtrol_linux_amd64.tar.gz" -C "$t/rel" kongtrol kongtrol-tray kongtrol.png kongtrol.desktop
printf '%s  kongtrol_linux_amd64.tar.gz\n' "$(sha256sum "$t/fix/kongtrol_linux_amd64.tar.gz" | awk '{print $1}')" > "$t/fix/checksums.txt"

# new_home NAME DIR: a home whose manifest says the last install was DIR,
# with an old 0.1.0 kongtrol there when DIR exists.
new_home() {
  local h="$t/$1"
  mkdir -p "$h/.kongtrol"
  printf 'file=%s/kongtrol\n' "$2" > "$h/.kongtrol/install.manifest"
  if [ -d "$2" ] && [ -w "$2" ]; then
    printf '#!/bin/sh\necho 0.1.0\n' > "$2/kongtrol"
    chmod +x "$2/kongtrol"
  fi
  printf '%s' "$h"
}

# run_sh HOME [VAR=value…] SCRIPT: install.sh offline, quiet extras off.
run_sh() {
  local h="$1"; shift
  env HOME="$h" XDG_DATA_HOME="$h/share" XDG_CONFIG_HOME="$h/config" PATH="$t/stub:$PATH" \
    KONGTROL_SKIP_SIGNATURE=1 KONGTROL_NO_LAUNCH=1 KONGTROL_NO_AUTOSTART=1 KT_LOG="$h/calls.log" \
    "$@" 2>&1 </dev/null || true
}

# 1a. Custom folder this user can write: updated in place, stop + resume run.
mkdir -p "$t/custom"
h="$(new_home h1 "$t/custom")"
out="$(run_sh "$h" env KT_FIX="$t/fix" sh "$root/install.sh")"
if grep -q "Updating Kongtrol 0.1.0 → 0.2.0" <<<"$out" && [ "$("$t/custom/kongtrol" version 2>/dev/null)" = 0.2.0 ] \
  && grep -q "^down --all --quiet --remember" "$h/calls.log" && grep -q "^resume --quiet" "$h/calls.log" \
  && grep -q "^file=$t/custom/kongtrol\$" "$h/.kongtrol/install.manifest" && ! grep -q "not reused" <<<"$out"; then
  pass "install.sh updates a writable custom folder in place (stop + resume)"
else
  bad "install.sh did not update the writable custom folder in place: $out"
fi

# 1b. Folder that would need sudo: warning, not reused.
locked="$t/locked"
mkdir -p "$locked"
printf '#!/bin/sh\necho 0.1.0\n' > "$locked/kongtrol"
chmod +x "$locked/kongtrol"
chmod a-w "$locked"
[ -w "$locked" ] && locked="$t/missing-dir" # root or a filesystem without modes
h="$(new_home h2 "$locked")"
out="$(run_sh "$h" sh "$root/install.sh")"
if grep -q "needs sudo, so it is not reused" <<<"$out" && ! grep -q "^sudo" <<<"$out"; then
  pass "install.sh ignores a manifest folder that needs sudo"
else
  bad "install.sh reused a folder that needs sudo: $out"
fi
# … and installs into a default folder (only checked when /usr/local/bin is
# not writable here, so the test never writes outside its temp folder).
if [ -w /usr/local/bin ]; then
  skip "default-folder install (/usr/local/bin is writable here)"
else
  out="$(run_sh "$h" env KT_FIX="$t/fix" sh "$root/install.sh")"
  if grep -q "Installed $h/.local/bin/kongtrol" <<<"$out" && ! grep -q "^sudo" <<<"$out"; then
    pass "install.sh installs into the default folder instead"
  else
    bad "install.sh did not fall back to the default folder: $out"
  fi
fi

# 1c. KONGTROL_INSTALL_DIR names the folder: no warning.
out="$(run_sh "$h" env KONGTROL_INSTALL_DIR="$locked" sh "$root/install.sh")"
if grep -q "not reused" <<<"$out"; then bad "install.sh warned with KONGTROL_INSTALL_DIR set"; else pass "install.sh accepts an explicit KONGTROL_INSTALL_DIR"; fi

# 2. Truncated download.
head -c "$(( $(wc -c <"$root/install.sh") * 6 / 10 ))" "$root/install.sh" > "$t/cut.sh"
out="$(run_sh "$t/h1" sh "$t/cut.sh")"
if grep -q "==>" <<<"$out"; then bad "truncated install.sh ran: $out"; else pass "truncated install.sh runs nothing"; fi

if command -v pwsh >/dev/null 2>&1; then
  # Git Bash on Windows: pwsh needs Windows paths.
  w="$t"; r="$root"
  if command -v cygpath >/dev/null 2>&1; then r="$(cygpath -w "$root")"; w="$(cygpath -w "$t")"; fi
  mkdir -p "$t/lad" "$t/pw/.kongtrol"
  # ps_run DIR: install.ps1 with a manifest naming DIR; downloads fail.
  ps_run() {
    printf 'file=%s/kongtrol.exe\n' "$1" > "$t/pw/.kongtrol/install.manifest"
    env HOME="$w/pw" USERPROFILE="$w/pw" LOCALAPPDATA="$w/lad" PROCESSOR_ARCHITECTURE=AMD64 \
      pwsh -NoProfile -Command "function Invoke-WebRequest { throw 'offline' }; & '$r/install.ps1'" 2>&1 || true
  }
  out="$(ps_run "$w/custom")"
  if grep -q "not reused" <<<"$out"; then bad "install.ps1 dropped a writable custom folder: $out"; else pass "install.ps1 reuses a folder you can write"; fi
  out="$(ps_run "$w/missing-dir")"
  if grep -q "without administrator rights, so it is not reused" <<<"$out"; then pass "install.ps1 ignores a folder you cannot write"; else bad "install.ps1 reused a folder you cannot write: $out"; fi
  # shellcheck disable=SC2016 # PowerShell code, expanded by pwsh
  out="$(env ROOT="$r" pwsh -NoProfile -Command '$s = Get-Content -Raw (Join-Path $env:ROOT "install.ps1"); $e = $null; $null = [System.Management.Automation.Language.Parser]::ParseInput($s.Substring(0, [int]($s.Length * 0.6)), [ref]$null, [ref]$e); $e.Count')"
  out="${out//[[:space:]]/}"
  if [[ "$out" =~ ^[1-9][0-9]*$ ]]; then pass "truncated install.ps1 does not parse (runs nothing)"; else bad "truncated install.ps1 parses or could not be read: $out"; fi
else
  skip "install.ps1 (pwsh not found)"
fi

exit "$fail"
