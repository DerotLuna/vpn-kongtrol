<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0064: Audit log kept 30 days within 500 MB, cleaned every midnight, and every API change audited

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

The dashboard's "Ajustes del registro" showed "Días que se guardan: 0" with "Se guardan hoy y
ayer": `retention_days: 0` meant 2 days (ADR-0042 kept the old behaviour). Old files were only
deleted when the logger opened a day (daemon start, first entry after midnight), and nothing
bounded the total size: with `max_size_mb` 1024 and a year of retention the log could take
hundreds of GB. The owner also asked whether every action is audited: only the tunnel lifecycle,
the watchdog, leaks and `on_tunnel_down` were; changes through the API (profiles, policies,
settings, security toggles, helper install, purge), sign-ins and refused credentials were not.

## Decision Drivers

- Never fill the user's disk; work with no tuning; older configs keep loading.
- AGENTS.md: CLI-GUI parity, guidance in short lines, i18n ES+EN, no secrets in logs.
- Standards: journald bounds the journal by space (`SystemMaxUse`, default 10 % of the file
  system, capped at 4G) and optionally by age (`MaxRetentionSec`); logrotate rotates by size;
  Kubernetes audits API requests at the `Metadata` level (who, when, resource, verb; no body)
  for secrets (research note).

## Considered Options

1. Keep `0` = today and yesterday; document it better.
2. Make `0` invalid (force a number).
3. **`0`/unset = 30 days, a new total cap `max_total_mb` (unset = 500 MB), cleanup at start,
   on every new file and on a midnight timer; audit every mutating API request at metadata
   level plus refused credentials, daemon start/stop and kill switch state.**

## Decision Outcome

Option 3. Option 1 keeps the confusing "0"; option 2 breaks configs that load today.

- **Retention:** `retention_days` 1–365; unset or `0` = 30 (the loader writes 30, the API
  answers the value in use, the dashboard and CLI accept 1–365). `1` = today only.
- **Total cap:** `max_total_mb` 1–10240; unset or `0` = 500, raised to `max_size_mb` when that is
  larger (so a config with `max_size_mb: 1024` keeps loading); a set value below one file is
  refused (`config.err.audit_total_small`). Pruning deletes the oldest files until closed files
  plus room for the open file to reach `max_size_mb` fit: the log never passes the cap. The open
  file is never deleted.
- **When:** at start, whenever a new file opens (new day or size segment), and every local
  midnight (+1 s) by a timer that also closes yesterday's file, so cleanup happens on idle days.
  The clock and timer are injectable (tests with a fake clock).
- **Surfaces:** dashboard Actividad → Ajustes del registro (new "Espacio máximo en total" field),
  `kongtrol config audit-log [--days N] [--max-size-mb N] [--max-total-mb N]` (through the
  daemon, else kongtrol.yaml with the same validation), `GET/PUT /api/v1/settings`
  `audit_log_max_total_mb`.
- **Coverage:** `internal/api/audit_requests.go` maps every mutating route to an action
  (`config.profile.update`, `security.killswitch.set`, `helper.install`, `data.purge`,
  `auth.session_create`…, fallback `api.request`; a test fails if a route has none) and writes
  who (`app` = file token, `dashboard` = session), the `{name}` in the path and the HTTP status;
  bodies are never read. Refused requests (`auth.denied`: credential, origin, host) are logged
  at most once per 10 s with the count skipped. The daemon adds `daemon.start`/`daemon.stop` and
  `security.killswitch.state`/`.error` (only on change). `data.purge` is written before it runs
  because the purge deletes the log. Inventory: `docs/SECURITY.md` "Audit log".

### Consequences

- Good: no "0" on screen; bounded disk use by default; a trail of every change and sign-in.
- Neutral: configs that relied on "0 = 2 days" now keep 30 days (more data, still bounded);
  more entries (one per change, a few per connect).
- Bad: CLI edits made with no daemon running and hand edits of kongtrol.yaml are not in the log
  (no daemon to write it); `auth.nonce` adds one line each time the panel opens.

### Confirmation

`go test ./internal/security -run 'Midnight|TotalCap|ZeroRetention'`,
`./internal/config -run Audit`, `./internal/api -run Audit`, `./cmd/kongtrol -run Audit`,
`web/ui` `ActivityPage.test.tsx`.

## More Information

- Code: `internal/security/audit.go`, `internal/config/{schema,loader,semantics}.go`,
  `internal/api/audit_requests.go`, `cmd/kongtrol/{audit_events,audit_runtime,cmd_config_audit}.go`,
  `internal/daemonctl/auditlog.go`, `web/ui/src/components/activity/AuditConfig.tsx`.
- Research: [2026-11-19-audit-log-retention-and-coverage](../research/2026-11-19-audit-log-retention-and-coverage.md).
- Builds on [ADR-0042](0042-dashboard-api-completeness.md) (size per file and retention_days);
  changes its "0 = today and yesterday".
