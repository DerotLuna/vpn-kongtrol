package monitor

import (
	"net"
	"sort"
	"strings"
	"time"
)

// NetworkFingerprint describes the computer's network without admin
// rights: every up, non-loopback interface with its IPv4 addresses (not
// link-local) and whether it has a global IPv6 address. A watchdog that
// gave up retries when this changes (ADR-0098): another Wi-Fi, a cable
// plugged in, back online. IPv6 addresses themselves are left out because
// temporary (privacy) addresses rotate without any network change
// (RFC 8981). "" means offline, or the list could not be read.
func NetworkFingerprint() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	seen := make([]InterfaceAddrs, 0, len(ifaces))
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		entry := InterfaceAddrs{Name: ifc.Name}
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				entry.IPs = append(entry.IPs, ipn.IP)
			}
		}
		seen = append(seen, entry)
	}
	return FingerprintOf(seen)
}

// sleepGap is how much more wall time than monotonic time must pass
// between two polls to count as a sleep (a few seconds of clock
// adjustment are not one).
const sleepGap = 30 * time.Second

// WokeFromSleep reports whether the computer slept between two polls:
// Go's monotonic clock does not advance while the system is suspended on
// Windows, Linux and macOS, the wall clock does (ADR-0098).
func WokeFromSleep(monoElapsed, wallElapsed time.Duration) bool {
	return wallElapsed-monoElapsed > sleepGap
}

// InterfaceAddrs is one interface's name and addresses (FingerprintOf).
type InterfaceAddrs struct {
	Name string
	IPs  []net.IP
}

// FingerprintOf is the pure part of NetworkFingerprint: a stable string
// for the same network, whatever the order of interfaces and addresses.
func FingerprintOf(ifaces []InterfaceAddrs) string {
	var parts []string
	for _, ifc := range ifaces {
		v6 := false
		for _, ip := range ifc.IPs {
			if ip4 := ip.To4(); ip4 != nil {
				if ip4.IsLinkLocalUnicast() || ip4.IsLoopback() {
					continue
				}
				parts = append(parts, ifc.Name+"="+ip4.String())
				continue
			}
			if ip.IsGlobalUnicast() {
				v6 = true
			}
		}
		if v6 {
			parts = append(parts, ifc.Name+"=v6")
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
