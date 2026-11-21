package privops

import (
	"context"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func CheckDefaultEgress(profileType, path, tunnelName string) DefaultEgressCheck {
	return checkWireGuardDefaultEgress(profileType, path, tunnelName)
}

func prepareWireGuardDefaultEgress(data []byte) []byte { return data }

// wg-quick removes all main-table suppression rules on down. Allow an
// existing pair only if it belongs to the managed tunnel being replaced.
func checkNativeDefaultRules(ctx context.Context, replacing string) error {
	var mark int
	if replacing != "" {
		exe, err := wgTool()
		if err != nil {
			return err
		}
		if out, err := run(ctx, exe, "show", replacing, "fwmark"); err == nil {
			n, _ := strconv.ParseUint(strings.TrimSpace(string(out)), 0, 31)
			mark = int(n)
		}
	}
	for _, family := range []int{netlink.FAMILY_V4, netlink.FAMILY_V6} {
		rules, err := netlink.RuleList(family)
		if err != nil {
			return err
		}
		if conflictingNativeDefaultRules(rules, mark) {
			return ErrWGDefaultEgressConflict
		}
	}
	return nil
}

func conflictingNativeDefaultRules(rules []netlink.Rule, mark int) bool {
	suppress, own := 0, 0
	for _, rule := range rules {
		if rule.Table == unix.RT_TABLE_MAIN && rule.SuppressPrefixlen == 0 {
			suppress++
		}
		if mark != 0 && rule.Table == mark && rule.Mark == mark && rule.Invert {
			own++
		}
	}
	return suppress > 0 && (suppress != 1 || own != 1)
}
