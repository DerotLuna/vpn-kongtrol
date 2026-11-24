<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0107: Show the routes each VPN brings by itself, read-only, with what wins against a rule

Version 1.0.0 · 2026-11-24

Status: Accepted · Date: 2026-11-24 · Decision-makers: Ruben Luna (owner), implemented by agent

Research: [routes a VPN brings by itself, and what wins](../research/2026-11-24-vpn-routes-precedence.md).

## Context and Problem Statement

The Reglas page lists only the user's rules. A VPN also routes destinations on its own: `route`
lines in the `.ovpn`, routes the OpenVPN server pushes, WireGuard `AllowedIPs`, and whatever a
provider app installs. Users see traffic leave through a VPN with no rule for it, cannot tell where
that comes from, and do not know whether their rule or the VPN's route wins. The existing
`GET /api/v1/tunnels/routes` answers "what goes through each connected tunnel" from the OS route
table (mixed with Kongtrol's policy routes) and says nothing for disconnected profiles or about the
file vs the server.

## Decision Drivers

- Kongtrol does not own these routes and must never look like it can edit them.
- Non-technical first: grouped, short lines, a full tunnel in words.
- Every "add" goes through the one "＋ Agregar" sheet (ADR-0055); CLI parity.
- Claim a winner only where the logic is verified (code + OS docs); say "the system picks" otherwise.

## Considered Options

1. Extend `GET /api/v1/tunnels/routes` with file routes and disconnected profiles.
2. A new read-only endpoint `GET /api/v1/vpns/routes`, computed by one shared function used by the
   API and the CLI (with or without a daemon).
3. Compute everything in the panel from `/api/v1/vpns` and the rules.

## Decision Outcome

Chosen: **option 2**. Option 1 would change the meaning of an endpoint the Inicio and Tráfico pages
and `kongtrol routes` rely on (live OS routes). Option 3 cannot read the files or the server's
PUSH_REPLY and would leave the CLI without the same answer.

- `app.OwnRoutes(cfg, adapters, only)` (`internal/app/vpnroutes.go`) builds per profile:
  `source` `config` | `app` | `unreadable`; routes with `origin` `file` (`openvpn.ConfigRoutes`:
  `route`, `route-ipv6`, `redirect-gateway` → 0.0.0.0/0, `net_gateway` skipped;
  `wireguard.ConfigAllowedIPs`) or `server` (the adapter's `RouteReporter`, only while connected;
  `server_pending` otherwise); `all_traffic` (0.0.0.0/0, both /1 halves, WireGuard
  `default_egress`, Proton/NordVPN/Surfshark apps); `rules_only` for a direct WireGuard profile
  without `default_egress`, whose AllowedIPs Kongtrol replaces with the rule destinations
  (`ProfileService.policyAllowedIPs`), so the file's routes are listed but marked unused.
- Conflicts: for every VPN route and every active rule IP range through **another** profile that
  overlaps it, the winner is the OS's longest prefix match: `rule` (rule prefix longer), `vpn`
  (VPN prefix longer), `system` (same prefix on two interfaces: metric decides; Kongtrol's side is
  `routeMetric = 10`, the VPN's side is not under its control). /0 and /1 routes never get conflicts:
  PolicyResolver never installs a /0 or /1 for a rule, so every rule route is narrower and wins
  (the panel says so once per full-tunnel profile). Domain rules are left out of the per-route
  list (their IPs are known only once resolved; each resolved IP is a /32, which no VPN route
  beats). Rules through the same profile and paused rules are not conflicts. `rules_only` profiles
  get none (their routes are not installed).
- `priority` does not take part: it only orders rules in the policy engine; the OS never sees it.
- `GET /api/v1/vpns/routes[?profile=]` (read-only, no audit entry: not a mutation) and
  `kongtrol routes vpn [perfil] [--json]` (daemon when running, else the files directly).
- Panel: a "Rutas que trae cada VPN" card under the rules table on Reglas
  (`web/ui/src/components/rules/VpnRoutes.tsx`, model `vpnRoutesModel.ts`): one native disclosure
  per profile ("ekiipago · 3 rutas · del servidor"), a lock icon and one read-only line, each route
  with an origin tag, no edit/delete; a full tunnel reads "Todo el tráfico"; the only action per
  route ("Hacer que esto salga por otra VPN") opens `?add=rule&target=<prefix>`; a "Qué gana" note
  explains the precedence; conflict sentences come from the daemon's `winner`.

### Consequences

- Good: users see where unexplained routes come from and what wins, in both GUI and CLI, without
  Kongtrol pretending to own them.
- Good: one function for API and CLI; the panel only words the daemon's verdict.
- Bad: equal-prefix outcomes stay "the system picks"; no per-OS metric reading yet.
- Bad: the panel re-reads every 30 s so server routes appear after a connect (no event for it).
- Neutral: a profile named `vpn` can no longer be shown with `kongtrol routes vpn` meaning
  `kongtrol routes <perfil>`; `kongtrol routes vpn vpn` or the panel cover it.

### Confirmation

- Go: `internal/app/vpnroutes_test.go` (file/server origin, `net_gateway`, `rules_only`, apps,
  unreadable, conflicts and winners, full tunnel never in conflict), `internal/vpn/openvpn/configroutes_test.go`,
  `internal/api/handlers_vpn_routes_test.go`, `cmd/kongtrol/cmd_routes_vpn_test.go` (ES+EN).
- Vitest: `web/ui/src/components/rules/vpnRoutesModel.test.ts` (grouping, full tunnel in words,
  conflict note).

## More Information

**No verificado:**
- Equal prefix on Windows (and Linux): whether an OpenVPN/WireGuard client route beats Kongtrol's
  metric-10 route on another interface depends on interface metrics; not tested on a machine.
- macOS route selection was not checked against a primary source.
- A `route-nopull` / `pull-filter ignore "route"` in the user's `.ovpn`: whether openvpn logs the
  PUSH_REPLY before filtering (the panel would then list a server route that was dropped).
- Provider apps: Proton's advanced kill switch may block other tunnels regardless of routes
  (ADR-0103), so the panel never claims a rule wins over an app tunnel.

**Proposed, not built:** the rules table does not show `priority`, and here it does not matter (it
never decides against a VPN route). Where it matters (overlapping domain rules through different
VPNs), the Overlaps note already names the winner and links to Perfiles. If the owner wants it
visible, add a small "orden" hint (VPN priority number, or "por posición") in the "Sale por" cell
and a Priority field in the profile editor, both reading `vpns.<name>.priority` from
`GET /api/v1/vpns`, with `kongtrol profile set <name> --priority N` for parity — a separate ADR.

Code: `internal/app/vpnroutes.go`, `internal/api/handlers_vpn_routes.go`,
`internal/vpn/openvpn/configroutes.go`, `internal/vpn/wireguard/routes.go`,
`cmd/kongtrol/cmd_routes_vpn.go`, `internal/i18n/vpn_routes.go`,
`web/ui/src/components/rules/VpnRoutes.tsx`, `web/ui/src/components/rules/vpnRoutesModel.ts`.
