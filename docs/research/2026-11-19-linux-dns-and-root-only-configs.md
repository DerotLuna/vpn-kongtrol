<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Linux: DNS of a dropped tunnel and root-only WireGuard files

Version 1.0.0 · 2026-11-19 · Backs [ADR-0058](../adr/0058-linux-dropped-tunnel-dns-and-root-only-configs.md)

Access date for every source: 2026-11-19. Sources read from the upstream repositories through the
GitHub API (`gh api repos/<owner>/<repo>/contents/...`); quotes are short.

## Questions

1. How do Ubuntu-style systems manage DNS, and how should a VPN set and undo its resolvers?
2. What does `wg-quick` do with `DNS =` and with a full-tunnel `AllowedIPs`?
3. Why can't a user process read `/etc/wireguard/wg0.conf`, and how do others get admin consent?

## Findings

| Source | What it says | Status |
|---|---|---|
| systemd `man/systemd-resolved.service.xml` — https://github.com/systemd/systemd/blob/main/man/systemd-resolved.service.xml | resolved provides `/run/systemd/resolve/stub-resolv.conf` (and `/run/systemd/resolve/resolv.conf`) for compatibility; "should not be used directly by applications" but through the `/etc/resolv.conf` symlink. Lookups are routed by per-link routing domains, "best matching" (most labels) wins; "to preferably route all DNS queries not explicitly matched by routing domain configuration to a specific link, configure a `~.` route-only domain on it". | verified |
| systemd `man/resolvectl.xml` — https://github.com/systemd/systemd/blob/main/man/resolvectl.xml | `revert LINK`: reverts the per-interface DNS configuration; "when a network interface disappears all configuration is lost automatically, an explicit reverting is not necessary". resolvconf compatibility: `-x` ("exclusive") "is mapped to an additional configured search domain of `~.`". `default-route` verb present. | verified |
| `resolvectl default-route` needs systemd ≥ 240 | Version when the verb appeared. | no verificado (from memory; handled as best effort in code) |
| wireguard-tools `src/wg-quick/linux.bash` — https://github.com/WireGuard/wireguard-tools/blob/master/src/wg-quick/linux.bash | `set_dns`: `printf 'nameserver %s\n' … \| resolvconf -a "$(resolvconf_iface_prefix)$INTERFACE" -m 0 -x`; `unset_dns`: `resolvconf -d … -f`. `add_default` for `*/0`: `wg set … fwmark $table`, `ip rule add not fwmark $table table $table`, `ip rule add table main suppress_prefixlength 0`. So with `DNS =` every lookup goes to the tunnel's servers while the interface exists, and a full tunnel sends everything into it even when its server is gone. | verified |
| wireguard-tools `src/man/wg-quick.8` — https://github.com/WireGuard/wireguard-tools/blob/master/src/man/wg-quick.8 | Configs are looked up as `/etc/wireguard/INTERFACE.conf`. | verified |
| `/etc/wireguard` is 0700 root and `wg0.conf` 0600 on Debian/Ubuntu packages; the WireGuard quick start recommends `umask 077` for keys | Permissions seen in the report (#5, #7) and common packaging. | no verificado (not checked on Ubuntu 26.04 here) |
| Kongtrol `cmd/kongtrol/cmd_service_linux.go` | Kongtrol already elevates GUI installs with `pkexec` (polkit dialog; exit 126 dismissed, 127 not authorized) and terminal ones with `sudo`. | verified (code) |
| Kongtrol `internal/privops/sys_unix.go` `readTrustedFile` | The helper refuses a root-owned file that is not world-readable unless the caller is root: "never reveals a root-only file the caller could not read". | verified (code) |
| Kongtrol v0.4.0 `internal/app/killswitch_service.go`, `internal/security/killswitch_linux.go` | Kill switch enabled whenever a protected tunnel was **connected**: `OUTPUT` accept tunnel/endpoints/LAN, then `-j DROP` (explains #6). | verified (git tag v0.4.0) |

## Conclusions

- Set VPN resolvers per link through resolved (`dns` + `domain ~.`), undo with `revert`; resolved
  forgets them when the link goes, so crashes cannot strand them. Rewriting the stub file is not
  what resolved expects.
- A dropped tunnel's resolvers must be released for split tunnels; for a full tunnel held by the
  kill switch keeping them avoids a lookup leak.
- `wg-quick`'s `DNS =` is exclusive by design; a split-DNS for policy profiles needs routing domains
  from the policies (follow-up).
- Root-only files: ask for admin consent each time (sudo / pkexec) rather than give the helper a
  standing read path.
