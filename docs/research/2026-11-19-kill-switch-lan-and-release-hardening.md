<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Kill switch LAN exception, resolved fallback and release hardening

Version 1.0.0 · 2026-11-19

Backs [ADR-0063](../adr/0063-kill-switch-lan-protected-read-and-release-hardening.md). Sources
accessed 2026-11-19.

## systemd-resolved: where a lookup goes once the tunnel link is gone

systemd-resolved.service(8), "Protocols and Routing"
(https://www.freedesktop.org/software/systemd/man/latest/systemd-resolved.service.html):

- A lookup that matches no routing domain "is sent to all DNS servers that are configured on links
  configured as the default route, as well as the globally configured DNS server". Only when none
  exists is a compiled-in fallback server used.
- A `~.` route-only domain on a link keeps other links out ("This will ensure that other links will
  not be considered for these queries"): that is what the helper sets on the tunnel link
  (ADR-0058). When the link disappears, its settings go with it, so the physical link (DHCP DNS,
  usually the router) answers.
- Consequence: a global `DNS=` pointing at the tunnel's resolver would be **added** to the physical
  link's servers, not replace them, so it does not stop the fallback. Global servers come from
  `resolved.conf` (file + reload); `resolvectl dns` takes a link. Not adopted (ADR-0063).

## What a full-tunnel block must leave open

- DHCP: renewing is a unicast DHCPREQUEST to the server; rebinding "MUST be broadcast to the
  0xffffffff IP broadcast address" (RFC 2131 §4.3.2, https://www.rfc-editor.org/rfc/rfc2131.html).
  255.255.255.255 and 0.0.0.0/8 stay in `neverBlock`, so a lease survives a long outage through
  rebinding even when the server's unicast address is blocked.
- ARP works below the IP firewall rules the helper writes (Windows Firewall, iptables `OUTPUT`, pf
  filter rules); IPv6 neighbor discovery uses link-local and multicast addresses, both exempt.
- IPv6 RDNSS: "The addresses for RDNSSes in the RDNSS option MAY be link-local addresses"
  (RFC 8106 §5.1, https://www.rfc-editor.org/rfc/rfc8106.html). Such a resolver stays reachable
  because fe80::/10 must stay exempt for neighbor discovery: documented residual.

## CI: caches, pinned tools, credentials

- zizmor audits `cache-poisoning` and `artipacked`: https://docs.zizmor.sh/audits/ (run locally:
  zizmor 1.32.0 `--offline`, the Windows release binary).
- `actions/setup-node` v4.4.0 at the pinned SHA `49933ea…`: `action.yml` has only the `cache`
  input (no `package-manager-cache`), so it caches nothing unless `cache` is set
  (`gh api repos/actions/setup-node/contents/action.yml?ref=49933ea5288caeca8642d1e84afbd3f7d6820020`).
  zizmor's low-confidence finding on it is a false positive for v4.
- `pnpm/action-setup` v4.4.0 (`fc06bc1…`): input `package_json_file` reads `packageManager`
  ("File path to the package.json to read "packageManager" configuration"), same API call.
- Renovate github-actions manager updates `with.version` of `goreleaser/goreleaser-action` and
  `pnpm/action-setup` (https://docs.renovatebot.com/modules/manager/github-actions/), so exact
  pins there do not go stale; it does not see `go install …@vX` in a `run:` script.
- GoReleaser v2.20.2: latest stable on 2026-11-19 (`gh api repos/goreleaser/goreleaser/releases`)
  and the version the v0.6.0 release run used (run 44583773287, log "Downloading
  …/v2.20.2/goreleaser_Linux_x86_64.tar.gz").
- govulncheck: `go list -m -versions golang.org/x/vuln` → v1.10.0 is the newest (tag commit
  2026-11-17).
- GitHub environments: "a job cannot access environment secrets until one of the required
  reviewers approves it"; deployment tag rules are matched against `GITHUB_REF`
  (https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments).
  Precedence between repository and environment secrets of the same name and auto-creation of a
  missing environment were not found in the pages read (**no verificado**); DEVELOPMENT.md asks to
  create the environment first and delete the repository copies, which does not depend on either.

## Not verified

- The LAN block on a real Ubuntu desktop with systemd-resolved during an outage, and a DHCP
  rebind while blocked (developed on Windows; unit tests only).
- The `release` environment in a tag run.
