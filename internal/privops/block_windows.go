//go:build windows

package privops

import (
	"context"
	"net/netip"
)

// blockRuleName names every Windows Firewall rule of the blocked set (one
// rule per chunk of addresses; `delete rule name=` removes them all).
const blockRuleName = "KongtrolBlock"

// sysBlockSet replaces the outbound BLOCK rules. Block rules win over any
// allow rule in Windows Defender Firewall, so a listed destination is
// unreachable on every interface while the set is in place.
func sysBlockSet(ctx context.Context, dsts []netip.Prefix) error {
	_, _ = netsh(ctx, "advfirewall", "firewall", "delete", "rule", "name="+blockRuleName)
	for _, list := range chunkPrefixes(dsts, 200, ",") {
		if _, err := netsh(ctx, "advfirewall", "firewall", "add", "rule", "name="+blockRuleName,
			"dir=out", "action=block", "enable=yes", "profile=any", "remoteip="+list); err != nil {
			return err
		}
	}
	return nil
}
