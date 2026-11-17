// Package vpn defines the common interface all VPN adapters must implement.
package vpn

import (
	"context"
	"errors"
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
	// ManualReconnect is true when the tunnel can only be brought up by the
	// user in the provider's app (no CLI, or auto_connect disabled). The
	// watchdog then notifies ("manual_reconnect_required") instead of
	// reconnecting.
	ManualReconnect bool
}

// AttachPhase is how a provider adapter (attach mode, internal/vpn/appattach)
// brought its tunnel up, or what it is waiting for.
type AttachPhase string

const (
	// PhaseAttached: the tunnel was already up in the provider app; Kongtrol
	// took it over.
	PhaseAttached AttachPhase = "attached"
	// PhaseAuto: connected through the official CLI / GUI automation.
	PhaseAuto AttachPhase = "auto"
	// PhaseUser: the user connected in the app while Kongtrol waited.
	PhaseUser AttachPhase = "user"
	// PhaseWaitingUser: waiting for the user to connect in the app (Status
	// stays StatusConnecting). Reason is set when automation failed first.
	PhaseWaitingUser AttachPhase = "waiting_user"
	// PhaseWaitingDisconnect: `down` without a CLI; waiting for the user to
	// disconnect in the app (Status stays StatusConnected until it drops).
	PhaseWaitingDisconnect AttachPhase = "waiting_disconnect"
	// PhaseDropped: the tunnel dropped and only the user can reconnect it
	// in the app; Kongtrol re-attaches when it comes back.
	PhaseDropped AttachPhase = "dropped"
)

// AttachState is the user-facing state of a provider adapter. Clients turn
// it into a translated one-liner with MessageKey + i18n at render time.
type AttachState struct {
	Phase    AttachPhase `json:"phase"`
	App      string      `json:"app"`                // provider app, e.g. "Surfshark"
	Location string      `json:"location,omitempty"` // requested target, e.g. "United States"
	Reason   string      `json:"reason,omitempty"`   // short reason automation failed
	// TimeoutSeconds bounds PhaseWaitingUser.
	TimeoutSeconds int       `json:"timeout_s,omitempty"`
	Since          time.Time `json:"since"`
}

// Waiting reports whether the user has to act in the app.
func (s AttachState) Waiting() bool {
	return s.Phase == PhaseWaitingUser || s.Phase == PhaseWaitingDisconnect || s.Phase == PhaseDropped
}

// MessageKey returns the i18n key ("attach.*", ES + EN in internal/i18n)
// and its arguments for a one-line message about s. Clients resolve it in
// their own language.
func (s AttachState) MessageKey() (string, []any) {
	switch s.Phase {
	case PhaseAttached:
		return "attach.attached", []any{s.App}
	case PhaseAuto:
		if s.Location != "" {
			return "attach.auto_location", []any{s.App, s.Location}
		}
		return "attach.auto", []any{s.App}
	case PhaseUser:
		return "attach.user", []any{s.App}
	case PhaseWaitingUser:
		if s.Reason != "" {
			return "attach.automation_failed", []any{s.Reason, s.App}
		}
		return "attach.waiting_user", []any{s.App}
	case PhaseWaitingDisconnect:
		return "attach.waiting_disconnect", []any{s.App}
	case PhaseDropped:
		return "attach.dropped", []any{s.App}
	}
	return "", nil
}

// AttachReporter is implemented by provider adapters (attach mode).
// AttachState returns nil when there is nothing to report.
type AttachReporter interface {
	AttachState() *AttachState
}

// ErrManualDisconnect is returned (wrapped) by Disconnect when the adapter
// cannot disconnect the provider's app itself: the user must disconnect in
// the app. The adapter keeps reporting StatusConnected until the tunnel
// interface drops, with AttachState PhaseWaitingDisconnect.
var ErrManualDisconnect = errors.New("disconnect it in the provider app")

// ErrUserActionTimeout is returned (wrapped) by Connect when the adapter
// waited for the user to connect in the provider app and the wait ran out.
var ErrUserActionTimeout = errors.New("timed out waiting for the tunnel to come up in the provider app")

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
	TunnelName string // named tunnel as configured in the VPN client
	CertPath   string // path to client certificate
	KeyPath    string // path to private key
	ConfigPath string // path to VPN config file (e.g. .ovpn)
	Username   string
	Password   string            // plaintext; zero this field after Connect() returns
	Extra      map[string]string // adapter-specific parameters
	// OnAttach, when set, is called by provider adapters (attach mode) on
	// each AttachState phase change during Connect, so the caller can
	// notify the user right away. Optional.
	OnAttach func(AttachState)
}

// Configurable is an optional interface adapters may implement to accept their
// AdapterConfig before Connect() is called. This allows Status() to work even
// when the adapter has never been connected in the current process lifetime
// (e.g. a tunnel started externally and queried via `kongtrol status`).
type Configurable interface {
	Configure(cfg AdapterConfig)
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
