//go:build linux

package sockowner

import (
	"encoding/hex"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

// procFiles are read in order; tests replace them.
var procFiles = []string{"/proc/net/tcp", "/proc/net/tcp6"}

// getuid is os.Getuid (a test seam).
var getuid = os.Getuid

func check(local, remote netip.AddrPort) (Result, error) {
	var rows []row
	for _, f := range procFiles {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		rows = append(rows, parseProcNet(string(b))...)
	}
	r, ok := match(rows, local, remote)
	if !ok || r.UID < 0 {
		return Result{}, ErrNotFound
	}
	res := Result{Owner: strconv.FormatInt(r.UID, 10)}
	// root can read this user's files anyway: trusting it leaks nothing.
	if r.UID == int64(getuid()) || r.UID == 0 {
		res.Verdict = Self
	} else {
		res.Verdict = Other
	}
	return res, nil
}

// parseProcNet reads /proc/net/tcp{,6}: sl local rem st tx:rx tr:when
// retrnsmt uid timeout inode. Addresses are hex with each 32-bit word in
// host (little-endian) order; ports are hex; st 0A is LISTEN.
func parseProcNet(text string) []row {
	var out []row
	for i, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if i == 0 || len(f) < 8 {
			continue
		}
		local, ok1 := parseProcAddrPort(f[1])
		remote, ok2 := parseProcAddrPort(f[2])
		if !ok1 || !ok2 {
			continue
		}
		uid, err := strconv.ParseInt(f[7], 10, 64)
		if err != nil {
			uid = -1
		}
		out = append(out, row{Local: local, Remote: remote, Listen: f[3] == "0A", UID: uid})
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
