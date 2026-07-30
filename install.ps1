# Installs the kongtrol CLI from the latest GitHub release.
#
#   iwr https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.ps1 -useb | iex
#
# Env overrides:
#   $env:KONGTROL_INSTALL_DIR   install directory (default: $env:LOCALAPPDATA\kongtrol\bin)
#   $env:KONGTROL_VERSION       release tag to install, e.g. v1.2.3 (default: latest)

$ErrorActionPreference = "Stop"

$Repo = "DerotLuna/vpn-kongtrol"
$Version = if ($env:KONGTROL_VERSION) { $env:KONGTROL_VERSION } else { "latest" }

function Write-Info($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Die($msg) { Write-Host "error: $msg" -ForegroundColor Red; exit 1 }

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

  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added $installDir to your user PATH. Restart your terminal for it to take effect." -ForegroundColor Yellow
  }

  & $destPath version 2>$null
} finally {
  Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
}
