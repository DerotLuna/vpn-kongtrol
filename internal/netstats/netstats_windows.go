//go:build windows

package netstats

import (
	"net"

	"golang.org/x/sys/windows"
)

// counters uses GetIfEntry2Ex (iphlpapi), which any user may call — unlike
// `wg show`, which needs admin. name is the interface alias, which is what
// net.InterfaceByName matches on Windows.
func counters(name string) (sent, received uint64, ok bool) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return 0, 0, false
	}
	row := windows.MibIfRow2{InterfaceIndex: uint32(iface.Index)}
	if err := windows.GetIfEntry2Ex(windows.MibIfEntryNormal, &row); err != nil {
		return 0, 0, false
	}
	return row.OutOctets, row.InOctets, true
}
