# Installs or updates the kongtrol CLI from the latest GitHub release.
# Version 1.5.0 · 2026-11-19 · docs/adr/0063-kill-switch-lan-protected-read-and-release-hardening.md, docs/adr/0057-installers-verify-release-signature.md, docs/adr/0034-uninstall-and-update-notice.md, docs/adr/0046-installers-detect-existing-install.md, docs/adr/0050-installer-resume-tunnels-and-open-request.md
#
#   iwr https://kongtrol.app/install.ps1 -useb | iex
#
# Run it again to update: an existing install is detected and the script says
# "Updating <old> -> <new>" (or "Reinstalling", or warns before an older
# version, asking only in an interactive session; -Yes or $env:KONGTROL_YES=1
# answers for scripts). It reuses the previous folder, keeps ~/.kongtrol, and
# updates the network helper only when it is outdated.
#
# Uninstall (your data stays unless you add -Purge):
#   & ([scriptblock]::Create((iwr https://kongtrol.app/install.ps1 -useb))) -Uninstall [-Purge]
#   or: $env:KONGTROL_UNINSTALL=1 (and $env:KONGTROL_PURGE=1) before the iwr | iex line
#
# Env overrides:
#   $env:KONGTROL_INSTALL_DIR   install directory (default: the previous one, else $env:LOCALAPPDATA\kongtrol\bin)
#   $env:KONGTROL_VERSION       release tag to install, e.g. v1.2.3 (default: latest)
#   $env:KONGTROL_YES           1 = same as -Yes
#   $env:KONGTROL_SKIP_SIGNATURE 1 = don't check the cosign signature of checksums.txt
#                               (only for releases before v0.6.0, which were not signed)
#
# Integrity: the download is checked against checksums.txt. When cosign 2.4+
# is installed, the Sigstore signature of checksums.txt is checked first (it
# must come from this repo's release workflow) and a bad signature stops the
# install. Without cosign the script says so and how to verify by hand.
param([switch]$Uninstall, [switch]$Purge, [switch]$Yes)

$ErrorActionPreference = "Stop"

$Repo = "DerotLuna/vpn-kongtrol"
$Version = if ($env:KONGTROL_VERSION) { $env:KONGTROL_VERSION } else { "latest" }
$AssumeYes = $Yes -or $env:KONGTROL_YES -eq "1"

