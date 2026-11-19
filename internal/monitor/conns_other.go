//go:build !windows && !linux

package monitor

import (
	"errors"
	"net/netip"
)

// tcpConnectionsSupported: macOS (and other OSes) would need lsof or
// sysctl parsing, not done: policy-traffic notices and the recent
// destinations list are off here (the API says supported:false).
const tcpConnectionsSupported = false

func activeRemoteAddrs() ([]netip.Addr, error) {
	return nil, errors.ErrUnsupported
}

func tcpConnections() ([]TCPConn, error) {
	return nil, errors.ErrUnsupported
}

func processNames([]TCPConn) map[TCPConn]string { return map[TCPConn]string{} }
