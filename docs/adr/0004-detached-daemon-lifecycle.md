<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0004: Detached daemon — views never stop tunnels, only `down`

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

If tunnels live inside the process of a terminal view, closing the terminal (or pressing
Ctrl+C in a live view) drops the VPN. With CLI, tray and dashboard as equal clients
([ADR-0003](0003-product-model-gui-first-cli-complete.md)), who owns the tunnels?

## Decision Drivers

- No terminal has to stay open.
- Leaving a view must never be confused with disconnecting.
- One state, shared by every client.

## Considered Options

1. Tunnels owned by the foreground `kongtrol up` process — Ctrl+C disconnects.
2. Tray process owns tunnels — CLI-only machines (servers) left out; tray exit kills tunnels.
3. One detached hidden daemon per user; CLI, tray and dashboard are clients of its local API.

## Decision Outcome

Chosen: **option 3** (`AGENTS.md`, "Daemon lifecycle"). `kongtrol up` starts the detached
`kongtrol daemon` (or reuses one) and opens a live view. Quitting any view (q / Ctrl+C,
`status --watch`, tray, dashboard) never stops the daemon or its tunnels; only `kongtrol down` or
the tray's "Salir y desconectar todo" does. The tray also has "Cerrar ícono (las VPN siguen)".
Local API on 127.0.0.1 with a token in `~/.kongtrol/api-token`; nothing leaves the host.

### Consequences

- Good: tunnels survive closed terminals; all clients show the same state.
- Bad: a background process the user may not notice — mitigated by explicit "VPN siguen" wording
  and `kongtrol down`.
- Watchdog intent: `ProfileService.Disconnect` calls `MarkIntended` before `Disconnect`, so an
  intended stop never triggers reconnect or the kill switch.

### Confirmation

Tests in `internal/daemonctl/spawn_test.go`; behaviour of `cmd/kongtrol/up_tui*.go` exit paths.

## More Information

- Code: `internal/daemonctl/spawn*.go`, `internal/daemonctl/client.go`,
  `cmd/kongtrol/daemon_lifecycle.go`, `cmd/kongtrol/daemon_runtime.go`,
  `internal/app/profile_service.go`, `cmd/kongtrol-tray/tray_menu.go`
- No external sources: internal design decision.
