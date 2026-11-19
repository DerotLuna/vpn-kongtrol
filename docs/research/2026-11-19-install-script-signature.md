<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Verifying the release signature in the install scripts

Version 1.0.0 · 2026-11-19

Used by: [ADR-0056](../adr/0056-installers-verify-release-signature.md). All sources accessed
2026-11-19.

## Findings

- **What the release publishes.** `.goreleaser.yaml` `signs:` signs `artifacts: checksum` with
  `cosign sign-blob --bundle=${signature}`, `signature: "${artifact}.sigstore.json"`. Release
  assets checked with `gh release view`: v0.6.0-rc.2 … rc.5 have `checksums.txt` and
  `checksums.txt.sigstore.json`; v0.5.0 has only `checksums.txt`. The bundle's `mediaType` is
  `application/vnd.dev.sigstore.bundle.v0.3+json`. (v0.6.0-rc.6 did not exist yet.)
- **cosign `verify-blob`.** Takes `--bundle <path>`, a certificate identity
  (`--certificate-identity` or `--certificate-identity-regexp`, Go RE2) and an issuer
  (`--certificate-oidc-issuer`, e.g. `https://token.actions.githubusercontent.com`); either form of
  each is required for keyless verification.
  https://github.com/sigstore/cosign/blob/main/doc/cosign_verify-blob.md
- **Bundle format by version.** v2.4.0 "begins … support for the newer Sigstore
  specification-compliant bundle format"; v2.6.0 shows
  `cosign verify-blob --new-bundle-format --bundle sigstore.json …`; v3.0.0 "Default to using the
  new protobuf format" with the new capabilities "on by default".
  https://github.com/sigstore/cosign/blob/main/CHANGELOG.md
  The `--new-bundle-format` flag is listed in the `verify-blob` docs of v2.4.0 and v2.6.1
  (`doc/cosign_verify-blob.md` at those tags) and is no longer listed by `cosign verify-blob --help`
  in v3.1.4 (checked locally).
- **Identity in the certificate.** The SAN of the rc.5 bundle is
  `https://github.com/DerotLuna/vpn-kongtrol/.github/workflows/release.yml@refs/tags/v0.6.0-rc.5`
  (printed by cosign on a deliberate mismatch), so the regexp
  `^https://github\.com/DerotLuna/vpn-kongtrol/\.github/workflows/release\.yml@refs/tags/v` pins
  repo, workflow file and a `v*` tag.
- **GitHub attestations.** `release.yml` attests `dist/checksums.txt` among others
  (`actions/attest-build-provenance`); `gh attestation verify checksums.txt --repo
  DerotLuna/vpn-kongtrol` exits 0 for rc.5 and 1 for a modified file (signed-in `gh`).
  https://cli.github.com/manual/gh_attestation_verify
- **Same pattern upstream.** GoReleaser's signing docs: "First, verify the checksum file's
  signature against the publisher's expected identity and OIDC issuer" with
  `cosign verify-blob --certificate-identity … --certificate-oidc-issuer
  'https://token.actions.githubusercontent.com' --bundle checksums.txt.sigstore.json checksums.txt`,
  "Only after that succeeds, verify the downloaded artifacts" with `sha256sum --check`.
  https://goreleaser.com/customization/sign/

## Test runs (cosign v3.1.4, Windows 11, Git Bash sh + PowerShell 7 + Windows PowerShell 5.1)

The scripts' signature blocks were run on their own (extracted from the files) against the
published assets; a full install was not run.

| Case | Result |
|---|---|
| rc.5, genuine | `Verified OK` → "Signature OK" |
| rc.2, genuine | "Signature OK" |
| rc.5, `checksums.txt` with one extra line | `invalid signature when validating ASN.1 encoded signature` → stop |
| identity regexp of another repo | `no matching CertificateIdentity found` → would stop |
| v0.5.0 (no bundle) | 404 → stop; with `KONGTROL_SKIP_SIGNATURE=1` → warning, continues |
| no cosign on PATH | one info line with the manual command |
| fake cosign reporting v2.2.4 / devel | "too old" warning; v2.4.0/v2.6.1 → `--new-bundle-format`; v3.x → no flag |

Not run: a real cosign 2.4–2.x binary; macOS / Linux hosts (the sh block ran under Git Bash with
a Windows cosign).
