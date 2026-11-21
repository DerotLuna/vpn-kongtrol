# Research: Domain input for VPN rules

Version 1.0.0 · 2026-11-21

| Source | Accessed |
|---|---|
| [RFC 4343, DNS case insensitivity](https://www.rfc-editor.org/rfc/rfc4343) | 2026-11-21 |
| [Go IDNA lookup profile](https://pkg.go.dev/golang.org/x/net/idna) | 2026-11-21 |
| [Cloudflare domain-based split tunnels](https://developers.cloudflare.com/cloudflare-one/team-and-resources/devices/cloudflare-one-client/configure/route-traffic/split-tunnels/) | 2026-11-21 |
| [RFC 5737, IPv4 documentation addresses](https://www.rfc-editor.org/rfc/rfc5737) | 2026-11-21 |
| [RFC 3849, IPv6 documentation addresses](https://www.rfc-editor.org/rfc/rfc3849) | 2026-11-21 |

DNS treats ASCII case as equivalent. IDNA provides the ASCII spelling for Unicode
labels. Standard browsers perform this conversion when parsing a URL. The common
classifier fixtures verify equivalent input in the CLI and browser.

The Kongtrol matcher already uses `*.` for one label. Its route resolver can query
known names, but a wildcard cannot enumerate every DNS name. Cloudflare's desktop
client illustrates a different approach, installing routes as answers arrive, and
documents limitations with cached or separately resolved names. That approach
would need its own DNS design and verification in Kongtrol.

## Monitor benchmark

`go test ./internal/monitor -run '^$' -bench BenchmarkPolicyResolverMatchAddr`
compares four destination matches with 8 or 128 rules. On Windows amd64, the
128-rule comparison against `72acab1` reduced allocations from 405 to 22 per
iteration in an intermediate build with cached policy destinations. CPU timings
varied under concurrent machine load. VPN latency remains **no verificado**.
[Kongtrol issue #45](https://github.com/DerotLuna/vpn-kongtrol/issues/45)
(accessed 2026-11-21).
