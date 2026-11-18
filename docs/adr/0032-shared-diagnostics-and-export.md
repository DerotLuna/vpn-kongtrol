<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0032: Shared diagnostics and config export for CLI and GUI

Version 1.0.0 · 2026-11-18

Status: Accepted · Date: 2026-11-18 · Decision-makers: project maintainer · Refines
[ADR-0005](0005-privileged-helper-service.md), [ADR-0025](0025-desktop-release-channels-first-run.md)

## Context and Problem Statement

`kongtrol doctor` and `kongtrol export` existed only in the terminal, against the product model
(GUI-first, CLI-complete: no feature only in one of them). Their logic lived in `cmd/kongtrol`,
so the daemon could not offer them. Two defects were found while moving them (verified in code
and with a round-trip test):

- `kongtrol export` printed invalid YAML: Go-formatted lists (`[a b]`), a second `match:` key
  when a policy had IPs and domains, unquoted `*.corp.com` (a YAML alias), random map order; it
  also dropped `security`, `country`, `auto_connect`, `on_tunnel_down`, the scheduler, etc.
- The tray's "Instalar ayudante (una vez)" item only appeared when the daemon answered and was
  missing from the first-run (no config) menu, so a fresh macOS/Linux install never offered it.

## Decision

- `internal/doctor`: `Run(Options)` returns translated checks; every warning/failure carries one
  "what to do" `Hint`; `Fix` names an automatic fix, and the only one is `helper_install`. The
  CLI renders the report (hint on its own line); `GET /api/v1/doctor` runs it inside the daemon
  with the daemon's tunnel and security state, and drops `fix` when the daemon cannot install
  the helper. The dashboard's Diagnóstico card runs it on demand only (it starts VPN clients'
  `--version`), never on a timer.
- `internal/configexport`: marshal the loaded `config.Config` with `gopkg.in/yaml.v3`, the same
  library the loader parses with (struct order, sorted map keys, quoting as needed —
  [yaml.v3 docs](https://pkg.go.dev/gopkg.in/yaml.v3), accessed 2026-11-18), plus a translated
  comment header. The schema holds no secrets (passwords are keychain references), so nothing
  is redacted. A test re-loads the output with `config.Load` and requires an identical config.
  Same output for `kongtrol export` (colours only on a terminal) and
  `GET /api/v1/config/export` (download `kongtrol-export.yaml`).
- Tray: decide "helper missing" from the daemon, or — when no daemon answers — by pinging the
  helper directly with the daemon's rule (not elevated and no helper with our protocol); show
  the item in the first-run menu too, and a one-time notice `tray.helper_first_run`.
- No GUI fix runs without the user: "Reparar" uses the existing `POST /api/v1/helper/install`
  (one OS permission prompt).

## Consequences

- One implementation, three surfaces (CLI, API, dashboard); strings in `internal/i18n`.
- Values containing `$` do not survive an export/import round trip, because the loader expands
  environment variables (`os.ExpandEnv`) before parsing; pre-existing loader behaviour, not
  changed here (no verificado en uso real).
- Exported paths are this machine's absolute paths; the header tells the user to adjust them.
