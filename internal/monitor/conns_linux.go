//go:build linux

package monitor

import (
	"net/netip"
	"os"
)

// activeRemoteAddrs returns the remote addresses of TCP connections that
// are established or being opened (/proc/net/tcp and tcp6).
func activeRemoteAddrs() ([]netip.Addr, error) {
	var out []netip.Addr
	for _, f := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		out = append(out, parseProcNetTCP(string(b))...)
	}
	return out, nil
}
