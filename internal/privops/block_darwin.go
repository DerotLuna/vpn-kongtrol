//go:build darwin

package privops

import (
	"context"
	"net/netip"
)

// sysBlockSet loads the blocked set into the Kongtrol pf anchor
// ("com.kongtrol", see the kill switch). The daemon uses this op instead of
// the legacy KillSwitchEnable, so both never hold the anchor at once;
// clearing it is the kill switch's own clean-up (anchor flushed, system
// ruleset reloaded, pf token released).
func sysBlockSet(ctx context.Context, dsts []netip.Prefix) error {
	if len(dsts) == 0 {
		return sysKillSwitchDisable(ctx)
	}
	base, err := readPFConf()
	if err != nil {
		return err
	}
	if _, err := runInput(ctx, pfMainRuleset(base), pfctl, "-f", "-"); err != nil {
		return err
	}
	pfState.loadedMain = true
	if _, err := runInput(ctx, []byte(pfBlockRules(dsts)), pfctl, "-a", pfAnchor, "-f", "-"); err != nil {
		return err
	}
	if pfState.token == "" {
		out, err := run(ctx, pfctl, "-E")
		if err != nil {
			return err
		}
		tok, err := parsePFToken(out)
		if err != nil {
			return err
		}
		pfState.token = tok
	}
	return nil
}
