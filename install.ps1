# Installs or updates Kongtrol (tray app + kongtrol CLI) from the latest GitHub release.
# Version 1.8.0 · 2026-11-20 · docs/adr/0075-install-scripts-install-cli-and-tray.md, docs/adr/0073-installers-stop-when-the-new-binary-cannot-run.md, docs/adr/0063-kill-switch-lan-protected-read-and-release-hardening.md, docs/adr/0057-installers-verify-release-signature.md, docs/adr/0034-uninstall-and-update-notice.md, docs/adr/0046-installers-detect-existing-install.md, docs/adr/0050-installer-resume-tunnels-and-open-request.md
#
#   iwr https://kongtrol.app/install.ps1 -useb | iex
#
# What it installs (docs/adr/0075-install-scripts-install-cli-and-tray.md):
# kongtrol.exe and kongtrol-tray.exe in %LOCALAPPDATA%\kongtrol\bin, a
# "Kongtrol" Start menu shortcut and the per-user autostart (HKCU Run value
# "Kongtrol", the same one the Windows installer uses), then opens Kongtrol.
# Only the command line: -CliOnly (or $env:KONGTROL_CLI_ONLY=1); Windows
# Server Core (no desktop) gets only the command line by itself. Updates keep
# that choice; -WithTray (or $env:KONGTROL_CLI_ONLY=0) adds the tray later.
# With flags, run it as:
#   & ([scriptblock]::Create((iwr https://kongtrol.app/install.ps1 -useb))) -CliOnly
# When Kongtrol was installed with the Windows installer (Kongtrol-Setup),
# this script installs nothing and says how to update that copy instead.
#
# Run it again to update: an existing install is detected and the script says
# "Updating <old> -> <new>" (or "Reinstalling", or warns before an older
# version, asking only in an interactive session; -Yes or $env:KONGTROL_YES=1
# answers for scripts). It reuses the previous folder, keeps ~/.kongtrol, and
# updates the network helper only when it is outdated. When Windows does not
# let the downloaded kongtrol.exe start (an antivirus false positive), it
# stops before touching the installed copy and says what to do.
#
# Uninstall (your data stays unless you add -Purge):
#   & ([scriptblock]::Create((iwr https://kongtrol.app/install.ps1 -useb))) -Uninstall [-Purge]
#   or: $env:KONGTROL_UNINSTALL=1 (and $env:KONGTROL_PURGE=1) before the iwr | iex line
#
# Env overrides:
#   $env:KONGTROL_INSTALL_DIR   install directory (default: the previous one, else $env:LOCALAPPDATA\kongtrol\bin)
#   $env:KONGTROL_VERSION       release tag to install, e.g. v1.2.3 (default: latest)
#   $env:KONGTROL_YES           1 = same as -Yes
#   $env:KONGTROL_CLI_ONLY      1 = same as -CliOnly, 0 = same as -WithTray
#   $env:KONGTROL_NO_AUTOSTART  1 = no autostart for the tray (remembered on updates)
#   $env:KONGTROL_NO_LAUNCH     1 = don't open Kongtrol at the end
#   $env:KONGTROL_SKIP_SIGNATURE 1 = don't check the cosign signature of checksums.txt
#                               (only for releases before v0.6.0, which were not signed)
#
# Integrity: the download is checked against checksums.txt. When cosign 2.4+
# is installed, the Sigstore signature of checksums.txt is checked first (it
# must come from this repo's release workflow) and a bad signature stops the
# install. Without cosign the script says so and how to verify by hand.
param([switch]$Uninstall, [switch]$Purge, [switch]$Yes, [switch]$CliOnly, [switch]$WithTray)

$ErrorActionPreference = "Stop"

$Repo = "DerotLuna/vpn-kongtrol"
$Version = if ($env:KONGTROL_VERSION) { $env:KONGTROL_VERSION } else { "latest" }
$AssumeYes = $Yes -or $env:KONGTROL_YES -eq "1"
$SecurityDoc = "https://github.com/$Repo/blob/main/docs/SECURITY.md#unsigned-binaries-and-antivirus-false-positives"

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

