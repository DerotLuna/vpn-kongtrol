SHELL    := /usr/bin/bash
MODULE   := github.com/vpn-kongtrol/kongtrol
VERSION  := $(shell v=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	if echo "$$v" | grep -q -- "-dirty$$"; then echo "$$v.$$(date +%Y%m%d%H%M%S)"; else echo "$$v"; fi)
LDFLAGS  := -ldflags "-X main.version=$(VERSION) -s -w"
DIST     := build/dist
SITE_DIR := landing
SITE_BIN := $(SITE_DIR)/_binaries

# On Windows (Git Bash), OS=Windows_NT is set by the environment.
ifeq ($(OS),Windows_NT)
  EXT := .exe
else
  EXT :=
endif

# ── Local build ──────────────────────────────────────────────────────────────

.PHONY: build
build:
	@mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/kongtrol$(EXT) ./cmd/kongtrol
	@echo "Built: $(DIST)/kongtrol$(EXT)  ($(VERSION))"

.PHONY: build-tray
build-tray:
	@mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/kongtrol-tray$(EXT) ./cmd/kongtrol-tray
	@echo "Built: $(DIST)/kongtrol-tray$(EXT)  ($(VERSION))"

.PHONY: build-all-cli
build-all-cli:
	@mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o $(DIST)/kongtrol-linux-amd64    ./cmd/kongtrol
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o $(DIST)/kongtrol-linux-arm64    ./cmd/kongtrol
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o $(DIST)/kongtrol-darwin-amd64   ./cmd/kongtrol
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o $(DIST)/kongtrol-darwin-arm64   ./cmd/kongtrol
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o $(DIST)/kongtrol-windows-amd64.exe ./cmd/kongtrol
	@echo "Cross-compiled CLI for all platforms → $(DIST)/"

# Tray app requires CGO and must be built natively on each target OS.
# Use the goreleaser workflow in CI for tray cross-compilation.
.PHONY: build-tray-native
build-tray-native:
	@mkdir -p $(DIST)
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(DIST)/kongtrol-tray ./cmd/kongtrol-tray
	@echo "Built tray (native only): $(DIST)/kongtrol-tray"

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
	go test ./...

.PHONY: test-verbose
test-verbose:
	go test -v ./...

.PHONY: test-race
test-race:
	go test -race ./...

# ── Lint ─────────────────────────────────────────────────────────────────────

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: vet
vet:
	go vet ./...

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
