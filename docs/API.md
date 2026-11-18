<img src="../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Local API reference

Version 1.3 · 2026-11-18

<!-- feature:api -->
The daemon's local API (IPC on this machine only — nothing leaves the host). The tray, the CLI and
both dashboards (`web/dashboard/`, `web/ui/`) are clients of it — everything they can do is
reachable directly too. All endpoints are under `http://127.0.0.1:9741` by default and require
the local capability stored in `~/.kongtrol/api-token`:

```bash
curl -H "X-Kongtrol-Token: $(cat ~/.kongtrol/api-token)" \
  http://127.0.0.1:9741/api/v1/tunnels
```

Commands: [CLI.en.md](CLI.en.md) (Español: [CLI.md](CLI.md)).

## Endpoints

```bash
# Tunnels & monitoring
GET  /api/v1/tunnels                         # live snapshot of all tunnels
GET  /api/v1/metrics/history                 # persistent per-profile history (reconnects, jitter, samples)
GET  /api/v1/routes                          # active OS routes with resolved policy/profile
GET  /api/v1/network/overview                # default egress, local IPs, last public IP (only if the lookup is on; never fetched here), connected count
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
POST /api/v1/security/publicip               # {"enabled": true|false} — opt-in public IP lookup (off by default)
POST /api/v1/security/publicip/check         # one lookup now ("Revisar"); 409 publicip.err.disabled while off
POST /api/v1/security/dnsguard               # {"enabled": true|false}

# Settings & scheduler
GET/PUT   /api/v1/settings                   # health check, scheduler, split DNS, kill switch/DNS guard tuning, public IP lookup (public_ip_check_enabled, public_ip_endpoints), audit log
GET/POST  /api/v1/scheduler/rules
PUT/DELETE /api/v1/scheduler/rules/{name}

# Audit log
GET  /api/v1/audit?profile=...&level=...&limit=...

POST /api/v1/shutdown                        # ask a running daemon to shut down gracefully

# Diagnostics, helper and export
GET  /api/v1/doctor                          # same checks as `kongtrol doctor`, in the daemon, translated (X-Kongtrol-Lang): {"checks":[{section,label,status ok|warn|fail,detail,hint,fix}],"failures","warnings"}; fix is only "helper_install"
GET  /api/v1/helper/status                   # {"installed","running","available","elevated","needs_install",…}
POST /api/v1/helper/install                  # installs the privileged helper (one OS permission prompt)
GET  /api/v1/config/export                   # kongtrol.yaml as a download (application/yaml, kongtrol-export.yaml; no passwords: they stay in the keychain); 404 export.err.no_config before the first profile

# First run and restart (ADR-0025)
GET  /api/v1/setup                           # {"setup_mode","needs_setup","restart_pending","config_path"}: daemon started with no kongtrol.yaml
POST /api/v1/restart                         # 202 {"status":"restarting"}: starts a fresh daemon (re-reads kongtrol.yaml, reconnects the tunnels that were up), then this one stops; 500 if the replacement could not start (this daemon keeps running)
```

<!-- feature:first_run -->
**Setup mode.** A daemon started with no config file (the tray spawns it on first run) serves the
dashboard with no profiles; `GET /api/v1/vpns` returns `[]` and the first valid
`POST /api/v1/vpns` writes `~/.kongtrol/kongtrol.yaml`. VPN profile changes answer
`restart_required: true` because the adapters are built at daemon start: the dashboard then
offers `POST /api/v1/restart` (after the first profile it restarts on its own). CLI:
`kongtrol restart`.
