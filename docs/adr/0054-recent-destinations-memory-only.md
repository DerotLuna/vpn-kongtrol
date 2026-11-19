<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0054: Recent destinations list, kept only in memory, on by default

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Users ask "did this site go through the VPN?". Kongtrol already reads the OS TCP table every
15 s to notice traffic for a policy whose tunnel is off (`monitor.PolicyWatch`). The owner
approved showing the recent destinations (host or IP, port, program, tunnel or direct, first and
last seen, count) on the condition that the data stays local and memory-only. Which tunnel a
connection used is privacy-sensitive: it is browsing history in all but name.

Research: [2026-11-19-recent-connections](../research/2026-11-19-recent-connections.md).

## Decision Drivers

- AGENTS.md "Local only": nothing leaves the machine; outbound calls must be opt-in. This feature
  makes no outbound call.
- No admin, no driver: read what `netstat`/`ss` read.
- Keep less than comparable tools (Little Snitch keeps up to a year on disk; GlassWire keeps a
  local database whose "clear" was reported to only hide data).
- GUI-first, CLI-complete.

## Considered Options

1. Off by default (opt-in), like the public IP lookup and the update notice.
2. **On by default, memory only, with a clear disclosure and a switch.**
3. Persist to disk (like Little Snitch / GlassWire).

## Decision Outcome

Option 2.

- `internal/monitor/recent.go` `RecentConnections`: every 15 s reads established / opening TCP
  connections (`tcpConnections`: Windows `GetExtendedTcpTable` with owning PID, Linux
  `/proc/net/tcp{,6}` with socket inodes; macOS not implemented → `supported: false`). Program
  name best effort without admin (Windows `OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` +
  `QueryFullProcessImageName`; Linux `/proc/<pid>/fd`, own processes only). The tunnel is the
  connected profile whose assigned IP is the connection's local address; otherwise "direct", with
  the local interface name. The host name only comes from what the policy resolver already
  resolved for policy domains (`PolicyResolver.HostOf`): **no reverse DNS**, which would send the
  addresses to a DNS server.
- Kept only in that struct: at most 500 entries (the one seen longest ago is evicted), forgotten
  1 h after last seen, emptied when turned off and on daemon shutdown. Never written to disk or
  logs, never in the audit log, never sent anywhere; served only on the authenticated loopback API
  `GET /api/v1/connections/recent?limit=n` (same token / session cookie as every route).
- Setting `monitor.recent_connections.enabled` (default **on**; stored only when off). Toggle:
  `PUT /api/v1/settings {"recent_connections_enabled": bool}` (applied live), `kongtrol config
  recent-connections on|off|status`. CLI view: `kongtrol traffic recent [--json] [--limit n]` (a
  new `traffic` group: there was none).
- Why on by default, not opt-in: the opt-in rule is for data that leaves the machine (public IP
  lookup, update check). This list leaves nothing and holds nothing the same user account cannot
  already read with `netstat -ano` / `ss -tnp` at any moment; the only addition is up to one hour
  of memory, gone on shutdown. Off by default would hide the feature that answers the user's
  question, for no protection against anyone who could not already see the same data. The
  disclosure is in `/privacidad` and `/en/privacy`, `docs/SECURITY.md`, the command help and the
  "on" message.
- Not option 3: a database on disk is exactly the trace users of other tools complain about.

### Consequences

- Good: "which tunnel did this go through" is answerable without admin, drivers or extensions.
- Good: turning it off empties it at once; nothing to clean up afterwards.
- Bad: a 15 s check misses connections that open and close between checks (short HTTP
  requests); the list is a sample, said in the help ("checked every 15 s").
- Bad: no macOS support yet (needs libproc / sysctl work).
- Bad: another local process running as the same user with the API token could read the list —
  the same user can already run `netstat`; the token file is user-only (ADR-0039).

### Confirmation

`internal/monitor/recent_test.go` (bound, expiry, count, off erases and reads nothing, shutdown
erases, a reflection test that the type has no path/file/log/writer field),
`internal/monitor/conns_windows_test.go` (row layout; real table read in a non-elevated
session), `internal/api/handlers_connections_test.go`, `cmd/kongtrol/cmd_traffic_test.go`.
