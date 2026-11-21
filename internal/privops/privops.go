// Package privops is the closed contract of privileged network operations
// Kongtrol needs (routes, DNS, kill switch, WireGuard, OpenVPN, hosts file).
//
// Two implementations exist:
//   - Local runs each operation directly. It is used by the privileged helper
//     service itself and by a process that is already elevated.
//   - Client forwards each operation to the helper service over local IPC
//     (named pipe on Windows, Unix socket elsewhere).
//
// Default picks the right one. Every request is validated by Local before
// anything touches the system, so the helper never trusts the caller.
package privops

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
)

// ProtocolVersion is the IPC wire version. Bump it on any incompatible
// change to the request/response shapes below.
// 4: wg.stats (read-only WireGuard handshake/transfer counters, ADR-0030).
const ProtocolVersion = 4

// BuildVersion is the Kongtrol release this binary was built from. The CLI
// sets it at startup; Ping reports the helper's value so `service status`
// can warn when the installed helper is older than the CLI.
var BuildVersion = "dev"

// Sentinel errors. Client maps wire error codes back to these so callers can
// use errors.Is regardless of transport.
var (
	ErrHelperNotInstalled = errors.New("privops: helper service not installed or not running")
	ErrHelperOutdated     = errors.New("privops: helper service speaks an older protocol")
	ErrInvalidRequest     = errors.New("privops: invalid request")
	ErrDenied             = errors.New("privops: denied")
	ErrNotFound           = errors.New("privops: not found")
	ErrUnsupported        = errors.New("privops: unsupported on this system")
	ErrFailed             = errors.New("privops: operation failed")
)

// Ops is the complete list of privileged operations. Nothing else can be
// asked of the helper: no command, no path to execute, no free-form text.
type Ops interface {
	Ping(ctx context.Context) (PingResult, error)

	AddRoute(ctx context.Context, r Route) error
	DeleteRoute(ctx context.Context, r Route) error
	// FlushRoutes removes every route this implementation added on iface.
	FlushRoutes(ctx context.Context, iface string) error

	// SetDNS forces iface's resolvers. The previous configuration is
	// snapshotted on the first call and put back by RestoreDNS.
	SetDNS(ctx context.Context, req DNSRequest) error
	RestoreDNS(ctx context.Context, iface string) error

	KillSwitchEnable(ctx context.Context, req KillSwitchRequest) error
	KillSwitchDisable(ctx context.Context) error

	WGInstallTunnel(ctx context.Context, req WGInstallRequest) (WGTunnel, error)
	WGRemoveTunnel(ctx context.Context, req WGRemoveRequest) error
	WGSetAllowedIPs(ctx context.Context, req WGAllowedIPsRequest) error
	// WGStats reads one managed tunnel's per-peer latest handshake and
	// transfer counters. Read-only: it never changes the tunnel and never
	// returns keys other than peer public keys.
	WGStats(ctx context.Context, req WGStatsRequest) (WGStats, error)

	OpenVPNStart(ctx context.Context, req OpenVPNStartRequest) (OpenVPNSession, error)
	OpenVPNStop(ctx context.Context, id string) error
	OpenVPNStatus(ctx context.Context, id string) (OpenVPNState, error)

	// WriteHostsBlock replaces only the Kongtrol-marked block of the hosts
	// file. An empty list removes the block.
	WriteHostsBlock(ctx context.Context, entries []HostEntry) error

	// SetBlockedDestinations replaces the whole set of outbound
	// destinations blocked on every interface (per-destination kill switch
	// and policies with on_tunnel_down: block). An empty list removes the
	// block. Leased: expiry removes it (fail-open).
	SetBlockedDestinations(ctx context.Context, req BlockRequest) error

	// RenewLease keeps the caller's lease alive (see LeaseTTL).
	RenewLease(ctx context.Context) error
}

// BlockRequest is the complete set of outbound destinations to block.
// Prefixes may not contain loopback or multicast addresses.
type BlockRequest struct {
	Destinations []netip.Prefix `json:"destinations"`
}

