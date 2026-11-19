//go:build !linux && !windows

package sockowner

import "net/netip"

// macOS and the rest: no no-root owner lookup was verified (ADR-0062).
// lsof only lists this user's processes without root and libproc needs cgo,
// so callers proceed and rely on the token rotating on every daemon start.
func check(netip.AddrPort, netip.AddrPort) (Result, error) {
	return Result{}, ErrUnsupported
}
