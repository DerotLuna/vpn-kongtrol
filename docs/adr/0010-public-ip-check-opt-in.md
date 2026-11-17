<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0010: Public IP check is opt-in (Local only)

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Showing the public IP and comparing it with expected IPs helps detect leaks, but it requires
asking an external service, which then sees the user's IP. The "Local only" rule (`AGENTS.md`):
the app never sends user data off the machine; any outbound call other than the VPN itself must
be opt-in, documented in `/privacidad` (EN `/en/privacy`) and listed in `docs/SECURITY.md`.

## Decision Drivers

- Local only by default.
- Useful leak check for users who want it.
- Minimal calls: no periodic polling.

## Considered Options

1. Always-on periodic lookup — violates Local only.
2. No public IP feature — loses a useful leak check.
3. Opt-in lookup, event-driven, one configurable https endpoint, default ipify.

## Decision Outcome

Chosen: **option 3**, as documented on 2026-11-17 in `docs/SECURITY.md` and `docs/CLI.md`:
off by default; enabled by `security.public_ip_check.enabled: true` (or legacy
`leak_detection.enabled: true`) or `kongtrol config public-ip on`; runs once after a tunnel
connects/disconnects (at most one lookup per 30 s, coalesced) and on demand (`kongtrol check`,
dashboard, tray); never periodic; https-only endpoints; 5 s timeout; answer must be a bare IP of
at most 64 bytes. Kill switch and DNS guard do not depend on it. Default `https://api.ipify.org`:
"No visitor information is ever logged. Period." and open source (ipify.org, verified
2026-11-17 — a vendor claim, not audited).

Suggested alternative endpoints: only icanhazip (verified 2026-11-17 as Cloudflare-operated, per
its creator: https://major.io/p/a-new-future-for-icanhazip). checkip.amazonaws.com was removed
from the suggestions (`docs/SECURITY.md`, `docs/CONFIGURATION.md`, `configs/example.yaml`): no
official AWS documentation found (**no verificado**), and third-party reports say it returns the
`X-Forwarded-For` address behind proxies.

### Consequences

- Good: no outbound call unless the user asks; documented third party.
- Bad: leak detection is off for users who never enable it.

### Confirmation

`cmd/kongtrol/publicip_test.go`; `docs/SECURITY.md` network-connections table. Re-check status
when the code is committed (files `cmd/kongtrol/publicip.go`, `cmd/kongtrol/cmd_config_publicip.go`
untracked, `internal/security/leaktest.go` modified at the time of writing).

## More Information

- Code: `internal/security/leaktest.go`, `cmd/kongtrol/publicip.go`,
  `cmd/kongtrol/cmd_config_publicip.go`, `internal/config/schema.go` (`PublicIPCheckConfig`),
  `internal/daemonctl/publicip.go`
- Research: [public IP services](../research/2026-11-17-public-ip-services.md)
