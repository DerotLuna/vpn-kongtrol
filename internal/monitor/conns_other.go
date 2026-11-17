//go:build !windows && !linux

package monitor

import (
	"errors"
	"net/netip"
)

// activeRemoteAddrs is not available here (macOS would need lsof or
// sysctl parsing): policy-traffic notices are skipped on this OS.
func activeRemoteAddrs() ([]netip.Addr, error) {
	return nil, errors.ErrUnsupported
}
