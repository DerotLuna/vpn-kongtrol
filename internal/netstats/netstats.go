// Package netstats reads per-interface byte counters from the OS. It needs
// no admin rights on any OS, so it is the generic source of tunnel traffic
// for adapters whose own tooling cannot report it unprivileged.
package netstats

import (
	"bufio"
	"strconv"
	"strings"
)

// Counters returns the bytes sent and received by the network interface
// called name (the OS name or, on Windows, the interface alias). ok is false
// when the interface does not exist or the OS gives no counters.
func Counters(name string) (sent, received uint64, ok bool) {
	if strings.TrimSpace(name) == "" {
		return 0, 0, false
	}
	return counters(name)
}

// parseNetstatIB reads `netstat -ibn -I <if>` (macOS/BSD) output and
// returns the link-level row's Ibytes/Obytes. Header columns are located by
// name because the Address column is empty on point-to-point (utun) links.
func parseNetstatIB(out, name string) (sent, received uint64, ok bool) {
	sc := bufio.NewScanner(strings.NewReader(out))
	ibytes, obytes := -1, -1
	var header []string
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "Name" {
			header = fields
			for i, f := range fields {
				switch f {
				case "Ibytes":
					ibytes = i
				case "Obytes":
					obytes = i
				}
			}
			continue
		}
		if ibytes < 0 || obytes < 0 || strings.TrimSuffix(fields[0], "*") != name {
			continue
		}
		// Only the <Link#N> row carries whole-interface totals. A row with
		// an empty Address column has one field fewer than the header, so
		// index the byte columns from the right.
		if len(fields) < 3 || !strings.HasPrefix(fields[2], "<Link") {
			continue
		}
		shift := len(header) - len(fields)
		ri, ro := ibytes-shift, obytes-shift
		if ri < 0 || ro < 0 || ri >= len(fields) || ro >= len(fields) {
			continue
		}
		in, errIn := strconv.ParseUint(fields[ri], 10, 64)
		outB, errOut := strconv.ParseUint(fields[ro], 10, 64)
		if errIn != nil || errOut != nil {
			continue
		}
		return outB, in, true
	}
	return 0, 0, false
}
