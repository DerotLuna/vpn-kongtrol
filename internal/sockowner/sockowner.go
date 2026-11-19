// Package sockowner tells whether the server end of a loopback TCP
// connection this process opened belongs to the current OS user, without
// admin rights (ADR-0062). The CLI and the tray use it before they send the
// local API token, so another user of the same machine who binds the
// daemon's port while it is stopped never receives it.
//
// Per OS: Windows reads the owning PID from GetExtendedTcpTable and the
// user SID from that process token; Linux reads the socket uid from
// /proc/net/tcp{,6}. Other systems (macOS) answer Unknown: no no-root API
// was verified there, so callers rely on the token rotating on every daemon
// start.
package sockowner

import (
	"errors"
	"net/netip"
)

// Verdict is who owns the server end of a connection.
type Verdict int

const (
	// Unknown: the OS gave no answer (unsupported OS, table read failed,
	// row not found). Callers proceed.
	Unknown Verdict = iota
	// Self: the current user (or root / this very process).
	Self
	// Other: a different user, read from the OS.
	Other
	// Denied: the owning process exists but this user may not inspect it
	// (Windows: another user's or an elevated process). PID is set so the
	// caller can compare it with its own daemon's PID file.
	Denied
)

func (v Verdict) String() string {
	switch v {
	case Self:
		return "self"
	case Other:
		return "other"
	case Denied:
		return "denied"
	}
	return "unknown"
}

// Result is the answer of Check.
type Result struct {
	Verdict Verdict
	// PID of the owning process (Windows; 0 elsewhere or unknown).
	PID uint32
	// Owner is the owner's id as the OS shows it (SID or uid), for logs.
	Owner string
}

// ErrUnsupported is returned with Unknown on systems without a check.
var ErrUnsupported = errors.New("sockowner: not supported on this OS")

// ErrNotFound is returned with Unknown when no table row matched.
var ErrNotFound = errors.New("sockowner: socket not found in the OS table")

// Check looks up the server end of the TCP connection local -> remote
// (local is this process's end, as net.Conn.LocalAddr reports it). It
// prefers the row of that very connection (no time-of-check gap) and falls
// back to the LISTEN row on remote's port.
func Check(local, remote netip.AddrPort) (Result, error) {
	return check(unmap(local), unmap(remote))
}

func unmap(ap netip.AddrPort) netip.AddrPort {
	return netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())
}

// row is one TCP table row: the fields every OS reader fills.
type row struct {
	Local, Remote netip.AddrPort
	Listen        bool
	PID           uint32 // Windows
	UID           int64  // Linux; -1 when unknown
}

// match finds the server row of the connection local -> remote, else the
// LISTEN row on remote's port (exact address first, then a wildcard).
func match(rows []row, local, remote netip.AddrPort) (row, bool) {
	for _, r := range rows {
		if !r.Listen && r.Local == remote && r.Remote == local {
			return r, true
		}
	}
	var wild row
	found := false
	for _, r := range rows {
		if !r.Listen || r.Local.Port() != remote.Port() {
			continue
		}
		if r.Local.Addr() == remote.Addr() {
			return r, true
		}
		if r.Local.Addr().IsUnspecified() && !found {
			wild, found = r, true
		}
	}
	return wild, found
}
