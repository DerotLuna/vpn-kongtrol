package monitor

import (
	"context"
	"net"
	"time"
)

// Waiting for the network before restoring VPNs at start (ADR-0101). At
// sign-in the tray (and so the daemon) often starts before Wi-Fi or DHCP
// are ready; a connect then fails and would spend the watchdog's few
// attempts on nothing. User-level code cannot depend on the OS's own
// "network online" signals on every OS (systemd user units cannot order
// after network-online.target; docs/research/2026-11-23-network-ready-at-logon.md),
// so the daemon polls a pure-Go check, no admin and no packet sent.

// routeProbes are documentation addresses (RFC 5737 TEST-NET-1, RFC 3849):
// never answered, but routed like any internet address, so asking the OS
// which local address would reach them tells whether a default route
// exists and through which interface.
var routeProbes = []string{"192.0.2.1:9", "[2001:db8::1]:9"}

// NetworkReady reports whether the computer has a usable way to the
// internet: a route to an internet address through an up interface with
// an address that is neither loopback nor link-local, and that is not one
// of Kongtrol's own tunnels (ownTunnel, may be nil). A UDP "connect" only
// asks the OS's routing table: nothing is sent.
func NetworkReady(ownTunnel func(iface string) bool) bool {
	for _, target := range routeProbes {
		ip, ok := routeSource(target)
		if !ok {
			continue
		}
		iface := interfaceWithIP(ip)
		if iface == "" || (ownTunnel != nil && ownTunnel(iface)) {
			continue
		}
		return true
	}
	return false
}

// routeSource returns the local address the OS would use to reach target.
func routeSource(target string) (net.IP, bool) {
	conn, err := net.Dial("udp", target)
	if err != nil {
		return nil, false
	}
	defer func() { _ = conn.Close() }()
	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || !UsableSource(addr.IP) {
		return nil, false
	}
	return addr.IP, true
}

// UsableSource reports whether ip can carry internet traffic: set, not
// loopback, not link-local (169.254/16, fe80::/10: what an interface gets
// with no DHCP answer yet).
func UsableSource(ip net.IP) bool {
	return ip != nil && !ip.IsUnspecified() && !ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() && !ip.IsMulticast()
}

// interfaceWithIP returns the name of the up interface that has ip.
func interfaceWithIP(ip net.IP) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.Equal(ip) {
				return ifc.Name
			}
		}
	}
	return ""
}

// NetworkWait polls Ready every Interval for at most Timeout. After is the
// clock (time.After when nil), replaced in tests.
type NetworkWait struct {
	Ready    func() bool
	Interval time.Duration
	Timeout  time.Duration
	After    func(time.Duration) <-chan time.Time
}

// The wait at start: every 2 s, up to 90 s.
const (
	NetworkWaitInterval = 2 * time.Second
	NetworkWaitTimeout  = 90 * time.Second
)

func (w NetworkWait) after(d time.Duration) <-chan time.Time {
	if w.After != nil {
		return w.After(d)
	}
	return time.After(d)
}

// Wait returns true as soon as Ready does, false after Timeout or when ctx
// ends. Ready is checked first, so a ready network costs no wait.
func (w NetworkWait) Wait(ctx context.Context) bool {
	if w.Ready() {
		return true
	}
	for waited := time.Duration(0); waited < w.Timeout; waited += w.Interval {
		select {
		case <-ctx.Done():
			return false
		case <-w.after(w.Interval):
		}
		if w.Ready() {
			return true
		}
	}
	return false
}

// WaitNetworkChange blocks until fingerprint returns something other than
// what it returned first (the network changed, as for the watchdog's
// re-arm, ADR-0098) and ready then reports true, or ctx ends (false). It
// polls every interval; after is the clock (time.After when nil).
func WaitNetworkChange(ctx context.Context, fingerprint func() string, ready func() bool, interval time.Duration, after func(time.Duration) <-chan time.Time) bool {
	if after == nil {
		after = time.After
	}
	last := fingerprint()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-after(interval):
		}
		now := fingerprint()
		if now == last {
			continue
		}
		last = now
		if ready() {
			return true
		}
	}
}
