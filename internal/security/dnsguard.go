package security

import "net"

// DNSGuard prevents DNS leaks by forcing DNS resolution through the active
// tunnel's resolvers instead of the system's default (ISP) resolver.
//
// OS implementations are in dnsguard_windows.go, dnsguard_unix.go.
//
// Apply() is called after a VPN connects and the tunnel's DNS servers are known.
// Restore() is called when the VPN disconnects or kongtrol shuts down.
// Always call Restore() in a defer to avoid leaving the system in a broken DNS state.
type DNSGuard interface {
	// Apply overrides the system DNS resolver with the provided servers
	// for the given network interface (or system-wide if iface is "").
	Apply(iface string, dnsServers []net.IP) error

	// Restore returns the system DNS configuration to its pre-Apply state.
	Restore() error

	// IsActive reports whether DNS guard is currently applied.
	IsActive() bool
}
