# CLI Reference

Full command reference for the `kongtrol` binary. For a guided first-run walkthrough, see [SETUP.md](SETUP.md).

## Setup & diagnostics

```bash
kongtrol init                        # interactive wizard (create or update config; re-run to add profiles)
kongtrol doctor                      # validate full stack: binaries · certs · keychain · permissions
kongtrol doctor --json               # machine-readable diagnostics output (CI/automation)
```

## Tunnel lifecycle

```bash
kongtrol up <profile> [profile...]   # connect one or more profiles
kongtrol up --all                    # connect all configured profiles
kongtrol up --group work             # connect all profiles in a group
kongtrol up --group work --dry-run   # simulate full connect flow without changing system state
kongtrol down <profile>              # disconnect a profile
kongtrol down --group work           # disconnect all profiles in a group
kongtrol down --all                  # disconnect everything
```

## Status

```bash
kongtrol status                      # table: profile · status · IP · uptime
kongtrol status --watch              # live auto-refreshing terminal view
kongtrol check                       # run leak + integrity test now
```

## Routes & policy

```bash
kongtrol routes list                 # list Kongtrol-managed routes
kongtrol policy explain <target>     # explain matched policy (IP, domain, or app:<exe>)
kongtrol policy explain <target> --json
```

## Dashboard

```bash
kongtrol dashboard                   # start web UI and open browser
```

## Config

```bash
kongtrol config validate             # validate kongtrol.yaml without connecting
kongtrol export                      # print sanitized config template (no secrets)
```

## Audit log

```bash
kongtrol audit                       # audit log subcommands
```

## Output language

CLI output is Spanish by default. Override per-invocation:

```bash
KONGTROL_LANG=es kongtrol doctor
KONGTROL_LANG=en kongtrol doctor
```

## Useful daemon/dashboard API endpoints

```bash
GET /api/v1/metrics/history                 # persistent per-profile history
GET /api/v1/dns/resolve?domain=...&via=...  # split-DNS resolution for a specific tunnel
GET /api/v1/resolve?target=...&app=...      # flow-aware policy resolution (app + target)
POST /api/v1/shutdown                       # ask a running daemon to shut down gracefully
```
