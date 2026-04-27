# ── Stage 1: Build ───────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w" \
    -o /out/kongtrol \
    ./cmd/kongtrol

# ── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM alpine:3.20

# OpenVPN binary (for OpenVPN adapter).
# protonvpn-cli must be installed separately if ProtonVPN is used.
RUN apk add --no-cache \
    openvpn \
    ca-certificates \
    iptables \
    ip6tables \
    iproute2

COPY --from=builder /out/kongtrol /usr/local/bin/kongtrol

# Create default config directory.
RUN mkdir -p /etc/kongtrol

EXPOSE 9741

# Mount your config + certs at /etc/kongtrol
VOLUME ["/etc/kongtrol"]

ENTRYPOINT ["kongtrol"]
CMD ["dashboard"]
