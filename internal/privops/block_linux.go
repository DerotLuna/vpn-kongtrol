//go:build linux

package privops

import (
	"context"
	"net/netip"
)

// blockChain holds the blocked destinations (iptables and ip6tables), apart
// from the kill switch chain; OUTPUT jumps to it first.
const blockChain = "KONGTROL_BLOCK"

func ip6tables() (string, error) {
	return trustedBinary("/usr/sbin/ip6tables", "/sbin/ip6tables", "/usr/bin/ip6tables")
}

func blockClear(ctx context.Context, ipt string) {
	for i := 0; i < 4; i++ { // remove every jump we may have inserted
		if _, err := run(ctx, ipt, "-D", "OUTPUT", "-j", blockChain); err != nil {
			break
		}
	}
	_, _ = run(ctx, ipt, "-F", blockChain)
	_, _ = run(ctx, ipt, "-X", blockChain)
}

// sysBlockSet replaces the blocked set: REJECT rules (apps fail fast
// instead of hanging) in a dedicated chain per address family.
func sysBlockSet(ctx context.Context, dsts []netip.Prefix) error {
	var v4, v6 []netip.Prefix
	for _, p := range dsts {
		if p.Addr().Is4() {
			v4 = append(v4, p)
		} else {
			v6 = append(v6, p)
		}
	}
	for _, fam := range []struct {
		bin  func() (string, error)
		dsts []netip.Prefix
	}{{iptables, v4}, {ip6tables, v6}} {
		ipt, err := fam.bin()
		if err != nil {
			if len(fam.dsts) == 0 {
				continue // nothing to block in a family whose tool is missing
			}
			return err
		}
		blockClear(ctx, ipt)
		if len(fam.dsts) == 0 {
			continue
		}
		if _, err := run(ctx, ipt, "-N", blockChain); err != nil {
			return err
		}
		for _, list := range chunkPrefixes(fam.dsts, 64, ",") {
			if _, err := run(ctx, ipt, "-A", blockChain, "-d", list, "-j", "REJECT"); err != nil {
				return err
			}
		}
		if _, err := run(ctx, ipt, "-I", "OUTPUT", "1", "-j", blockChain); err != nil {
			return err
		}
	}
	return nil
}
