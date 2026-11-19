<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0063: Full-tunnel block covers the LAN without allow_lan; generic root read; release jobs without caches; install scripts trust only default folders

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

An independent review (2026-11-19) confirmed six findings in code that ADR-0058, ADR-0059 and the
install scripts introduced or kept:

1. **DNS leak during a full-tunnel outage (medium, regression from #37).** On Linux with
   systemd-resolved, the tunnel's DNS lives on the tunnel link (ADR-0058). When the interface
   disappears, resolved forgets it and sends lookups to the DNS of the remaining default-route
   links: the physical link, usually the router in the local subnet. The full-tunnel block always
   exempted the directly connected subnets (`localSubnets()`), even with `allow_lan: false`, so
   those lookups (names of every site the user opens) left outside the tunnel while the kill switch
   said it held all traffic. Windows and macOS fall back the same way once the tunnel's resolver
   is unreachable.
2. **Root read releases any text named `.ovpn` (low).** `profileimport.ReadProtected` (run as root
   after one sudo/pkexec prompt) released any `.ovpn` without a denied directive: the OpenVPN
   check is a denylist, so unknown lines pass. It did not check hard links either.
3. **Root read echoes file fragments (low).** Its refusals named WireGuard sections and keys and
   quoted an OpenVPN line (`wgconf.go`, `ovpnconf.go` validator messages), on the elevated
   command's stderr.
4. **Release jobs restore caches written by `main` runs (medium).** `setup-go` caches by default
   and the release UI job used the pnpm cache; a cache entry written by a `main` run (any merged
   PR) fed the tag build that is signed and published (zizmor `cache-poisoning`).
5. **Floating tool versions in CI (low).** GoReleaser `~> v2`, pnpm `10`, `govulncheck@latest`;
   checkouts persisted the job token in `.git/config` (zizmor `artipacked`).
6. **install.sh trusts the folder in `~/.kongtrol/install.manifest` (low).** Any program of the
   user can rewrite that file; the next install ran `sudo mv` / `sudo ln` into the folder it
   names. A `curl | sh` cut short could also run half the script.

## Decision Drivers

- No traffic or DNS outside the tunnel while the kill switch says it holds everything.
- Nothing crosses the root boundary but a VPN profile, and nothing about a file that is not one.
- Release artifacts are built only from the tag's sources and pinned tools.
- Root (sudo/UAC) never writes where a user-writable file says.
- Standards first; keep DHCP, NDP/ARP and the VPN endpoints working during a block.

## Considered Options

For 1:
1. **Exempt the local subnets from a full-tunnel block only with `allow_lan`** (broadcast,
   multicast and link-local stay exempt as before).
2. Keep a systemd-resolved global DNS pointing at the tunnel DNS while the block holds.
3. Block only port 53 to the LAN.

For 6: drop the manifest folder entirely / **reuse it only when it is a default folder; another
folder only through `KONGTROL_INSTALL_DIR`**.

## Decision Outcome

1. **Option 1.** `KillSwitchService.Apply` keeps `localSubnets()` out of split-tunnel blocks (as
   before) and, for a full-tunnel block, adds them to the exceptions only when
   `KillSwitchConfig.LANAllowed()` (`allow_lan: true` or legacy `mode: loose`). With
   `allow_lan: false` a drop now also blocks the LAN, the router included, so a fallback lookup to
   the router fails closed. Still exempt: `neverBlock` (0.0.0.0/8 and 255.255.255.255 for DHCP,
   169.254/16, fe80::/10 for NDP, multicast, loopback) and every VPN endpoint. ARP is below the IP
   firewall. A unicast DHCP renewal to the server in the blocked subnet fails; the client then
   rebinds by broadcast to 255.255.255.255 (RFC 2131 §4.3.2), which stays exempt.
   The notice says so: `killswitch.event.blocking_full_lan` ("blocks the internet and your local
   network (printers, NAS) … turn on "Allow the local network"") when the LAN is not allowed,
   `blocking_full` otherwise.
   Option 2 was **not implemented**: resolved sends a lookup that matches no routing domain to the
   default-route links' servers **and** the global ones (systemd-resolved.service(8)), so a global
   server does not stop the fallback; global servers are also only set in `resolved.conf` (a file
   edit + reload, new helper operation, protocol bump). The firewall block of option 1 is what makes
   the fallback fail closed. Option 3 needs a port-aware helper operation (protocol change); the
   prefix block covers it with today's protocol.
   Residual, documented: with `allow_lan: true`, lookups to a resolver in the LAN still leave during
   an outage (the user chose to keep the LAN). A resolver at an IPv6 link-local address (RA RDNSS,
   RFC 8106) stays reachable because fe80::/10 must stay exempt for neighbor discovery.
2. **`ReadProtected`** also refuses a file whose link count is not 1 (`Stat_t.Nlink`, Unix; Windows
   has no root-only profile files) and an `.ovpn` without a client profile shape: at least one
   `remote` (top level or in `<connection>`) and `client` or `tls-client`, outside inline blocks.
   WireGuard keeps the helper's validator (one `[Interface]` with a `PrivateKey`, only WireGuard
   keys).
3. **Every refusal of the root read is the generic `api.err.import_protected_path`** (the path the
   caller typed + what to do), also too big and validator failures; the hidden
   `profile read-protected` command prints only that. The normal import of a file the user can read
   keeps the detailed errors (the file is theirs).
4. **Release jobs restore no cache:** `cache: false` on every `setup-go` of `release.yml`, no pnpm
   cache in the `ui` job. The jobs that hold signing or publishing secrets (`windows`, `release`)
   run in the GitHub environment `release`; the owner creates it protected (tag rule `v*`, optional
   reviewer) and moves the secrets there (`docs/DEVELOPMENT.md`). The OIDC subject of those jobs
   becomes `repo:…:environment:release`; cosign keyless and attestations identify the workflow file
   and ref (`release.yml@refs/tags/v…`), which does not change.
5. **Exact versions:** GoReleaser `v2.20.2` (the version the v0.6.0 release ran, run 44583773287) in
   `release.yml` and `ci.yml`, kept current by Renovate (its github-actions manager updates
   `goreleaser/goreleaser-action` `with.version`); pnpm from `web/ui/package.json`
   `packageManager` (`pnpm/action-setup` `package_json_file`, exact `10.33.2`, bumped by Renovate);
   `govulncheck@v1.10.0` (bumped by hand; advisories come from the online database).
   `persist-credentials: false` on every checkout: nothing pushes with git (publishers use their
   own tokens through the API). **No Dependabot:** `renovate.json` (`config:recommended`,
   `helpers:pinGitHubActionDigestsToSemver`) already updates the SHA-pinned actions; two bots
   would open duplicate PRs.
6. **install.sh / install.ps1** reuse a manifest folder only when it is a default one
   (`/usr/local/bin`, `~/.local/bin`; `%LOCALAPPDATA%\kongtrol\bin`; macOS app folder
   `~/Applications` or `/Applications`); otherwise they say so and how to choose it
   (`KONGTROL_INSTALL_DIR`). sudo is used only for that explicit or a default folder. Both scripts
   run from a function called on the last line (`main "$@"`, `Invoke-KongtrolInstall`), so a cut
   download is a syntax error that runs nothing.

### Consequences

- Good: a full-tunnel drop with `allow_lan: false` leaks no lookups to the LAN resolver; the root
  prompt releases only a VPN profile and says nothing about any other file; release builds use only
  the tag's sources and pinned tools; a rewritten manifest cannot steer sudo.
- Bad: with `allow_lan: false` (the schema default; the wizard writes `allow_lan: true`) printers,
  NAS and other LAN devices are unreachable until the VPN returns, the notice says how to keep
  them. A refused root read no longer says which line failed; the hint says to import a user copy
  to see it. Release jobs are slower without caches (minutes, once per tag). Each release may wait
  for an environment approval once the owner adds a reviewer. An install outside the default
  folders needs `KONGTROL_INSTALL_DIR` on every update.
- Neutral: zizmor still reports `cache-poisoning` (low confidence) on the two `setup-node` steps of
  `release.yml`: v4.4.0 at the pinned SHA has no implicit cache (only the `cache` input, unset).

### Confirmation

- `internal/app/killswitch_service_test.go` `TestKillSwitch_FullTunnelLocalSubnetsFollowAllowLAN`;
  `cmd/kongtrol/daemon_watchdog_test.go` `TestKillSwitchBlockKey`.
- `internal/profileimport/protected_test.go` (generic refusal, no echo, `.ovpn` shapes, hard link
  on Unix); `cmd/kongtrol/protected_read_test.go` `TestProfileReadProtectedCmd_GenericRefusal`.
- `packaging/tests/install-scripts.sh` (CI `packaging` job): manifest folder and truncation for both
  scripts, offline.
- actionlint 1.7.7 + shellcheck 0.11.0 clean on the workflows; zizmor 1.32.0 offline: no
  `artipacked`, `release.yml` `cache-poisoning` only on the two `setup-node` steps above.
- Not verified on a real Linux desktop or in a tag run: the full-tunnel LAN block with
  systemd-resolved, DHCP rebinding during a long outage, the `release` environment (**no
  verificado**).

## More Information

- Research: [Kill switch LAN exception, resolved fallback and release hardening](../research/2026-11-19-kill-switch-lan-and-release-hardening.md).
- Code: `internal/app/killswitch_service.go`, `cmd/kongtrol/daemon_watchdog.go`,
  `internal/profileimport/protected.go`, `internal/profileimport/linkcount_*.go`,
  `cmd/kongtrol/protected_read.go`, `.github/workflows/{release,ci,docs,landing}.yml`,
  `install.sh`, `install.ps1`, `packaging/tests/install-scripts.sh`.
- Related: ADR-0009 (kill switch per tunnel), ADR-0025, ADR-0034, ADR-0046, ADR-0057, ADR-0058,
  ADR-0059.
