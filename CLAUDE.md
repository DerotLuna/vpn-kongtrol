# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Module

`github.com/vpn-kongtrol/kongtrol` — Go 1.22+

## Common Commands

```bash
# Build
make build                  # current platform → build/dist/kongtrol
make build-all-cli          # cross-compile CLI for all platforms (CGO_ENABLED=0)
make build-tray-native      # tray app, native only (requires CGO)

# Test
go test ./...               # all packages
go test ./internal/policy/  # single package
go test -run TestEngine_ResolveIP ./internal/policy/  # single test
make test-race              # with race detector

# Vet / lint
go vet ./...
golangci-lint run ./...

# Tidy
go mod tidy
```

The tray binary (`cmd/kongtrol-tray`) requires `CGO_ENABLED=1` and can only be built natively per OS. The CLI binary (`cmd/kongtrol`) is pure Go and cross-compiles freely.

## Architecture

### Adapter pattern

Every VPN is a `vpn.VPNAdapter` (defined in `internal/vpn/adapter.go`). Each adapter package registers itself via `init()` using `vpn.Register(name, factory)`. The binary activates adapters through blank imports in `cmd/kongtrol/main.go`.

When adding a new adapter:
1. Create `internal/vpn/<name>/adapter.go` + `cli.go` + `util.go`
2. Call `vpn.Register("name", ...)` inside `func init()`
3. Add a blank import `_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/<name>"` in `cmd/kongtrol/main.go`
4. Add the type key to the `oneof` validator in `internal/config/schema.go`
5. Add an example entry in `configs/example.yaml`

`Status()` must call `.Normalize()` on the internal status field before returning — zero value (`""`) must never escape to callers.

### Data flow: connect lifecycle

```
cmd/kongtrol/main.go:connectProfile()
  → resolves credentials from OS keychain (config.GetCredential)
  → adapter.Connect(ctx, AdapterConfig)          // blocks until tunnel up
  → watchdog.MarkActive(name)                    // re-arms reconnect
  → dnsMgr.OnConnect(name, info.DNS)             // applies DNS guard if enabled
```

```
cmd/kongtrol/main.go:disconnect()
  → watchdog.MarkIntended(name)                  // suppresses reconnect
  → adapter.Disconnect(ctx)
  → dnsMgr.OnDisconnect(name)                    // restores DNS when last tunnel drops
```

### Key subsystems

| Package | Responsibility |
|---|---|
| `internal/vpn` | `VPNAdapter` interface, `Status` type, registry |
| `internal/config` | YAML schema (`Config`, `VPNConfig`), loader (env expansion, `~/` paths), keychain access |
| `internal/policy` | IP longest-prefix match + domain glob → VPN profile name |
| `internal/routing` | OS route table management; build-tagged per OS (`windows.go`, `linux.go`, `darwin.go`) |
| `internal/security` | Kill switch, DNS guard, leak tester, HMAC-signed audit log; OS implementations are build-tagged |
| `internal/monitor` | `Collector` (metrics snapshot), `Watchdog` (auto-reconnect with backoff), `DNSManager` (reference-counted DNS guard) |
| `internal/api` | Embedded HTTP server + REST handlers + WebSocket live feed; imports `web` package for `//go:embed` |
| `web/` | `//go:embed dashboard` in `web/assets.go`; dashboard files in `web/dashboard/` |
| `assets/` | `//go:embed logo.png` in `assets/embed.go`; `TrayIcon(size)` in `assets/icon.go` |
| `cmd/kongtrol` | cobra CLI: `main.go` (commands + wiring) + `wizard.go` (`kongtrol init`) |
| `cmd/kongtrol-tray` | systray app; starts the full daemon internally |

### OS-specific files

Build tags control OS implementations — never use runtime `switch runtime.GOOS` for system-level operations:

- `internal/routing/windows.go` / `linux.go` / `darwin.go`
- `internal/security/killswitch_windows.go` / `killswitch_linux.go` / `killswitch_darwin.go`
- `internal/security/dnsguard_windows.go` / `dnsguard_unix.go` (covers both Linux and macOS)
- `internal/vpn/openvpn/process_windows.go` / `process_unix.go`
- `internal/vpn/wireguard/wg_windows.go` / `wg_unix.go`

### `//go:embed` constraints

Go embed paths are relative to the source file — no `../` allowed. The layout solves this with two embed points:
- `web/assets.go` embeds `dashboard/` (adjacent directory)
- `assets/embed.go` embeds `logo.png` (adjacent file)
- `web/dashboard/logo.png` is a copy of `assets/logo.png` for web serving

### Config validation

`internal/config/schema.go` uses `go-playground/validator` tags. The `vpns` field has `validate:"required,min=1,dive"` — the `dive` tag is required for validator to recurse into `map[string]VPNConfig` values. Adding a new adapter type requires updating the `oneof` tag on `VPNConfig.Type`.

### Credential rule

`AdapterConfig.Password` must be zeroed immediately after `Connect()` returns. Passwords are fetched from the OS keychain (`config.GetCredential`) immediately before calling `Connect()` — never stored in config structs at rest.

### Watchdog internals

`monitor.Watchdog` polls adapter `Status()` every 5 seconds per profile goroutine. Call `MarkIntended(name)` **before** `Disconnect()` to suppress reconnect. Call `MarkActive(name)` **after** a successful `Connect()` to re-arm it. Backoff: `2s × 2^attempt`, capped at 5 minutes.

### DNS manager internals

`monitor.DNSManager` is reference-counted across simultaneous tunnels. `OnConnect` merges + deduplicates DNS servers from all active profiles and applies the guard. `OnDisconnect` re-applies with the remaining set, or restores original DNS when the last profile disconnects. Always call `ForceRestore()` on SIGTERM/panic path (it's deferred in `upCmd`).