// PingResult identifies the helper.
type PingResult struct {
	Version  string `json:"version"`
	Protocol int    `json:"protocol"`
	Caller   string `json:"caller"` // identity the helper authenticated
}

// Route is one route-table entry. Gateway is optional (zero Addr = none).
type Route struct {
	Destination netip.Prefix `json:"destination"`
	Gateway     netip.Addr   `json:"gateway,omitzero"`
	Interface   string       `json:"interface"`
	Metric      int          `json:"metric"`
}

// DNSRequest sets the resolvers of one interface.
type DNSRequest struct {
	Interface string       `json:"interface"`
	Servers   []netip.Addr `json:"servers"`
}

// KillSwitchRequest blocks all outbound traffic except loopback, the tunnel
// interfaces, the VPN endpoints and (optionally) RFC 1918 LAN ranges.
type KillSwitchRequest struct {
	Interfaces []string     `json:"interfaces,omitempty"`
	Endpoints  []netip.Addr `json:"endpoints,omitempty"`
	AllowLAN   bool         `json:"allow_lan"`
}

// WGInstallRequest brings a WireGuard tunnel up from a user .conf file.
// The helper reads and validates the file itself and runs a private copy.
type WGInstallRequest struct {
	ConfigPath    string `json:"config_path"`
	Name          string `json:"name,omitempty"` // default: config file base name
	DefaultEgress bool   `json:"default_egress,omitempty"`
}

// WGTunnel is the tunnel the helper brought up.
type WGTunnel struct {
	Name string `json:"name"`
}

// WGRemoveRequest stops and removes a WireGuard tunnel.
//
// ConfigPath is the caller's profile .conf for that tunnel. It lets the
// helper adopt a tunnel it did not start itself (brought up by an older
// Kongtrol through UAC, or by the WireGuard app): allowed only when the
// tunnel is a genuine WireGuard tunnel, the file passes the same ownership
// checks as WGInstallTunnel and its base name equals the tunnel name.
type WGRemoveRequest struct {
	Name       string `json:"name"`
	ConfigPath string `json:"config_path,omitempty"`
}

// WGAllowedIPsRequest replaces a peer's AllowedIPs on a managed tunnel.
// ConfigPath adopts an existing tunnel as in WGRemoveRequest.
type WGAllowedIPsRequest struct {
	Interface     string         `json:"interface"`
	PeerPublicKey string         `json:"peer_public_key"`
	AllowedIPs    []netip.Prefix `json:"allowed_ips"`
	ConfigPath    string         `json:"config_path,omitempty"`
}

// WGStatsRequest asks for the runtime counters of one WireGuard tunnel.
// ConfigPath (the caller's profile .conf) lets the helper accept a tunnel
// it did not start, with the same checks as WGRemoveRequest, but nothing
// is adopted or written: the request stays read-only.
type WGStatsRequest struct {
	Interface  string `json:"interface"`
	ConfigPath string `json:"config_path,omitempty"`
}

// WGPeerStats is one peer of a tunnel as `wg show <iface> dump` reports
// it, without the preshared key, endpoint or allowed IPs.
type WGPeerStats struct {
	PublicKey string `json:"public_key"`
	// LatestHandshake is the Unix time (seconds) of the last completed
	// handshake; 0 means never.
	LatestHandshake int64  `json:"latest_handshake"`
	RxBytes         uint64 `json:"rx_bytes"`
	TxBytes         uint64 `json:"tx_bytes"`
	// PersistentKeepalive in seconds; 0 means off.
	PersistentKeepalive int `json:"persistent_keepalive"`
}

// WGStats is the runtime state of one WireGuard tunnel.
type WGStats struct {
	Peers []WGPeerStats `json:"peers"`
}

// OpenVPNStartRequest starts openvpn for a user .ovpn file. The binary is
// resolved by the helper, never taken from the request.
type OpenVPNStartRequest struct {
	ConfigPath string `json:"config_path"`
	CertPath   string `json:"cert_path,omitempty"`
	KeyPath    string `json:"key_path,omitempty"`
}

