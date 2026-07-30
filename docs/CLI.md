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

## Reload

`kongtrol reload` picks up a hand-edited `kongtrol.yaml` in an already-running `kongtrol up`
daemon, without restarting the whole process. It always talks to the running daemon's embedded
API — if none is reachable, it fails clearly instead of silently reloading a throwaway in-process
copy nobody is using.

```bash
kongtrol reload                      # reload the policy engine, then restart every active group in place
kongtrol reload --policy             # reload only the policy engine (routing rules) — no tunnels touched
kongtrol reload --group work         # reload policies, then restart only this group's active tunnels
kongtrol reload --tunnel office      # reload policies, then restart only this one tunnel (if connected)
```

`--group`, `--tunnel`, and `--policy` are mutually exclusive — pick the narrowest scope for the
edit you made. A profile that isn't already registered with the running daemon (e.g. a brand-new
VPN type added by the hand edit — the adapters map is built once at boot) can't be restarted this
way; `reload` reports it as needing a full process restart (`kongtrol down` then `kongtrol up`)
instead of silently doing nothing.

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

kongtrol config dashboard show               # show the dashboard bind/port override, if any
kongtrol config dashboard set-port <port>    # override the dashboard's local port (restart to apply)
kongtrol config dashboard set-bind <address> # override the dashboard's bind address (restart to apply)
kongtrol config dashboard reset              # clear the override, fall back to kongtrol.yaml
```

`kongtrol config dashboard` writes to `~/.kongtrol/preferences.json` — the same machine-local file
that stores `kongtrol config lang`, favorites, and the default group. It's intentionally **not**
editable from the dashboard itself: changing the port from the page serving that request would cut
the connection mid-response.

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

The embedded dashboard (`web/dashboard/`) is a full client of this API — everything it can do is
reachable directly too. All endpoints are under `http://127.0.0.1:9741` by default and require
the local capability stored in `~/.kongtrol/api-token`:

```bash
curl -H "X-Kongtrol-Token: $(cat ~/.kongtrol/api-token)" \
  http://127.0.0.1:9741/api/v1/tunnels
```

```bash
# Tunnels & monitoring
GET  /api/v1/tunnels                         # live snapshot of all tunnels
GET  /api/v1/metrics/history                 # persistent per-profile history (reconnects, jitter, samples)
GET  /api/v1/routes                          # active OS routes with resolved policy/profile
GET  /api/v1/network/overview                # default egress, local IPs, public IP, connected count
POST /api/v1/tunnels/{name}/connect
POST /api/v1/tunnels/{name}/cancel_connect
POST /api/v1/tunnels/{name}/disconnect
POST /api/v1/tunnels/{name}/reload           # reload kongtrol.yaml + restart this tunnel in place if connected
GET  /api/v1/dns/resolve?domain=...&via=...  # split-DNS resolution for a specific tunnel
GET  /api/v1/resolve?target=...&app=...      # flow-aware policy resolution (app + target)

# Policies
GET/POST  /api/v1/policies
PUT/DELETE /api/v1/policies/{name}
POST /api/v1/policies/reload                 # reload kongtrol.yaml from disk, hot-swap the policy engine
POST /api/v1/policies/test                   # dry-run a policy against a target/app before saving

# VPN profiles (config + OS keychain only — new/edited profiles need a daemon restart)
GET/POST  /api/v1/vpns
PUT/DELETE /api/v1/vpns/{name}
PUT  /api/v1/vpns/{name}/killswitch          # {"override": "inherit"|"on"|"off"}

# Groups
GET/POST  /api/v1/groups
PUT/DELETE /api/v1/groups/{name}
POST /api/v1/groups/{name}/connect
POST /api/v1/groups/{name}/disconnect
POST /api/v1/groups/{name}/reload            # reload kongtrol.yaml + restart the group's active tunnels in place

# Security — live toggles, immediately re-applied (not just persisted)
GET  /api/v1/security/status
POST /api/v1/security/killswitch             # {"enabled": true|false}
POST /api/v1/security/dnsguard               # {"enabled": true|false}

# Settings & scheduler
GET/PUT   /api/v1/settings                   # health check, scheduler, split DNS, kill switch/DNS guard tuning, audit log
GET/POST  /api/v1/scheduler/rules
PUT/DELETE /api/v1/scheduler/rules/{name}

# Audit log
GET  /api/v1/audit?profile=...&level=...&limit=...

POST /api/v1/shutdown                        # ask a running daemon to shut down gracefully
```
