# Development

See [ARCHITECTURE.md](ARCHITECTURE.md) for system design and project layout, and [CLAUDE.md](../CLAUDE.md) for coding conventions used by contributors and AI assistants working in this repo.

## Build

```bash
make build             # current platform → build/dist/kongtrol
make build-all-cli      # cross-compile CLI for all platforms (CGO_ENABLED=0)
make build-tray-native  # tray app, native only (requires CGO)
```

> **Windows:** `make` targets use `SHELL := /usr/bin/bash` and Unix env-var syntax — run them from **Git Bash**, not PowerShell or cmd.exe.

## Test

```bash
go test ./cmd/... ./internal/... ./assets ./web    # all product packages
go test ./internal/policy/                          # single package
go test -run TestEngine_ResolveIP ./internal/policy/ # single test
make test-race                                       # with the race detector (requires a C toolchain)
go test -tags e2e ./internal/vpn/...                 # E2E tests against live daemons (opt-in)
```

## Lint

```bash
go vet ./cmd/... ./internal/... ./assets ./web
golangci-lint run ./cmd/... ./internal/... ./assets ./web
```

## Release

```bash
make release   # requires goreleaser
```

## Landing site

The public marketing site under `landing/` is independent from the embedded dashboard (`web/dashboard`):

```bash
make build-all-cli          # produce CLI binaries in build/dist/
make landing-sync-binaries  # copy + refresh checksums in landing/_binaries/
cd landing && pnpm install && pnpm run dev
```

## Docker

```bash
make docker-build
make docker-up
```

See [DOCKER.md](DOCKER.md) for deployment details.
