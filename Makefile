SHELL    := /usr/bin/bash
MODULE   := github.com/vpn-kongtrol/kongtrol
GO_PACKAGES := ./cmd/... ./internal/... ./assets ./web
VERSION  := $(shell v=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	if echo "$$v" | grep -q -- "-dirty$$"; then echo "$$v.$$(date +%Y%m%d%H%M%S)"; else echo "$$v"; fi)
LDFLAGS  := -ldflags "-X main.version=$(VERSION) -s -w"
DIST     := build/dist
SITE_DIR := landing
SITE_BIN := $(SITE_DIR)/_binaries

# On Windows (Git Bash), OS=Windows_NT is set by the environment.
ifeq ($(OS),Windows_NT)
  EXT := .exe
  # windowsgui: no console window attached, so the tray keeps running after
  # whatever terminal launched it closes (a plain console-subsystem build
  # dies with its parent console, which is wrong for a background tray app).
  TRAY_LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w -H=windowsgui"
else
  EXT :=
  TRAY_LDFLAGS := $(LDFLAGS)
endif

# ── Dashboard UI (web/ui: Vite + React + TypeScript, pnpm) ─────────────────────────
#
# web/ui/dist is embedded by web/ui_embed.go and served at /. It is
# rebuilt only when a UI source is newer than dist/index.html (plain make
# file dependencies), so `make build` with an unchanged UI never runs Node.
# Without pnpm, the UI step is skipped with a warning and the binary serves a
# "UI not built" page at /; release.yml always builds it.

