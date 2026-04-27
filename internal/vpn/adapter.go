// Package vpn defines the common interface all VPN adapters must implement.
package vpn

import (
	"context"
	"net"
	"time"
)

// Status represents the current lifecycle state of a VPN tunnel.
type Status string

const (
	StatusDisconnected Status = "disconnected"
	StatusConnecting   Status = "connecting"
	StatusConnected    Status = "connected"
	StatusError        Status = "error"
)

// Normalize returns s if non-empty, or StatusDisconnected for the zero value.
// Adapters should call this before returning their internal status field to
// ensure callers never observe an empty string.
func (s Status) Normalize() Status {
	if s == "" {
		return StatusDisconnected
	}
	return s
}

// Capabilities describes optional features an adapter supports.
type Capabilities struct {
	SupportsSplitTunnel bool // adapter can limit routing to specific prefixes
	SupportsKillSwitch  bool // adapter has a built-in kill switch
	SupportsMultiConn   bool // can run alongside other active VPN tunnels
	SupportsReconnect   bool // adapter can reconnect without full teardown
}

// TunnelInfo contains live state for an active tunnel.
// Fields may be zero-valued when the tunnel is not connected.
type TunnelInfo struct {
	InterfaceName string
	AssignedIP    net.IP
	RemoteIP      net.IP
	DNS           []net.IP
	ConnectedAt   time.Time
	BytesSent     uint64
	BytesReceived uint64
}

// AdapterConfig holds resolved connection parameters passed to Connect().
// Credentials are only populated at connect time — callers must fetch them
// from the OS keychain immediately before calling Connect and zero them after.
type AdapterConfig struct {
	Host       string
	Port       int
	TunnelName string            // named tunnel as configured in the VPN client
	CertPath   string            // path to client certificate
	KeyPath    string            // path to private key
	ConfigPath string            // path to VPN config file (e.g. .ovpn)
	Username   string
	Password   string            // plaintext; zero this field after Connect() returns
	Extra      map[string]string // adapter-specific parameters
}

// VPNAdapter is the contract every VPN adapter must satisfy.
// Implementations live in sub-packages (openvpn, forticlient, protonvpn, …)
// and register themselves via init() using Register().
type VPNAdapter interface {
	// Connect establishes the VPN tunnel using the provided config.
	// It blocks until the tunnel is up or ctx is cancelled.
	Connect(ctx context.Context, cfg AdapterConfig) error

	// Disconnect tears down the tunnel cleanly.
	Disconnect(ctx context.Context) error

	// Reconnect tears down and re-establishes the tunnel using the last config.
	Reconnect(ctx context.Context) error

	// Status returns the current tunnel state without blocking.
	Status() Status

	// TunnelInfo returns live tunnel statistics.
	// Returns nil, nil when the tunnel is not connected.
	TunnelInfo() (*TunnelInfo, error)

	// Name returns the adapter type name (e.g. "openvpn", "forticlient").
	Name() string

	// Version returns the detected version of the underlying VPN client binary.
	Version() string

	// Capabilities returns what optional features this adapter supports.
	Capabilities() Capabilities
}
