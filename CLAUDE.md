# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Module

`github.com/vpn-kongtrol/kongtrol` — Go 1.25+

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
| `internal/api` | Embedded HTTP server + REST handlers (tunnels, routes, policies, VPN profiles, groups, security toggles, settings, scheduler rules, audit) + WebSocket live feed; imports `web` package for `//go:embed` |
| `web/` | `//go:embed dashboard` in `web/assets.go`; dashboard files (full management UI, vanilla JS/CSS, no bundler) in `web/dashboard/` |
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

`config.Validate(cfg)` (in `internal/config/loader.go`) exposes the same struct-tag + semantic
validation `Load()` runs, for callers that mutate an already-loaded `*Config` in memory before
persisting it — used by the dashboard API's trial-then-commit pattern (build a modified copy,
`config.Validate(&trial)`, only write if it passes).

### Dashboard REST API + machine-local preferences

`internal/api/handlers.go` follows one repeated pattern for every dashboard CRUD endpoint (policies,
VPN profiles, groups, scheduler rules, settings): `s.loadRuntimeConfig()` → mutate a copy →
`config.Validate` → `s.saveRuntimeConfig()` (marshals YAML, hot-swaps the policy engine, calls
`onPolicyUpdate`). Security toggles (`POST /api/v1/security/killswitch|dnsguard`) additionally call
`s.onSecurityToggle`, which re-runs `applyKillSwitchState()`/`applyDNSGuardState()` in
`cmd/kongtrol/main.go` so the change takes effect immediately, not just on the next connect.

VPN profile CRUD writes `kongtrol.yaml` + the OS keychain only — it does **not** hot-register the
adapter (the `adapters` map in `cmd/kongtrol/main.go` is built once at boot and shared,
unsynchronized, with the collector/watchdog goroutines), so responses include a
`restart_required` flag.

Settings that are genuinely local to one machine — not the shared `kongtrol.yaml` — live in
`~/.kongtrol/preferences.json` instead (`cmd/kongtrol/preferences.go`): CLI display language,
favorites, default group, and the dashboard's own bind/port override
(`kongtrol config dashboard set-port/set-bind`, applied in `loadConfig()` via
`applyDashboardPreferences`). This is intentionally **not** editable from the dashboard itself —
changing the port from the page serving that request would cut the connection mid-response.

### Credential rule

`AdapterConfig.Password` must be zeroed immediately after `Connect()` returns. Passwords are fetched from the OS keychain (`config.GetCredential`) immediately before calling `Connect()` — never stored in config structs at rest.

### Watchdog internals

`monitor.Watchdog` polls adapter `Status()` every 5 seconds per profile goroutine. Call `MarkIntended(name)` **before** `Disconnect()` to suppress reconnect. Call `MarkActive(name)` **after** a successful `Connect()` to re-arm it. Backoff: `2s × 2^attempt`, capped at 5 minutes.

### DNS manager internals

`monitor.DNSManager` is reference-counted across simultaneous tunnels. `OnConnect` merges + deduplicates DNS servers from all active profiles and applies the guard. `OnDisconnect` re-applies with the remaining set, or restores original DNS when the last profile disconnects. Always call `ForceRestore()` on SIGTERM/panic path (it's deferred in `upCmd`).

## Internationalization (i18n)

All user-visible strings — prompts, error messages, status output, banners — must be internationalized using `internal/i18n`. **Never hardcode strings directly in user-facing code.**

Rules:
- Add new keys to **both** `ES` and `EN` maps in `internal/i18n/i18n.go`
- In wizard/CLI code use `i18n.T(lang, key)` or `i18n.F(lang, key, args...)` — never `fmt.Printf("hardcoded string")`
- Spanish (`ES`) is the default language; English (`EN`) must always be present as well
- `confirm()` must use `i18n.YesNo(lang, def)` for the hint and `i18n.IsYes(lang, input)` for parsing — Spanish accepts `s/si/sí`, English accepts `y/yes`
- The language is selected once at wizard startup via `selectLanguage()` and stored on the `wizard` struct; all subsequent output goes through `w.t(key)` / `w.tF(key, args...)`

## Build on Windows

`make` targets use `SHELL := /usr/bin/bash` and Unix env-var syntax — they must be run from **Git Bash**, not from PowerShell or cmd.exe. Running `make build-all-cli` from PowerShell will fail with "The syntax of the command is incorrect".