# Test-KongtrolExe: runs `<exe> version` (it prints only the version when its
# output is redirected) and tells "it ran" apart from "it could not start"
# (docs/adr/0073-installers-stop-when-the-new-binary-cannot-run.md). Returns
# Ran (started and exited 0), Version (or $null), Blocked (an antivirus
# stopped it: Win32 error 225 ERROR_VIRUS_INFECTED or 226 ERROR_VIRUS_DELETED,
# or the file was gone right after the attempt, i.e. quarantined) and Reason.
# The tray answers the same way to `kongtrol-tray.exe --version` ($Arg).
function Test-KongtrolExe([string]$Exe, [string]$Arg = "version") {
  $r = [pscustomobject]@{ Ran = $false; Version = $null; Blocked = $false; Reason = "" }
  if (-not (Test-Path -LiteralPath $Exe)) {
    $r.Blocked = $true; $r.Reason = "the file is gone (moved to quarantine?)"; return $r
  }
  try {
    # Pipeline form (it gives the Win32 error code, ADR-0073), read to the
    # end: Select-Object -First 1 here stopped the program before its exit
    # code was set (PowerShell 7 on Linux, measured).
    $out = @(& $Exe $Arg 2>$null | ForEach-Object { "$_" }) | Select-Object -First 1
    if ($LASTEXITCODE -eq 0) { $r.Ran = $true } else { $r.Reason = "it exited with code $LASTEXITCODE" }
    if ($out) { $v = ($out.ToString().Trim() -replace '^[vV]', ''); if ($v -match '^[0-9]') { $r.Version = $v } }
  } catch {
    $r.Reason = ($_.Exception.Message -replace '(?s)\s*At line:.*$', '')
    for ($e = $_.Exception; $e; $e = $e.InnerException) {
      if ($e -is [ComponentModel.Win32Exception]) {
        $r.Reason = "$($e.Message) (Windows error $($e.NativeErrorCode))"
        if ($e.NativeErrorCode -in 225, 226) { $r.Blocked = $true }
        break
      }
    }
  }
  if (-not $r.Ran -and -not (Test-Path -LiteralPath $Exe)) { $r.Blocked = $true }
  return $r
}

# Test-KongtrolTray: the same check for kongtrol-tray.exe. Reading the file
# back (its SHA256 must still be $Hash) makes the antivirus scan it without
# starting it; a blocked read is Win32 error 225/226 like a blocked start.
# Only a tray that knows --version (releases after 0.7.0-rc.3, $CanRun) is
# also started with it: an older tray ignores the flag and would open
# (docs/adr/0075-install-scripts-install-cli-and-tray.md).
function Test-KongtrolTray([string]$Exe, [string]$Hash, [bool]$CanRun) {
  $r = [pscustomobject]@{ Ran = $false; Version = $null; Blocked = $false; Reason = "" }
  if (-not (Test-Path -LiteralPath $Exe)) {
    $r.Blocked = $true; $r.Reason = "the file is gone (moved to quarantine?)"; return $r
  }
  try {
    $h = (Get-FileHash -LiteralPath $Exe -Algorithm SHA256).Hash.ToLower()
    if ($h -ne $Hash) { $r.Reason = "the file changed after it was checked"; return $r }
  } catch {
    $r.Reason = ($_.Exception.Message -replace '(?s)\s*At line:.*$', '')
    for ($e = $_.Exception; $e; $e = $e.InnerException) {
      if ($e -is [ComponentModel.Win32Exception] -or $e -is [IO.IOException]) {
        $code = if ($e -is [ComponentModel.Win32Exception]) { $e.NativeErrorCode } else { $e.HResult -band 0xFFFF }
        if ($code -in 225, 226) { $r.Blocked = $true; $r.Reason = "$($e.Message) (Windows error $code)"; break }
      }
    }
    if (-not (Test-Path -LiteralPath $Exe)) { $r.Blocked = $true }
    return $r
  }
  if ($CanRun) { return Test-KongtrolExe $Exe "--version" }
  $r.Ran = $true
  return $r
}

