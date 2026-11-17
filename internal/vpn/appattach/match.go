// Package appattach is the shared "attach mode" of provider adapters
// (forticlient, protonvpn, nordvpn, surfshark): however the provider's
// official app brings its tunnel up — an official CLI, GUI automation
// (corporate clients only) or the user clicking Connect — Kongtrol finds
// the tunnel by its network interface and attaches to it.
//
// Connect flow (Session.Connect):
//  1. tunnel already up → attach;
//  2. else, if auto_connect (default true) and the provider has an
//     automatic path → launch it; failure is not an error, fall to 3;
//  3. launch the app, report a vpn.UserAction{Kind: connect} and wait
//     (ctx-cancellable, wait_timeout) until the interface appears.
//
// Disconnect: the provider's CLI when there is one; otherwise report a
// vpn.UserAction{Kind: disconnect} and return vpn.ErrManualDisconnect.
package appattach

import (
	"net"
	"strings"
)

// Iface is one network interface as seen by the OS. On Windows Name is the
// adapter's friendly name and Description its driver description (e.g.
// "TAP-NordVPN Windows Adapter V9"); elsewhere Description is empty.
type Iface struct {
	Name        string
	Description string
	Up          bool
	Addrs       []net.IP
}

// Tunnel is an attached tunnel: interface name and its address (IPv4
// preferred, may be nil when the OS reports none).
type Tunnel struct {
	Name string
	IP   net.IP
}

// Matcher selects a provider's tunnel interface. Patterns are
// case-insensitive globs (`*` any run, `?` one char) matched against the
// interface name and description; Exclude removes matches (e.g. Proton's
// kill-switch dummies). RequireIPv4 skips interfaces without an IPv4
// address. Services are macOS network-service name globs checked with
// `scutil --nc list` when no interface matches (Network Extension VPNs run
// on generic utunN interfaces that carry no provider name).
type Matcher struct {
	Patterns    []string
	Exclude     []string
	RequireIPv4 bool
	Services    []string
}

// WithOverride returns m with its interface patterns replaced by the
// profile's `interface:` value (comma-separated globs) when it is set.
// The override also disables macOS service matching: the user named the
// interface explicitly.
func (m Matcher) WithOverride(override string) Matcher {
	pats := ParsePatterns(override)
	if len(pats) == 0 {
		return m
	}
	m.Patterns = pats
	m.Services = nil
	return m
}

// ParsePatterns splits a comma-separated pattern list, trimming blanks.
func ParsePatterns(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Match reports whether ifc is an up interface that m selects.
func (m Matcher) Match(ifc Iface) bool {
	if !ifc.Up {
		return false
	}
	if !anyGlob(m.Patterns, ifc.Name, ifc.Description) || anyGlob(m.Exclude, ifc.Name, ifc.Description) {
		return false
	}
	if m.RequireIPv4 {
		return preferredIP(ifc.Addrs).To4() != nil
	}
	return true
}

// Find returns the first interface in ifaces that m selects.
func Find(ifaces []Iface, m Matcher) (Tunnel, bool) {
	for _, ifc := range ifaces {
		if m.Match(ifc) {
			return Tunnel{Name: ifc.Name, IP: preferredIP(ifc.Addrs)}, true
		}
	}
	return Tunnel{}, false
}

func anyGlob(patterns []string, values ...string) bool {
	for _, p := range patterns {
		for _, v := range values {
			if v != "" && Glob(p, v) {
				return true
			}
		}
	}
	return false
}

// Glob reports whether s matches pattern case-insensitively. `*` matches
// any run of characters (including none), `?` exactly one; everything else
// is literal.
func Glob(pattern, s string) bool {
	p := []rune(strings.ToLower(pattern))
	r := []rune(strings.ToLower(s))
	pi, ri := 0, 0
	star, mark := -1, 0
	for ri < len(r) {
		switch {
		case pi < len(p) && (p[pi] == '?' || p[pi] == r[ri]):
			pi++
			ri++
		case pi < len(p) && p[pi] == '*':
			star, mark = pi, ri
			pi++
		case star >= 0:
			pi = star + 1
			mark++
			ri = mark
		default:
			return false
		}
	}
	for pi < len(p) && p[pi] == '*' {
		pi++
	}
	return pi == len(p)
}

// preferredIP returns the first IPv4 address, else the first address.
func preferredIP(addrs []net.IP) net.IP {
	var first net.IP
	for _, ip := range addrs {
		if v4 := ip.To4(); v4 != nil {
			return v4
		}
		if first == nil {
			first = ip
		}
	}
	return first
}

// Interfaces lists the OS network interfaces (Windows: with descriptions).
func Interfaces() []Iface { return osInterfaces() }
