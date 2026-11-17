package privops

import (
	"context"
	"fmt"
	"net/netip"
	"slices"
	"strings"
)

// Blocked destinations: the per-destination kill switch (a protected
// tunnel dropped unexpectedly: only its destinations are blocked) and
// policies with on_tunnel_down: block. One op replaces the whole set, so
// the daemon is the single owner of what is blocked and a retry is always
// idempotent. Only BLOCK rules are used, never "block all + allow
// exceptions": in Windows Defender Firewall a block rule always beats an
// allow rule, so exceptions cannot be expressed with allow rules. A
// full-tunnel block is sent as the complement of its exceptions instead
// (see the app layer). The set is leased like the kill switch: a daemon
// that dies without clearing it gets it removed (fail-open).

// blockForbidden are ranges a blocked prefix may never overlap: blocking
// loopback or multicast would break the machine, not a destination.
var blockForbidden = []netip.Prefix{
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("255.255.255.255/32"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("ff00::/8"),
}

// NormalizeBlock validates req and returns its prefixes masked, unmapped,
// deduplicated and sorted.
func NormalizeBlock(req BlockRequest) (BlockRequest, error) {
	if len(req.Destinations) > MaxBlockPrefixes {
		return req, invalid("more than %d blocked destinations", MaxBlockPrefixes)
	}
	seen := map[netip.Prefix]bool{}
	out := make([]netip.Prefix, 0, len(req.Destinations))
	for _, p := range req.Destinations {
		if !p.IsValid() {
			return req, invalid("blocked destination is not a valid CIDR")
		}
		p = netip.PrefixFrom(p.Addr().Unmap(), p.Bits())
		if !p.IsValid() || p.Addr().Zone() != "" {
			return req, invalid("blocked destination %s is not a valid CIDR", p)
		}
		p = p.Masked()
		for _, f := range blockForbidden {
			if p.Overlaps(f) {
				return req, invalid("blocked destination %s overlaps loopback or multicast", p)
			}
		}
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b netip.Prefix) int {
		if c := a.Addr().Compare(b.Addr()); c != 0 {
			return c
		}
		return a.Bits() - b.Bits()
	})
	req.Destinations = out
	return req, nil
}

// SetBlockedDestinations replaces the blocked set; empty removes it.
func (l *Local) SetBlockedDestinations(ctx context.Context, req BlockRequest) error {
	req, err := NormalizeBlock(req)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(req.Destinations) == 0 {
		l.leases.release(resource{resBlock, ""})
		return failed("clear blocked destinations", sysBlockSet(ctx, nil))
	}
	if err := sysBlockSet(ctx, req.Destinations); err != nil {
		_ = sysBlockSet(ctx, nil) // never leave half a set behind
		l.leases.release(resource{resBlock, ""})
		return failed("block destinations", err)
	}
	l.leases.own(l.leaseOf(ctx), resource{resBlock, ""})
	return nil
}

// chunkPrefixes splits ps into strings of at most n comma-joined prefixes
// (firewall rules take an address list; command lines have a limit).
func chunkPrefixes(ps []netip.Prefix, n int, sep string) []string {
	var out []string
	for i := 0; i < len(ps); i += n {
		end := min(i+n, len(ps))
		parts := make([]string, 0, end-i)
		for _, p := range ps[i:end] {
			parts = append(parts, p.String())
		}
		out = append(out, strings.Join(parts, sep))
	}
	return out
}

// pfBlockRules returns the pf anchor rules for a block set: a table and
// one rule that rejects outbound traffic to it ("return" answers with a
// TCP reset / ICMP unreachable so apps fail fast instead of hanging).
func pfBlockRules(dsts []netip.Prefix) string {
	var b strings.Builder
	b.WriteString("# Kongtrol blocked destinations\n")
	fmt.Fprintf(&b, "table <kongtrol_block> const { %s }\n", strings.Join(chunkPrefixes(dsts, len(dsts)+1, " "), " "))
	b.WriteString("block return out quick to <kongtrol_block>\n")
	return b.String()
}
