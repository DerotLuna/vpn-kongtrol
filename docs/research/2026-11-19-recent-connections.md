<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Recent destinations: how others expose per-connection data

Version 1.0.0 · 2026-11-19

Backs [ADR-0054](../adr/0054-recent-destinations-memory-only.md). Sources accessed 2026-11-19.

## Question

Kongtrol wants to show "where did this computer connect lately, through which tunnel, by which
program" (`GET /api/v1/connections/recent`, `kongtrol traffic recent`). How do established tools
read this without admin rights, and what do they do about privacy (what is kept, where, for how
long, how to turn it off)?

## How the data is read

| Tool | Source | Admin? |
|---|---|---|
| Windows `netstat -ano`, Resource Monitor (Network › TCP Connections), TCPView | IP Helper `GetExtendedTcpTable` (`TCP_TABLE_OWNER_PID_ALL`: every TCP endpoint with the owning PID) | Not stated in the Microsoft reference; **verified here**: a standard (non-elevated) Windows 11 session read 80 connections, 66 with a program name (`internal/monitor/conns_windows_test.go`) |
| Linux `ss -tnp`, `netstat -tnp` | `/proc/net/tcp{,6}` (any user) + socket inodes matched in `/proc/<pid>/fd` for the program | Table: no. Program: only for the user's own processes without root (others need root) — reasoned from `proc(5)` permissions, **no verificado** against the man page text in this session (man7.org not reachable from this environment) |
| macOS `lsof -i`, `nettop` | libproc / sysctl | `lsof` without root sees only the user's processes (**no verificado**); not implemented in Kongtrol |
| Little Snitch Network Monitor | Network Extension (kernel-level filter) | Installs a system extension |
| GlassWire | Windows filtering driver/service | Installed as a service |

Sources:
- `GetExtendedTcpTable`: https://learn.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getextendedtcptable
- `TCP_TABLE_CLASS`: https://learn.microsoft.com/en-us/windows/desktop/api/iprtrmib/ne-iprtrmib-tcp_table_class
- `MIB_TCPROW_OWNER_PID`: https://learn.microsoft.com/windows/win32/api/tcpmib/ns-tcpmib-mib_tcprow_owner_pid ;
  `MIB_TCP6TABLE_OWNER_PID`: https://learn.microsoft.com/en-us/windows/win32/api/tcpmib/ns-tcpmib-mib_tcp6table_owner_pid
  (row layouts used by `parseTCPTable`; the pages were found by search, the field offsets were
  checked against the structures and a unit test, the page text itself **no verificado** in this session)
- `PROCESS_QUERY_LIMITED_INFORMATION` + `QueryFullProcessImageName` for the program name:
  https://learn.microsoft.com/en-us/windows/win32/procthread/process-security-and-access-rights
  (page not reachable from this environment: **no verificado**; behaviour verified by the test above)
- `ss(8)`: https://man7.org/linux/man-pages/man8/ss.8.html (not reachable: **no verificado**)
- `proc(5)` `/proc/net/tcp`: https://man7.org/linux/man-pages/man5/proc.5.html (**no verificado**)

## Privacy precedent

- **Little Snitch** (Objective Development) records every connection in its Network Monitor,
  stored locally and encrypted; by default up to one year and 50,000 distinct connections
  (older ones merged); "Rules to Hide Connections" for sites the user does not want traces of;
  turning Network Monitor off stops collection while the firewall keeps working. Sources (found by
  search; pages not fetchable here, content from search excerpts: **no verificado** word for word):
  https://help.obdev.at/littlesnitch6/pref-monitor ,
  https://help.obdev.at/littlesnitch6/lsm-connection-list
- **GlassWire** keeps network history locally (never sent to its servers per its FAQ), offers
  "Incognito" to keep no history and "clear history"; a forum report says clearing only hid the
  data (user report, unverified). Sources: https://www.glasswire.com/faq/ ,
  https://forum.glasswire.com/t/why-glasswire-keep-months-of-hostname-data-and-applications-in-db-even-after-wiping-it-from-settings/10224
- **Resource Monitor / TCPView / `ss`** show only what is open now and keep nothing.

## What Kongtrol takes from this

- Read the same OS tables `netstat`/`ss` read, without admin, on the same 15 s check the policy
  notices already use (`monitor.PolicyWatch`) — no driver, no extension.
- Keep less than Little Snitch/GlassWire: memory only (no database at all, so "clear" cannot
  leave data behind, the GlassWire complaint), 500 entries, 1 h after last seen, erased on off and
  on shutdown.
- A switch like Little Snitch's "disable Network Monitor": `monitor.recent_connections.enabled`.
- Host names only from what Kongtrol already resolved for policy domains: no reverse DNS lookups
  (they would send the addresses to a DNS server).
