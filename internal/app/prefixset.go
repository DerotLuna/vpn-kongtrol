package app

import (
	"net"
	"net/netip"
	"slices"
)

// Prefix set helpers for the block set (kill switch + on_tunnel_down).

var (
	allIPv4 = netip.MustParsePrefix("0.0.0.0/0")
	allIPv6 = netip.MustParsePrefix("::/0")

	// neverBlock are ranges the block set must never cover: loopback,
	// link-local, multicast, broadcast and "this network" (DHCP). The
	// helper rejects loopback/multicast overlaps anyway.
	neverBlock = []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("169.254.0.0/16"),
		netip.MustParsePrefix("224.0.0.0/4"),
		netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("fe80::/10"),
		netip.MustParsePrefix("ff00::/8"),
	}

	// lanRanges stay reachable while a full tunnel is blocked when the
	// kill switch allows the LAN.
	lanRanges = []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.168.0.0/16"),
		netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("fc00::/7"),
	}
)

// subtractPrefix returns p minus every prefix in excl, as a minimal list
// of prefixes (p is split in halves around each excluded range).
func subtractPrefix(p netip.Prefix, excl []netip.Prefix) []netip.Prefix {
	for _, x := range excl {
		if !p.Overlaps(x) {
			continue
		}
		if x.Bits() <= p.Bits() {
			return nil // x covers p
		}
		lo, hi := splitPrefix(p)
		return append(subtractPrefix(lo, excl), subtractPrefix(hi, excl)...)
	}
	return []netip.Prefix{p}
}

// subtractAll returns ps minus excl.
func subtractAll(ps, excl []netip.Prefix) []netip.Prefix {
	var out []netip.Prefix
	for _, p := range ps {
		out = append(out, subtractPrefix(p, excl)...)
	}
	return out
}

// splitPrefix returns the two halves of p.
func splitPrefix(p netip.Prefix) (netip.Prefix, netip.Prefix) {
	bits := p.Bits() + 1
	lo := netip.PrefixFrom(p.Addr(), bits)
	b := p.Addr().AsSlice()
	idx := (bits - 1) / 8
	b[idx] |= 0x80 >> ((bits - 1) % 8)
	a, _ := netip.AddrFromSlice(b)
	return lo, netip.PrefixFrom(a, bits)
}

// hostPrefix is the single-address prefix of a.
func hostPrefix(a netip.Addr) netip.Prefix {
	a = a.Unmap()
	return netip.PrefixFrom(a, a.BitLen())
}

// prefixFromIPNet converts a net.IPNet (v4 may be 16-byte) to a masked
// prefix; ok is false when it is not valid.
func prefixFromIPNet(n net.IPNet) (netip.Prefix, bool) {
	a, ok := netip.AddrFromSlice(n.IP)
	if !ok {
		return netip.Prefix{}, false
	}
	ones, bits := n.Mask.Size()
	if bits == 0 {
		return netip.Prefix{}, false
	}
	a = a.Unmap()
	if a.Is4() && bits == 128 {
		ones -= 96
	}
	p := netip.PrefixFrom(a, ones)
	if !p.IsValid() {
		return netip.Prefix{}, false
	}
	return p.Masked(), true
}

// isDefaultish reports a route that sends everything through a tunnel:
// a default route or one of the two halves (OpenVPN def1, WireGuard).
func isDefaultish(p netip.Prefix) bool { return p.Bits() <= 1 }

// dedupePrefixes sorts and removes duplicates.
func dedupePrefixes(ps []netip.Prefix) []netip.Prefix {
	ps = slices.Clone(ps)
	slices.SortFunc(ps, func(a, b netip.Prefix) int {
		if c := a.Addr().Compare(b.Addr()); c != 0 {
			return c
		}
		return a.Bits() - b.Bits()
	})
	return slices.Compact(ps)
}

// localSubnets are the networks this machine is directly connected to
// (LAN, Wi-Fi). They stay reachable even if a dropped tunnel routed a
// larger range over them; overridable in tests.
var localSubnets = func() []netip.Prefix {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []netip.Prefix
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifi.Addrs()
		for _, a := range addrs {
			n, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if p, ok := prefixFromIPNet(*n); ok && p.Bits() >= 8 && !p.Addr().IsLinkLocalUnicast() {
				out = append(out, p)
			}
		}
	}
	return out
}
