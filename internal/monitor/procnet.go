package monitor

import (
	"encoding/hex"
	"net/netip"
	"strings"
)

// parseProcNetTCP extracts the remote addresses of established (01) and
// SYN_SENT (02) connections from Linux /proc/net/tcp{,6} text. Addresses
// are hex, each 32-bit word in host (little-endian) order.
func parseProcNetTCP(text string) []netip.Addr {
	var out []netip.Addr
	for i, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if i == 0 || len(f) < 4 || (f[3] != "01" && f[3] != "02") {
			continue
		}
		host, _, ok := strings.Cut(f[2], ":")
		if !ok {
			continue
		}
		raw, err := hex.DecodeString(host)
		if err != nil || (len(raw) != 4 && len(raw) != 16) {
			continue
		}
		for w := 0; w+4 <= len(raw); w += 4 {
			raw[w], raw[w+1], raw[w+2], raw[w+3] = raw[w+3], raw[w+2], raw[w+1], raw[w]
		}
		a, ok := netip.AddrFromSlice(raw)
		if !ok {
			continue
		}
		out = append(out, a.Unmap())
	}
	return out
}
