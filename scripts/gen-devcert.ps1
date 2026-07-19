<#
Windows-only. Generates a local, self-signed Authenticode code-signing certificate
for Kongtrol development builds, and trusts it on this machine only.

Scope / safety, so it's clear this isn't doing anything invasive:
  - No admin/elevation required — everything happens under Cert:\CurrentUser, not
    Cert:\LocalMachine. It cannot affect other user accounts on this machine.
  - No network calls. The cert is generated and self-signed entirely offline.
  - Nothing here is Linux/macOS-relevant: Authenticode signing only applies to
    Windows PE binaries, so this script has no equivalent on those platforms and
    is simply not part of their build/verify flow. `make sign` already no-ops on
    non-Windows for the same reason.
  - Reversible: delete $HOME/.kongtrol/codesign/ and remove the cert from
    Cert:\CurrentUser\My, Root, and TrustedPublisher (certmgr.msc, or
    `certutil -user -delstore Root/TrustedPublisher <thumbprint>`) to undo it.

This does NOT remove SmartScreen/Defender warnings for other users who download a
prebuilt binary — a self-signed cert has no chain to a public CA, so Windows on
another machine has no reason to trust it. What it does:
  - Lets you locally sign your own builds so this machine stops flagging binaries
    you just compiled (matching hash + matching, now-trusted signer).
  - Gives `make sign` something to sign with, so the release pipeline is already
    wired for a real CA-issued cert if/when the project gets one — swap the .pfx,
    nothing else changes.

Do NOT sign binaries meant for distribution (releases, landing page downloads)
with this cert — it gives everyone but you a false sense of "signed" without
actually vouching for anything. Release artifacts should stay unsigned until
there's a real CA cert; see docs/SECURITY.md.

Usage:
    pwsh scripts/gen-devcert.ps1

Output (not committed to git — see .gitignore):
    $HOME/.kongtrol/codesign/kongtrol-devsign.pfx           (private key, password-protected)
    $HOME/.kongtrol/codesign/kongtrol-devsign.pfx.password  (random password, local only)
    $HOME/.kongtrol/codesign/kongtrol-devsign.cer           (public cert)
#>

$ErrorActionPreference = "Stop"

$dir = Join-Path $HOME ".kongtrol\codesign"
New-Item -ItemType Directory -Force -Path $dir | Out-Null

$pfxPath = Join-Path $dir "kongtrol-devsign.pfx"
$cerPath = Join-Path $dir "kongtrol-devsign.cer"
$pwPath  = Join-Path $dir "kongtrol-devsign.pfx.password"

if (Test-Path $pfxPath) {
    Write-Host "Dev signing cert already exists at $pfxPath — nothing to do."
    Write-Host "Delete it first if you want to regenerate."
    exit 0
}

$cert = New-SelfSignedCertificate `
    -Type CodeSigningCert `
    -Subject "CN=Kongtrol Dev, O=Kongtrol Open Source, C=MX" `
    -KeyUsage DigitalSignature `
    -FriendlyName "Kongtrol Code Signing (self-signed, dev)" `
    -CertStoreLocation "Cert:\CurrentUser\My" `
    -NotAfter (Get-Date).AddYears(5) `
    -KeyExportPolicy Exportable `
    -KeyAlgorithm RSA `
    -KeyLength 2048

$plainPw = -join ((48..57) + (65..90) + (97..122) | Get-Random -Count 32 | ForEach-Object { [char]$_ })
$securePw = ConvertTo-SecureString -String $plainPw -Force -AsPlainText

Export-PfxCertificate -Cert $cert -FilePath $pfxPath -Password $securePw | Out-Null
Set-Content -Path $pwPath -Value $plainPw -NoNewline
Export-Certificate -Cert $cert -FilePath $cerPath | Out-Null

# Trust it locally so signed binaries stop triggering SmartScreen/Defender on this machine.
certutil -user -addstore Root "$cerPath" | Out-Null
certutil -user -addstore TrustedPublisher "$cerPath" | Out-Null

Write-Host "Generated and trusted dev signing cert:"
Write-Host "  $pfxPath"
Write-Host "Thumbprint: $($cert.Thumbprint)"
Write-Host ""
Write-Host "Next: make build sign"
