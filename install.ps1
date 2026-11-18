# Installs the kongtrol CLI from the latest GitHub release.
# Version 1.1.0 · 2026-11-18 · docs/adr/0033-uninstall-keeps-data-by-default.md
#
#   iwr https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.ps1 -useb | iex
#
# Uninstall (your data stays unless you add -Purge):
#   & ([scriptblock]::Create((iwr https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.ps1 -useb))) -Uninstall [-Purge]
#   or: $env:KONGTROL_UNINSTALL=1 (and $env:KONGTROL_PURGE=1) before the iwr | iex line
#
# Env overrides:
#   $env:KONGTROL_INSTALL_DIR   install directory (default: $env:LOCALAPPDATA\kongtrol\bin)
#   $env:KONGTROL_VERSION       release tag to install, e.g. v1.2.3 (default: latest)
param([switch]$Uninstall, [switch]$Purge)

$ErrorActionPreference = "Stop"

$Repo = "DerotLuna/vpn-kongtrol"
$Version = if ($env:KONGTROL_VERSION) { $env:KONGTROL_VERSION } else { "latest" }

function Write-Info($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Die($msg) { Write-Host "error: $msg" -ForegroundColor Red; exit 1 }

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

  $installDir = if ($env:KONGTROL_INSTALL_DIR) { $env:KONGTROL_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "kongtrol\bin" }
  New-Item -ItemType Directory -Path $installDir -Force | Out-Null

  $destPath = Join-Path $installDir "kongtrol.exe"
  Move-Item -Path $binaryPath -Destination $destPath -Force

  Write-Info "Installed to $destPath"

  # What this script installed, so `kongtrol uninstall` removes exactly that.
  $dataDir = Join-Path $HOME ".kongtrol"
  $manifest = Join-Path $dataDir "install.manifest"
  $previous = if (Test-Path $manifest) { Get-Content $manifest } else { @() }
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

  & $destPath version 2>$null
} finally {
  Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
}
