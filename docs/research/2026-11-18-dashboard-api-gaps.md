<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: dashboard API gaps (audit rotation, config import safety, compression)

Version 1.0.0 · 2026-11-18 · Used by [ADR-0042](../adr/0042-dashboard-api-completeness.md)

Access date for every source: 2026-11-18. Sources marked *no verificado* were cited from prior
knowledge and not re-opened in this session.

## Size-based log rotation

- logrotate(8) `maxsize` / `size`: rotate when the file grows past a size, independently of the
  time-based schedule. https://man7.org/linux/man-pages/man8/logrotate.8.html (*no verificado*).
- natefinch/lumberjack (Go): `MaxSize` (MB per file), `MaxAge` (days), `MaxBackups`.
  https://pkg.go.dev/gopkg.in/natefinch/lumberjack.v2 (*no verificado*). Not adopted as a
  dependency: its backup names (`name-<timestamp>.ext`) would break the daily
  `audit-YYYY-MM-DD.log` names that `doctor`, purge/uninstall and `logs --follow` rely on; the
  same semantics (size per file + age) are implemented in `internal/security/audit.go` with
  `.N` segments inside the day.

## OpenVPN directives that run code or touch files

OpenVPN 2.6 reference manual, "Script hooks" and "--script-security":
https://openvpn.net/community-resources/reference-manual-for-openvpn-2-6/ (*no verificado* in
this session). `--up`, `--down`, `--route-up`, `--route-pre-down`, `--ipchange`,
`--client-connect`, `--client-disconnect`, `--learn-address`, `--auth-user-pass-verify`,
`--tls-verify` run external programs; `--plugin` loads a shared library; `--script-security 2`
or higher is required for user scripts to run; `--log`, `--status`, `--writepid`, `--cd`,
`--config` read or write paths. Kongtrol already rejects these in
`internal/privops/ovpnconf.go` (helper side); the import endpoint reuses that function and
additionally refuses any file reference (`ca /path`…), so an imported file is self-contained.

## WireGuard hooks

wg-quick(8): `PreUp`, `PostUp`, `PreDown`, `PostDown` run shell commands as root.
https://man7.org/linux/man-pages/man8/wg-quick.8.html (*no verificado*). Rejected by
`privops.ValidateWireGuardConfig`, reused by the import.

## HTTP content coding

- RFC 9110 §12.5.3 `Accept-Encoding` (q-values, `*`, `q=0` = not acceptable) and §12.5.5 `Vary`.
  https://www.rfc-editor.org/rfc/rfc9110 (*no verificado*).
- Go standard library has `compress/gzip`; Brotli needs a third-party package
  (e.g. `github.com/andybalholm/brotli`), not in `go.mod`, so only gzip is served.
- WOFF2 is already Brotli-compressed internally (W3C WOFF2 recommendation), so it is not
  re-compressed. https://www.w3.org/TR/WOFF2/ (*no verificado*).

## Stable sorting

Go `sort.Slice` is not guaranteed stable; `sort.SliceStable` keeps equal elements in their
original order. https://pkg.go.dev/sort#Slice (*no verificado*). Verified in code: the new test
fails with `sort.Slice` and passes with `sort.SliceStable`.
