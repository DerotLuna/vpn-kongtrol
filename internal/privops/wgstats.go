package privops

import (
	"strconv"
	"strings"
)

// maxWGPeers bounds the peers one WGStats reply carries.
const maxWGPeers = 256

// ParseWGDump parses `wg show <iface> dump` (wg(8)): the first line is the
// interface (private-key, public-key, listen-port, fwmark) and is skipped;
// every other line is a peer: public-key, preshared-key, endpoint,
// allowed-ips, latest-handshake, transfer-rx, transfer-tx,
// persistent-keepalive, tab separated. Only the public key, handshake,
// counters and keepalive are kept, so no secret ever leaves the helper.
func ParseWGDump(out string) (WGStats, error) {
	st := WGStats{Peers: []WGPeerStats{}}
	lines := strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
	first := true
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if first { // interface line
			first = false
			if n := len(strings.Split(line, "\t")); n != 4 {
				return WGStats{}, invalid("wg dump: interface line has %d fields, want 4", n)
			}
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 8 {
			return WGStats{}, invalid("wg dump: peer line has %d fields, want 8", len(f))
		}
		if err := ValidateWGKey(f[0]); err != nil {
			return WGStats{}, invalid("wg dump: peer public key")
		}
		hs, err := strconv.ParseInt(f[4], 10, 64)
		if err != nil || hs < 0 {
			return WGStats{}, invalid("wg dump: latest-handshake %q", f[4])
		}
		rx, err := strconv.ParseUint(f[5], 10, 64)
		if err != nil {
			return WGStats{}, invalid("wg dump: transfer-rx %q", f[5])
		}
		tx, err := strconv.ParseUint(f[6], 10, 64)
		if err != nil {
			return WGStats{}, invalid("wg dump: transfer-tx %q", f[6])
		}
		ka := 0
		if f[7] != "off" {
			ka, err = strconv.Atoi(f[7])
			if err != nil || ka < 0 || ka > 65535 {
				return WGStats{}, invalid("wg dump: persistent-keepalive %q", f[7])
			}
		}
		if len(st.Peers) == maxWGPeers {
			break
		}
		st.Peers = append(st.Peers, WGPeerStats{PublicKey: f[0], LatestHandshake: hs,
			RxBytes: rx, TxBytes: tx, PersistentKeepalive: ka})
	}
	if first {
		return WGStats{}, invalid("wg dump: empty output")
	}
	return st, nil
}
