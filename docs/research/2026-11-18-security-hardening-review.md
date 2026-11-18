<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: local hardening review (purge, uninstall, WireGuard binding, update link, CI)

Version 1.0.0 · 2026-11-18

## Question

A source review found several local-attack paths (findings 2–6 and A–C of the 2026-11-18
review). What do the primary sources say for each fix, and what stays unverified?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | wg(8) man page, `show … dump` format | https://git.zx2c4.com/wireguard-tools/about/src/man/wg.8 | 2026-11-18 |
| 2 | RFC 7748 (X25519) | https://www.rfc-editor.org/rfc/rfc7748 | 2026-11-18 |
| 3 | `golang.org/x/crypto/curve25519` | https://pkg.go.dev/golang.org/x/crypto/curve25519 | 2026-11-18 |
| 4 | GitHub Docs, Security hardening for GitHub Actions | https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions | 2026-11-18 |
| 5 | Renovate helper presets (`helpers:pinGitHubActionDigestsToSemver`) | https://docs.renovatebot.com/presets-helpers/ | 2026-11-18 |
| 6 | OWASP Top 10 2021 | https://owasp.org/Top10/ | 2026-11-18 |
| 7 | OWASP Desktop App Security Top 10 | https://owasp.org/www-project-desktop-app-security-top-10/ | 2026-11-18 |
| 8 | Microsoft, well-known SIDs (TrustedInstaller service SID) | `sc.exe showsid` / `LookupAccountName("NT SERVICE\TrustedInstaller")` on Windows 11 | 2026-11-18 |
| 9 | Microsoft, `cmd /?` (`/c` quote handling, `/d` AutoRun) | https://learn.microsoft.com/windows-server/administration/windows-commands/cmd | 2026-11-18 |

## Findings

- **WireGuard binding (1–3).** `wg show <if> dump` prints the interface line as private-key,
  public-key, listen-port, fwmark. A WireGuard public key is X25519(private, basepoint); the
  example key pair of the wg(8) man page (`yAnz5TF…` → `HIgo9xNz…`) is reproduced by
  `curve25519.X25519` (test `TestWGConfigPublicKey_ManPageExample`). `all` and `interfaces`
  are `wg show` keywords, so they cannot be tunnel names for the helper.
- **TrustedInstaller SID (8).** The system answers
  `S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464`; the constant in
  `internal/privops/sys_windows.go` had two wrong sub-authorities (found because the real
  `C:\Program Files\WireGuard` ACL check failed). Fixed with a test that asks the OS.
- **Program Files ACL (verified on this machine).** `C:\Program Files\WireGuard` grants
  FullControl to TrustedInstaller, SYSTEM, Administrators; Users and app packages get
  ReadAndExecute; CREATOR OWNER is inherit-only. `adminOnlyWritable` accepts it.
- **cmd.exe (9).** With `/c`, a command line that starts with a quote and has more than two quotes
  loses its first and last quote, so the whole command is wrapped in one extra pair; `/d` skips
  the AutoRun registry commands. Go's own argument quoting (`\"`) is not cmd's, so the raw
  `SysProcAttr.CmdLine` is used. A bare program name in cmd is searched in the current
  directory first (documented cmd behavior), hence full paths and `Dir = System32`.
- **CI (4, 5).** GitHub: "Pin references to actions to a commit SHA" and verify the SHA comes
  from the action's own repository (not a fork); workflow dependency lockfiles, where
  available, supersede SHA pinning (not adopted yet — **no verificado** for this repo).
  Step-level secret scoping is reasoned (least exposure), not quoted from GitHub. Renovate's
  `helpers:pinGitHubActionDigestsToSemver` pins digests and keeps the `# vX.Y.Z` comment.
- **OWASP mapping (6, 7).** Used for the section "OWASP mapping" in `docs/SECURITY.md`.

## Confidence

High for wg dump format, X25519 and the SID (checked against the OS). Medium for cmd.exe quote
rules (documented, verified by `TestDeleteAfterExitCmd_DeletesFile` with a path with spaces).
Unverified: sudo `HOME` handling per OS (`env_keep`), so `kongtrol uninstall` refuses sudo
instead of guessing; Inno Setup `[UninstallRun]` working directory (issue).

## Open questions

- Should a path containing `%` be deleted by the delayed delete? cmd expands `%VAR%` even
  inside quotes; the install folder never has one, but a renamed one could (issue).

## Used by

- ADR-0040
