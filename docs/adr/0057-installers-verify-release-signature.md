<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0057: Install scripts verify the Sigstore signature of `checksums.txt` when cosign is present

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer

## Context and Problem Statement

`install.sh` and `install.ps1` check the downloaded binary against `checksums.txt` from the same
release. Someone who could replace release assets (a compromised account or token) could
replace the binary and `checksums.txt` together and the check would still pass. Every release
since v0.6.0-rc.2 also publishes `checksums.txt.sigstore.json`, a keyless Sigstore bundle made by
`.github/workflows/release.yml` (`.goreleaser.yaml` `signs:` `artifacts: checksum`,
`signature: "${artifact}.sigstore.json"`, `sign-blob --bundle`), plus GitHub build provenance
attestations (ADR-0025, ADR-0035). Issue #38.

## Decision Drivers

- Verify against an identity the attacker cannot forge (the workflow's OIDC certificate), not
  against a file shipped next to the binary.
- Never download a verifier automatically (it would need its own trust anchor) and never make
  installing depend on an extra tool: most users have neither cosign nor `gh`.
- A failed verification must stop the install with a "what to do" line (AGENTS.md "User
  guidance"); a skipped one must say so in one line.
- Standards first: cosign's documented keyless `verify-blob` flow, not our own code.

## Considered Options

1. Keep checksum only; document manual verification.
2. **Verify with cosign when it is installed; otherwise checksum only plus a one-line hint.**
3. Verify with `gh attestation verify` when `gh` is installed.
4. Download cosign during install.

## Decision Outcome

Option 2. Sources and test runs in
[the research note](../research/2026-11-19-install-script-signature.md).

- After `checksums.txt` is downloaded, if `cosign` is on `PATH` and is 2.4 or newer, the script
  downloads `checksums.txt.sigstore.json` and runs
  `cosign verify-blob [--new-bundle-format] --bundle checksums.txt.sigstore.json
  --certificate-identity-regexp '^https://github\.com/DerotLuna/vpn-kongtrol/\.github/workflows/release\.yml@refs/tags/v'
  --certificate-oidc-issuer https://token.actions.githubusercontent.com checksums.txt`.
  `--new-bundle-format` is passed only to cosign 2.4–2.x (3.0 made the bundle format the default
  and its `verify-blob` no longer lists the flag). Only then is the binary hashed.
- Signature fails → stop, nothing installed, cosign's last error line, and: offline → try again;
  otherwise do not use the download and report it.
- Bundle missing while cosign is present → stop too (removing the bundle must not downgrade the
  check). Releases before v0.6.0 were not signed, so `KONGTROL_SKIP_SIGNATURE=1` installs one
  anyway, with a warning. An attacker controls the downloads, not the user's environment.
- cosign missing → one line "Signature not checked (cosign is not installed)" with the exact
  manual command and `gh attestation verify checksums.txt --repo DerotLuna/vpn-kongtrol` as the
  alternative. cosign older than 2.4 (or a version it does not report) → same, as a warning
  asking to update cosign.
- Option 3 stays a documented manual alternative only: `gh attestation verify` needs a signed-in
  `gh` (GitHub API), which a one-line installer cannot assume. Option 4 rejected (driver 2).
- Nothing new leaves the machine from Kongtrol itself: cosign is the user's own tool and talks to
  the public Sigstore services (TUF trust root) when it verifies.

### Consequences

- Good: with cosign, a swapped binary + `checksums.txt` is caught; the identity pins this repo,
  this workflow file and a `v*` tag.
- Good: no new dependency, no behavior change for users without cosign besides one line.
- Bad: without cosign the original weakness remains; the line tells the user how to close it.
- Neutral: `KONGTROL_SKIP_SIGNATURE` is a deliberate escape hatch for old releases.

### Confirmation

Run against the real release v0.6.0-rc.5 (and rc.2) with cosign v3.1.4: genuine bundle →
"Signature OK"; tampered `checksums.txt` → stop; v0.5.0 (no bundle) → stop, and installs with
`KONGTROL_SKIP_SIGNATURE=1`; no cosign → hint line. Both `install.sh` (sh) and `install.ps1`
(PowerShell 7 and Windows PowerShell 5.1). The cosign 2.4–2.x path (`--new-bundle-format`) was
checked against the 2.4.0 and 2.6.1 `verify-blob` docs but not run (**no verificado**).

## More Information

- Code: `install.sh` (`COSIGN_IDENTITY`, `cosign_bundle_flag`), `install.ps1` (signature block
  after the `checksums.txt` download); `.goreleaser.yaml` `signs:`.
- Docs: `docs/SECURITY.md` "Verifying releases", `docs/SETUP.md`, README.
- Related: ADR-0025 (distribution), ADR-0035 (install scripts as release assets), issue #38.
