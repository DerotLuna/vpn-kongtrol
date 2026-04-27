MODULE   := github.com/vpn-kongtrol/kongtrol
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-X main.version=$(VERSION) -s -w"
DIST     := build/dist

# ── Local build ──────────────────────────────────────────────────────────────

.PHONY: build
build:
	go build $(LDFLAGS) -o $(DIST)/kongtrol       ./cmd/kongtrol
	@echo "Built: $(DIST)/kongtrol"

.PHONY: build-tray
build-tray:
	go build $(LDFLAGS) -o $(DIST)/kongtrol-tray  ./cmd/kongtrol-tray
	@echo "Built: $(DIST)/kongtrol-tray"

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
