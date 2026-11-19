# Audit log retention, total size and what to audit

Version 1.0 · 2026-11-19 · cited by ADR-0064

## How others bound log storage

- systemd-journald (`journald.conf`): `SystemMaxUse=` caps the disk space the persistent journal
  may use; default 10 % of the file system, each default capped at 4G. `MaxRetentionSec=`
  deletes journal files with entries older than a time span; default 0 (off). Only archived
  files are deleted to reduce space (the active file is kept). The page does not state the
  deletion order. Source: https://www.freedesktop.org/software/systemd/man/latest/journald.conf.html
  (accessed 2026-11-19).
- logrotate: `maxsize`/`size` rotate by size, `rotate N`/`maxage` bound how many or how old.
  Kongtrol's per-file `.1`, `.2` segments already follow `maxsize` (ADR-0042). No verificado
  again in this session.
- Takeaway: bound both age and total space, never delete the file being written, and default to
  a cap that needs no tuning. Kongtrol: 30 days and 500 MB (a fixed cap instead of a percentage:
  simpler to explain to non-technical users; reasoned, not measured).

## What to audit and how

- Kubernetes audit levels: `None`; `Metadata` = "log events with metadata (requesting user,
  timestamp, resource, verb, etc.) but not request or response body"; `Request`;
  `RequestResponse`. Its example policy logs secrets and configmaps at `Metadata` so their
  contents never reach the log. Source:
  https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/ (accessed 2026-11-19).
- OWASP Logging Cheat Sheet: log authentication failures, access control failures and
  configuration changes; never log passwords, session identifiers, access tokens or keys.
  Source: https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html
  (no verificado in this session; cited from the project's existing OWASP mapping).
- Takeaway for Kongtrol: one entry per mutating API request at metadata level (who = credential
  type, what = route action + name in the path, when, result = status), refused credentials
  throttled so a local program retrying cannot flood the log.