function Write-Info($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-Warn($msg) { Write-Host "warning: $msg" -ForegroundColor Yellow }
function Die($msg) { Write-Host "error: $msg" -ForegroundColor Red; exit 1 }

# Compare-KongtrolVersion A B returns -1, 0 or 1 with SemVer 2.0.0 precedence
# (https://semver.org section 11), the same rules as install.sh and the
# Windows installer: leading "v" and "+build" ignored, X.Y.Z numeric
# (missing = 0), a pre-release is lower than its release, pre-release
# identifiers dot by dot (numbers numerically and below text), fewer
# identifiers lower. 0.6.0-rc.3 < 0.6.0-rc.4 < 0.6.0 < 0.10.0.
function Compare-KongtrolVersion([string]$A, [string]$B) {
  $norm = { param($s) (($s.Trim() -replace '^[vV]', '') -split '\+', 2)[0] }
  $a = & $norm $A; $b = & $norm $B
  $ca, $pa = $a -split '-', 2
  $cb, $pb = $b -split '-', 2
  $xa = $ca -split '\.'; $xb = $cb -split '\.'
  for ($i = 0; $i -lt [Math]::Max($xa.Count, $xb.Count); $i++) {
    $na = if ($i -lt $xa.Count -and $xa[$i] -match '^[0-9]+$') { [long]$xa[$i] } else { 0 }
    $nb = if ($i -lt $xb.Count -and $xb[$i] -match '^[0-9]+$') { [long]$xb[$i] } else { 0 }
    if ($na -ne $nb) { if ($na -lt $nb) { return -1 } else { return 1 } }
  }
  if (-not $pa -and -not $pb) { return 0 }
  if (-not $pa) { return 1 }
  if (-not $pb) { return -1 }
  $ia = $pa -split '\.'; $ib = $pb -split '\.'
  for ($i = 0; $i -lt [Math]::Min($ia.Count, $ib.Count); $i++) {
    $numA = $ia[$i] -match '^[0-9]+$'; $numB = $ib[$i] -match '^[0-9]+$'
    if ($numA -and $numB) {
      $c = ([long]$ia[$i]).CompareTo([long]$ib[$i])
    } elseif ($numA) { $c = -1 } elseif ($numB) { $c = 1 } else {
      $c = [string]::CompareOrdinal($ia[$i], $ib[$i])
    }
    if ($c -ne 0) { if ($c -lt 0) { return -1 } else { return 1 } }
  }
  if ($ia.Count -lt $ib.Count) { return -1 }
  if ($ia.Count -gt $ib.Count) { return 1 }
  return 0
}

# Get-KongtrolVersion: the version an exe reports (`kongtrol version` prints
# only the version when its output is redirected), or $null.
function Get-KongtrolVersion([string]$Exe) {
  try {
    $v = (& $Exe version 2>$null | Select-Object -First 1)
    if ($v) { $v = ($v.ToString().Trim() -replace '^[vV]', '') }
    if ($v -match '^[0-9]') { return $v }
  } catch {}
  return $null
}

# Everything below runs from Invoke-KongtrolInstall, called on the last line:
# a download cut short is a parse error that runs nothing, never half a
# script (ADR-0063). The body of the function is not indented.
function Invoke-KongtrolInstall {

# Uninstall delegates to `kongtrol uninstall`, the same logic the app uses.
if ($Uninstall -or $env:KONGTROL_UNINSTALL -eq "1") {
  $cmd = Get-Command kongtrol -ErrorAction SilentlyContinue
  $exe = if ($cmd) { $cmd.Source } else {
    $dir = if ($env:KONGTROL_INSTALL_DIR) { $env:KONGTROL_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "kongtrol\bin" }
    Join-Path $dir "kongtrol.exe"
  }
  if (-not (Test-Path $exe)) { Die "kongtrol is not installed here: nothing to remove" }
  if ($Purge -or $env:KONGTROL_PURGE -eq "1") { & $exe uninstall --purge } else { & $exe uninstall }
  exit $LASTEXITCODE
}
if ($Purge) { Die "-Purge only works with -Uninstall" }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { Die "unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)" }
}

$binary = "kongtrol-windows-$arch.exe"

$baseUrl = if ($Version -eq "latest") {
  "https://github.com/$Repo/releases/latest/download"
} else {
  "https://github.com/$Repo/releases/download/$Version"
}

# Existing install (docs/adr/0046-installers-detect-existing-install.md): the
# manifest an earlier run wrote says where kongtrol.exe went, so an update
# reuses that folder instead of leaving a second copy behind. Only the
# default folder is reused from it: the manifest is a file any program of
# this user can rewrite, and the update runs the exe it finds there with a
# permission prompt (service install); KONGTROL_INSTALL_DIR chooses another.
$dataDir = Join-Path $HOME ".kongtrol"
$manifest = Join-Path $dataDir "install.manifest"
$previous = if (Test-Path $manifest) { @(Get-Content $manifest) } else { @() }
$prevExe = $previous | Where-Object { $_ -like "file=*kongtrol.exe" } | ForEach-Object { $_.Substring(5) } | Select-Object -First 1
$defaultDir = Join-Path $env:LOCALAPPDATA "kongtrol\bin"
if ($prevExe) {
  $prevDir = try { [IO.Path]::GetFullPath((Split-Path -Parent $prevExe)).TrimEnd('\') } catch { "" }
  if ($prevDir -ine [IO.Path]::GetFullPath($defaultDir).TrimEnd('\')) {
    if (-not $env:KONGTROL_INSTALL_DIR) {
      Write-Warn "the last install was in $prevDir, which is not the default folder, so it is not reused. To install there again, set `$env:KONGTROL_INSTALL_DIR=`"$prevDir`" first"
    }
    $prevExe = $null
  }
}
if (-not $prevExe -or -not (Test-Path $prevExe)) {
  $prevExe = $null
  foreach ($d in @($env:KONGTROL_INSTALL_DIR, $defaultDir)) {
    if ($d -and (Test-Path (Join-Path $d "kongtrol.exe"))) { $prevExe = Join-Path $d "kongtrol.exe"; break }
  }
}
$prevVersion = if ($prevExe) { Get-KongtrolVersion $prevExe } else { $null }

$tmpDir = Join-Path $env:TEMP "kongtrol-install-$(Get-Random)"
New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
try {
  $binaryPath = Join-Path $tmpDir $binary
  Write-Info "Downloading $binary ($Version)..."
  try {
    Invoke-WebRequest -Uri "$baseUrl/$binary" -OutFile $binaryPath -UseBasicParsing
  } catch {
    Die "download failed - check that a release has been published: https://github.com/$Repo/releases"
  }

  Write-Info "Verifying checksum..."
  $checksumsPath = Join-Path $tmpDir "checksums.txt"
  try {
    Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -OutFile $checksumsPath -UseBasicParsing
  } catch {
    Die "could not fetch checksums.txt"
  }

  # Signature of checksums.txt (docs/adr/0057-installers-verify-release-signature.md):
  # signed keyless by .github/workflows/release.yml on a v* tag.
  $cosignIdentity = '^https://github\.com/DerotLuna/vpn-kongtrol/\.github/workflows/release\.yml@refs/tags/v'
  $cosignIssuer = 'https://token.actions.githubusercontent.com'
  $verifyHint = "cosign verify-blob --bundle checksums.txt.sigstore.json --certificate-identity-regexp '$cosignIdentity' --certificate-oidc-issuer $cosignIssuer checksums.txt (or: gh attestation verify checksums.txt --repo $Repo)"
  $cosign = Get-Command cosign -ErrorAction SilentlyContinue
  if ($env:KONGTROL_SKIP_SIGNATURE -eq "1") {
    Write-Warn "signature check skipped (KONGTROL_SKIP_SIGNATURE=1): checked with checksums.txt only."
  } elseif (-not $cosign) {
    Write-Info "Signature not checked (cosign is not installed): checked with checksums.txt only. To verify by hand: $verifyHint"
  } else {
    # Sigstore bundles (.sigstore.json) need cosign 2.4+; 2.4-2.x reads them
    # with --new-bundle-format, 3+ by default.
    # cosign may write to stderr: run it with errors not terminating (Windows
    # PowerShell 5.1 turns native stderr into errors under "Stop").
    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    $cosignVersion = $null
    $line = (& $cosign.Source version 2>$null | Select-String 'GitVersion:' | Select-Object -First 1)
    if ($line -and $line.Line -match 'GitVersion:\s+v?(\d+\.\d+\.\d+)') { $cosignVersion = $Matches[1] }
    $ErrorActionPreference = $prevEap
    if (-not $cosignVersion -or (Compare-KongtrolVersion $cosignVersion "2.4.0") -lt 0) {
      Write-Warn "cosign is too old to check the release signature (needs 2.4 or newer): checked with checksums.txt only. Update cosign, or verify by hand: $verifyHint"
    } else {
      Write-Info "Verifying the signature of checksums.txt (cosign)..."
      $bundlePath = Join-Path $tmpDir "checksums.txt.sigstore.json"
      try {
        Invoke-WebRequest -Uri "$baseUrl/checksums.txt.sigstore.json" -OutFile $bundlePath -UseBasicParsing
      } catch {
        Die "this release has no signature (checksums.txt.sigstore.json), so nothing was installed. Releases before v0.6.0 were not signed: to install one anyway, set `$env:KONGTROL_SKIP_SIGNATURE=1 and run again. Otherwise report it: https://github.com/$Repo/issues"
      }
      $cosignArgs = @("verify-blob")
      if ((Compare-KongtrolVersion $cosignVersion "3.0.0") -lt 0) { $cosignArgs += "--new-bundle-format" }
      $cosignArgs += @("--bundle", $bundlePath, "--certificate-identity-regexp", $cosignIdentity, "--certificate-oidc-issuer", $cosignIssuer, $checksumsPath)
      $prevEap = $ErrorActionPreference
      $ErrorActionPreference = "Continue"
      $cosignOut = & $cosign.Source @cosignArgs 2>&1 | ForEach-Object { "$_" }
      $cosignExit = $LASTEXITCODE
      $ErrorActionPreference = $prevEap
      if ($cosignExit -ne 0) {
        $reason = ($cosignOut | Where-Object { $_ -and $_ -notmatch '^error during' } | Select-Object -Last 1)
        Die "the signature of checksums.txt could not be verified, so nothing was installed (cosign: $reason). If you are offline, connect and try again; otherwise do not use this download and report it: https://github.com/$Repo/security"
      }
      Write-Info "Signature OK: checksums.txt comes from the Kongtrol release workflow."
    }
  }

  $expectedLine = Select-String -Path $checksumsPath -Pattern "  $binary$" | Select-Object -First 1
  if (-not $expectedLine) { Die "no checksum entry found for $binary" }
  $expected = ($expectedLine.Line -split '\s+')[0]

  $actual = (Get-FileHash -Path $binaryPath -Algorithm SHA256).Hash.ToLower()
  if ($expected -ne $actual) { Die "checksum mismatch: expected $expected, got $actual" }

  # The new binary says which version this is: update, reinstall or older.
  $newVersion = Get-KongtrolVersion $binaryPath
  $kind = "install"
  if ($prevExe) {
    $kind = "update"
    if ($prevVersion -and $newVersion) {
      switch (Compare-KongtrolVersion $prevVersion $newVersion) {
        0 { $kind = "repair" }
        1 { $kind = "downgrade" }
      }
    }
  }
  $shownNew = if ($newVersion) { $newVersion } else { $Version }
  switch ($kind) {
    "update" { Write-Info "Updating Kongtrol $prevVersion -> $shownNew (your VPNs and settings are kept)..." }
    "repair" { Write-Info "Reinstalling Kongtrol $newVersion (repair: your VPNs and settings are kept)..." }
    "downgrade" {
      Write-Warn "This installs an OLDER version ($prevVersion -> $newVersion). Your VPNs and settings are kept."
      $interactive = [Environment]::UserInteractive -and -not [Console]::IsInputRedirected
      if (-not $AssumeYes -and $interactive) {
        $answer = Read-Host "Install the older version? [y/N]"
        if ($answer -notmatch '^(y|yes|s|si)$') {
          Write-Info "Nothing changed. To install it anyway, set `$env:KONGTROL_YES=1 and run the same command."
          exit 0
        }
      }
    }
  }

  $installDir = if ($env:KONGTROL_INSTALL_DIR) { $env:KONGTROL_INSTALL_DIR }
    elseif ($prevExe) { Split-Path -Parent $prevExe }
    else { $defaultDir }
  New-Item -ItemType Directory -Path $installDir -Force | Out-Null

  $destPath = Join-Path $installDir "kongtrol.exe"
  # A running daemon locks kongtrol.exe against overwriting, not renaming:
  # move the old copy aside, put the new one in place, then stop and resume below.
  $oldPath = "$destPath.old"
  if (Test-Path $destPath) {
    Remove-Item $oldPath -Force -ErrorAction SilentlyContinue
    Move-Item -Path $destPath -Destination $oldPath -Force
  }
  Move-Item -Path $binaryPath -Destination $destPath -Force

  Write-Info "Installed to $destPath"

  # What this script installed, so `kongtrol uninstall` removes exactly that.
  $lines = @("# Written by install.ps1: files kongtrol uninstall may remove.", "file=$destPath")

  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added $installDir to your user PATH. Restart your terminal for it to take effect." -ForegroundColor Yellow
    $lines += "path=$installDir"
  } elseif ($previous -contains "path=$installDir") {
    $lines += "path=$installDir"  # added by an earlier run of this script
  }
  New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
  Set-Content -Path $manifest -Value $lines -Encoding UTF8

  if ($kind -ne "install") {
    # Stop the old Kongtrol and save which VPNs were connected, BEFORE the
    # helper update (a helper restart could drop a VPN and it would then be
    # forgotten); `resume` below reconnects exactly those on the new version
    # (docs/adr/0050-installer-resume-tunnels-and-open-request.md).
    Write-Info "Stopping Kongtrol for the update (connected VPNs reconnect at the end)..."
    try { & $destPath down --all --quiet --remember *> $null } catch {}
    # Update the network helper only when it speaks an older protocol: a
    # missing or current helper is left alone and Windows asks nothing.
    Write-Info "Checking the network helper (Windows asks for permission only if it is out of date)..."
    & $destPath service install --if-outdated --quiet
    if ($LASTEXITCODE -ne 0) {
      Write-Warn "the network helper was not updated: open Kongtrol and choose `"Actualizar ayudante (una vez)`", or run: kongtrol service install"
    }
    # Reconnect the VPNs saved above (nothing when none was connected).
    try {
      & $destPath resume --quiet *> $null
      if ($LASTEXITCODE -eq 0) { Write-Info "Kongtrol is reconnecting the VPNs you had connected (if any). Check with: kongtrol status" }
      else { Write-Warn "your VPNs were not reconnected: open Kongtrol and connect them, or run: kongtrol up" }
    } catch {}
  }
  # Gone once the old daemon exited; a leftover is harmless and replaced next time.
  Remove-Item $oldPath -Force -ErrorAction SilentlyContinue

  switch ($kind) {
    "update" { Write-Info "Kongtrol updated to ${shownNew}: your VPNs and settings are kept. Next: kongtrol up" }
    "repair" { Write-Info "Kongtrol $newVersion reinstalled: your VPNs and settings are kept. Next: kongtrol up" }
    "downgrade" { Write-Info "Kongtrol $newVersion installed: your VPNs and settings are kept. Next: kongtrol up" }
  }
  & $destPath version 2>$null
} finally {
  Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
}
}

Invoke-KongtrolInstall
