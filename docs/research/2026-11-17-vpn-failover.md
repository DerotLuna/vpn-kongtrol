<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: how VPN clients define failover

Version 1.0.0 · 2026-11-17

## Question

When a tunnel cannot reconnect, which other connection (if any) do established VPN tools bring
up, and who decides? Needed for [ADR-0018](../adr/0018-failover-only-to-declared-profiles.md):
Kongtrol's watchdog connected *any* other profile ordered by `priority`, which brought up an
unrelated FortiClient profile when an OpenVPN profile died.

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | OpenVPN — Reference manual for OpenVPN 2.6 (`--remote`, connection profiles, `--connect-retry`) | https://openvpn.net/community-resources/reference-manual-for-openvpn-2-6/ | 2026-11-17 |
| 2 | NetworkManager — `connection` setting (`autoconnect-priority`, `autoconnect-retries`, `secondaries`) | https://networkmanager.dev/docs/api/latest/settings-connection.html | 2026-11-17 |
| 3 | WireGuard — wg(8) man page (peer `Endpoint`) | https://man7.org/linux/man-pages/man8/wg.8.html | 2026-11-17 |
| 4 | Cisco Community — "Using Backup Server List in AnyConnect XML" (third party, forum) | https://community.cisco.com/t5/vpn/using-quot-backup-server-list-quot-in-anyconnect-xml-for-client/m-p/5149318 | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- OpenVPN [1]: redundancy lives **inside one profile**, declared by whoever writes the config:
  "On the client, multiple --remote options may be specified for redundancy"; they are tried "in
  the order specified by the list of --remote options"; "The client will move on to the next
  host in the list, in the event of connection failure." Multiple `--remote` is "a special case
  of the more general connection-profile feature" (`<connection>` blocks, also inside one
  config). Retries are slowed down and capped ("default `300`" seconds), and a successful
  connection resets the counter. All alternatives are servers of the *same* VPN.
- NetworkManager [2]: `autoconnect-priority` ("in range -999 to 999") only ranks connections
  that NetworkManager is *allowed to auto-activate* (`autoconnect`); `autoconnect-retries` is
  "The number of times a connection should be tried when autoactivating before giving up".
  Linking connections is explicit: `secondaries` is a "List of connection UUIDs that should be
  activated when the base connection itself is activated". Priority is never a reason to bring
  up a different VPN after a failure (reasoned from the property semantics; NetworkManager's
  activation code was not read: **no verificado** beyond the docs).
- WireGuard [3]: one `Endpoint` per peer, "updated automatically to the most recent source IP
  address and port" (roaming). No failover to another peer or tunnel is built in.
- Cisco AnyConnect / Secure Client [4]: the profile carries an explicit "Backup Server List"
  for the same headend, tried in order when the selected server fails. Taken from a forum
  thread; the Cisco administrator guide itself was not read: **no verificado** with a primary
  source.
- Common pattern: failover targets are **declared explicitly and ordered**, and they reach the
  **same network/service**. No tool reviewed infers alternatives from names, types or a global
  priority, or brings up an unrelated VPN.

## Confidence

High for OpenVPN, NetworkManager and WireGuard (primary docs). Medium for Cisco (forum thread).

## Open questions

- Whether Kongtrol should later support multiple servers *inside* one profile (as OpenVPN
  `remote` does) for adapters that lack it. Not needed for ADR-0018.
