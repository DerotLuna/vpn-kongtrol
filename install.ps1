# Installs or updates the kongtrol CLI from the latest GitHub release.
# Version 1.2.0 · 2026-11-19 · docs/adr/0034-uninstall-and-update-notice.md, docs/adr/0046-installers-detect-existing-install.md
#
#   iwr https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.ps1 -useb | iex
#
# Run it again to update: an existing install is detected and the script says
# "Updating <old> -> <new>" (or "Reinstalling", or warns before an older
# version, asking only in an interactive session; -Yes or $env:KONGTROL_YES=1
# answers for scripts). It reuses the previous folder, keeps ~/.kongtrol, and
# updates the network helper only when it is outdated.
#
# Uninstall (your data stays unless you add -Purge):
#   & ([scriptblock]::Create((iwr https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.ps1 -useb))) -Uninstall [-Purge]
#   or: $env:KONGTROL_UNINSTALL=1 (and $env:KONGTROL_PURGE=1) before the iwr | iex line
#
# Env overrides:
#   $env:KONGTROL_INSTALL_DIR   install directory (default: the previous one, else $env:LOCALAPPDATA\kongtrol\bin)
#   $env:KONGTROL_VERSION       release tag to install, e.g. v1.2.3 (default: latest)
#   $env:KONGTROL_YES           1 = same as -Yes
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
# reuses that folder instead of leaving a second copy behind.
$dataDir = Join-Path $HOME ".kongtrol"
$manifest = Join-Path $dataDir "install.manifest"
$previous = if (Test-Path $manifest) { @(Get-Content $manifest) } else { @() }
$prevExe = $previous | Where-Object { $_ -like "file=*kongtrol.exe" } | ForEach-Object { $_.Substring(5) } | Select-Object -First 1
$defaultDir = Join-Path $env:LOCALAPPDATA "kongtrol\bin"
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
  # move the old copy aside, put the new one in place, then restart below.
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
    # Update the network helper only when it speaks an older protocol: a
    # missing or current helper is left alone and Windows asks nothing.
    Write-Info "Checking the network helper (Windows asks for permission only if it is out of date)..."
    & $destPath service install --if-outdated --quiet
    if ($LASTEXITCODE -ne 0) {
      Write-Warn "the network helper was not updated: open Kongtrol and choose `"Actualizar ayudante (una vez)`", or run: kongtrol service install"
    }
    # A running daemon (old version) restarts on the new binary with its
    # connected VPNs; with no daemon running this does nothing.
    try {
      & $destPath restart --quiet *> $null
      if ($LASTEXITCODE -eq 0) { Write-Info "Kongtrol restarted on the new version (your connected VPNs reconnected)." }
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