UI_DIR      := web/ui
UI_INDEX    := $(UI_DIR)/dist/index.html
UI_DEPS     := $(UI_DIR)/node_modules/.modules.yaml
# Recursive wildcard in pure make (no $(shell find)), so the file list also
# works with make builds whose $(shell) cannot start bash.
rwildcard    = $(foreach d,$(wildcard $(1:=/*)),$(call rwildcard,$d,$2) $(filter $(subst *,%,$2),$d))
UI_SOURCES  := $(call rwildcard,$(UI_DIR)/src,*) $(call rwildcard,$(UI_DIR)/public,*) \
	$(call rwildcard,assets/brand,*) \
	$(wildcard $(UI_DIR)/index.html $(UI_DIR)/vite.config.ts $(UI_DIR)/tsconfig*.json \
	           $(UI_DIR)/package.json $(UI_DIR)/pnpm-lock.yaml)

$(UI_DEPS): $(UI_DIR)/package.json $(UI_DIR)/pnpm-lock.yaml
	cd $(UI_DIR) && pnpm install --frozen-lockfile
	@touch $@

$(UI_INDEX): $(UI_SOURCES) | $(UI_DEPS)
	cd $(UI_DIR) && pnpm run build

.PHONY: ui-install
ui-install: $(UI_DEPS)

.PHONY: ui-build
ui-build: $(UI_DEPS) $(UI_INDEX)

# Dev server at http://127.0.0.1:5173/ with hot reload; /api (REST +
# WebSocket) is proxied to the running daemon (KONGTROL_DAEMON_URL, default
# http://127.0.0.1:9741). Open the dashboard once at http://127.0.0.1:9741/
# first so the browser holds its login cookie.
.PHONY: ui-dev
ui-dev: $(UI_DEPS)
	cd $(UI_DIR) && pnpm run dev

.PHONY: ui-check
ui-check: $(UI_DEPS)
	cd $(UI_DIR) && pnpm run lint && pnpm run test && pnpm run build

# Prerequisite of every binary that embeds the UI (not the tray).
.PHONY: ui-if-available
ui-if-available:
	@if command -v pnpm >/dev/null 2>&1; then \
		"$(MAKE)" --no-print-directory ui-build; \
	else \
		echo "⚠ pnpm not found: building without the dashboard UI (/ shows how to build it)."; \
	fi

# ── Local build ──────────────────────────────────────────────────────────────

.PHONY: build
build: ui-if-available
	@mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/kongtrol$(EXT) ./cmd/kongtrol
	@echo "Built: $(DIST)/kongtrol$(EXT)  ($(VERSION))"

.PHONY: build-tray
build-tray:
	@mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/kongtrol-tray$(EXT) ./cmd/kongtrol-tray
	@echo "Built: $(DIST)/kongtrol-tray$(EXT)  ($(VERSION))"

.PHONY: build-all-cli
build-all-cli: ui-if-available
	@mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o $(DIST)/kongtrol-linux-amd64    ./cmd/kongtrol
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o $(DIST)/kongtrol-linux-arm64    ./cmd/kongtrol
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o $(DIST)/kongtrol-darwin-amd64   ./cmd/kongtrol
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o $(DIST)/kongtrol-darwin-arm64   ./cmd/kongtrol
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o $(DIST)/kongtrol-windows-amd64.exe ./cmd/kongtrol
	@echo "Cross-compiled CLI for all platforms → $(DIST)/"

# Tray app: only macOS needs CGO (Cocoa) and a native build. On Windows and
# Linux fyne.io/systray is pure Go (Win32 / D-Bus), so release builds cross-
# compile it (.goreleaser.yaml); the macOS .app comes from
# packaging/macos/build-app.sh (docs/adr/0025).
.PHONY: build-tray-native
build-tray-native:
	@mkdir -p $(DIST)
ifeq ($(OS),Windows_NT)
	@echo "Embedding exe icon (assets/logo.ico)..."
	go run github.com/akavel/rsrc@v0.10.2 -arch $$(go env GOARCH) -ico assets/logo.ico -o cmd/kongtrol-tray/rsrc_windows_$$(go env GOARCH).syso
endif
	go build $(TRAY_LDFLAGS) -o $(DIST)/kongtrol-tray$(EXT) ./cmd/kongtrol-tray
	@echo "Built tray: $(DIST)/kongtrol-tray$(EXT)"

# Signs every .exe in $(DIST) with a local Authenticode cert (Windows only).
# Generate one first: pwsh scripts/gen-devcert.ps1
#
# ⚠ LOCAL ONLY unless KONGTROL_SIGN_PFX points at a CA-issued cert: the default dev
# cert is self-signed and only trusted on the machine that generated it (see
# docs/SECURITY.md#unsigned-binaries-and-antivirus-false-positives). Signing a
# release binary with it before distributing it does NOT stop SmartScreen/Defender
# warnings for anyone else — don't sign release artifacts with the dev cert.
.PHONY: sign
sign:
ifneq ($(OS),Windows_NT)
	@echo "Code signing targets Windows .exe artifacts only; nothing to sign on this OS."
else
	@PFX="$${KONGTROL_SIGN_PFX:-$$HOME/.kongtrol/codesign/kongtrol-devsign.pfx}"; \
	if [ ! -f "$$PFX" ]; then \
		echo "No signing cert at $$PFX — generate one with: pwsh scripts/gen-devcert.ps1"; \
		exit 1; \
	fi; \
	if [ -z "$$KONGTROL_SIGN_PFX" ]; then \
		echo "⚠ Using the local self-signed dev cert — this only suppresses warnings on THIS machine."; \
		echo "  Do not distribute binaries signed this way as if they were verified for others."; \
	fi; \
	PW="$${KONGTROL_SIGN_PFX_PASSWORD:-$$(cat "$$PFX.password" 2>/dev/null)}"; \
	SIGNTOOL=$$(find "/c/Program Files (x86)/Windows Kits/10/bin" -iname signtool.exe -path "*x64*" 2>/dev/null | head -1); \
	if [ -z "$$SIGNTOOL" ]; then \
		echo "signtool.exe not found. Install the Windows SDK (App Certification Kit)."; \
		exit 1; \
	fi; \
	found=0; \
	for f in $(DIST)/*.exe; do \
		[ -f "$$f" ] || continue; \
		found=1; \
		echo "Signing $$f"; \
		MSYS_NO_PATHCONV=1 "$$SIGNTOOL" sign /f "$$PFX" /p "$$PW" /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 "$$f"; \
	done; \
	if [ "$$found" = "0" ]; then echo "No .exe files found in $(DIST)/ — run make build first."; fi
endif

# ── Test ─────────────────────────────────────────────────────────────────────

.PHONY: test
test:
	go test $(GO_PACKAGES)

.PHONY: test-verbose
test-verbose:
	go test -v $(GO_PACKAGES)

.PHONY: test-race
test-race:
	go test -race ./cmd/kongtrol ./internal/...

# ── Lint ─────────────────────────────────────────────────────────────────────

.PHONY: lint
lint:
	golangci-lint run $(GO_PACKAGES)

.PHONY: vet
vet:
	go vet $(GO_PACKAGES)

# ── Docker ───────────────────────────────────────────────────────────────────

.PHONY: docker-build
docker-build:
	docker build -f build/docker/Dockerfile -t vpn-kongtrol:$(VERSION) .

.PHONY: docker-up
docker-up:
	docker compose -f build/docker/docker-compose.yml up -d

.PHONY: docker-down
docker-down:
	docker compose -f build/docker/docker-compose.yml down

.PHONY: docker-logs
docker-logs:
	docker compose -f build/docker/docker-compose.yml logs -f

# ── Dev ───────────────────────────────────────────────────────────────────────

.PHONY: dev
dev:
	@which air > /dev/null || go install github.com/air-verse/air@latest
	air -c .air.toml

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: landing-sync-binaries
landing-sync-binaries:
	@if ! ls $(DIST)/kongtrol-* > /dev/null 2>&1; then \
		echo "No binaries found in $(DIST)/. Run: make build-all-cli"; \
		exit 1; \
	fi
	@mkdir -p $(SITE_BIN)
	@cp -f $(DIST)/kongtrol-* $(SITE_BIN)/
	@cd $(SITE_BIN) && \
		if command -v sha256sum > /dev/null 2>&1; then \
			sha256sum kongtrol-* > checksums.txt; \
		elif command -v shasum > /dev/null 2>&1; then \
			shasum -a 256 kongtrol-* > checksums.txt; \
		else \
			echo "No SHA256 tool found (sha256sum/shasum)."; \
			exit 1; \
		fi
	@echo "Updated landing binaries + checksums in $(SITE_BIN)/"

# ── Release ───────────────────────────────────────────────────────────────────

.PHONY: release
release:
	goreleaser release --clean

.PHONY: release-snapshot
release-snapshot:
	goreleaser release --snapshot --clean

# ── Clean ─────────────────────────────────────────────────────────────────────

.PHONY: clean
clean:
	rm -rf $(DIST)
	@echo "Cleaned $(DIST)/"

# ── Docs that must never drift (see docs/DOCS-TOOLING.md) ────────────────────
#
# Regenerates docs/CLI.md, docs/CLI.en.md, docs/cli.json (from the cobra
# command tree + internal/i18n) and docs/CONFIG.md, docs/CONFIG.en.md (from
# internal/config/schema.go). Never edit those files by hand.
.PHONY: docs
docs:
	go generate ./cmd/kongtrol

# Report-only checks, like CI's docs job (never fails): generated docs,
# change impact against main, landing i18n parity (web/ui's is in its tests).
.PHONY: docs-check
docs-check:
	go run ./cmd/kongtrol gen-docs --check
	go run ./internal/impact/impactcheck -base main -head HEAD
	@if [ -d landing/node_modules ]; then cd landing && pnpm run check:i18n; fi
