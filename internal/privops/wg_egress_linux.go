package privops

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// CheckDefaultEgress checks the selected profile's native wg-quick settings.
// It reads the config with the caller's own privileges and changes no state.
func CheckDefaultEgress(profileType, path, tunnelName string) DefaultEgressCheck {
	if profileType != "wireguard" {
		return DefaultEgressCheck{Reason: "unsupported_profile"}
	}
	if path == "" {
		return DefaultEgressCheck{Reason: "config_unavailable"}
	}
	abs, err := filepath.Abs(path)
	if err != nil || ValidateConfigPath(abs, ".conf") != nil {
		return DefaultEgressCheck{Reason: "config_unavailable"}
	}
	if tunnelName == "" {
		tunnelName = strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	}
	if ValidateWGName(tunnelName) != nil {
		return DefaultEgressCheck{Reason: "invalid_config"}
	}
	data, _, err := readTrustedFile(selfCaller(), abs, MaxConfigSize, nil)
	if err != nil {
		return DefaultEgressCheck{Reason: "config_unavailable"}
	}
	defer clear(data)
	return inspectWireGuardDefaultEgress(data)
}

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
