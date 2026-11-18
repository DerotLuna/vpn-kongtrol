<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0040: Local hardening after the 2026-11 source review

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

A source review (2026-11-18) found, besides the browser token disclosure fixed by
[ADR-0039](0039-browser-sessions-one-time-nonce.md), several ways a local process, a crafted
setting or a compromised dependency could harm the user: recursive deletes outside
`~/.kongtrol`, a manifest that could name `/`, a helper that bound foreign WireGuard tunnels by
name only, an update link taken from the network, stored XSS in the legacy dashboard through
profile names, and CI secrets visible to every step. Product rule: nothing leaves the machine
unless the user opted in, and Kongtrol must not be usable against its own users.

## Decision Drivers

- Least privilege and smallest effective fix at the last trusted decision point.
- Verify against primary sources ([research note](../research/2026-11-18-security-hardening-review.md)).
- No new user-facing burden; every refusal says what to do (ES+EN).

## Decision Outcome

1. **Purge** removes extra paths outside `~/.kongtrol` only when `Lstat` says regular file,
   with `os.Remove`; the audit log is purged through the files the logger wrote
   (`security.AuditLogFiles`); `security.audit_log.path` may not be a folder
   (`config.err.audit_log_dir`).
2. **Uninstall** trusts only manifest entries with Kongtrol's own base names; the executable
   must match exactly (inside `Kongtrol.app` only for the bundle); only `Kongtrol.app` is
   removed recursively. `sudo kongtrol uninstall` is refused (`cli.uninstall.no_sudo`). The
   Windows delayed delete runs `%SystemRoot%\System32\cmd.exe /d` and `PING.EXE` by full path
   from System32.
3. **Privileged helper**: a tunnel Kongtrol did not start is accepted for `wg.stats`,
   `wg.remove`, `wg.allowed_ips` only if the public key derived from the caller's `.conf`
   equals the interface's (`wg show dump`); such a stats success is audited (1/h/tunnel);
   names `all`/`interfaces` are refused; on Windows `wg.exe`/`wireguard.exe` and their folder
   must be admin-owned and admin-writable only. The TrustedInstaller SID constant is corrected.
   No wire change (`ProtocolVersion` unchanged).
4. **Update notice**: the link is built from the strict `vX.Y.Z` tag under a fixed prefix;
   `html_url` is ignored; remembered state is re-validated; redirects only https to
   `api.github.com`. Public IP lookups follow redirects only https to the same host.
5. **Names** of profiles, policies, groups and rules: 1–64 characters without control
   characters or `< > " ' \` & \\` (`config.err.name_chars`); the legacy dashboard escapes quotes
   and passes names through `data-*` + `addEventListener`, never inline handlers.
6. **CI**: secrets only on the steps that use them (conditions read non-secret step outputs);
   third-party actions pinned to full commit SHAs with `# vX.Y.Z`; Renovate keeps them current.
7. **Inbound**: the API stays loopback-only (non-loopback binds refused since before this ADR);
   `localhost` is pinned to `127.0.0.1`; helper transport unchanged (pipe SDDL: SYSTEM,
   Administrators, installing user, network logons denied; unix socket `0660 root:<gid>` +
   peer uid).

### Consequences

- Good: each path above now fails closed with guidance; tests pin every case.
- Bad: configs with names using the refused characters stop loading until renamed (the error
  says how). A same-named foreign WireGuard tunnel can no longer be "adopted" (intended).
- Neutral: Renovate will also pin `actions/*` digests.

### Confirmation

Tests listed in each commit: `TestPurge_ExtraFilesOnlyRegularFiles`,
`TestAuditLogFiles_OnlyDailyFiles`, `TestValidateAuditLogPath`,
`TestReadManifest_OnlyKongtrolNames`, `TestRunViaSudo`, `TestDeleteAfterExitCmd_*`,
`TestWGAuthorize_ForeignSameNameTunnelDenied`, `TestWGConfigPublicKey_ManPageExample`,
`TestValidateWGName_Reserved`, `TestWGStats_AuditMark`, `TestTrustedInstallerSID_MatchesSystem`,
`TestWGExe_RealInstall`, `TestValidReleaseURL`, `TestCheck_IgnoresHTMLURL`,
`TestLoadState_DropsTamperedValues`, `TestCheckRedirect`, `TestPublicIPRedirect`,
`TestValidateName`; `actionlint` clean.

## More Information

`docs/SECURITY.md` ("Network connections", "Privileged helper", "OWASP mapping").