# Test-ServerCore: Windows Server without the desktop shell (Server Core or
# Nano Server), where a tray icon has nowhere to show. Microsoft's
# documented check for Windows Server 2012 and later: the ServerLevels
# registry values (ServerCore=1 without Server-Gui-Shell=1, or NanoServer=1).
# Absent on Windows 10/11, so every desktop gets the tray.
function Test-ServerCore {
  $l = Get-ItemProperty -LiteralPath 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Server\ServerLevels' -ErrorAction SilentlyContinue
  if (-not $l) { return $false }
  if ($l.NanoServer -eq 1) { return $true }
  return ($l.ServerCore -eq 1 -and $l.'Server-Gui-Shell' -ne 1)
}

# Write-CannotRun: what happened and what to do when Windows does not let a
# kongtrol.exe start. It never turns Defender off or adds exclusions: the
# user decides, in Windows Security, after checking the file.
function Write-CannotRun($Probe, [string]$What) {
  if ($Probe.Blocked) {
    Write-Host "error: Windows blocked $What (Microsoft Defender or another antivirus reported it as a threat): $($Probe.Reason)" -ForegroundColor Red
    Write-Host "It is the official Kongtrol file: its SHA256 matches the release's checksums.txt. New programs without a paid code-signing certificate are a known false positive (for example Trojan:Win32/Bearfoos.A!ml)." -ForegroundColor Yellow
  } else {
    Write-Host "error: Windows did not let $What start: $($Probe.Reason)" -ForegroundColor Red
    Write-Host "Its SHA256 matches the release's checksums.txt, so the download is complete and unchanged. Security software or a policy (Smart App Control, AppLocker) may be stopping it." -ForegroundColor Yellow
  }
  Write-Host "What to do: check the release yourself (gh attestation verify checksums.txt --repo $Repo, or cosign: $SecurityDoc), then in Windows Security > Virus & threat protection > Protection history open the Kongtrol entry and choose Allow on device (or Restore) only if you trust it, and run the same command again. Please report the false positive: https://www.microsoft.com/wdsi/filesubmission" -ForegroundColor Yellow
}

# Test-UserWritableDir: this user can create a file in $Dir (no elevation).
function Test-UserWritableDir([string]$Dir) {
  if (-not $Dir -or -not (Test-Path -LiteralPath $Dir -PathType Container)) { return $false }
  $probe = Join-Path $Dir ".kongtrol-write-test-$(Get-Random)"
  try {
    [IO.File]::WriteAllText($probe, "")
    Remove-Item -LiteralPath $probe -Force -ErrorAction SilentlyContinue
    return $true
  } catch { return $false }
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
if ($CliOnly -and $WithTray) { Die "use -CliOnly or -WithTray, not both" }

# Kongtrol from the Windows installer (Kongtrol-Setup) owns its folder, its
# shortcut, the autostart value and its own uninstaller: a second copy from
# this script would fight it over the same daemon, autostart value and PATH.
# Its uninstall key is the one kongtrol.iss reads in InitializeSetup
# (Inno names it "<AppId>_is1"; PrivilegesRequired=lowest puts it in HKCU).
$innoKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\{BA277280-D8B2-4309-8620-BCEECBAA4B41}_is1'
$inno = Get-ItemProperty -LiteralPath $innoKey -ErrorAction SilentlyContinue
if ($inno) {
  $where = if ($inno.InstallLocation) { " in $($inno.InstallLocation.TrimEnd('\'))" } else { "" }
  Write-Host "error: Kongtrol $($inno.DisplayVersion) is already installed with the Windows installer (Kongtrol-Setup)$where, so this script installs nothing (it would be a second copy)." -ForegroundColor Red
  Write-Host "What to do: to update it, download and run the new Kongtrol-Setup-windows-amd64.exe from https://kongtrol.app or https://github.com/$Repo/releases/latest (your VPNs and settings are kept; for scripts add /VERYSILENT). To use this script instead, first uninstall Kongtrol in Windows Settings > Apps (your data stays), then run this command again." -ForegroundColor Yellow
  exit 1
}

# Windows-only steps (Start menu, autostart, PATH, opening the tray) are
# skipped when PowerShell runs on another OS, as the offline tests do.
$onWindows = ($PSVersionTable.PSEdition -ne 'Core') -or $IsWindows

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { Die "unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)" }
}

$baseUrl = if ($Version -eq "latest") {
  "https://github.com/$Repo/releases/latest/download"
} else {
  "https://github.com/$Repo/releases/download/$Version"
}

