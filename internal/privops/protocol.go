package privops

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// MaxFrame caps one request or response (length prefix excluded).
const MaxFrame = 1 << 20

// Operation names on the wire. One request type per operation.
const (
	OpPing                         = "ping"
	OpRouteAdd                     = "route.add"
	OpRouteDelete                  = "route.delete"
	OpRouteFlush                   = "route.flush"
	OpDNSSet                       = "dns.set"
	OpDNSRestore                   = "dns.restore"
	OpKillSwitchOn                 = "killswitch.enable"
	OpKillSwitchOff                = "killswitch.disable"
	OpWGInstall                    = "wg.install"
	OpWGRemove                     = "wg.remove"
	OpWGAllowedIPs                 = "wg.allowed_ips"
	OpWGStats                      = "wg.stats"
	OpOpenVPNStart                 = "openvpn.start"
	OpOpenVPNStop                  = "openvpn.stop"
	OpOpenVPNStatus                = "openvpn.status"
	OpHostsWrite                   = "hosts.write"
	OpLeaseRenew                   = "lease.renew"
	OpBlockSet                     = "block.set"
	errCodeInvalid                 = "invalid"
	errCodeDenied                  = "denied"
	errCodeNotFound                = "not_found"
	errCodeUnsupported             = "unsupported"
	errCodeProtocol                = "protocol"
	errCodeWGDefaultEgressConflict = "wg_default_egress_conflict"
	errCodeFailed                  = "failed"
)

// Request is the envelope for every call.
type Request struct {
	V    int             `json:"v"`
	Op   string          `json:"op"`
	Args json.RawMessage `json:"args,omitempty"`
	// Lease is the client's lease id (see LeaseTTL). State a request
	// leaves behind is undone when the lease expires.
	Lease string `json:"lease,omitempty"`
}

// Response is the envelope for every reply.
type Response struct {
	V      int             `json:"v"`
	OK     bool            `json:"ok"`
	Code   string          `json:"code,omitempty"`
	Error  string          `json:"error,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	// Reason refines Code "denied" (DeniedError.Reason).
	Reason string `json:"reason,omitempty"`
}

// Single-value argument shapes.
type (
	IfaceArgs struct {
		Interface string `json:"interface"`
	}
	NameArgs struct {
		Name string `json:"name"`
	}
	SessionArgs struct {
		ID string `json:"id"`
	}
	HostsArgs struct {
		Entries []HostEntry `json:"entries"`
	}
)

// WriteFrame writes v as a 4-byte big-endian length followed by JSON.
func WriteFrame(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(b) > MaxFrame {
		return fmt.Errorf("privops: frame of %d bytes exceeds %d", len(b), MaxFrame)
	}
	buf := make([]byte, 4+len(b))
	binary.BigEndian.PutUint32(buf, uint32(len(b)))
	copy(buf[4:], b)
	_, err = w.Write(buf)
	return err
}

// ReadFrame reads one frame into v, rejecting oversize frames and unknown
// JSON fields.
func ReadFrame(r io.Reader, v any) error {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 || n > MaxFrame {
		return fmt.Errorf("privops: frame length %d outside 1-%d", n, MaxFrame)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return err
	}
	return DecodeStrict(b, v)
}

// DecodeStrict unmarshals JSON rejecting unknown fields and trailing data.
func DecodeStrict(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("privops: trailing data after JSON")
	}
	return nil
}

// ErrorCode maps an error to its wire code.
func ErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		return errCodeInvalid
	case errors.Is(err, ErrDenied):
		return errCodeDenied
	case errors.Is(err, ErrNotFound):
		return errCodeNotFound
	case errors.Is(err, ErrUnsupported):
		return errCodeUnsupported
	case errors.Is(err, ErrHelperOutdated):
		return errCodeProtocol
	case errors.Is(err, ErrWGDefaultEgressConflict):
		return errCodeWGDefaultEgressConflict
	}
	return errCodeFailed
}

// errorFromWire rebuilds a sentinel-wrapped error from a response.
func errorFromWire(code, msg, reason string) error {
	if code == errCodeDenied && knownDenyReason(reason) {
		return &DeniedError{Reason: reason, Detail: fmt.Sprintf("%s (helper)", strings.TrimPrefix(msg, "privops: denied: "))}
	}
	var base error
	switch code {
	case errCodeInvalid:
		base = ErrInvalidRequest
	case errCodeDenied:
		base = ErrDenied
	case errCodeNotFound:
		base = ErrNotFound
	case errCodeUnsupported:
		base = ErrUnsupported
	case errCodeProtocol:
		base = ErrHelperOutdated
	case errCodeWGDefaultEgressConflict:
		base = ErrWGDefaultEgressConflict
	default:
		base = ErrFailed
	}
	if msg == "" {
		return base
	}
	return fmt.Errorf("%w (helper: %s)", base, msg)
}
