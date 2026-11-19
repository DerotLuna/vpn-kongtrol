package monitor

import (
	"encoding/hex"
	"net/netip"
	"strconv"
	"strings"
)

// TCPConn is one TCP connection that is established or being opened, as
// the OS lists it without admin rights. PID is 0 when unknown; Inode is
// the Linux socket inode (0 elsewhere).
type TCPConn struct {
	Local  netip.AddrPort
	Remote netip.AddrPort
	PID    uint32
	Inode  uint64
}

// parseProcNetTCP extracts the remote addresses of established (01) and
// SYN_SENT (02) connections from Linux /proc/net/tcp{,6} text.
func parseProcNetTCP(text string) []netip.Addr {
	conns := parseProcNetTCPConns(text)
	out := make([]netip.Addr, 0, len(conns))
	for _, c := range conns {
		out = append(out, c.Remote.Addr())
	}
	return out
}

// parseProcNetTCPConns parses established (01) and SYN_SENT (02) rows of
// Linux /proc/net/tcp{,6} text. Addresses are hex, each 32-bit word in
// host (little-endian) order; ports are hex numbers.
func parseProcNetTCPConns(text string) []TCPConn {
	var out []TCPConn
	for i, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if i == 0 || len(f) < 4 || (f[3] != "01" && f[3] != "02") {
			continue
		}
		remote, ok := parseProcAddrPort(f[2])
		if !ok {
			continue
		}
		local, _ := parseProcAddrPort(f[1])
		c := TCPConn{Local: local, Remote: remote}
		if len(f) > 9 {
			c.Inode, _ = strconv.ParseUint(f[9], 10, 64)
		}
		out = append(out, c)
	}
	return out
}

func parseProcAddrPort(s string) (netip.AddrPort, bool) {
	host, port, ok := strings.Cut(s, ":")
	if !ok {
		return netip.AddrPort{}, false
	}
	raw, err := hex.DecodeString(host)
	if err != nil || (len(raw) != 4 && len(raw) != 16) {
		return netip.AddrPort{}, false
	}
	for w := 0; w+4 <= len(raw); w += 4 {
		raw[w], raw[w+1], raw[w+2], raw[w+3] = raw[w+3], raw[w+2], raw[w+1], raw[w]
	}
	a, ok := netip.AddrFromSlice(raw)
	if !ok {
		return netip.AddrPort{}, false
	}
	p, err := strconv.ParseUint(port, 16, 16)
	if err != nil {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(a.Unmap(), uint16(p)), true
}