# Existing install (docs/adr/0046-installers-detect-existing-install.md): the
# manifest an earlier run wrote says where kongtrol.exe went, so an update
# reuses that folder instead of leaving a second copy behind. The manifest
# is a file any program of this user can rewrite and the update runs the exe
# it finds there with a permission prompt (service install), so its folder
# is reused only when this user can already write it (as the default
# %LOCALAPPDATA% folder): then nothing is gained by rewriting the manifest.
# Otherwise KONGTROL_INSTALL_DIR chooses it explicitly (ADR-0063).
$dataDir = Join-Path $HOME ".kongtrol"
$manifest = Join-Path $dataDir "install.manifest"
$previous = if (Test-Path $manifest) { @(Get-Content $manifest) } else { @() }
$prevExe = $previous | Where-Object { $_ -like "file=*kongtrol.exe" } | ForEach-Object { $_.Substring(5) } | Select-Object -First 1
$defaultDir = Join-Path $env:LOCALAPPDATA "kongtrol\bin"
if ($prevExe) {
  $prevDir = try { [IO.Path]::GetFullPath((Split-Path -Parent $prevExe)).TrimEnd('\') } catch { "" }
  if (-not (Test-UserWritableDir $prevDir)) {
    if (-not $env:KONGTROL_INSTALL_DIR) {
      Write-Warn "the last install was in $prevDir, which you cannot write without administrator rights, so it is not reused. To install there again, set `$env:KONGTROL_INSTALL_DIR=`"$prevDir`" first"
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
# The previous version: from the old exe, else from the manifest (an exe an
# antivirus now blocks cannot say it), so an older download is still seen
# as one (ADR-0073).
$prevVersion = $null
$prevUnreadable = ""
if ($prevExe) {
  $prevProbe = Test-KongtrolExe $prevExe
  $prevVersion = $prevProbe.Version
  if (-not $prevVersion) {
    $prevVersion = $previous | Where-Object { $_ -match '^version=[0-9]' } | ForEach-Object { $_.Substring(8).Trim() } | Select-Object -First 1
    $prevUnreadable = if ($prevProbe.Reason) { $prevProbe.Reason } else { "it printed no version" }
  }
}

# CLI + tray (default) or only the CLI (docs/adr/0075-install-scripts-install-cli-and-tray.md):
# a flag or KONGTROL_CLI_ONLY decides; otherwise an update keeps what the
# manifest says the user chose ("components=cli"), and an earlier install
# with the tray keeps it; a fresh install on Server Core gets only the CLI.
# Manifests from before 1.8.0 list only kongtrol.exe (there was no choice),
# so those updates add the tray like a new install.
$prevComponents = $previous | Where-Object { $_ -match '^components=' } | ForEach-Object { $_.Substring(11).Trim() } | Select-Object -First 1
$prevTray = $previous | Where-Object { $_ -like "file=*kongtrol-tray.exe" } | ForEach-Object { $_.Substring(5) } | Select-Object -First 1
$prevLnk = $previous | Where-Object { $_ -like "file=*Kongtrol.lnk" } | ForEach-Object { $_.Substring(5) } | Select-Object -First 1
$choice = "default"
if ($CliOnly -or $env:KONGTROL_CLI_ONLY -eq "1") { $withTray = $false; $choice = "flag" }
elseif ($WithTray -or $env:KONGTROL_CLI_ONLY -eq "0") { $withTray = $true; $choice = "flag" }
elseif ($prevComponents -eq "cli") { $withTray = $false; $choice = "kept" }
elseif ($prevTray) { $withTray = $true; $choice = "kept" }
elseif (Test-ServerCore) { $withTray = $false; $choice = "auto" }
else { $withTray = $true }
# Autostart: KONGTROL_NO_AUTOSTART, else the previous choice (an install
# with the tray but no runvalue= line had it off).
$prevNoAutostart = if ($prevTray -and -not ($previous | Where-Object { $_ -like "runvalue=*" })) { "1" } else { "0" }
$noAutostart = if ($env:KONGTROL_NO_AUTOSTART) { $env:KONGTROL_NO_AUTOSTART } else { $prevNoAutostart }
$addTrayHint = "To add the Kongtrol app (tray icon) later: `$env:KONGTROL_CLI_ONLY=0; iwr https://kongtrol.app/install.ps1 -useb | iex"
if ($choice -eq "auto") {
  Write-Info "No desktop on this Windows (Server Core): installing only the command line (kongtrol). $addTrayHint"
}

# The tray ships only inside the release zip (kongtrol.exe + kongtrol-tray.exe);
# the CLI alone is also a plain .exe asset.
$asset = if ($withTray) { "kongtrol_windows_$arch.zip" } else { "kongtrol-windows-$arch.exe" }
$binary = "kongtrol-windows-$arch.exe"

$tmpDir = Join-Path $env:TEMP "kongtrol-install-$(Get-Random)"
New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
try {
  $assetPath = Join-Path $tmpDir $asset
  Write-Info "Downloading $asset ($Version)..."
  try {
    Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile $assetPath -UseBasicParsing
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

  $expectedLine = Select-String -Path $checksumsPath -SimpleMatch -Pattern "  $asset" | Where-Object { $_.Line.EndsWith("  $asset") } | Select-Object -First 1
  if (-not $expectedLine) { Die "no checksum entry found for $asset" }
  $expected = ($expectedLine.Line -split '\s+')[0]

  $actual = (Get-FileHash -Path $assetPath -Algorithm SHA256).Hash.ToLower()
  if ($expected -ne $actual) { Die "checksum mismatch: expected $expected, got $actual" }

  # The zip is trusted now (its SHA256 matched): take exactly the two
  # programs out of it, by name, nothing else.
  $binaryPath = $assetPath
  $trayPath = $null
  if ($withTray) {
    $xDir = Join-Path $tmpDir "x"
    New-Item -ItemType Directory -Path $xDir -Force | Out-Null
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [IO.Compression.ZipFile]::OpenRead($assetPath)
    try {
      foreach ($name in @("kongtrol.exe", "kongtrol-tray.exe")) {
        $entry = $zip.Entries | Where-Object { $_.FullName -eq $name } | Select-Object -First 1
        if (-not $entry) { Die "the release archive $asset has no $name, so nothing was installed. Report it: https://github.com/$Repo/issues" }
        [IO.Compression.ZipFileExtensions]::ExtractToFile($entry, (Join-Path $xDir $name), $true)
      }
    } finally { $zip.Dispose() }
    $binaryPath = Join-Path $xDir "kongtrol.exe"
    $trayPath = Join-Path $xDir "kongtrol-tray.exe"
    $trayHash = (Get-FileHash -LiteralPath $trayPath -Algorithm SHA256).Hash.ToLower()
  }

  # The new binary says which version this is: update, reinstall or older.
  # It must also start: when Windows blocks it (an antivirus false positive
  # on an unsigned binary, ADR-0073) stop here, before the installed copy is
  # touched, instead of installing a file that cannot run.
  $newProbe = Test-KongtrolExe $binaryPath
  if (-not $newProbe.Ran) {
    Write-CannotRun $newProbe "the Kongtrol it just downloaded ($binary)"
    if ($prevExe) { Die "nothing was installed: your current Kongtrol$(if ($prevVersion) { " $prevVersion" }) was not changed." }
    Die "nothing was installed."
  }
  $newVersion = $newProbe.Version
  # The tray must not be blocked either (ADR-0073 for the tray, ADR-0075).
  $trayCanRun = [bool]($newVersion -and (Compare-KongtrolVersion $newVersion "0.7.0-rc.3") -gt 0)
  if ($withTray) {
    $trayProbe = Test-KongtrolTray $trayPath $trayHash $trayCanRun
    if (-not $trayProbe.Ran) {
      Write-CannotRun $trayProbe "the Kongtrol tray app it just downloaded (kongtrol-tray.exe)"
      Write-Host "Meanwhile you can install only the command line: `$env:KONGTROL_CLI_ONLY=1, then the same command." -ForegroundColor Yellow
      if ($prevExe) { Die "nothing was installed: your current Kongtrol$(if ($prevVersion) { " $prevVersion" }) was not changed." }
      Die "nothing was installed."
    }
  }
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
  if ($prevExe -and $prevUnreadable) {
    Write-Warn "could not read the installed version from $prevExe ($prevUnreadable)$(if ($prevVersion) { "; the last install recorded $prevVersion" })."
  }
  switch ($kind) {
    "update" {
      if ($prevVersion) { Write-Info "Updating Kongtrol $prevVersion -> $shownNew (your VPNs and settings are kept)..." }
      else { Write-Info "Updating Kongtrol to $shownNew (your VPNs and settings are kept)..." }
    }
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
  $trayDest = Join-Path $installDir "kongtrol-tray.exe"
  # Close this copy's running tray (it would keep the old file in use and
  # restart the old daemon); it is opened again at the end.
  Get-Process -Name kongtrol-tray -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -and $_.Path -ieq $trayDest } |
    Stop-Process -Force -ErrorAction SilentlyContinue
  # A running daemon locks kongtrol.exe against overwriting, not renaming:
  # move the old copy aside, put the new one in place, then stop and resume below.
  $payload = @(@{ Src = $binaryPath; Dest = $destPath })
  if ($withTray) { $payload += @{ Src = $trayPath; Dest = $trayDest } }
  foreach ($p in $payload) {
    if (Test-Path $p.Dest) {
      Remove-Item "$($p.Dest).old" -Force -ErrorAction SilentlyContinue
      Move-Item -Path $p.Dest -Destination "$($p.Dest).old" -Force
    }
    Move-Item -Path $p.Src -Destination $p.Dest -Force
  }
  $oldPath = "$destPath.old"

  # Restore-Previous: put the old copies (and the old manifest) back when
  # the new ones cannot run; a tray that was not there before is removed.
  function Restore-Previous {
    if (-not (Test-Path $oldPath)) { return $false }
    foreach ($p in $payload) {
      Remove-Item $p.Dest -Force -ErrorAction SilentlyContinue
      if (Test-Path "$($p.Dest).old") {
        try { Move-Item -Path "$($p.Dest).old" -Destination $p.Dest -Force } catch { return $false }
      }
    }
    if ($previous.Count) { Set-Content -Path $manifest -Value $previous -Encoding UTF8 }
    return $true
  }
  $placed = Test-KongtrolExe $destPath
  if (-not $placed.Ran) {
    Write-CannotRun $placed "the new kongtrol.exe in $installDir"
    if (Restore-Previous) { Die "nothing was installed: your previous Kongtrol$(if ($prevVersion) { " $prevVersion" }) is back in place." }
    Die "Kongtrol cannot be used until Windows lets it run."
  }
  if ($withTray) {
    $placedTray = Test-KongtrolTray $trayDest $trayHash $trayCanRun
    if (-not $placedTray.Ran) {
      Write-CannotRun $placedTray "the new kongtrol-tray.exe in $installDir"
      if (Restore-Previous) { Die "nothing was installed: your previous Kongtrol$(if ($prevVersion) { " $prevVersion" }) is back in place." }
      Remove-Item $trayDest -Force -ErrorAction SilentlyContinue
      Die "the Kongtrol tray app cannot be used until Windows lets it run. Meanwhile: `$env:KONGTROL_CLI_ONLY=1, then the same command, installs only the command line."
    }
    Write-Info "Installed to $destPath and $trayDest"
  } else {
    Write-Info "Installed to $destPath"
  }

  # What this script installed, so `kongtrol uninstall` removes exactly that.
  $lines = @("# Written by install.ps1: files kongtrol uninstall may remove.", "file=$destPath")
  # The installed version, read back when that exe cannot say it (ADR-0073).
  if ($newVersion) { $lines += "version=$newVersion" }
  # The user's choice, kept by updates (ADR-0075): "cli" only when asked for.
  $lines += if ($withTray) { "components=cli,tray" } elseif ($choice -eq "auto") { "components=auto-cli" } else { "components=cli" }
  $lnkPath = $null
  if ($withTray) {
    $lines += "file=$trayDest"
    if ($onWindows) {
      $lnkPath = Join-Path ([Environment]::GetFolderPath('Programs')) "Kongtrol.lnk"
      $lines += "file=$lnkPath"
      if ($noAutostart -ne "1") { $lines += "runvalue=$trayDest" }
    }
  }

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
  # Success only when the installed copy still runs: an antivirus can
  # quarantine it after its first start (ADR-0073).
  $final = Test-KongtrolExe $destPath
  if (-not $final.Ran) {
    Write-CannotRun $final "the installed kongtrol.exe"
    if (Restore-Previous) { Die "the update was undone: your previous Kongtrol$(if ($prevVersion) { " $prevVersion" }) is back in place. Next: kongtrol up" }
    Die "Kongtrol cannot be used until Windows lets it run."
  }
  # Gone once the old daemon exited; a leftover is harmless and replaced next time.
  foreach ($p in $payload) { Remove-Item "$($p.Dest).old" -Force -ErrorAction SilentlyContinue }

  if ($withTray -and $onWindows) {
    # Start menu shortcut and autostart, like the Windows installer
    # ({autoprograms}\Kongtrol and HKCU Run "Kongtrol", kongtrol.iss).
    try {
      $ws = New-Object -ComObject WScript.Shell
      $sc = $ws.CreateShortcut($lnkPath)
      $sc.TargetPath = $trayDest
      $sc.WorkingDirectory = $installDir
      $sc.Description = "Kongtrol"
      $sc.Save()
    } catch { Write-Warn "the Start menu shortcut was not created ($($_.Exception.Message)); open Kongtrol with: $trayDest" }
    $runKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
    if ($noAutostart -ne "1") {
      try {
        if (-not (Test-Path $runKey)) { New-Item -Path $runKey -Force | Out-Null }
        New-ItemProperty -Path $runKey -Name "Kongtrol" -Value "`"$trayDest`"" -PropertyType String -Force | Out-Null
      } catch { Write-Warn "Kongtrol will not start by itself when you sign in ($($_.Exception.Message)); open it from the Start menu." }
    }
  }
  if (-not $withTray -and $prevTray) {
    # Switched to the command line only: the tray this script installed
    # earlier goes, so nothing stale stays (its shortcut and autostart too).
    Get-Process -Name kongtrol-tray -ErrorAction SilentlyContinue |
      Where-Object { $_.Path -and $_.Path -ieq $prevTray } |
      Stop-Process -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $prevTray -Force -ErrorAction SilentlyContinue
    if ($prevLnk) { Remove-Item -LiteralPath $prevLnk -Force -ErrorAction SilentlyContinue }
    if ($onWindows) {
      $runKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
      $run = (Get-ItemProperty -LiteralPath $runKey -ErrorAction SilentlyContinue).Kongtrol
      if ($run -and ($run.Trim('"') -ieq $prevTray)) { Remove-ItemProperty -LiteralPath $runKey -Name "Kongtrol" -ErrorAction SilentlyContinue }
    }
    Write-Info "Removed the Kongtrol tray app installed earlier (you chose only the command line)."
  }
  if ($withTray -and $onWindows -and $env:KONGTROL_NO_LAUNCH -ne "1") {
    # Opens Kongtrol: on a first install the tray starts it and opens the
    # page to add your first VPN (ADR-0025).
    try {
      Start-Process -FilePath $trayDest -WorkingDirectory $installDir
      Write-Info "Kongtrol is open: look for its icon next to the clock (in the ^ overflow if hidden)."
    } catch { Write-Warn "Kongtrol did not open ($($_.Exception.Message)); open it from the Start menu." }
  }

  $next = if ($withTray) { "Next: open Kongtrol from the Start menu or its icon (or: kongtrol up)" } else { "Next: kongtrol up" }
  switch ($kind) {
    "update" { Write-Info "Kongtrol updated to ${shownNew}: your VPNs and settings are kept. $next" }
    "repair" { Write-Info "Kongtrol $newVersion reinstalled: your VPNs and settings are kept. $next" }
    "downgrade" { Write-Info "Kongtrol $newVersion installed: your VPNs and settings are kept. $next" }
    default { Write-Info "Kongtrol $shownNew installed. $next" }
  }
  # Every run ends by saying what is installed and how to get the other part.
  if ($withTray) {
    Write-Info "Installed: the Kongtrol app (tray icon) and the command line (kongtrol). Only the command line? `$env:KONGTROL_CLI_ONLY=1, then the same command."
  } else {
    Write-Info "Installed: only the command line (kongtrol). $addTrayHint"
  }
  if ($final.Version) { Write-Host $final.Version }
} finally {
  Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
}
}

Invoke-KongtrolInstall
