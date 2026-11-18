<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0042: Dashboard API completeness — audit verification and rotation, leak action removed, profile rename/delete/import, scheduler hot reload

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer

## Context and Problem Statement

Building the new dashboard (`web/ui`, [ADR-0038](0038-new-dashboard-architecture-and-patterns.md))
exposed API gaps and settings that did nothing (`docs/dashboard-parity.md` §5.2, items 16, 22–26):
the audit HMAC was never checked, `security.audit_log.max_size_mb` was not enforced, the audit API
read only two days, `security.leak_detection.action` had no runtime effect, `PUT /api/v1/vpns/{name}`
ignored renames, `DELETE` missed failover and schedule references, profile files could not be
imported from the GUI, group/scheduler errors were English only, scheduler edits needed a
restart, equal-priority policies sorted unstably, and the watchdog's `manual_reconnect_required`
never reached the clients. Product rules: GUI-first and CLI-complete, nothing stale stays, local
only, every error says what to do (ES+EN).

## Decision Drivers

- A setting the dashboard shows must do what it says, or be removed.
- Reuse what is already vetted in the codebase (the helper's config validators) and standard
  practice ([research note](../research/2026-11-18-dashboard-api-gaps.md)).
- No new outbound calls; secrets (audit HMAC key) never leave the daemon.

## Decision Outcome

1. **Audit verification.** `GET /api/v1/audit` returns `signed` and `verified` per entry. The
   daemon checks the HMAC with the key in the OS keychain (`auditVerifier`, existing key only);
   the API gets a yes/no function, never the key. `kongtrol logs` shows the same (`--json` fields,
   a warning line for a bad signature).
2. **Audit size and retention: enforce, not remove.** `max_size_mb` (default 100, 1–1024) caps one
   file: the day continues in `audit-YYYY-MM-DD.1.log`, `.2`… (logrotate `maxsize` /
   lumberjack `MaxSize` semantics, keeping the daily names every tool already uses). New
   `retention_days` (0 = today and yesterday, the previous behaviour; max 365). Both apply live
   (the logger is reopened atomically on settings save).
3. **Audit query.** `from` / `to` (local day or RFC 3339), `before` cursor, `limit` 1–1000,
   `X-Kongtrol-Audit-More`; one reader (`security.QueryAuditEvents`) for API and CLI (`logs
   --since/--until`).
4. **Leak action: remove.** `security.leak_detection.action` and the already ignored `interval`
   are removed from the schema, API and both dashboards; old configs still load. A leak only
   notifies (alert + audit). Making the kill switch depend on an external public IP service would
   contradict [ADR-0010](0010-public-ip-check-opt-in.md) (the kill switch never needs that lookup)
   and the per-destination design of [ADR-0009](0009-kill-switch-split-tunneling.md) (an IP
   mismatch does not say which destinations to block). The leak check now compares only with
   `expected_ips` of **connected** profiles (before, any configured profile that was off made a
   permanent "leak").
5. **Security status.** `security/status` adds the DNS guard servers in use and a translated
   `kill_switch_error`; new read-only `GET /api/v1/security/blocked`; `expected_ips` editable in
   settings with validation. CLI: `kongtrol config killswitch|dns-guard status`.
6. **Profiles.** Rename is implemented, not rejected: references (policies, groups, failover,
   schedules, expected IPs), keychain entries (copy → save → delete old, rollback on failure) and
   favorites follow; refused while the tunnel is up or connecting. Delete lists every reference
   at once and validates the trial config.
7. **Profile file import** (`POST /api/v1/vpns/import`): JSON + base64 (the API middleware
   already enforces JSON and 1 MiB), stored 0600 in `~/.kongtrol/configs/`, validated with the
   helper's own `ValidateWireGuardConfig` / `SanitizeOpenVPNConfig` (no hook scripts, no
   directives that run programs or write files, `script-security` ≤ 1) and, for an import, no
   references to other files; never overwrites without `overwrite` (`O_EXCL`), never through a
   link.
8. **Localized errors** for groups, scheduler and profiles (`config.err.*`, `api.err.*`).
9. **Scheduler hot reload** (`Scheduler.Reconfigure`) and per-rule `enabled`.
10. **Policy order:** `sort.SliceStable`; the config position is the last key, so equal
    priorities keep the written order (a name key would reorder what the user wrote).
11. **Watchdog notice** `watchdog.event.manual_reconnect` on `/api/v1/events`.
12. **`/app/` compression:** gzip with `Accept-Encoding` negotiation and `Vary`, cached per file;
    no Brotli because no vetted Brotli library is in `go.mod`.

### Consequences

- Good: every audit/leak/scheduler setting shown in the dashboards is truthful; the dashboard can
  edit names, import files and show what the kill switch blocks.
- Bad: rename touches the keychain; a crash between copy and delete can leave a stale old entry
  (harmless, removed by purge). Longer audit retention keeps more local data (user's choice).
- Neutral: `leak_detection_action` / `leak_detection_interval` disappear from `/api/v1/settings`.