// OpenVPNSession is a running openvpn. The caller talks to the management
// interface at MgmtAddr itself and must answer its "ENTER PASSWORD:" prompt
// with MgmtPassword before sending any command.
type OpenVPNSession struct {
	ID           string `json:"id"`
	PID          int    `json:"pid"`
	MgmtAddr     string `json:"mgmt_addr"`
	MgmtPassword string `json:"mgmt_password"`
}

// OpenVPNState reports a session's process state and its recent log lines.
type OpenVPNState struct {
	Running   bool     `json:"running"`
	ExitError string   `json:"exit_error,omitempty"`
	LogTail   []string `json:"log_tail,omitempty"`
	// PushReply holds the session's latest PUSH_REPLY lines, kept apart
	// from LogTail so the server's routes survive the log scrolling.
	PushReply []string `json:"push_reply,omitempty"`
}

// HostEntry is one "IP host" line inside the Kongtrol hosts block.
type HostEntry struct {
	IP   netip.Addr `json:"ip"`
	Host string     `json:"host"`
}

// Caller is the authenticated identity a request runs on behalf of.
type Caller struct {
	ID    string // Windows SID or Unix uid
	Name  string // display name for the audit log
	Admin bool   // elevated administrator / root
	UID   int    // Unix only (-1 on Windows)
	// Token is a Windows impersonation token (0 = act as the current
	// process). Files named in a request are opened while impersonating it,
	// so the helper can never read a file the caller could not read.
	Token uintptr
}

type callerKey struct{}

// WithCaller attaches the authenticated caller to ctx.
func WithCaller(ctx context.Context, c Caller) context.Context {
	return context.WithValue(ctx, callerKey{}, c)
}

// CallerFrom returns the caller attached to ctx, or the current process.
func CallerFrom(ctx context.Context) Caller {
	if c, ok := ctx.Value(callerKey{}).(Caller); ok {
		return c
	}
	return selfCaller()
}

// Denial reasons. Each has an i18n key "privops.denied.<reason>" with a
// one-line "what to do".
const (
	DenyWGNotManaged = "wg_not_managed" // tunnel not Kongtrol's and no matching profile
	DenyFileOwner    = "file_owner"     // config file owned by someone else
	DenyFileWritable = "file_writable"  // config file writable by other users
	DenySessionOwner = "session_owner"  // OpenVPN session of another user
	DenyNotWireGuard = "not_wireguard"  // name is not a genuine WireGuard tunnel
	denyKeyPrefix    = "privops.denied."
)

// DeniedError is a typed ErrDenied: Reason tells clients which guidance
// line to show; Detail stays in English for logs.
type DeniedError struct {
	Reason string
	Detail string
}

func (e *DeniedError) Error() string        { return "privops: denied: " + e.Detail }
func (e *DeniedError) Is(target error) bool { return target == ErrDenied }

// denied returns a typed denial.
func denied(reason, format string, args ...any) error {
	return &DeniedError{Reason: reason, Detail: fmt.Sprintf(format, args...)}
}

func knownDenyReason(r string) bool {
	switch r {
	case DenyWGNotManaged, DenyFileOwner, DenyFileWritable, DenySessionOwner, DenyNotWireGuard:
		return true
	}
	return false
}

// GuidanceKey returns the i18n key of the one-line "what to do" message for
// err, so callers can show it translated. ok is false for other errors.
func GuidanceKey(err error) (key string, ok bool) {
	var de *DeniedError
	if errors.As(err, &de) && knownDenyReason(de.Reason) {
		return denyKeyPrefix + de.Reason, true
	}
	switch {
	case errors.Is(err, ErrHelperNotInstalled):
		return "service.error.not_installed", true
	case errors.Is(err, ErrHelperOutdated):
		return "service.error.outdated", true
	case errors.Is(err, ErrDenied):
		return "service.error.denied", true
	}
	return "", false
}
